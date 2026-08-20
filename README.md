# API de Sorteos de Instagram

API en Go para gestionar sorteos de Instagram: reemplaza la recopilación manual de participantes (en Excel) por un flujo automatizado — captura participantes vía la Graph API y webhooks de Instagram, los asocia a una campaña, y ejecuta el sorteo de ganadores de forma reproducible y auditable.

## Qué permite

- **Gestionar campañas** de sorteo asociadas a un post o una historia de Instagram, con un ciclo de vida controlado (`draft → activa → cerrada → sorteada`) y reglas de participación propias por campaña.
- **Capturar participantes automáticamente**: sincronizando comentarios de un post contra la Graph API, o en tiempo real vía webhook para menciones en historias.
- **Excluir participantes manualmente** (cuenta falsa, empleado, etc.) sin borrarlos — quedan marcados para trazabilidad, pero el sorteo los ignora.
- **Ejecutar el sorteo**: 1, 2 o 3 ganadores al azar entre los elegibles, con la semilla aleatoria siempre persistida para que el resultado sea reproducible y demostrable ante un reclamo. Soporta re-sorteos sin perder el historial.
- **Refrescar solo** el long-lived access token de Instagram (expira cada ~60 días) vía un módulo de tareas programadas.

**Fuera de alcance, a propósito:** soporte multi-cliente/multi-cuenta (gestiona una única cuenta de Instagram Business) y notificación automática al ganador por DM.

## Stack

- Go + [Fiber](https://gofiber.io/)
- PostgreSQL ([pgx](https://github.com/jackc/pgx) v5)
- Instagram Graph API (Meta for Developers)
- JWT (autenticación) + bcrypt
- [robfig/cron](https://github.com/robfig/cron) (módulo de tareas programadas)
- Arquitectura hexagonal (puertos y adaptadores)

## Estructura del proyecto

```
cmd/
  api/                  → servidor HTTP (Fiber)
  scheduler/             → módulo de tareas programadas (refresh del token de Instagram)
  seedadmin/              → alta manual de administradores
  seedinstagramtoken/      → alta manual del primer access token de Instagram

internal/
  domain/                → entidades y reglas de negocio, sin dependencias externas
  application/usecase/    → casos de uso, contra interfaces de domain
  infrastructure/         → Postgres, Fiber, Graph API, JWT, bcrypt, cron
  config/                 → carga de variables de entorno
```

Ver [`ARCHITECTURE.md`](./ARCHITECTURE.md) para el diseño completo.

## Requisitos

- Go 1.25+
- PostgreSQL 16+
- (Opcional) Docker + Docker Compose para correrlo empaquetado

## Uso rápido (desarrollo local)

```bash
# 1. Variables de entorno mínimas
export DATABASE_URL="postgres://user:pass@localhost:5432/giveaways?sslmode=disable"
export JWT_SECRET="algo-largo-y-random"

# 2. Migraciones
migrate -path internal/infrastructure/persistence/postgres/migrations -database "$DATABASE_URL" up

# 3. Alta del primer administrador (no hay endpoint de registro)
go run ./cmd/seedadmin -username=admin -password=algo-seguro

# 4. Levantar la API
go run ./cmd/api
```

A partir de ahí, `POST /auth/login` devuelve un JWT para el resto de los endpoints — ver la [guía de uso completa](./docs/RUNBOOK.md) para el flujo entero con `curl` (crear campaña → activar → capturar participantes → sortear).

## Deploy con Docker

```bash
cp .env.example .env   # completar los valores
docker compose -f docker-compose.prod.yml up --build -d
```

Ver [`docs/RUNBOOK.md`](./docs/RUNBOOK.md) para el detalle de variables/secrets y el bootstrap (alta de administrador y del token de Instagram).

## Tests

```bash
go test ./...           # unitarios (con fakes) + de integración contra Postgres si DATABASE_URL está seteada
go test ./... -race
```

Los tests de integración de la capa de Postgres se saltean automáticamente si no hay `DATABASE_URL` disponible.

## Documentación

| Documento | Contenido |
|---|---|
| [`ARCHITECTURE.md`](./ARCHITECTURE.md) | Diseño completo: entidades, ciclo de vida, límites de la integración con Instagram, estructura de carpetas |
| [`docs/USE_CASES.md`](./docs/USE_CASES.md) | Casos de uso detallados (actor, flujo principal, flujos alternativos, reglas de negocio) |
| [`docs/RUNBOOK.md`](./docs/RUNBOOK.md) | Variables de entorno, secrets, y guía de uso con ejemplos de `curl` |
| [`CLAUDE.md`](./CLAUDE.md) | Contexto y convenciones para trabajar en este repo con Claude Code |

## Licencia

[MIT](./LICENSE)
