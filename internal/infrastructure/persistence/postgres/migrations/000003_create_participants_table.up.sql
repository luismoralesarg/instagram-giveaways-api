CREATE TABLE participants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    campaign_id UUID NOT NULL REFERENCES campaigns(id),
    instagram_user_id TEXT NOT NULL,
    username TEXT NOT NULL DEFAULT '',
    source_type TEXT NOT NULL CHECK (source_type IN ('comment', 'story_mention')),
    is_excluded BOOLEAN NOT NULL DEFAULT false,
    excluded_reason TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- UC-2.1/UC-2.2: un instagram_user_id no puede generar más de un
-- Participant en la misma campaña.
CREATE UNIQUE INDEX participants_campaign_instagram_user_idx ON participants (campaign_id, instagram_user_id);
