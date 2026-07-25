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
```

(Ajustar esta sección a medida que se agreguen Makefile, migraciones, docker-compose, etc.)

## Variables de entorno esperadas

```
DATABASE_URL=
INSTAGRAM_ACCESS_TOKEN=
INSTAGRAM_APP_SECRET=
INSTAGRAM_WEBHOOK_VERIFY_TOKEN=
JWT_SECRET=
PORT=
```

## Al implementar un caso de uso nuevo

1. Definir/ajustar la entidad y las interfaces necesarias en `domain`.
2. Escribir el caso de uso en `application/usecase`, contra las interfaces (no contra implementaciones concretas).
3. Implementar o ajustar el adapter correspondiente en `infrastructure`.
4. Conectar todo en `cmd/api/main.go`.
5. Agregar el handler HTTP en `infrastructure/http/fiber` si corresponde.

## Qué preguntar antes de asumir

- Si una tarea toca las reglas de validación de participantes o el algoritmo de sorteo, confirmar con el usuario antes de cambiar el comportamiento — son reglas de negocio explícitamente definidas, no detalles de implementación libres.
