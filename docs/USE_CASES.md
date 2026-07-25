# Casos de Uso — API de Sorteos de Instagram

---

## EPIC-00: Autenticación

### UC-0.1: Iniciar Sesión

- **Resumen:** El administrador se autentica con usuario y contraseña para obtener un token de acceso (JWT) que usará en las siguientes solicitudes a la API.
- **Actor:** Administrador.
- **Precondición:** Existe un usuario en la tabla `users` con las credenciales provistas (dado de alta manualmente, vía seed/migración).
- **Disparador:** El administrador envía `POST /auth/login` con `username` y `password`.
- **Flujo Principal:**
  1. El sistema busca el usuario por `username` en la tabla `users`.
  2. El sistema compara la contraseña provista contra el `password_hash` almacenado (bcrypt).
  3. Si coincide, el sistema genera un JWT firmado con expiración y lo devuelve.
- **Flujos Alternativos:**
  - **FA-0.1.1 — Usuario inexistente o contraseña incorrecta:** el sistema devuelve error 401, sin distinguir cuál de las dos condiciones falló (para no revelar qué usernames existen).
  - **FA-0.1.2 — Datos incompletos:** el sistema devuelve un error de validación sin consultar la base.
- **Reglas de Negocio:**
  - Las contraseñas nunca se almacenan ni se comparan en texto plano.
  - No existe endpoint de auto-registro; los usuarios se crean manualmente fuera de la API.
  - Todos los endpoints de administración (excepto `/auth/login` y `/webhooks/instagram`) requieren un JWT válido en el header `Authorization`.
- **Postcondición:** El administrador obtiene un token válido para autenticar solicitudes subsiguientes.

---

## EPIC-01: Gestión de Campañas

### UC-1.1: Crear Campaña

- **Resumen:** El administrador crea una nueva campaña de sorteo, asociándola a una publicación o historia de Instagram y definiendo sus reglas de participación.
- **Actor:** Administrador.
- **Precondición:** El administrador está autenticado en la API. La cuenta de Instagram Business está vinculada y el token de acceso es válido.
- **Disparador:** El administrador envía una solicitud de creación de campaña (`POST /campaigns`).
- **Flujo Principal:**
  1. El administrador provee: tipo de campaña (post o historia), identificador de la publicación de Instagram (`media_id`), nombre/descripción, y reglas de validación (`must_follow`, `min_mentions`).
  2. El sistema valida que el `media_id` no esté ya asociado a otra campaña activa o en draft.
  3. El sistema crea la campaña en estado `draft`.
  4. El sistema devuelve el identificador de la campaña creada.
- **Flujos Alternativos:**
  - **FA-1.1.1 — media_id duplicado:** si ya existe una campaña activa o en draft con el mismo `media_id`, el sistema rechaza la solicitud con un error de conflicto.
  - **FA-1.1.2 — Datos incompletos:** si falta el tipo de campaña o el `media_id`, el sistema devuelve un error de validación sin crear el registro.
- **Reglas de Negocio:**
  - Una publicación de Instagram (`media_id`) no puede tener más de una campaña activa/draft simultánea.
  - Las reglas de validación (`must_follow`, `min_mentions`) son inmutables una vez que la campaña pasa a estado `activa`.
- **Postcondición:** La campaña queda creada en estado `draft`, sin participantes.

---

### UC-1.2: Activar Campaña

- **Resumen:** El administrador activa una campaña en estado `draft` para que empiece a capturar participantes.
- **Actor:** Administrador.
- **Precondición:** La campaña existe y está en estado `draft`. Si es de tipo historia, el webhook de Instagram debe estar configurado y verificado.
- **Disparador:** El administrador envía `POST /campaigns/:id/activate`.
- **Flujo Principal:**
  1. El sistema valida que la campaña esté en estado `draft`.
  2. El sistema cambia el estado de la campaña a `activa`.
  3. A partir de este momento, el sistema acepta sincronizaciones de comentarios y eventos de webhook para esta campaña.
- **Flujos Alternativos:**
  - **FA-1.2.1 — Estado inválido:** si la campaña no está en `draft` (ya está activa, cerrada o sorteada), el sistema rechaza la operación con un error de transición inválida.
- **Reglas de Negocio:**
  - Solo se puede activar una campaña que esté en `draft`.
- **Postcondición:** La campaña queda en estado `activa` y lista para capturar participantes.

---

### UC-1.3: Cerrar Campaña

- **Resumen:** El administrador cierra una campaña activa para dejar de aceptar nuevos participantes, como paso previo obligatorio al sorteo.
- **Actor:** Administrador.
- **Precondición:** La campaña existe y está en estado `activa`.
- **Disparador:** El administrador envía `POST /campaigns/:id/close`.
- **Flujo Principal:**
  1. El sistema valida que la campaña esté en estado `activa`.
  2. El sistema cambia el estado de la campaña a `cerrada`.
  3. El sistema deja de aceptar nuevos participantes para esta campaña; cualquier evento posterior se descarta y se registra en log.
- **Flujos Alternativos:**
  - **FA-1.3.1 — Estado inválido:** si la campaña no está en `activa`, el sistema rechaza la operación.
- **Reglas de Negocio:**
  - Una campaña solo puede cerrarse desde el estado `activa`.
  - Un sorteo (UC-3.1) solo puede ejecutarse sobre una campaña en estado `cerrada`.
- **Postcondición:** La campaña queda en estado `cerrada`, con la lista de participantes definitiva.

---

## EPIC-02: Captura de Participantes

### UC-2.1: Sincronizar Comentarios del Post

- **Resumen:** El sistema trae los comentarios de la publicación de Instagram asociada a la campaña, valida cada uno contra las reglas de participación, y persiste a los participantes válidos.
- **Actor:** Administrador (dispara la sincronización) / Sistema (ejecuta la lógica).
- **Precondición:** La campaña existe, es de tipo post, y está en estado `activa`. El token de acceso a la Graph API es válido.
- **Disparador:** El administrador envía `POST /campaigns/:id/sync-comments` (manual, o disparado por un job programado).
- **Flujo Principal:**
  1. El sistema solicita a la Graph API los comentarios del `media_id` de la campaña, paginando hasta traer todos los disponibles.
  2. Para cada comentario, el sistema verifica si el usuario ya está registrado como participante en esta campaña (deduplicación por `instagram_user_id`).
  3. Si es un participante nuevo, el sistema aplica las reglas de la campaña (ej. cantidad mínima de menciones en el texto del comentario).
  4. Los comentarios que cumplen las reglas se persisten como `Participant`, con `source_type = comment`.
  5. El sistema informa cuántos participantes nuevos se agregaron.
- **Flujos Alternativos:**
  - **FA-2.1.1 — Comentario no cumple reglas:** no se persiste como participante válido.
  - **FA-2.1.2 — Token expirado:** si la Graph API rechaza la solicitud por token inválido/expirado, el sistema devuelve un error indicando que se requiere refrescar el token, sin perder el progreso ya sincronizado.
  - **FA-2.1.3 — Campaña no activa:** el sistema rechaza la sincronización si la campaña está en `draft` o `cerrada`.
- **Reglas de Negocio:**
  - Un mismo `instagram_user_id` no puede generar más de un `Participant` en la misma campaña.
  - Solo se sincronizan comentarios mientras la campaña está en estado `activa`.
- **Postcondición:** Los participantes válidos nuevos quedan persistidos y asociados a la campaña.

---

### UC-2.2: Capturar Mención/Compartido en Historia (Webhook)

- **Resumen:** El sistema recibe en tiempo real, vía webhook, el evento de que un usuario mencionó a la cuenta en su historia, y lo asocia como participante de la campaña correspondiente.
- **Actor:** Sistema (Instagram como emisor del evento).
- **Precondición:** La campaña de tipo historia está en estado `activa`. El webhook de la aplicación está suscripto y verificado ante Meta.
- **Disparador:** Instagram envía un evento a `POST /webhooks/instagram` cuando un usuario menciona a la cuenta en su historia.
- **Flujo Principal:**
  1. El sistema recibe el payload del webhook y valida su firma/token de verificación.
  2. El sistema identifica que el evento corresponde a una mención de historia y extrae el `instagram_user_id` del emisor.
  3. El sistema identifica la campaña de tipo historia en estado `activa` a la que corresponde el evento.
  4. El sistema verifica que el usuario no esté ya registrado como participante de esa campaña.
  5. El sistema persiste al usuario como `Participant`, con `source_type = story_mention`.
- **Flujos Alternativos:**
  - **FA-2.2.1 — Firma inválida:** el sistema descarta el evento y responde con error, sin persistir nada.
  - **FA-2.2.2 — Ninguna campaña activa de tipo historia:** el evento se descarta y se registra en log para trazabilidad.
  - **FA-2.2.3 — Usuario ya registrado:** el evento se ignora, no genera duplicado.
- **Reglas de Negocio:**
  - Un evento de webhook no procesado en el momento (servidor caído, endpoint no configurado) se pierde de forma permanente — no existe mecanismo de recuperación posterior contra la API de Instagram.
  - Solo se procesan eventos mientras la campaña de historia está `activa`.
- **Postcondición:** El usuario que mencionó a la cuenta queda registrado como participante de la campaña de historia correspondiente.

---

### UC-2.3: Excluir Participante Manualmente

- **Resumen:** El administrador excluye manualmente a un participante que cumple las reglas automáticas pero no debe ser elegible para el sorteo (cuenta falsa, empleado del negocio, etc.).
- **Actor:** Administrador.
- **Precondición:** El participante existe y pertenece a la campaña indicada.
- **Disparador:** El administrador envía `PATCH /campaigns/:id/participants/:pid/exclude`, indicando un motivo.
- **Flujo Principal:**
  1. El sistema valida que el participante pertenezca a la campaña indicada.
  2. El sistema marca al participante con `is_excluded = true` y guarda el motivo (`excluded_reason`).
- **Flujos Alternativos:**
  - **FA-2.3.1 — Participante inexistente:** el sistema devuelve error 404.
  - **FA-2.3.2 — Campaña ya sorteada:** el sistema advierte que existe un `Draw` previo y que la exclusión no modifica resultados ya emitidos, pero permite guardarla igual para futuros re-sorteos.
- **Reglas de Negocio:**
  - Un participante excluido nunca se borra del sistema, solo se marca — se conserva para trazabilidad.
  - El motor de sorteo (UC-3.1) ignora a todo participante con `is_excluded = true`.
- **Postcondición:** El participante queda marcado como excluido y no será considerado en sorteos futuros de esa campaña.

---

## EPIC-03: Sorteo

### UC-3.1: Ejecutar Sorteo

- **Resumen:** El administrador ejecuta el sorteo de una campaña cerrada, obteniendo 1, 2 o 3 ganadores seleccionados al azar entre los participantes elegibles.
- **Actor:** Administrador.
- **Precondición:** La campaña existe y está en estado `cerrada`. Existe al menos un participante elegible (`is_excluded = false`).
- **Disparador:** El administrador envía `POST /campaigns/:id/draws`, indicando `winners_count` (1, 2 o 3).
- **Flujo Principal:**
  1. El sistema valida que la campaña esté en estado `cerrada`.
  2. El sistema obtiene la lista de participantes elegibles (`is_excluded = false`).
  3. El sistema valida que la cantidad de elegibles sea mayor o igual a `winners_count`.
  4. El sistema genera una semilla aleatoria (`random_seed`) y la usa para seleccionar `winners_count` participantes sin repetición.
  5. El sistema persiste el `Draw` (con la semilla) y los `Winner` asociados, cada uno con su posición si `winners_count > 1`, o sin posición si es un único ganador.
  6. El sistema cambia el estado de la campaña a `sorteada`.
  7. El sistema devuelve la lista de ganadores.
- **Flujos Alternativos:**
  - **FA-3.1.1 — Campaña no cerrada:** el sistema rechaza la ejecución si la campaña no está en estado `cerrada`.
  - **FA-3.1.2 — Participantes insuficientes:** si la cantidad de elegibles es menor a `winners_count`, el sistema rechaza la operación indicando cuántos participantes elegibles hay.
  - **FA-3.1.3 — Re-sorteo:** si el administrador ejecuta un nuevo `Draw` sobre una campaña ya `sorteada` (ej. un ganador no respondió y se decide resortear con suplentes), el sistema permite crear un nuevo `Draw`, quedando ambos sorteos en el historial.
- **Reglas de Negocio:**
  - El sorteo debe ser reproducible: dado el mismo conjunto de participantes elegibles y la misma `random_seed`, el resultado debe ser idéntico.
  - Los participantes excluidos (`is_excluded = true`) nunca son elegibles.
  - `winners_count` debe ser 1, 2 o 3.
- **Postcondición:** Queda registrado un `Draw` con sus `Winner` correspondientes, y la campaña pasa (o permanece) en estado `sorteada`.

---

### UC-3.2: Consultar Resultado del Sorteo

- **Resumen:** El administrador consulta los ganadores y la información de un sorteo ya ejecutado, incluyendo la semilla usada para poder demostrar su aleatoriedad.
- **Actor:** Administrador.
- **Precondición:** Existe al menos un `Draw` ejecutado para la campaña.
- **Disparador:** El administrador envía `GET /campaigns/:id/draws/:draw_id`.
- **Flujo Principal:**
  1. El sistema busca el `Draw` solicitado.
  2. El sistema devuelve los datos del `Draw`: fecha de ejecución, `winners_count`, `random_seed`, y la lista de `Winner` con su posición y datos del participante (`username`, `instagram_user_id`).
- **Flujos Alternativos:**
  - **FA-3.2.1 — Draw inexistente:** el sistema devuelve error 404.
- **Reglas de Negocio:** (ninguna adicional a las definidas en UC-3.1)
- **Postcondición:** Ninguna — operación de solo lectura.
