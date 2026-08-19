CREATE TABLE campaigns (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    type TEXT NOT NULL CHECK (type IN ('post', 'historia')),
    media_id TEXT NOT NULL,
    name TEXT NOT NULL DEFAULT '',
    must_follow BOOLEAN NOT NULL DEFAULT false,
    min_mentions INTEGER NOT NULL DEFAULT 0,
    status TEXT NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'activa', 'cerrada', 'sorteada')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- UC-1.1: un media_id no puede tener más de una campaña draft/activa a la vez.
-- Índice único parcial en vez de constraint a nivel aplicación para que la regla
-- se sostenga incluso frente a inserts concurrentes.
CREATE UNIQUE INDEX campaigns_media_id_open_idx ON campaigns (media_id) WHERE status IN ('draft', 'activa');
