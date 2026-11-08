ALTER TABLE risk_gates
    ADD COLUMN rules_hit jsonb NOT NULL DEFAULT '[]' CONSTRAINT risk_gates_rules_hit_lista CHECK (jsonb_typeof(rules_hit) = 'array'),
    ADD COLUMN new_dependencies jsonb NOT NULL DEFAULT '[]' CONSTRAINT risk_gates_new_dependencies_lista CHECK (jsonb_typeof(new_dependencies) = 'array');
