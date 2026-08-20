# RUNBOOK — Variables de entorno y guía de uso

Referencia operativa: qué variables/secrets hacen falta y cómo usar la API de punta a punta. Para el diseño y las decisiones de arquitectura, ver [`ARCHITECTURE.md`](../ARCHITECTURE.md).

---

## 1. Variables de entorno de la app

Las lee `internal/config.Load()`, usado por `cmd/api` y `cmd/scheduler`:

| Variable | Obligatoria para arrancar | Para qué sirve |
|---|---|---|
| `DATABASE_URL` | Sí | Connection string de Postgres (`postgres://user:pass@host:5432/db?sslmode=disable`) |
| `JWT_SECRET` | Sí | Firma y valida los JWT de `POST /auth/login` |
| `INSTAGRAM_APP_ID` | No | App ID de Meta — solo lo usa `cmd/scheduler` para refrescar el token (`grant_type=fb_exchange_token`) |
| `INSTAGRAM_APP_SECRET` | No | App Secret de Meta — lo usa `cmd/api` para validar la firma del webhook (`X-Hub-Signature-256`) y `cmd/scheduler` para el refresh |
| `INSTAGRAM_WEBHOOK_VERIFY_TOKEN` | No | El valor que Meta manda como `hub.verify_token` al suscribir el webhook |
| `PORT` | No (default `8080`) | Puerto HTTP de `cmd/api` |

Notas:
- **No existe `INSTAGRAM_ACCESS_TOKEN`** — el token en sí vive en Postgres (tabla `instagram_token`), no es una env var. Se carga una vez con `cmd/seedinstagramtoken` y `cmd/scheduler` lo mantiene fresco (ver §3).
- `cmd/seedadmin` y `cmd/seedinstagramtoken` **no pasan por `config.Load()`** — solo leen `DATABASE_URL` directo del entorno; los datos reales (usuario/contraseña, token) van por flags de línea de comandos, no por env vars.

### Solo para `docker-compose.prod.yml`

| Variable | Para qué sirve |
|---|---|
| `POSTGRES_PASSWORD` | Contraseña del servicio `postgres` del compose — tiene que coincidir con la que uses dentro de `DATABASE_URL` |

Se completan en un `.env` (copiando [`.env.example`](../.env.example)) al lado de `docker-compose.prod.yml`. Ese archivo **no se commitea**.

### Secrets de GitHub Actions

Usados por [`.github/workflows/release.yml`](../.github/workflows/release.yml), job `deploy` — se cargan en **Settings → Environments → production**:

| Secret | Obligatorio | Para qué sirve |
|---|---|---|
| `PROD_SSH_HOST` | Sí | Host del servidor de producción |
| `PROD_SSH_USER` | Sí | Usuario SSH |
| `PROD_SSH_PRIVATE_KEY` | Sí | Clave privada SSH |
| `PROD_SSH_PORT` | No (default `22`) | Puerto SSH |
| `PROD_APP_PATH` | Sí | Path del clone del repo en el servidor (donde vive `docker-compose.prod.yml` + `.env`) |

---

## 2. Levantar la API

**Local (Go directo), para desarrollo:**

```bash
export DATABASE_URL="postgres://user:pass@localhost:5432/giveaways?sslmode=disable"
export JWT_SECRET="algo-largo-y-random"
migrate -path internal/infrastructure/persistence/postgres/migrations -database "$DATABASE_URL" up
go run ./cmd/api
```

**Con Docker (producción o staging):**

```bash
cp .env.example .env   # completar los valores
docker compose -f docker-compose.prod.yml up --build -d
```

`api` y `scheduler` quedan corriendo; `postgres` no expone puerto al host.

---

## 3. Bootstrap (una sola vez)

No hay endpoints de alta — todo es manual, vía CLI:

```bash
# 1. Crear el primer administrador
go run ./cmd/seedadmin -username=admin -password=algo-seguro
# (con Docker: docker compose -f docker-compose.prod.yml --profile tools run --rm seedadmin -username=admin -password=algo-seguro)

# 2. Cargar el access token de Instagram (obtenido a mano vía Meta for Developers)
go run ./cmd/seedinstagramtoken -token=EAAB... [-expires-in-days=60]
# A partir de acá, cmd/scheduler lo refresca solo (job diario a las 3am).
```

---

## 4. Flujo completo (con `curl`)

**1. Login** — devuelve el JWT que va en `Authorization: Bearer <token>` en todo lo demás:

```bash
TOKEN=$(curl -s -X POST http://localhost:8080/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"admin","password":"algo-seguro"}' | jq -r .token)
```

**2. Crear una campaña** (queda en `draft`):

```bash
curl -s -X POST http://localhost:8080/campaigns \
  -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d '{"type":"post","media_id":"17123...","name":"Sorteo verano","must_follow":true,"min_mentions":2}'
# {"id": "..."}
```

`type` es `"post"` o `"historia"`. `must_follow` queda solo documentado (no se verifica automáticamente — la Graph API no lo permite para comentaristas arbitrarios, ver ARCHITECTURE.md §2.2).

**3. Activar** (empieza a aceptar participantes):

```bash
curl -s -X POST http://localhost:8080/campaigns/$CID/activate -H "Authorization: Bearer $TOKEN"
```

**4. Capturar participantes** — dos caminos según el tipo de campaña:

- **Post**: disparar la sincronización de comentarios contra la Graph API.

  ```bash
  curl -s -X POST http://localhost:8080/campaigns/$CID/sync-comments -H "Authorization: Bearer $TOKEN"
  # {"new_participants": N}
  ```

- **Historia**: llegan solas por `POST /webhooks/instagram` cuando Instagram dispara el evento — nada que hacer manualmente. El webhook tiene que estar suscripto y accesible desde afuera **antes** de que la campaña arranque: un evento perdido no se recupera nunca (ARCHITECTURE.md §3).

**5. Ver participantes cargados:**

```bash
curl -s http://localhost:8080/campaigns/$CID/participants -H "Authorization: Bearer $TOKEN"
```

**6. Excluir a alguien** (no lo borra, lo marca):

```bash
curl -s -X PATCH http://localhost:8080/campaigns/$CID/participants/$PID/exclude \
  -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d '{"reason":"cuenta falsa"}'
```

**7. Cerrar la campaña** (deja de aceptar participantes, paso obligatorio antes de sortear):

```bash
curl -s -X POST http://localhost:8080/campaigns/$CID/close -H "Authorization: Bearer $TOKEN"
```

**8. Sortear:**

```bash
curl -s -X POST http://localhost:8080/campaigns/$CID/draws \
  -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d '{"winners_count":2}'
# {"id":"...", "winners_count":2, "random_seed":..., "winners":[{"instagram_user_id":"...","position":1}, ...]}
```

`winners_count` es 1, 2 o 3. Se puede re-sortear las veces que haga falta — queda historial de todos los `Draw`, no se pisan.

**9. Consultar un sorteo ya hecho:**

```bash
curl -s http://localhost:8080/campaigns/$CID/draws/$DRAW_ID -H "Authorization: Bearer $TOKEN"
```

---

## 5. Todos los endpoints

| Método | Ruta | Auth |
|---|---|---|
| POST | `/auth/login` | pública |
| GET/POST | `/webhooks/instagram` | firma de Meta (no JWT) |
| POST | `/campaigns` | JWT |
| POST | `/campaigns/:id/activate` | JWT |
| POST | `/campaigns/:id/close` | JWT |
| POST | `/campaigns/:id/sync-comments` | JWT |
| GET | `/campaigns/:id/participants` | JWT |
| PATCH | `/campaigns/:id/participants/:pid/exclude` | JWT |
| POST | `/campaigns/:id/draws` | JWT |
| GET | `/campaigns/:id/draws/:draw_id` | JWT |
