# ARCHITECTURE.md — API de Sorteos de Instagram

## 1. Propósito

API en Go que reemplaza la recopilación manual de participantes de sorteos de Instagram (en Excel) por un flujo automatizado: captura participantes vía la Graph API / webhooks de Instagram, los asocia a una campaña, y ejecuta el sorteo de ganadores de forma reproducible y auditable.

**Fuera de alcance (por decisión explícita):**
- Soporte multi-cliente / multi-cuenta. La API gestiona una única cuenta de Instagram Business.
- Notificación automática al ganador vía DM (Instagram es estricto con mensajería automatizada; se evalúa a futuro).

## 2. Conceptos del dominio

| Concepto | Descripción |
|---|---|
| **Campaign** | Una campaña de sorteo, asociada a una publicación (post) o historia de Instagram. Tiene un ciclo de vida y reglas de validación propias. |
| **Participant** | Una persona que interactuó con la campaña (comentario, o mención/compartido en historia) y quedó registrada como candidata a ganar. |
| **Draw** | La ejecución de un sorteo sobre una campaña. Define cuántos ganadores se buscan (1, 2 o 3) y guarda la semilla aleatoria usada. |
| **Winner** | Un participante seleccionado por un Draw, con su posición (1°, 2°, 3°, o sin posición si es un único ganador). |

### 2.1 Ciclo de vida de una Campaign

```
draft → activa → cerrada → sorteada
```

- **draft**: se está configurando (reglas, tipo de publicación). No captura participantes todavía.
- **activa**: capturando participantes en tiempo real (webhook habilitado, o sincronización manual de comentarios).
- **cerrada**: se dejó de aceptar nuevos participantes. Paso obligatorio antes de sortear.
- **sorteada**: tiene al menos un Draw ejecutado. Una campaña puede tener más de un Draw (ej. si hay que resortear), pero no se puede volver a un estado anterior.

Regla de negocio: **no se puede ejecutar un Draw si la campaña no está `cerrada`.**

### 2.2 Reglas de validación por campaña

Cada Campaign define, al crearse, sus reglas de participación válida. Ejemplos:
- Debe seguir la cuenta (`must_follow: bool`)
- Cantidad mínima de menciones/etiquetados en el comentario (`min_mentions: int`)
- Un participante no puede aparecer dos veces (deduplicación por `instagram_user_id`)

Estas reglas se aplican en el momento de sincronizar/capturar participantes, no en el sorteo — un participante inválido ni siquiera se persiste como elegible (o se persiste marcado como `is_excluded`, ver 2.3).

**Decisión sobre `must_follow` (confirmada con el usuario):** la Graph API de Instagram no permite verificar si un comentarista arbitrario sigue la cuenta sin el access token de ESE usuario — no hay endpoint para chequear el estado de "sigue" de un tercero. Por eso `must_follow` se guarda en `Campaign` solo a título informativo/documental; `sync-comments` (UC-2.1) **no lo aplica como filtro automático**. `min_mentions` sí se verifica (cuenta de menciones `@usuario` en el texto del comentario).

### 2.3 Exclusión manual

Un participante válido según las reglas automáticas puede igual necesitar exclusión manual (cuenta falsa, empleado del negocio, etc.). Se modela como un campo en `Participant`: `is_excluded bool`, `excluded_reason string`. No se borra el registro — se mantiene para trazabilidad, pero el motor de sorteo lo ignora.

### 2.4 Reproducibilidad del sorteo

Cada `Draw` persiste la semilla aleatoria (`random_seed`) usada para seleccionar ganadores. Dado el mismo conjunto de participantes elegibles + la misma semilla, el resultado debe ser reproducible — esto permite demostrar que el sorteo fue al azar ante un reclamo.

Implementación: `domain.SelectWinners` hace un partial Fisher-Yates sobre los elegibles (obtenidos siempre con `ORDER BY id`, para que el orden de entrada también sea determinístico) usando `math/rand` sembrado con `random_seed`. La semilla en sí se genera con `crypto/rand` (`infrastructure/random`, puerto `RandomGenerator`) — aleatoriedad criptográfica al generarla, determinismo matemático al reproducirla.

**Precondición real de UC-3.1 (decisión confirmada):** un `Draw` se puede ejecutar con la campaña en `cerrada` **o** `sorteada` — nunca en `draft`/`activa`. Permitir `sorteada` es necesario para que el re-sorteo (FA-3.1.3) sea posible: una campaña sorteada no tiene forma de volver a `cerrada` con el modelo de estados actual.

**Snapshot en Winner (decisión confirmada):** `Winner` guarda su propio `instagram_user_id`/`username`, copiados de `Participant` en el momento del sorteo — no hace join en vivo contra `participants`. Si el username cambiara después en Instagram, el resultado histórico anunciado no cambia.

### 2.5 Autenticación (User)

Existe una entidad `User` (id, username, password_hash, created_at) usada exclusivamente para autenticar al administrador contra la API — no participa en las reglas de negocio de sorteos.

- El administrador se autentica con usuario y contraseña (`POST /auth/login`) y recibe un JWT (Bearer token).
- Todos los endpoints de administración requieren `Authorization: Bearer <token>`, **excepto** `/auth/login` y `/webhooks/instagram` (este último se autentica con la firma de Meta vía `INSTAGRAM_APP_SECRET`, no con JWT).
- No hay endpoint de auto-registro: los usuarios se dan de alta manualmente (seed/migración), coherente con el alcance de una única cuenta/administrador.
- Las contraseñas se almacenan hasheadas (bcrypt) — nunca en texto plano ni en logs.

## 3. Límites de la integración con Instagram

- **Comentarios en posts**: se obtienen vía Graph API (`GET /{media-id}/comments`), sincronización activa (polling o disparada manualmente), no hay webhook nativo por comentario nuevo salvo suscripción a `comments` en el webhook de la app.
- **Menciones/compartidos en historias**: **no existe** un endpoint para consultar el historial. Solo se conocen en el momento en que Instagram dispara el webhook de `messages` (story mention). Si el servidor no está escuchando en ese momento, esa mención se pierde para siempre — no hay forma de recuperarla después.
- **Token de acceso**: el long-lived token de la cuenta expira cada ~60 días. El sistema debe soportar refresh (manual o programado) y alertar si está por vencer.

Esto implica que el webhook debe estar activo y probado **antes** de que arranque cualquier campaña que dependa de historias.

### 3.1 Refresh del token (implementado)

El token **ya no es una variable de entorno**: vive en Postgres (tabla `instagram_token`, de una sola fila — no hay soporte multi-cuenta). `GraphClient` lo consulta en cada llamada, así un refresh queda reflejado sin reiniciar el proceso de la API.

- **Alta inicial**: `cmd/seedinstagramtoken -token=... [-expires-in-days=60]` — el long-lived token se obtiene a mano (flujo de Meta for Developers, fuera de esta app) y se carga una única vez, mismo patrón que `cmd/seedadmin`.
- **Refresh automático**: `cmd/scheduler` es un módulo de tareas programadas (`internal/infrastructure/scheduler`, sobre `robfig/cron/v3`) — un registro de jobs con expresión cron, cada uno disparando un caso de uso. El job `refresh-instagram-token` corre todos los días a las 3am y solo actúa si al token le quedan 10 días o menos para vencer (`domain.InstagramToken.NeedsRefresh`); si está dentro de esa ventana, llama a la Graph API (`grant_type=fb_exchange_token`, requiere `INSTAGRAM_APP_ID` además de `INSTAGRAM_APP_SECRET`) y persiste el token nuevo con su expiración fresca.
- **"Alertar si está por vencer"**: por ahora es el log de `cmd/scheduler` (éxito, salteo, o error del refresh) — no hay canal de notificaciones (email/Slack) en el proyecto todavía.
- `cmd/scheduler -run-once=<job>` corre un job ya, sin esperar su cron — útil para operar a demanda o para probar que el job funciona.

**Decisión sobre campañas de historia concurrentes (confirmada con el usuario):** el payload del webhook de story mention no trae ningún identificador que permita saber a cuál campaña de tipo historia corresponde una mención — solo una URL de imagen. Por eso **como máximo una campaña de tipo historia puede estar `activa` a la vez**: `POST /campaigns/:id/activate` rechaza activar una segunda campaña de historia mientras otra siga activa (mismo mecanismo que la deduplicación de `media_id` en UC-1.1 — pre-chequeo en el caso de uso + índice único parcial en Postgres como respaldo).

## 4. Arquitectura (hexagonal)

```
/cmd
  /api
    main.go                    → wiring de dependencias, arranque del servidor Fiber
  /scheduler
    main.go                    → módulo de tareas programadas (cron); -run-once=<job>
  /seedadmin
    main.go                    → alta manual de administradores
  /seedinstagramtoken
    main.go                    → alta manual del primer InstagramToken

/internal
  /domain
    campaign.go          → entidad Campaign + reglas de transición de estado
    participant.go        → entidad Participant
    draw.go               → entidades Draw y Winner + lógica de selección random
    user.go                → entidad User (autenticación)
    instagram_token.go      → entidad InstagramToken + NeedsRefresh()
    errors.go              → errores de dominio (sentinel errors) de todas las entidades
    ports.go               → interfaces (CampaignRepository, ParticipantRepository,
                              DrawRepository, UserRepository, InstagramClient,
                              InstagramTokenRepository, InstagramTokenRefresher,
                              RandomGenerator, PasswordHasher, TokenIssuer)

  /application
    /usecase
      login.go                  → autentica usuario/contraseña y emite un JWT
      create_campaign.go
      activate_campaign.go
      close_campaign.go
      sync_comments.go          → trae comentarios del post vía Graph API (UC-2.1)
      handle_story_mention.go   → procesa el evento del webhook de historias (UC-2.2),
                                   agnóstico de HTTP
      exclude_participant.go
      list_participants.go      → GET /campaigns/:id/participants (sin UC dedicado)
      run_draw.go                → UC-3.1: valida precondición, arma el pool de
                                    elegibles, delega en domain.SelectWinners
      get_draw_result.go         → UC-3.2
      refresh_instagram_token.go → job "refresh-instagram-token" (§3.1): chequea
                                    NeedsRefresh y, si corresponde, refresca y persiste

  /infrastructure
    /auth
      jwt.go               → firma y valida JWT (implementa TokenIssuer)
      password_hasher.go   → hashing de contraseñas con bcrypt
    /random
      generator.go          → RandomGenerator con crypto/rand (la seed que
                               persiste cada Draw)
    /scheduler
      scheduler.go           → módulo de tareas programadas: registro de Job
                                (nombre + cron + función) sobre robfig/cron/v3
    /instagram
      graph_client.go    → implementa InstagramClient contra graph.facebook.com
                            (UC-2.1; lee el token vigente de InstagramTokenRepository
                            en cada llamada, no un string fijo)
      token_refresher.go → implementa InstagramTokenRefresher (fb_exchange_token,
                            ver §3.1)
      errors.go           → graphAPIError, compartido por ambos clientes
      signature.go        → verifica la firma HMAC-SHA256 (X-Hub-Signature-256) de
                            los webhooks de Meta
    /persistence/postgres
      db.go                → pool de conexión (pgxpool)
      errors.go             → helpers para traducir errores de pgx/postgres
      campaign_repo.go
      participant_repo.go
      draw_repo.go
      user_repo.go
      instagram_token_repo.go → tabla de una sola fila (id fijo en 1)
      migrations/
    /http/fiber
      router.go
      auth_handlers.go
      auth_middleware.go     → valida el Bearer token en las rutas protegidas
      campaign_handlers.go
      participant_handlers.go
      webhook_handlers.go     → GET/POST /webhooks/instagram: valida firma y
                                delega en handle_story_mention.go (UC-2.2)
      draw_handlers.go

  /config
    config.go             → carga de variables de entorno
```

**Regla de dependencia:** `domain` no importa nada de `application` ni `infrastructure`. `application` depende solo de las interfaces (`ports`) definidas en `domain`. `infrastructure` implementa esas interfaces. `main.go` es el único lugar que conecta implementaciones concretas con casos de uso.

## 5. Endpoints principales (borrador)

| Método | Ruta | Descripción | Auth |
|---|---|---|---|
| POST | `/auth/login` | Autentica usuario/contraseña, devuelve un JWT | pública |
| POST | `/campaigns` | Crea una campaña en estado `draft`, con sus reglas | JWT |
| POST | `/campaigns/:id/activate` | Pasa a `activa` | JWT |
| POST | `/campaigns/:id/close` | Pasa a `cerrada` | JWT |
| POST | `/campaigns/:id/sync-comments` | Sincroniza comentarios del post (Graph API) | JWT |
| POST | `/webhooks/instagram` | Recibe eventos de Instagram (story mentions, etc.) | firma Meta |
| PATCH | `/campaigns/:id/participants/:pid/exclude` | Exclusión manual | JWT |
| POST | `/campaigns/:id/draws` | Ejecuta un sorteo (`winners_count: 1\|2\|3`) | JWT |
| GET | `/campaigns/:id/participants` | Lista participantes (con estado de exclusión) | JWT |
| GET | `/campaigns/:id/draws/:draw_id` | Resultado de un sorteo (ganadores + semilla) | JWT |

## 6. Stack

- Go + Fiber
- PostgreSQL
- Instagram Graph API (Meta for Developers)
- JWT (autenticación) + bcrypt (hashing de contraseñas)
- robfig/cron/v3 (módulo de tareas programadas, `cmd/scheduler`)
- Arquitectura hexagonal (puertos y adaptadores)
