ALTER TABLE agent_profiles DROP CONSTRAINT IF EXISTS fk_agent_profiles_active_config;
DROP TABLE IF EXISTS agent_configs;
DROP TYPE IF EXISTS config_status;
