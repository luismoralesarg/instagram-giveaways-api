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

### 2.3 Exclusión manual

Un participante válido según las reglas automáticas puede igual necesitar exclusión manual (cuenta falsa, empleado del negocio, etc.). Se modela como un campo en `Participant`: `is_excluded bool`, `excluded_reason string`. No se borra el registro — se mantiene para trazabilidad, pero el motor de sorteo lo ignora.

### 2.4 Reproducibilidad del sorteo

Cada `Draw` persiste la semilla aleatoria (`random_seed`) usada para seleccionar ganadores. Dado el mismo conjunto de participantes elegibles + la misma semilla, el resultado debe ser reproducible — esto permite demostrar que el sorteo fue al azar ante un reclamo.

## 3. Límites de la integración con Instagram

- **Comentarios en posts**: se obtienen vía Graph API (`GET /{media-id}/comments`), sincronización activa (polling o disparada manualmente), no hay webhook nativo por comentario nuevo salvo suscripción a `comments` en el webhook de la app.
- **Menciones/compartidos en historias**: **no existe** un endpoint para consultar el historial. Solo se conocen en el momento en que Instagram dispara el webhook de `messages` (story mention). Si el servidor no está escuchando en ese momento, esa mención se pierde para siempre — no hay forma de recuperarla después.
- **Token de acceso**: el long-lived token de la cuenta expira cada ~60 días. El sistema debe soportar refresh (manual o programado) y alertar si está por vencer.

Esto implica que el webhook debe estar activo y probado **antes** de que arranque cualquier campaña que dependa de historias.

## 4. Arquitectura (hexagonal)

```
/cmd
  /api
    main.go              → wiring de dependencias, arranque del servidor Fiber

/internal
  /domain
    campaign.go          → entidad Campaign + reglas de transición de estado
    participant.go        → entidad Participant
    draw.go               → entidades Draw y Winner + lógica de selección random
    ports.go               → interfaces (CampaignRepository, ParticipantRepository,
                              DrawRepository, InstagramClient, RandomGenerator)

  /application
    /usecase
      create_campaign.go
      close_campaign.go
      sync_comments.go          → trae comentarios del post vía Graph API
      handle_story_mention.go   → procesa el evento del webhook de historias
      exclude_participant.go
      run_draw.go
      list_campaign_results.go

  /infrastructure
    /instagram
      graph_client.go     → implementa InstagramClient contra graph.facebook.com
      webhook_handler.go  → recibe y valida los webhooks de Instagram
      token_refresher.go  → maneja el ciclo de vida del token
    /persistence/postgres
      campaign_repo.go
      participant_repo.go
      draw_repo.go
      migrations/
    /http/fiber
      router.go
      campaign_handlers.go
      draw_handlers.go
      webhook_handlers.go

  /config
    config.go             → carga de variables de entorno
```

**Regla de dependencia:** `domain` no importa nada de `application` ni `infrastructure`. `application` depende solo de las interfaces (`ports`) definidas en `domain`. `infrastructure` implementa esas interfaces. `main.go` es el único lugar que conecta implementaciones concretas con casos de uso.

## 5. Endpoints principales (borrador)

| Método | Ruta | Descripción |
|---|---|---|
| POST | `/campaigns` | Crea una campaña en estado `draft`, con sus reglas |
| POST | `/campaigns/:id/activate` | Pasa a `activa` |
| POST | `/campaigns/:id/close` | Pasa a `cerrada` |
| POST | `/campaigns/:id/sync-comments` | Sincroniza comentarios del post (Graph API) |
| POST | `/webhooks/instagram` | Recibe eventos de Instagram (story mentions, etc.) |
| PATCH | `/campaigns/:id/participants/:pid/exclude` | Exclusión manual |
| POST | `/campaigns/:id/draws` | Ejecuta un sorteo (`winners_count: 1\|2\|3`) |
| GET | `/campaigns/:id/participants` | Lista participantes (con estado de exclusión) |
| GET | `/campaigns/:id/draws/:draw_id` | Resultado de un sorteo (ganadores + semilla) |

## 6. Stack

- Go + Fiber
- PostgreSQL
- Instagram Graph API (Meta for Developers)
- Arquitectura hexagonal (puertos y adaptadores)
