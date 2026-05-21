-- Bind an active agent_config to the data source whose route_configs the
-- LLM tools should hit via /api/v1/internal/tenants/:id/data-sources/:did/execute.
-- Nullable so legacy configs (created before this migration) still resolve.
ALTER TABLE agent_configs
    ADD COLUMN data_source_id UUID REFERENCES data_sources(id) ON DELETE SET NULL;
