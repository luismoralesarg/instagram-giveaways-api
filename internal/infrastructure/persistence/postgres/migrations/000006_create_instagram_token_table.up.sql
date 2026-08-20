-- Tabla de una sola fila: no hay soporte multi-cuenta (CLAUDE.md), así que
-- existe como mucho un InstagramToken. id fijo en 1 + CHECK fuerza el
-- singleton a nivel de base de datos.
CREATE TABLE instagram_token (
    id SMALLINT PRIMARY KEY DEFAULT 1 CHECK (id = 1),
    access_token TEXT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
