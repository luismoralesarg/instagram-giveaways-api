-- Regla acordada para UC-2.2: el webhook de mención de historia no trae en
-- el payload un identificador que permita desambiguar a qué campaña
-- corresponde si hay más de una campaña de tipo historia activa a la vez,
-- así que solo se permite una activa por vez.
--
-- Índice único parcial sobre `type`: como el filtro ya fija type = 'historia'
-- para toda fila que matchee, el valor indexado es constante entre esas
-- filas — por lo tanto el índice único permite, como mucho, una.
CREATE UNIQUE INDEX campaigns_single_active_story_idx ON campaigns (type) WHERE type = 'historia' AND status = 'activa';
