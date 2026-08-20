#!/bin/sh
# Entrypoint compartido por los 4 binarios de la imagen (api, scheduler,
# seedadmin, seedinstagramtoken): aplica las migraciones pendientes y
# después ejecuta lo que se le haya pasado como comando. `migrate ... up`
# usa un advisory lock de Postgres, así que es seguro que dos contenedores
# (api y scheduler) lo corran al arrancar al mismo tiempo — el segundo
# espera a que el primero termine y no encuentra nada pendiente.
set -e

echo "entrypoint: aplicando migraciones..."
./migrate -path ./migrations -database "$DATABASE_URL" up

exec "$@"
