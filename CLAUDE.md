# CLAUDE.md

Este archivo le da contexto a Claude Code para trabajar en este repo.

## Qué es este proyecto

API en Go para gestionar sorteos de Instagram (campañas basadas en un post o historia, captura de participantes, y ejecución de sorteos). Ver `ARCHITECTURE.md` para el diseño completo — **leerlo siempre antes de tocar código**, es la fuente de verdad sobre entidades, ciclo de vida y estructura de carpetas.

## Stack y convenciones

- **Go + Fiber** para el HTTP layer.
- **PostgreSQL** como persistencia.
- **Arquitectura hexagonal**: `domain` no depende de nada externo; `application` (casos de uso) depende solo de interfaces (`ports`) definidas en `domain`; `infrastructure` implementa esas interfaces. No romper esta dirección de dependencias.
- Un caso de uso = un archivo en `/internal/application/usecase`, con un único método público (`Execute` o similar).
- Entidades del dominio son structs simples en `/internal/domain`, con sus propios métodos de validación de transición de estado (ej. `Campaign.Activate() error`, que valida que esté en `draft` antes de pasar a `activa`).
- **Autenticación**: usuario/contraseña vía `POST /auth/login`, que devuelve un JWT. Todas las rutas de administración requieren `Authorization: Bearer <token>`, salvo `/auth/login` y `/webhooks/instagram` (este último se valida con la firma de Instagram, no con JWT). Los usuarios se dan de alta manualmente (seed/migración) — no hay endpoint de registro.

## Reglas de negocio no negociables (ver ARCHITECTURE.md §2 y §3)

- Un `Draw` no puede ejecutarse si la `Campaign` no está en estado `cerrada`.
- Un `Draw` siempre persiste la `random_seed` usada — el sorteo debe ser reproducible.
- Los participantes excluidos manualmente **no se borran**, se marcan (`is_excluded`) y se ignoran en el sorteo.
- No existe forma de recuperar menciones de historia después del hecho — cualquier código que dependa del webhook de historias debe asumir que un evento perdido es un evento perdido para siempre (no hay reintento posible contra la API de Instagram).
- No implementar soporte multi-cliente/multi-cuenta — está fuera de alcance a propósito.

## Comandos

```bash
go run ./cmd/api              # levantar la API localmente
go test ./...                 # correr todos los tests
go build ./...                # compilar

# Migraciones (golang-migrate, instalar con:
# go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@v4.19.1)
migrate -path internal/infrastructure/persistence/postgres/migrations -database "$DATABASE_URL" up

# Alta manual de un administrador (no hay endpoint de registro, ver "Autenticación" arriba)
go run ./cmd/seedadmin -username=admin -password=algo-seguro

# Alta manual del primer InstagramToken (ver ARCHITECTURE.md §3.1) — a partir de
# acá, cmd/scheduler lo mantiene fresco automáticamente
go run ./cmd/seedinstagramtoken -token=EAAB... [-expires-in-days=60]

# Módulo de tareas programadas (cron). Sin flags: arranca y corre en background
# (bloquea). Con -run-once: corre ese job ya y sale.
go run ./cmd/scheduler
go run ./cmd/scheduler -run-once=refresh-instagram-token

# Deploy (ver Dockerfile, docker-compose.prod.yml, .env.example — copiar a .env y completar)
docker compose -f docker-compose.prod.yml up --build -d
docker compose -f docker-compose.prod.yml --profile tools run --rm seedadmin -username=admin -password=algo-seguro
docker compose -f docker-compose.prod.yml --profile tools run --rm seedinstagramtoken -token=EAAB...
```

(Ajustar esta sección a medida que se agregue un Makefile.)

## Variables de entorno esperadas

```
DATABASE_URL=
INSTAGRAM_APP_ID=
INSTAGRAM_APP_SECRET=
INSTAGRAM_WEBHOOK_VERIFY_TOKEN=
JWT_SECRET=
PORT=
```

Nota: el access token de Instagram en sí **no** es una variable de entorno — vive en Postgres (tabla `instagram_token`) desde que existe `cmd/scheduler`. `INSTAGRAM_APP_ID` e `INSTAGRAM_APP_SECRET` son los que identifican la app de Meta para poder refrescarlo (ver ARCHITECTURE.md §3.1).

## Al implementar un caso de uso nuevo

1. Definir/ajustar la entidad y las interfaces necesarias en `domain`.
2. Escribir el caso de uso en `application/usecase`, contra las interfaces (no contra implementaciones concretas).
3. Implementar o ajustar el adapter correspondiente en `infrastructure`.
4. Conectar todo en `cmd/api/main.go`.
5. Agregar el handler HTTP en `infrastructure/http/fiber` si corresponde.

## Qué preguntar antes de asumir

- Si una tarea toca las reglas de validación de participantes o el algoritmo de sorteo, confirmar con el usuario antes de cambiar el comportamiento — son reglas de negocio explícitamente definidas, no detalles de implementación libres.
