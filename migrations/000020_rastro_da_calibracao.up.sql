ALTER TABLE risk_gates
    ADD COLUMN calibration_status text NOT NULL DEFAULT 'desligada'
        CONSTRAINT risk_gates_calibration_status_check CHECK (calibration_status IN (
            'desligada', 'sem_motivo_a_calibrar', 'motivo_nao_calibravel', 'embedding_indisponivel',
            'historico_indisponivel', 'historico_insuficiente', 'vizinho_negativo', 'calibrada'
        )),
    ADD COLUMN calibration_reason text NOT NULL DEFAULT 'calibração desligada'
        CONSTRAINT risk_gates_calibration_reason_check CHECK (calibration_reason <> ''),
    ADD COLUMN calibration_neighbors jsonb NOT NULL DEFAULT '[]'
        CONSTRAINT risk_gates_calibration_neighbors_lista CHECK (jsonb_typeof(calibration_neighbors) = 'array');
