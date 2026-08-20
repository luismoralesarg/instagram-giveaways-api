CREATE TABLE draws (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    campaign_id UUID NOT NULL REFERENCES campaigns(id),
    winners_count INTEGER NOT NULL CHECK (winners_count IN (1, 2, 3)),
    random_seed BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Snapshot de instagram_user_id/username al momento del sorteo (decisión
-- confirmada): si el participante cambia de username después, el
-- resultado histórico anunciado no cambia.
CREATE TABLE winners (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    draw_id UUID NOT NULL REFERENCES draws(id),
    participant_id UUID NOT NULL REFERENCES participants(id),
    instagram_user_id TEXT NOT NULL,
    username TEXT NOT NULL DEFAULT '',
    position INTEGER NOT NULL
);

CREATE INDEX winners_draw_id_idx ON winners (draw_id);
