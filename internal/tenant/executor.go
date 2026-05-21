package tenant

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/UNagent-1D/Tenant/config"
	"github.com/UNagent-1D/Tenant/pkg/db"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

func ExecuteDataSource(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID := c.Param("id")
		dataSourceID := c.Param("did")

		var req ExecuteRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Resolve tenant — check it exists and is active
		var slug, status string
		err := db.Pool.QueryRow(c.Request.Context(),
			"SELECT slug, status FROM tenants WHERE id = $1", tenantID,
		).Scan(&slug, &status)
		if err != nil {
			if err == pgx.ErrNoRows {
				c.JSON(http.StatusNotFound, gin.H{"error": "tenant not found"})
				return
			}
			internalError(c)
			return
		}
		if status != "active" {
			c.JSON(http.StatusNotFound, gin.H{"error": "tenant is not active"})
			return
		}

		// Quote the schema identifier so slugs with hyphens
		// (e.g. "demo-hospital") parse as a single identifier.
		schema := fmt.Sprintf("%q", "tenant_"+slug)

		// Fetch data source
		var ds DataSource
		err = db.Pool.QueryRow(c.Request.Context(),
			fmt.Sprintf(`SELECT id, base_url, credential_ref, route_configs
			             FROM %s.data_sources WHERE id = $1 AND is_active = true`, schema),
			dataSourceID,
		).Scan(&ds.ID, &ds.BaseURL, &ds.CredentialRef, &ds.RouteConfigs)
		if err != nil {
			if err == pgx.ErrNoRows {
				c.JSON(http.StatusNotFound, gin.H{"error": "data source not found"})
				return
			}
			internalError(c)
			return
		}

		// Decode route configs and find the operation
		var routes map[string]struct {
			Method string `json:"method"`
			Path   string `json:"path"`
		}
		if err := json.Unmarshal(ds.RouteConfigs, &routes); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid route_configs", "request_id": c.GetString("request_id")})
			return
		}
		route, ok := routes[req.Operation]
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("operation %q not found in route_configs", req.Operation)})
			return
		}

		// Resolve path template {key} → path_params[key]
		resolvedPath, err := resolvePath(route.Path, req.PathParams)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Build full URL with query params
		fullURL := ds.BaseURL + resolvedPath
		if len(req.QueryParams) > 0 {
			q := url.Values{}
			for k, v := range req.QueryParams {
				q.Set(k, v)
			}
			fullURL += "?" + q.Encode()
		}

		// Build HTTP request
		resp, err := doExternalRequest(c.Request.Context(), cfg, route.Method, fullURL, req.Body, ds.CredentialRef)
		if err != nil {
			if strings.Contains(err.Error(), "timeout") || strings.Contains(err.Error(), "deadline") {
				c.JSON(http.StatusGatewayTimeout, gin.H{"error": "external system did not respond in time"})
				return
			}
			c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("failed to connect to external system: %s", err.Error())})
			return
		}
		defer resp.Body.Close()

		bodyBytes, _ := io.ReadAll(resp.Body)

		headers := map[string]string{}
		for k, v := range resp.Header {
			if len(v) > 0 {
				headers[k] = v[0]
			}
		}

		var bodyJSON any
		if err := json.Unmarshal(bodyBytes, &bodyJSON); err != nil {
			bodyJSON = string(bodyBytes)
		}

		c.JSON(http.StatusOK, ExecuteResponse{
			StatusCode: resp.StatusCode,
			Headers:    headers,
			Body:       bodyJSON,
		})
	}
}

func resolvePath(template string, pathParams map[string]string) (string, error) {
	result := template
	for strings.Contains(result, "{") {
		start := strings.Index(result, "{")
		end := strings.Index(result, "}")
		if end < start {
			break
		}
		key := result[start+1 : end]
		val, ok := pathParams[key]
		if !ok {
			return "", fmt.Errorf("missing path_param: %s", key)
		}
		result = result[:start] + val + result[end+1:]
	}
	return result, nil
}

func doExternalRequest(ctx context.Context, cfg *config.Config, method, fullURL string, body map[string]any, credentialRef *string) (*http.Response, error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, cfg.HTTPTimeout)
	defer cancel()

	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(timeoutCtx, method, fullURL, bodyReader)
	if err != nil {
		return nil, err
	}

	if credentialRef != nil {
		req.Header.Set("X-API-Key", *credentialRef)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return http.DefaultClient.Do(req)
}
