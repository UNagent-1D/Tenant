package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/UNagent-1D/Tenant/config"
	"github.com/UNagent-1D/Tenant/internal/auth"
	"github.com/UNagent-1D/Tenant/pkg/db"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// ─── Constants ────────────────────────────────────────────────────────────────

const (
	testSecret      = "super-secreto-cambiar-en-produccion"
	testAdminID     = "00000000-0000-0000-0000-000000000001"
	testTenantAID   = "00000000-0000-0000-0000-000000000010"
	testTenantBID   = "00000000-0000-0000-0000-000000000011"
	testProfileID   = "00000000-0000-0000-0000-000000000020"
	testConfigID    = "00000000-0000-0000-0000-000000000030"
	testChannelID   = "00000000-0000-0000-0000-000000000040"
	testDataSrcID   = "00000000-0000-0000-0000-000000000050"
	testToolID      = "00000000-0000-0000-0000-000000000060"
	testTenantASlug = "hospital-test-a"
	testTenantBSlug = "hospital-test-b"
)

// ─── Globals set in TestMain ──────────────────────────────────────────────────

var (
	testRouter      *gin.Engine
	appAdminToken   string
	tenantAAdminTok string
	tenantAOperTok  string
	tenantBAdminTok string
)

// ─── Setup ────────────────────────────────────────────────────────────────────

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)

	// Provide defaults for required env vars so tests don't panic
	if os.Getenv("INTERNAL_API_KEY") == "" {
		os.Setenv("INTERNAL_API_KEY", "test-internal-key")
	}

	cfg := config.Load()

	// Init DB only for integration tests
	if dbURL := os.Getenv("TEST_DATABASE_URL"); dbURL != "" {
		os.Setenv("DATABASE_URL", dbURL)
		cfg = config.Load()
		if err := db.Init(cfg.DatabaseURL); err == nil {
			seedDB()
		}
	}

	tA := testTenantAID
	tB := testTenantBID

	appAdminToken = makeToken(testAdminID, "app_admin", nil)
	tenantAAdminTok = makeToken(testAdminID, "tenant_admin", &tA)
	tenantAOperTok = makeToken(testAdminID, "tenant_operator", &tA)
	tenantBAdminTok = makeToken(testAdminID, "tenant_admin", &tB)

	testRouter = SetupRouter(cfg)
	os.Exit(m.Run())
}

// seedDB inserts minimal rows for integration tests.
func seedDB() {
	ctx := context.Background()
	db.Pool.Exec(ctx, `TRUNCATE users, tenants, tool_registry CASCADE`)

	hash, _ := bcrypt.GenerateFromPassword([]byte("Test1234!"), bcrypt.DefaultCost)
	db.Pool.Exec(ctx,
		`INSERT INTO users (id, email, password_hash, role) VALUES ($1,$2,$3,'app_admin')`,
		testAdminID, "admin@test.internal", string(hash),
	)
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func makeToken(userID, role string, tenantID *string) string {
	claims := &auth.Claims{
		UserID:   userID,
		Email:    "test@example.com",
		TenantID: tenantID,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	tok, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(testSecret))
	return "Bearer " + tok
}

func req(t *testing.T, method, path string, body any, token string) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		json.NewEncoder(&buf).Encode(body)
	}
	r := httptest.NewRequest(method, path, &buf)
	r.Header.Set("Content-Type", "application/json")
	if token != "" {
		r.Header.Set("Authorization", token)
	}
	w := httptest.NewRecorder()
	testRouter.ServeHTTP(w, r)
	return w
}

func assertStatus(t *testing.T, got, want int) {
	t.Helper()
	if got != want {
		t.Errorf("status: got %d, want %d", got, want)
	}
}

func skipIfNoDB(t *testing.T) {
	t.Helper()
	if db.Pool == nil {
		t.Skip("set TEST_DATABASE_URL to run integration tests")
	}
}

// ─── 1. Public routes ─────────────────────────────────────────────────────────

func TestHealth(t *testing.T) {
	skipIfNoDB(t)
	w := req(t, http.MethodGet, "/api/v1/health", nil, "")
	assertStatus(t, w.Code, http.StatusOK)
}

func TestLogin_BadPayload(t *testing.T) {
	cases := []struct {
		name string
		body any
	}{
		{"empty body", nil},
		{"missing password", map[string]string{"email": "a@b.com"}},
		{"missing email", map[string]string{"password": "pass"}},
		{"invalid email format", map[string]string{"email": "notanemail", "password": "pass"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := req(t, http.MethodPost, "/api/v1/auth/login", tc.body, "")
			assertStatus(t, w.Code, http.StatusBadRequest)
		})
	}
}

// ─── 2. Auth middleware — all protected routes return 401 without token ───────

func TestUnauthenticated(t *testing.T) {
	routes := []struct{ method, path string }{
		{http.MethodGet, "/api/v1/tenants"},
		{http.MethodPost, "/api/v1/tenants"},
		{http.MethodGet, "/api/v1/tenants/" + testTenantAID},
		{http.MethodPatch, "/api/v1/tenants/" + testTenantAID},
		{http.MethodGet, "/api/v1/tenants/" + testTenantAID + "/channels"},
		{http.MethodPost, "/api/v1/tenants/" + testTenantAID + "/channels"},
		{http.MethodGet, "/api/v1/tenants/" + testTenantAID + "/profiles"},
		{http.MethodPost, "/api/v1/tenants/" + testTenantAID + "/profiles"},
		{http.MethodGet, "/api/v1/tenants/" + testTenantAID + "/profiles/" + testProfileID + "/configs"},
		{http.MethodPost, "/api/v1/tenants/" + testTenantAID + "/profiles/" + testProfileID + "/configs"},
		{http.MethodGet, "/api/v1/tenants/" + testTenantAID + "/profiles/" + testProfileID + "/configs/active"},
		{http.MethodGet, "/api/v1/tenants/" + testTenantAID + "/data-sources"},
		{http.MethodPost, "/api/v1/tenants/" + testTenantAID + "/data-sources"},
		{http.MethodGet, "/api/v1/users"},
		{http.MethodPost, "/api/v1/users"},
		{http.MethodGet, "/api/v1/tool-registry"},
		{http.MethodPost, "/api/v1/tool-registry"},
	}
	for _, r := range routes {
		t.Run(r.method+" "+r.path, func(t *testing.T) {
			w := req(t, r.method, r.path, nil, "")
			assertStatus(t, w.Code, http.StatusUnauthorized)
		})
	}
}

func TestInvalidToken(t *testing.T) {
	w := req(t, http.MethodGet, "/api/v1/tenants", nil, "Bearer invalid.token.here")
	assertStatus(t, w.Code, http.StatusUnauthorized)
}

func TestExpiredToken(t *testing.T) {
	claims := &auth.Claims{
		UserID: testAdminID,
		Email:  "x@x.com",
		Role:   "app_admin",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Hour)),
		},
	}
	tok, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(testSecret))
	w := req(t, http.MethodGet, "/api/v1/tenants", nil, "Bearer "+tok)
	assertStatus(t, w.Code, http.StatusUnauthorized)
}

func TestMalformedAuthHeader(t *testing.T) {
	cases := []string{"nobearer", "Bearer", "Token abc"}
	for _, hdr := range cases {
		t.Run(hdr, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/api/v1/tenants", nil)
			r.Header.Set("Authorization", hdr)
			w := httptest.NewRecorder()
			testRouter.ServeHTTP(w, r)
			assertStatus(t, w.Code, http.StatusUnauthorized)
		})
	}
}

// ─── 3. Role enforcement — wrong role returns 403 ────────────────────────────

func TestRoleEnforcement(t *testing.T) {
	cases := []struct {
		name   string
		token  string
		method string
		path   string
	}{
		{"operator POST tenant", tenantAOperTok, http.MethodPost, "/api/v1/tenants"},
		{"operator POST user", tenantAOperTok, http.MethodPost, "/api/v1/users"},
		{"operator POST tool", tenantAOperTok, http.MethodPost, "/api/v1/tool-registry"},
		{"operator GET all tenants", tenantAOperTok, http.MethodGet, "/api/v1/tenants"},
		{"operator GET all users", tenantAOperTok, http.MethodGet, "/api/v1/users"},
		{"tenant_admin GET all tenants", tenantAAdminTok, http.MethodGet, "/api/v1/tenants"},
		{"tenant_admin POST tool", tenantAAdminTok, http.MethodPost, "/api/v1/tool-registry"},
		{"tenant_admin PATCH tool", tenantAAdminTok, http.MethodPatch, "/api/v1/tool-registry/" + testToolID},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := req(t, tc.method, tc.path, nil, tc.token)
			assertStatus(t, w.Code, http.StatusForbidden)
		})
	}
}

// ─── 4. Cross-tenant enforcement ─────────────────────────────────────────────

func TestCrossTenantEnforcement(t *testing.T) {
	routes := []struct{ method, path string }{
		{http.MethodGet, "/api/v1/tenants/" + testTenantAID},
		{http.MethodPatch, "/api/v1/tenants/" + testTenantAID},
		{http.MethodGet, "/api/v1/tenants/" + testTenantAID + "/channels"},
		{http.MethodPost, "/api/v1/tenants/" + testTenantAID + "/channels"},
		{http.MethodGet, "/api/v1/tenants/" + testTenantAID + "/profiles"},
		{http.MethodPost, "/api/v1/tenants/" + testTenantAID + "/profiles"},
		{http.MethodGet, "/api/v1/tenants/" + testTenantAID + "/profiles/" + testProfileID + "/configs"},
		{http.MethodPost, "/api/v1/tenants/" + testTenantAID + "/profiles/" + testProfileID + "/configs"},
		{http.MethodGet, "/api/v1/tenants/" + testTenantAID + "/data-sources"},
		{http.MethodPost, "/api/v1/tenants/" + testTenantAID + "/data-sources"},
	}
	for _, r := range routes {
		t.Run(r.method+" "+r.path, func(t *testing.T) {
			w := req(t, r.method, r.path, nil, tenantBAdminTok)
			assertStatus(t, w.Code, http.StatusForbidden)
		})
	}
}

// ─── 5. Request validation ────────────────────────────────────────────────────

func TestCreateTenant_Validation(t *testing.T) {
	cases := []struct {
		name string
		body any
	}{
		{"empty body", nil},
		{"missing slug", map[string]string{"name": "Hospital X", "plan": "free"}},
		{"missing name", map[string]string{"slug": "hospital-x", "plan": "free"}},
		{"missing plan", map[string]string{"slug": "hospital-x", "name": "Hospital X"}},
		{"invalid plan", map[string]string{"slug": "hospital-x", "name": "Hospital X", "plan": "gold"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := req(t, http.MethodPost, "/api/v1/tenants", tc.body, appAdminToken)
			assertStatus(t, w.Code, http.StatusBadRequest)
		})
	}
}

func TestCreateUser_Validation(t *testing.T) {
	cases := []struct {
		name string
		body any
	}{
		{"empty body", nil},
		{"missing email", map[string]string{"password": "pass123", "role": "tenant_operator"}},
		{"invalid email", map[string]string{"email": "bad", "password": "pass123", "role": "tenant_operator"}},
		{"missing password", map[string]string{"email": "a@b.com", "role": "tenant_operator"}},
		{"short password", map[string]string{"email": "a@b.com", "password": "12", "role": "tenant_operator"}},
		{"missing role", map[string]string{"email": "a@b.com", "password": "pass123"}},
		{"invalid role", map[string]string{"email": "a@b.com", "password": "pass123", "role": "superuser"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := req(t, http.MethodPost, "/api/v1/users", tc.body, appAdminToken)
			assertStatus(t, w.Code, http.StatusBadRequest)
		})
	}
}

func TestCreateChannel_Validation(t *testing.T) {
	skipIfNoDB(t)
	cases := []struct {
		name string
		body any
	}{
		{"empty body", nil},
		{"missing channel_type", map[string]string{"channel_key": "+573001234567"}},
		{"invalid channel_type", map[string]string{"channel_type": "telegram", "channel_key": "+573001234567"}},
		{"missing channel_key", map[string]string{"channel_type": "whatsapp"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := req(t, http.MethodPost, "/api/v1/tenants/"+testTenantAID+"/channels", tc.body, tenantAAdminTok)
			assertStatus(t, w.Code, http.StatusBadRequest)
		})
	}
}

func TestCreateDataSource_Validation(t *testing.T) {
	skipIfNoDB(t)
	cases := []struct {
		name string
		body any
	}{
		{"empty body", nil},
		{"missing name", map[string]any{"source_type": "scheduling", "base_url": "http://x.com", "route_configs": map[string]any{}}},
		{"invalid source_type", map[string]any{"name": "X", "source_type": "other", "base_url": "http://x.com", "route_configs": map[string]any{}}},
		{"missing base_url", map[string]any{"name": "X", "source_type": "scheduling", "route_configs": map[string]any{}}},
		{"missing route_configs", map[string]any{"name": "X", "source_type": "scheduling", "base_url": "http://x.com"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := req(t, http.MethodPost, "/api/v1/tenants/"+testTenantAID+"/data-sources", tc.body, tenantAAdminTok)
			assertStatus(t, w.Code, http.StatusBadRequest)
		})
	}
}

func TestCreateAgentConfig_Validation(t *testing.T) {
	skipIfNoDB(t)
	path := "/api/v1/tenants/" + testTenantAID + "/profiles/" + testProfileID + "/configs"
	cases := []struct {
		name string
		body any
	}{
		{"empty body", nil},
		{"missing conversation_policy", map[string]any{
			"escalation_rules": []map[string]string{{"c": "x"}},
			"tool_permissions": []string{},
			"llm_params":       map[string]any{"model": "gpt-4o", "temperature": 0.3, "max_tokens": 100},
		}},
		{"missing escalation_rules", map[string]any{
			"conversation_policy": map[string]any{},
			"tool_permissions":    []string{},
			"llm_params":          map[string]any{"model": "gpt-4o", "temperature": 0.3, "max_tokens": 100},
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := req(t, http.MethodPost, path, tc.body, tenantAAdminTok)
			assertStatus(t, w.Code, http.StatusBadRequest)
		})
	}
}

// ─── 6. LLM params validation ─────────────────────────────────────────────────

func TestLLMParamsValidation(t *testing.T) {
	skipIfNoDB(t)
	path := "/api/v1/tenants/" + testTenantAID + "/profiles/" + testProfileID + "/configs"

	makeBody := func(model string, temp float64, maxTokens int) map[string]any {
		return map[string]any{
			"conversation_policy": map[string]any{},
			"escalation_rules":    []map[string]string{{"c": "x"}},
			"tool_permissions":    []string{},
			"llm_params": map[string]any{
				"model":       model,
				"temperature": temp,
				"max_tokens":  maxTokens,
			},
		}
	}

	cases := []struct {
		name string
		body map[string]any
		want int
	}{
		{"unknown model", makeBody("claude-3", 0.5, 100), http.StatusBadRequest},
		{"temperature below 0", makeBody("gpt-4o", -0.1, 100), http.StatusBadRequest},
		{"temperature above 2", makeBody("gpt-4o", 2.1, 100), http.StatusBadRequest},
		{"max_tokens 0", makeBody("gpt-4o", 0.5, 0), http.StatusBadRequest},
		{"max_tokens above 4096", makeBody("gpt-4o", 0.5, 5000), http.StatusBadRequest},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := req(t, http.MethodPost, path, tc.body, tenantAAdminTok)
			assertStatus(t, w.Code, tc.want)
		})
	}
}

func TestEscalationRulesValidation(t *testing.T) {
	skipIfNoDB(t)
	path := "/api/v1/tenants/" + testTenantAID + "/profiles/" + testProfileID + "/configs"
	body := map[string]any{
		"conversation_policy": map[string]any{},
		"escalation_rules":    []any{},
		"tool_permissions":    []string{},
		"llm_params":          map[string]any{"model": "gpt-4o", "temperature": 0.5, "max_tokens": 512},
	}
	w := req(t, http.MethodPost, path, body, tenantAAdminTok)
	assertStatus(t, w.Code, http.StatusBadRequest)
}

// ─── 7. Operator read-only scope ──────────────────────────────────────────────

func TestOperatorReadOnly(t *testing.T) {
	skipIfNoDB(t)
	reads := []string{
		"/api/v1/tenants/" + testTenantAID + "/profiles",
		"/api/v1/tenants/" + testTenantAID + "/profiles/" + testProfileID + "/configs",
		"/api/v1/tenants/" + testTenantAID + "/profiles/" + testProfileID + "/configs/active",
	}
	for _, path := range reads {
		t.Run(path, func(t *testing.T) {
			w := req(t, http.MethodGet, path, nil, tenantAOperTok)
			if w.Code == http.StatusForbidden {
				t.Errorf("tenant_operator must be able to read %s, got 403", path)
			}
		})
	}

	writes := []struct{ method, path string }{
		{http.MethodPost, "/api/v1/tenants/" + testTenantAID + "/channels"},
		{http.MethodPost, "/api/v1/tenants/" + testTenantAID + "/profiles"},
		{http.MethodPost, "/api/v1/tenants/" + testTenantAID + "/profiles/" + testProfileID + "/configs"},
		{http.MethodPost, "/api/v1/tenants/" + testTenantAID + "/data-sources"},
	}
	for _, r := range writes {
		t.Run("DENY "+r.method+" "+r.path, func(t *testing.T) {
			w := req(t, r.method, r.path, nil, tenantAOperTok)
			assertStatus(t, w.Code, http.StatusForbidden)
		})
	}
}

// ─── 9. Integration tests (require TEST_DATABASE_URL) ────────────────────────

func TestIntegration_Login(t *testing.T) {
	skipIfNoDB(t)
	t.Run("success", func(t *testing.T) {
		w := req(t, http.MethodPost, "/api/v1/auth/login",
			map[string]string{"email": "admin@test.internal", "password": "Test1234!"}, "")
		assertStatus(t, w.Code, http.StatusOK)
		var body map[string]string
		json.NewDecoder(w.Body).Decode(&body)
		if body["token"] == "" {
			t.Error("expected token in response")
		}
	})
	t.Run("wrong password", func(t *testing.T) {
		w := req(t, http.MethodPost, "/api/v1/auth/login",
			map[string]string{"email": "admin@test.internal", "password": "wrong"}, "")
		assertStatus(t, w.Code, http.StatusUnauthorized)
	})
	t.Run("unknown user", func(t *testing.T) {
		w := req(t, http.MethodPost, "/api/v1/auth/login",
			map[string]string{"email": "nobody@test.internal", "password": "pass"}, "")
		assertStatus(t, w.Code, http.StatusUnauthorized)
	})
}

func TestIntegration_TenantCRUD(t *testing.T) {
	skipIfNoDB(t)

	var tenantID string

	t.Run("create tenant", func(t *testing.T) {
		w := req(t, http.MethodPost, "/api/v1/tenants", map[string]any{
			"slug": "integration-test-tenant",
			"name": "Integration Test Hospital",
			"plan": "starter",
		}, appAdminToken)
		assertStatus(t, w.Code, http.StatusCreated)

		var body map[string]any
		json.NewDecoder(w.Body).Decode(&body)
		id, ok := body["id"].(string)
		if !ok || id == "" {
			t.Fatal("expected id in response")
		}
		tenantID = id
	})

	t.Run("duplicate slug", func(t *testing.T) {
		w := req(t, http.MethodPost, "/api/v1/tenants", map[string]any{
			"slug": "integration-test-tenant",
			"name": "Another Hospital",
			"plan": "free",
		}, appAdminToken)
		assertStatus(t, w.Code, http.StatusConflict)
	})

	t.Run("get tenant", func(t *testing.T) {
		if tenantID == "" {
			t.Skip("tenant not created")
		}
		w := req(t, http.MethodGet, "/api/v1/tenants/"+tenantID, nil, appAdminToken)
		assertStatus(t, w.Code, http.StatusOK)
	})

	t.Run("get tenant not found", func(t *testing.T) {
		w := req(t, http.MethodGet, "/api/v1/tenants/00000000-0000-0000-0000-999999999999", nil, appAdminToken)
		assertStatus(t, w.Code, http.StatusNotFound)
	})

	t.Run("update tenant", func(t *testing.T) {
		if tenantID == "" {
			t.Skip("tenant not created")
		}
		w := req(t, http.MethodPatch, "/api/v1/tenants/"+tenantID,
			map[string]string{"status": "suspended"}, appAdminToken)
		assertStatus(t, w.Code, http.StatusOK)
	})

	t.Run("update tenant no fields", func(t *testing.T) {
		if tenantID == "" {
			t.Skip("tenant not created")
		}
		w := req(t, http.MethodPatch, "/api/v1/tenants/"+tenantID, map[string]any{}, appAdminToken)
		assertStatus(t, w.Code, http.StatusBadRequest)
	})
}

func TestIntegration_UserCRUD(t *testing.T) {
	skipIfNoDB(t)

	wt := req(t, http.MethodPost, "/api/v1/tenants", map[string]any{
		"slug": "user-test-tenant",
		"name": "User Test Hospital",
		"plan": "free",
	}, appAdminToken)
	var tenantResp map[string]any
	json.NewDecoder(wt.Body).Decode(&tenantResp)
	tenantID, _ := tenantResp["id"].(string)

	var userID string

	t.Run("create tenant_admin", func(t *testing.T) {
		w := req(t, http.MethodPost, "/api/v1/users", map[string]any{
			"email":     "tadmin@user-test.internal",
			"password":  "Pass1234!",
			"role":      "tenant_admin",
			"tenant_id": tenantID,
		}, appAdminToken)
		assertStatus(t, w.Code, http.StatusCreated)
		var body map[string]any
		json.NewDecoder(w.Body).Decode(&body)
		userID, _ = body["id"].(string)
	})

	t.Run("duplicate email", func(t *testing.T) {
		w := req(t, http.MethodPost, "/api/v1/users", map[string]any{
			"email":     "tadmin@user-test.internal",
			"password":  "Pass1234!",
			"role":      "tenant_admin",
			"tenant_id": tenantID,
		}, appAdminToken)
		assertStatus(t, w.Code, http.StatusConflict)
	})

	t.Run("app_admin creates app_admin without tenant_id", func(t *testing.T) {
		w := req(t, http.MethodPost, "/api/v1/users", map[string]any{
			"email":    "admin2@test.internal",
			"password": "Pass1234!",
			"role":     "app_admin",
		}, appAdminToken)
		assertStatus(t, w.Code, http.StatusCreated)
	})

	t.Run("app_admin creates tenant_admin without tenant_id → 400", func(t *testing.T) {
		w := req(t, http.MethodPost, "/api/v1/users", map[string]any{
			"email":    "ta2@test.internal",
			"password": "Pass1234!",
			"role":     "tenant_admin",
		}, appAdminToken)
		assertStatus(t, w.Code, http.StatusBadRequest)
	})

	t.Run("deactivate user", func(t *testing.T) {
		if userID == "" {
			t.Skip("user not created")
		}
		inactive := false
		w := req(t, http.MethodPatch, "/api/v1/users/"+userID,
			map[string]any{"is_active": &inactive}, appAdminToken)
		assertStatus(t, w.Code, http.StatusOK)
	})
}

func TestIntegration_ChannelCRUD(t *testing.T) {
	skipIfNoDB(t)

	wt := req(t, http.MethodPost, "/api/v1/tenants", map[string]any{
		"slug": "channel-test-tenant",
		"name": "Channel Test Hospital",
		"plan": "free",
	}, appAdminToken)
	var tenantResp map[string]any
	json.NewDecoder(wt.Body).Decode(&tenantResp)
	tenantID, _ := tenantResp["id"].(string)

	var channelID string

	t.Run("create channel", func(t *testing.T) {
		w := req(t, http.MethodPost, "/api/v1/tenants/"+tenantID+"/channels", map[string]any{
			"channel_type": "whatsapp",
			"channel_key":  "+573001234567",
		}, appAdminToken)
		assertStatus(t, w.Code, http.StatusCreated)
		var body map[string]any
		json.NewDecoder(w.Body).Decode(&body)
		channelID, _ = body["id"].(string)
	})

	t.Run("list channels", func(t *testing.T) {
		w := req(t, http.MethodGet, "/api/v1/tenants/"+tenantID+"/channels", nil, appAdminToken)
		assertStatus(t, w.Code, http.StatusOK)
	})

	t.Run("update channel", func(t *testing.T) {
		if channelID == "" {
			t.Skip("channel not created")
		}
		active := false
		w := req(t, http.MethodPatch, "/api/v1/tenants/"+tenantID+"/channels/"+channelID,
			map[string]any{"is_active": &active}, appAdminToken)
		assertStatus(t, w.Code, http.StatusOK)
	})
}

func TestIntegration_ProfileAndACRFlow(t *testing.T) {
	skipIfNoDB(t)

	wt := req(t, http.MethodPost, "/api/v1/tenants", map[string]any{
		"slug": "acr-test-tenant",
		"name": "ACR Test Hospital",
		"plan": "pro",
	}, appAdminToken)
	var tenantResp map[string]any
	json.NewDecoder(wt.Body).Decode(&tenantResp)
	tenantID, _ := tenantResp["id"].(string)

	var profileID, configID string

	t.Run("create profile", func(t *testing.T) {
		w := req(t, http.MethodPost, "/api/v1/tenants/"+tenantID+"/profiles", map[string]any{
			"name":                "Scheduling Bot",
			"description":         "Books appointments",
			"allowed_specialties": []string{"cardiology", "general"},
			"allowed_locations":   []string{"bogota"},
		}, appAdminToken)
		assertStatus(t, w.Code, http.StatusCreated)
		var body map[string]any
		json.NewDecoder(w.Body).Decode(&body)
		profileID, _ = body["id"].(string)
	})

	t.Run("list profiles", func(t *testing.T) {
		w := req(t, http.MethodGet, "/api/v1/tenants/"+tenantID+"/profiles", nil, appAdminToken)
		assertStatus(t, w.Code, http.StatusOK)
	})

	t.Run("create draft config", func(t *testing.T) {
		if profileID == "" {
			t.Skip("profile not created")
		}
		w := req(t, http.MethodPost,
			"/api/v1/tenants/"+tenantID+"/profiles/"+profileID+"/configs",
			map[string]any{
				"conversation_policy": map[string]any{"steps": []string{"greet", "ask_specialty"}},
				"escalation_rules":    []map[string]any{{"condition": "user_frustrated", "after_turns": 3}},
				"tool_permissions":    []map[string]any{{"tool_name": "list_doctors", "constraints": map[string]any{}}},
				"llm_params": map[string]any{
					"model":         "gpt-4o",
					"temperature":   0.3,
					"max_tokens":    1024,
					"system_prompt": "You are a scheduling assistant.",
				},
			}, appAdminToken)
		assertStatus(t, w.Code, http.StatusCreated)
		var body map[string]any
		json.NewDecoder(w.Body).Decode(&body)
		configID, _ = body["id"].(string)
		status, _ := body["status"].(string)
		if status != "draft" {
			t.Errorf("new config must be draft, got %q", status)
		}
	})

	t.Run("no active config yet", func(t *testing.T) {
		if profileID == "" {
			t.Skip("profile not created")
		}
		w := req(t, http.MethodGet,
			"/api/v1/tenants/"+tenantID+"/profiles/"+profileID+"/configs/active",
			nil, appAdminToken)
		assertStatus(t, w.Code, http.StatusNotFound)
	})

	t.Run("activate config", func(t *testing.T) {
		if configID == "" {
			t.Skip("config not created")
		}
		w := req(t, http.MethodPost,
			"/api/v1/tenants/"+tenantID+"/profiles/"+profileID+"/configs/"+configID+"/activate",
			nil, appAdminToken)
		assertStatus(t, w.Code, http.StatusOK)
		var body map[string]any
		json.NewDecoder(w.Body).Decode(&body)
		if body["status"] != "active" {
			t.Errorf("config must be active after activation, got %v", body["status"])
		}
	})

	t.Run("active config is now retrievable", func(t *testing.T) {
		if configID == "" {
			t.Skip("config not created")
		}
		w := req(t, http.MethodGet,
			"/api/v1/tenants/"+tenantID+"/profiles/"+profileID+"/configs/active",
			nil, appAdminToken)
		assertStatus(t, w.Code, http.StatusOK)
	})

	t.Run("cannot activate a non-draft config", func(t *testing.T) {
		if configID == "" {
			t.Skip("config not created")
		}
		w := req(t, http.MethodPost,
			"/api/v1/tenants/"+tenantID+"/profiles/"+profileID+"/configs/"+configID+"/activate",
			nil, appAdminToken)
		assertStatus(t, w.Code, http.StatusNotFound)
	})
}

func TestIntegration_DataSourceCRUD(t *testing.T) {
	skipIfNoDB(t)

	wt := req(t, http.MethodPost, "/api/v1/tenants", map[string]any{
		"slug": "ds-test-tenant",
		"name": "DS Test Hospital",
		"plan": "free",
	}, appAdminToken)
	var tenantResp map[string]any
	json.NewDecoder(wt.Body).Decode(&tenantResp)
	tenantID, _ := tenantResp["id"].(string)

	var dsID string

	t.Run("create data source", func(t *testing.T) {
		w := req(t, http.MethodPost, "/api/v1/tenants/"+tenantID+"/data-sources", map[string]any{
			"name":        "Hospital Mock API",
			"source_type": "scheduling",
			"base_url":    "https://mock-hospital.internal",
			"route_configs": map[string]any{
				"list_doctors":     map[string]string{"method": "GET", "path": "/doctors"},
				"book_appointment": map[string]string{"method": "POST", "path": "/appointments"},
			},
		}, appAdminToken)
		assertStatus(t, w.Code, http.StatusCreated)
		var body map[string]any
		json.NewDecoder(w.Body).Decode(&body)
		dsID, _ = body["id"].(string)
	})

	t.Run("list data sources", func(t *testing.T) {
		w := req(t, http.MethodGet, "/api/v1/tenants/"+tenantID+"/data-sources", nil, appAdminToken)
		assertStatus(t, w.Code, http.StatusOK)
	})

	t.Run("update route_configs", func(t *testing.T) {
		if dsID == "" {
			t.Skip("data source not created")
		}
		w := req(t, http.MethodPatch, "/api/v1/tenants/"+tenantID+"/data-sources/"+dsID, map[string]any{
			"route_configs": map[string]any{
				"list_doctors":       map[string]string{"method": "GET", "path": "/doctors"},
				"book_appointment":   map[string]string{"method": "POST", "path": "/appointments"},
				"cancel_appointment": map[string]string{"method": "DELETE", "path": "/appointments/{id}"},
			},
		}, appAdminToken)
		assertStatus(t, w.Code, http.StatusOK)
	})
}

func TestIntegration_ToolRegistry(t *testing.T) {
	skipIfNoDB(t)

	var toolID string

	t.Run("create tool", func(t *testing.T) {
		w := req(t, http.MethodPost, "/api/v1/tool-registry", map[string]any{
			"name":        "list_doctors",
			"description": "Returns available doctors",
			"openai_function_def": map[string]any{
				"name":        "list_doctors",
				"description": "Get list of available doctors",
				"parameters":  map[string]any{"type": "object", "properties": map[string]any{}},
			},
		}, appAdminToken)
		assertStatus(t, w.Code, http.StatusCreated)
		var body map[string]any
		json.NewDecoder(w.Body).Decode(&body)
		toolID, _ = body["id"].(string)
	})

	t.Run("duplicate name → 409", func(t *testing.T) {
		w := req(t, http.MethodPost, "/api/v1/tool-registry", map[string]any{
			"name":                "list_doctors",
			"openai_function_def": map[string]any{"name": "list_doctors"},
		}, appAdminToken)
		assertStatus(t, w.Code, http.StatusConflict)
	})

	t.Run("list tools", func(t *testing.T) {
		w := req(t, http.MethodGet, "/api/v1/tool-registry", nil, appAdminToken)
		assertStatus(t, w.Code, http.StatusOK)
	})

	t.Run("deactivate tool", func(t *testing.T) {
		if toolID == "" {
			t.Skip("tool not created")
		}
		inactive := false
		w := req(t, http.MethodPatch, "/api/v1/tool-registry/"+toolID,
			map[string]any{"is_active": &inactive}, appAdminToken)
		assertStatus(t, w.Code, http.StatusOK)
	})
}
