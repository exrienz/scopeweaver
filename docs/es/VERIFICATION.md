# Verificación y limitaciones conocidas

[English](../VERIFICATION.md) · [한국어](../ko/VERIFICATION.md) · [简体中文](../zh-Hans/VERIFICATION.md) · [繁體中文](../zh-Hant/VERIFICATION.md) · [日本語](../ja/VERIFICATION.md) · Español

<a id="current-dev-pr-40-reconstruction--2026-10-25"></a>
## Reconstrucción de la PR #40 sobre el dev actual — 2026-10-25

Primero se reconstruyeron únicamente los cuatro commits previstos de `aebea3e..9b04754` sobre `f354c920`; no se reprodujo el historial divergente de la rama. Después se reaplicó el candidato sobre `7af06f1`, conservando su integración de seis idiomas. La integración actual exige exactamente **1,987** claves de producción con traducciones propias en inglés, coreano, chino simplificado, chino tradicional, japonés y español, sin recurrir a alternativas de idioma. Se conservan el aislamiento de claves de entorno de OpenCode, la prioridad de las cabeceras de sesión y los valores predeterminados de User-Agent.

La ampliación de las pruebas de cancelación y condiciones de carrera reveló un resultado del planificador que ya carecía de sincronización: la sesión en streaming puede retornar durante una pausa mientras el manejador real de `prove_goal` sigue estableciendo `GoalMet` y su motivo. Los dos escritores de herramientas terminales y la instantánea del resultado del planificador ahora comparten un mutex privado. Se conservan los campos exportados, los comprobantes auténticos de herramientas y terminación y la prioridad actual entre pausa y finalización; no se añaden solicitudes al proveedor ni cambios en el esquema de producción.

El control de prueba terminal también exige un **comprobante auténtico de resultado satisfactorio de herramienta del SDK**, coincidente con el ID de uso de herramienta marcado y la respuesta del modelo que lo originó. El observador privado de captura se ejecuta antes del siguiente límite de compresión o inferencia; ni la finalización del manejador ni un hook `PostToolUse` bastan por sí solos. La regresión adversaria usó `goal_met` real, un presupuesto de ejecución de **30 ms** y un hook posterior a la herramienta de **120 ms**: el control anterior omitía la resolución final tras un comprobante interrumpido y realizaba **1 llamada al proveedor en lugar de 2**, aunque el motivo nativo seguía siendo `timeout` y el error seguía siendo nil. La ruta corregida conserva ese comprobante interrumpido, permite la llamada real de resolución y su compresión, y rechaza que un hook tardío cumpla la condición. Una segunda regresión sincronizada retiene el hook hasta que la captura observa la interrupción, demostrando que el manejador real se ejecutó antes del comprobante de timeout; se mantiene la cobertura de prueba satisfactoria normal, cancelación del padre, desbordamiento reactivo y protecciones contra orígenes tardíos. Consulta las [regresiones de prueba terminal](../../agent/terminal_proof_test.go).

La reconstrucción inicial aún tenía una carencia heredada en el apagado: sobre `f354c920` sin modificar, `TestMainServeFailureRunsShutdown` falló en **3 de 10** ejecuciones aisladas, incluidas escrituras del planificador tras cerrar la base de datos. La corrección posterior registra la propiedad del planificador antes de iniciarlo. `Manager.Close` cancela los planificadores y espera su terminación fuera del bloqueo del gestor durante un máximo de **5 segundos**, antes de detener el escritor de registros y cerrar la base de datos. Si el SQL del planificador sigue bloqueado, Close devuelve `context.DeadlineExceeded` y **mantiene abierta la base de datos**; una vez terminado el trabajo, un reintento puede cerrarla. La CLI ya informa del fallo de cierre y sale con código 1. Esto no afirma que las llamadas SQL existentes sin contexto se vuelvan cancelables. Un planificador cancelado de antemano ya no inicializa marcas de progreso ni inicia otro ciclo. La [regresión con SQL retenido](../../server/scheduler_shutdown_test.go) comprueba tiempos de espera agotados en cierres simultáneos, que la base siga abierta mientras haya trabajo activo y que los reintentos simultáneos tengan éxito tras la liberación. Una primera ejecución paralela de todos los paquetes también falló en la aserción del recuento global de filas del test del receptor de registros mientras subprocesos de otros paquetes usaban la misma base; la interferencia entre paquetes es una hipótesis, por lo que la verificación final sigue el patrón de CI de una base desechable distinta por paquete. Estos comprobantes históricos no afirman una verificación final en CI alojada del commit exacto, en Windows nativo ni con proveedores reales.

El candidato integrado usa Next.js **16.4.3** y Zod **4.6.7** en web, y Zod **4.6.7** en el adaptador, con correcciones compatibles del archivo de bloqueo para body-parser **2.4.0**, fast-uri **3.1.9**, ip-address **10.7.4** y qs **6.16.1**. La versión fijada del SDK del adaptador sigue siendo **1.32.0**. Tras la reaplicación sobre la integración de seis idiomas, una instalación nueva con `npm ci` pasó en ambos paquetes; las pruebas web pasaron **75/75**, la compilación estática en frío exportó **31 páginas**, las pruebas del adaptador pasaron **18/18** y su comprobación de sintaxis pasó. Los comprobantes anteriores de 72/7 pruebas siguen siendo evidencia histórica. Ambas comprobaciones actuales de `npm audit --audit-level=low` informan de **0 vulnerabilidades**. Estos resultados sustituyen la limitación inicial por el aviso de seguridad y el fallo de compilación de Next.js 16.4.0; los comprobantes originales se conservan como evidencia histórica. Las teclas nativas de pegar/deshacer del navegador ahora usan `ControlOrMeta+V` y `ControlOrMeta+Z`, conservando aserciones y tiempos de espera. Las ejecuciones finales del navegador canónico y la CI alojada del commit exacto son condiciones independientes que no se dan por cumplidas aquí.

Las comprobaciones de compatibilidad tras la integración de seis idiomas deben cubrir los prompts estándar anteriores del planificador y los trabajadores y los esquemas `add_intent` de cada idioma de origen publicado y cada idioma de salida solicitado. Los casos adicionales de zh-Hant/ja/es proceden del dev inmutable anterior a PR40, `7af06f1`. El reconocimiento exacto debe conservar prompts editados, campos y valores predeterminados personalizados de esquemas, filas almacenadas e historial sin procesar. Esto amplía la verificación original de tres idiomas descrita más abajo.

<a id="patched-sdk-and-model-specific-context-windows"></a>
### SDK corregido y ventanas de contexto específicas de cada modelo

La batería de Norma anidada es obligatoria en el trabajo CI de backend. Ejecuta `(cd third_party/norma && go test ./... -count=1)` por separado de los paquetes raíz; consulta los [avisos de origen y modificación](PROVENANCE.md#norma-execution-and-compaction-fixes--2026-10-25).

Las regresiones cubren la carrera original de herramienta interrumpida de 30 ms / 120 ms, el ritmo de compresión en ventanas pequeñas y los límites de sesiones largas, la consolidación útil de resúmenes, las vistas reales de reintento por desbordamiento, los límites aprendidos que no aumentan y los errores de salida que no deben activar la recuperación de contexto. Se conservan los límites numéricos existentes y el historial original. La aserción anterior de que 40K no podía alcanzar el segundo nivel ya era falsa en el código original sin modificar; su reemplazo exige consolidación real, recuperación de tokens y resúmenes acotados usando el mismo caso de prueba.

`TestNoaRetainsConfiguredModelContextWindow` ejecuta sesiones reales de Noa con 40K, 128K, 200K, 250K, 400K y 1M. Antes de corregir el host, todos los casos distintos de 200K se convertían en 200K. `enableNoa` ahora pasa la ventana positiva del perfil resuelto antes de eliminar el compresor sustituido. Pruebas separadas cubren la alternativa cuando falta la ventana, Noa desactivado, recuperación de estado dañado y la política local existente de Ollama. `MaxTokens` sigue limitando la salida de forma independiente. La alternativa de 200K es para una ventana sin configurar; no es un límite universal del modelo. Los valores predeterminados del SDK a partir de 200K no cambian, mientras que la condición por cantidad de resúmenes útiles cambia de forma intencionada en las distintas ventanas.

<a id="trigger-evidence-in-the-first-planner-request-2026-10-18"></a>
## Evidencia del disparador en la primera solicitud del planificador (2026-10-18)

Candidato local sobre el commit original `ffe260435fed6576394d613991aaf6e8010d3f48`, con la corrección compartida de la CLI adaptada. Normalmente, un evento de hallazgo o de trabajador finalizado despierta al planificador con resúmenes de disparadores y un `graph_overview` compacto; `prove_goal` sigue exigiendo la evidencia registrada para cada condición del objetivo. Las entradas compactas carecían de los campos completos de hallazgos/hechos, por lo que leer `node_detail` antes de probar era legítimo.

Cambio real abreviado en [Planner.Plan](../../agent/planner.go), omitiendo la configuración intermedia del contexto de tarea que no cambió:

```go
// Before
situational := renderTriggers(ts, triggers) + renderGraphOverview(tsx.graphOverviewData())
// After
overview := tsx.graphOverviewData()
situational := renderTriggers(ts, triggers) + renderGraphOverview(overview)
// After ensureRunDir and WithReviewContext:
situational += renderPlannerEvidence(ctx, overview, triggers, tools, ts.ID(), taskDir)
```

La primera línea nueva captura una sola vez la vista general acotada existente; la segunda representa esa misma instantánea. La última añade los detalles completos de sus candidatos de hechos/hallazgos de la exploración actual, obtenidos de intents disparadores concretos `finding`/`done`, una vez disponible el contexto de tarea/revisión. No recorre `related_tasks` ni amplía la genealogía global. La [implementación del adjunto](../../agent/planner_evidence.go) realiza como máximo **4 intentos de lectura** y solo admite nodos enteros dentro de **32,768 bytes UTF-8 para todo el adjunto**, incluidos su encabezado traducido, envoltura JSON y `omitted_count`; un nodo demasiado grande se omite en lugar de truncarse. Se eliminan los candidatos duplicados.

Solo puede leer la capacidad final resuelta de `node_detail`: su esquema debe aceptar la entrada, debe cumplirse `IsReadOnly` y los permisos deben permitir explícitamente una entrada sin cambios. Envoltorios, valores predeterminados, medición, resultados restringidos y errores siguen vigentes; los resultados inseguros o incompletos se omiten sin recurrir a auxiliares sin procesar. El encabezado está completo en EN/KO/zh-Hans e identifica el contenido registrado como observaciones no fiables. Las lecturas explícitas existentes, incluidos los nodos asociados de fuente directa, conservan sus reglas de acceso originales.

Caso concreto del **laboratorio R22 de loopback sintético controlado por la CLI**, no de un incidente en producción original: el objetivo `2` siguió abierto después de que el intent `3` completara un GET propio y guardara el hecho `4` y el hallazgo `5`. El objetivo exigía el nombre exacto de hallazgo `SYNTHETIC LAB ONLY public debug marker`, gravedad `low`, marcador y observaciones. A las **2026-10-17 22:42:31+09:00**, el planificador leyó legítimamente `node_detail` `5` y `4` para obtener esos campos completos que faltaban. La limpieza del arnés lo pausó a las **22:42:54+09:00**, antes de la prueba; no fue una pausa humana. R22 falló tras **470.50 segundos en total**, con el **presupuesto de tarea de 420 segundos** sin cambios. La causa subyacente de la latencia del modelo/proveedor es **desconocida**. Con la corrección, los detalles completos admisibles de esos mismos candidatos de la instantánea llegan a la primera solicitud de la ronda activada; no constituyen por sí solos una prueba.

| Comprobación del candidato local | Resultado observado |
| --- | --- |
| Paquete `agent` original completo, código exacto de producción sin superposición | 598 pruebas/subpruebas pasadas, 0 fallos/omisiones; equivalente CLI: 607 pasadas, 0 fallos/omisiones |
| Nuevas regresiones en tres archivos `planner_evidence*test.go` | 31 pruebas/subpruebas pasadas; igualdad de la primera solicitud real de `Planner.Plan` con/sin streaming, nueve casos de permisos, límites de cantidad/bytes, deduplicación/aislamiento, evidencia ausente/eliminada/tardía/oculta, recuperación explícita, ausencia de prueba automática y varios objetivos |
| Comprobaciones originales específicas del servidor de prueba/idiomas y paquete `locale` completo | 5 y 19 pasadas respectivamente, 0 fallos/omisiones; se conserva la integridad estricta de **1,986 claves × 3 idiomas**, actualizando el recuento esperado desde 1,985 por el nuevo encabezado |
| `go build ./...`, `go vet ./...` originales | Ambos con código de salida nativo 0 |
| Linux `-race`, código CLI equivalente actual | Comprobaciones específicas: 49 pasadas; paquete `agent` completo: 607 pasadas, 0 fallos/omisiones; ambas salidas reales de Go son 0, sin `DATA RACE`. Verificados los 1,175 archivos fuente de la instantánea; no es una ejecución del módulo original |
| Aceptación real por loopback de la CLI empaquetada, los cinco casos | Salida nativa 0 en core 19.14s, GLM LOW 82.63s, Flash LOW 114.84s, OAuth LOW 108.12s y Flash HIGH 317.83s, cada uno con el presupuesto de tarea inalterado de 420 segundos. Estas ejecuciones únicas de laboratorio no son una comparación de rendimiento |

La base original sin cambios falló las aserciones de contexto ausente con y sin streaming (dos fallos hoja y su padre), sin fallos de preparación. El uso de proveedores controlados en los casos de regresión es sintético, no evidencia de tokens de pago. Siguen siendo necesarios `prove_goal` real, uso/comprobantes auténticos y el control de prueba terminal existente; no cambian fase, trabajador, esfuerzo, plazo ni reintentos. Contraejemplo: un nodo desactivado, restringido, ausente, demasiado grande, no activado por un disparador o heredado no recibe detalles completos automáticamente. Cuando sea posible, una lectura explícita puede recuperar evidencia omitida/reciente; leer datos por sí solo no puede completar un objetivo, y probar uno de varios objetivos no puede completar la tarea.

Son resultados del árbol de trabajo local, no CI del nuevo commit exacto. El fallo inalterado de privilegios de Windows en `TestChatBranchCopyRejectsDestinationSymlink` también se reprodujo en la base CLI `650dc82910e5e36668161a5276455506f64713fe`; no se debilitó ninguna comprobación ni se afirma éxito completo del servidor nativo. Siguen pendientes las siete comprobaciones alojadas obligatorias del nuevo candidato. La PR #40 sigue sin integrarse y pendiente de revisión humana final; esta sección no afirma publicación ni lanzamiento. No se introducen cambios de API pública, esquema ni migraciones. Las secciones de fechas anteriores conservan sus resultados históricos.

<a id="chat-context-readiness-2026-10-17"></a>
## Disponibilidad del contexto de chat (2026-10-17)

Candidato local de interfaz sobre el commit c9fb642 de PR #40. El inspector puede abrirse antes de que termine su solicitud inicial de controles. Antes identificaba como borrador una sesión sin resolver, trataba después la primera sesión resuelta como un cambio y cerraba el panel. El caso chino de tarea en escritorio falló con un botón de cierre inestable y después separado del DOM en la [ejecución de comprobaciones 39031048389](https://github.com/cskwork/scopeweaver/actions/runs/39031048389). Ese commit alojado sigue siendo evidencia de fallo; estos resultados nuevos son locales.

Lógica de reinicio abreviada (binding es la clave de controles solicitada; session es su identidad resuelta):
~~~ts
// Before: every change from draft to resolved identity ran the reset.
epoch.current++;
setOpen(false);
// After: qualify the change before the existing reset.
const previous = hydrated.current;
hydrated.current = { binding, session };
if (previous.binding === binding && !previous.session) return;
epoch.current++;
setOpen(false);
~~~
La primera línea añadida lee binding y session anteriores; la segunda recuerda el par actual. La tercera conserva el panel abierto solo mientras el mismo binding no hubiera resuelto aún su primera session. Otro binding solicitado o un cambio posterior de session resuelta siguen llegando al incremento de epoch y al reinicio completo de vista/formulario, incluidos el foco y el número de elementos retenidos. El hook expone su clave solicitada existente; no hay cambios de API del servidor ni de etiquetas traducidas.

Caso controlado concreto: tarea 1, exploración 1, segmento 0 se pausó mediante la API real de tareas. Su primera respuesta de controles se retuvo hasta que se abrió /context. Liberar esa respuesta real antes cerraba el disparador y hacía que el clic normal agotara su espera tras 15 segundos (salida Go 1); el mismo comprobador de navegador, idéntico byte a byte, ahora mantiene abierto el disparador y cierra normalmente (salida 0), con cero llamadas al proveedor. La pausa de tarea es distinta del campo paused del control de chat.

| Comprobación local del código corregido congelado | Resultado observado |
| --- | --- |
| Pruebas existentes del frontend | 65 pasadas, 0 fallos/omisiones |
| Compilación estática nueva del frontend | TypeScript y 31 páginas completados; salida nativa 0 verificada |
| Comprobador independiente sin cambios de primera respuesta retrasada | 1 prueba Go pasada, 0 fallos/omisiones; cero llamadas al proveedor |
| Caso canónico de navegador para contexto | Pasaron los 12 casos EN/KO/zh-Hans × 1440/390 × independiente/tarea, más los controles de hidratación y binding solicitado (14 pasos de navegador; 1 prueba Go, 0 fallos/omisiones) |

La protección añadida usa la primera respuesta real del backend, y luego estado sintético de ruta para un cambio posterior de session resuelta y una respuesta antigua de contexto retenida. Cada respuesta de sustitución permanece retenida hasta la aserción de contenido obsoleto, para impedir que un resultado nuevo oculte una filtración. El foco no predeterminado y la retención 7 se reinician a foco vacío y valor predeterminado 2. Contraejemplo: un cambio real de binding solicitado o de session resuelta posterior debe seguir reiniciando; los resultados tardíos no deben restaurar el contexto anterior. Un segundo cambio real de consulta de tarea (de tarea 7 a 8 en la ejecución completa) observó la nueva URL de controles con su respuesta retenida, conservó el mismo disparador del inspector conectado y el documento, reinició foco/retención y rechazó el contexto anterior tardío antes de que los datos sustitutos pudieran ocultarlo. Esto demuestra un cambio del binding solicitado con el componente montado, no un remontaje. Las ramas de error tardío/compresión satisfactoria siguen revisadas en código, sin ejecutarse por separado. Las protecciones de identidad sintética demuestran el manejo del cliente, no la sustitución de exploración del servidor. Los demás casos de contexto usan el proveedor de loopback controlado; no se llamó a proveedores de pago/externos.

Los intentos iniciales del envoltorio shell se detuvieron en la preparación del entorno/captura; las salidas finales de compilación/pruebas se registran por separado, conservando avisos y sin eludir código ni aserciones. Se mantienen los cambios anteriores de planificador/apagado. La documentación emparejada registra estos resultados locales; siguen siendo necesarias la revisión final independiente y las siete comprobaciones alojadas sobre el siguiente commit exacto. Que una PR esté lista no equivale a aprobación del mantenedor, integración ni lanzamiento público.

<a id="log-writer-shutdown-ownership-2026-10-17"></a>
## Propiedad del apagado del escritor de registros (2026-10-17)

El servidor mantiene registros en memoria y los persiste mediante un escritor asíncrono de base de datos. Antes, cancelar el contexto del servidor no esperaba a que terminara ese escritor: un insert con contexto de fondo podía seguir activo mientras `Manager.Close` cerraba su pool. La última ejecución de backend del commit `14b5403` de PR #40 reveló esto en la prueba inalterada de apagado con puerto ocupado.

Flujo abreviado entre escritor y gestor:
```go
// Before: writer, then manager shutdown
pg.InsertLog(level, tag, text) // background context
return m.pg.Close()           // no writer join
// After: writer, then manager shutdown
pg.InsertLogContext(writeCtx, level, tag, text)
logSink.StopDB(m.pg)
return m.pg.Close()
```
La primera línea nueva permite cancelar el insert real; la restauración inicial del historial usa el mismo contexto. La segunda desvincula únicamente este pool exacto, cancela el SQL pendiente y espera la finalización del escritor fuera del bloqueo del receptor. Cada parada del mismo propietario comparte esa finalización; el pool de otro gestor no puede detenerlo. Solo entonces la tercera línea cierra el pool. Los métodos de BD existentes sin contexto delegan en el mismo SQL; siguen disponibles la entrega al búfer circular, stderr y SSE, y el canal de encolado nunca se cierra.

Un caso observado: la ejecución de backend [39028154916](https://github.com/cskwork/scopeweaver/actions/runs/39028154916), trabajo `117383132313`, falló en `TestMainServeFailureRunsShutdown` a las **2026-10-17 11:18:45 UTC**. Al fallo esperado de enlace por puerto ocupado siguieron el apagado y `[logsink] db write: sql: database is closed`. De forma independiente, un bloqueo real de tabla PostgreSQL reprodujo que `Manager.Close` retornaba con su escritor aún bloqueado sobre `14b5403` sin cambios (salida Go 1). El mismo caso exacto ahora pasa mientras el bloqueo sigue retenido, antes de cerrar el pool; no se filtra ningún diagnóstico.

| Comprobación local original R19 | Resultado observado |
| --- | --- |
| Regresión independiente sin cambios de insert bloqueado | 1 pasada, 0 fallos/omisiones; se conserva el RED de comportamiento de la base |
| Protección de rechazo SQL auténtico | 1 pasada, 0 fallos/omisiones; SQLSTATE `23514` visible, fila rechazada ausente y fila válida posterior persistida |
| Casos congelados `TestStopDB*` | 6 pasados, 0 fallos/omisiones: sin vinculación, insert bloqueado, pool ajeno, contexto cancelado, paradas simultáneas y continuidad de stderr |
| Prueba inalterada de apagado con puerto ocupado | Pasaron 1 ejecución más 20 repeticiones; se conserva el fallo esperado del hijo, salida Go del padre 0 |
| `go build ./...`, `go vet ./...` | Ambos con salida 0 |

Contraejemplo: un rechazo SQL real con contexto activo debe seguir notificándose; cancelar/esperar no lo convierte en un insert satisfactorio. El registro sigue siendo de mejor esfuerzo, sin nueva garantía de vaciado para líneas en cola. El caso concurrente comprueba la finalización conjunta; la espera compartida por cada solicitante también se verifica mediante revisión de código, sin afirmarla como un comprobador de ejecución separado. Todas las ejecuciones con BD usan bases de prueba desechables; los casos independientes ahora usan esquemas propios en la base desechable indicada y no necesitan privilegios CREATE DATABASE. No cambian los privilegios de producción, contratos HTTP/esquema ni pruebas existentes del planificador. Son resultados locales, no una afirmación de éxito completo del servidor nativo ni de lanzamiento: sigue la limitación heredada de privilegios de enlaces simbólicos en Windows y deben pasar las siete comprobaciones alojadas del siguiente candidato confirmado.

<a id="terminal-goal-completion-2026-10-17"></a>
## Finalización terminal del objetivo (2026-10-17)

Son resultados locales del candidato R17; la revisión final independiente del código y las condiciones alojadas del candidato exacto son independientes. La corrección compartida R17 mantiene un turno de prueba completado en la ruta normal del SDK y después termina localmente el planificador antes de otra solicitud al proveedor. Normalmente, `prove_goal` vincula un hecho/hallazgo confirmado con su objetivo y lo marca como cumplido; solo la prueba de todos los objetivos establece `GoalMet`. Antes, `Plan` esperaba a que terminara `captureRun` antes de devolver ese indicador al motor, por lo que el motor no podía completar mientras la sesión esperaba otra respuesta del modelo.

Flujo de control abreviado (`...` es el callback de actividad sin cambios):
```go
// Before
_, _, err = captureRun(ctx, opts, input, ...)
return tsx.GoalMet, tsx.Reason, err
// After
installTerminalProofGateOn(&opts, tsx)
_, _, err = captureRun(ctx, opts, input, ...)
return tsx.GoalMet, tsx.Reason, err
```
La primera línea anterior espera toda la sesión; la segunda expone el resultado del objetivo solo después. La línea añadida instala el límite de la siguiente solicitud antes de iniciar la captura. Deja terminar la respuesta actual, el resultado real de herramienta, el uso tardío y la transcripción; después proporciona un `end_turn` local sin contenido y con uso cero del proveedor, y omite el `Compactor.Pre` innecesario posterior a la prueba. Conserva el compresor seleccionado, `ContextView` opcional, comportamiento normal previo a la prueba/de recuperación y envoltorios existentes de proveedores/herramientas. El `ToolUseID` original y la validación de respuestas satisfactorias/fallidas impiden que un manejador retrasado atribuya una prueba antigua fallida a una respuesta posterior satisfactoria. Se conserva el origen de prueba más reciente mediante compare-and-swap; un origen anterior o desconocido no puede borrar una prueba válida más reciente. El registro de fallos se ejecuta antes de entregar un error, incluso si su consumidor se detiene inmediatamente.

Caso concreto de laboratorio: en el **candidato R11 de la CLI independiente**, no en un incidente original de producción, el nodo de hallazgo `5` probó el objetivo `2` a las **2026-10-17 18:36:24+09:00**. La salida del modelo seguía activa cuando el arnés inalterado de 420 segundos pausó a las **18:37:48+09:00**, 84 segundos después. Esa ejecución tuvo exactamente un GET, un intent y un hallazgo. El comprobante es `scopeweaver-cli/dist/pi-glm-flash-fix-20261017/r11-flash-high-digest.txt`; demuestra la espera de finalización, no la causa subyacente de latencia del proveedor. Contraejemplo: una prueba parcial/rechazada debe continuar normalmente; una respuesta de prueba fallida conserva su ruta real de error/recuperación y no puede habilitar la finalización local.

La verificación del código original usa proveedores controlados y PostgreSQL desechable, sin inferencia de pago:

| Comprobación del candidato local | Resultado observado |
| --- | --- |
| Paquete `agent` completo, R17 con caso de hecho propio corregido | 567 pruebas/subpruebas pasadas, 0 fallos, 0 omisiones; incluye streaming/sin streaming, prueba parcial/rechazada, cancelación del padre, error/recuperación, compresión/vista y casos de origen retrasado |
| `TestWireTools`, runtime estándar S7 y motor real S18 | 28 pasadas, 0 fallos, 0 omisiones; S18 persiste done tras una llamada real al proveedor, con comprobantes auténticos de prueba/terminación y uso exacto de entrada/salida/lectura de caché/escritura de caché **11/7/3/2**, también en la transcripción sin procesar |
| Protecciones reales de pausa/error del proveedor en S18 | Los comprobantes fieles de error/causa conservan el uso emitido **11/0/3/0** para pausa del usuario y **11/7/3/2** para fallo del proveedor; se observa la prioridad del estado entre pausa posterior a la prueba y finalización de tarea sin añadir una nueva aserción de política |
| `locale` completo, compilación y vet | 19 pruebas pasadas, 0 fallos, 0 omisiones; `go build ./...` y `go vet ./...` con salida 0 |

La base original S18 guardada falló porque intentó la llamada 2 al proveedor y no tuvo resultado terminal/finalización; sus protecciones de comprobantes de cancelación/error del proveedor pasaron. La regresión R14 independiente e inmutable también falló con calls=2 en lugar de 3 cuando un manejador antiguo fallido tomó la identidad de una respuesta posterior; el candidato conserva la regresión de tres llamadas. Después, el revisor reprodujo el orden inverso en el candidato R16 original: un manejador antiguo de response-1 fallida terminó después de una prueba válida de response-2 y borró su origen, causando una tercera solicitud innecesaria. La superposición inalterada de sesión real fallaba con calls=3 antes de R17 y ahora pasa con calls=2/origin=2 y un nuevo resultado de prueba auténtico. Ambos órdenes de manejadores retrasados pasan en las [regresiones de límites](../../agent/terminal_boundary_test.go); la [regresión del motor real](../../server/engine_terminal_proof_test.go) conserva por separado las comprobaciones de finalización/comprobantes/uso. La primera ejecución integrada de agent falló solo porque un caso nuevo esperaba que `CreateExploration` creara un hecho de origen. R16 prepara su propio hecho confirmado; ni las aserciones ni el código de producción cambian por esa corrección del caso. Se conservan los comprobantes anteriores de fallo.

S18 ahora reutiliza el auxiliar de esquema propio de S7 dentro de la base desechable indicada, cerrando los almacenes antes de eliminar su propio esquema; basta un rol de prueba propietario de la base con NOCREATEDB y los privilegios de producción no cambian. `TestChatBranchCopyRejectsDestinationSymlink`, sin cambios, sigue fallando localmente en `server/chat_history_test.go:270` porque Windows deniega crear enlaces simbólicos. No se afirma éxito completo del servidor nativo ni de las siete comprobaciones alojadas del candidato exacto; la entrega CI/frontend/navegador sigue pendiente. Root se encarga por separado de la aceptación de carreras y en vivo; esta sección solo informa de estas comprobaciones locales originales.

<a id="plannertool-contract-and-stored-stock-compatibility-2026-10-17"></a>
## Contrato del planificador/herramientas y compatibilidad de valores estándar almacenados (2026-10-17)

Resultados locales en Windows de `fix/glm-agent-contracts`, basado en el commit `dev` `d7365ec`. La explicación de `parent_ids` del planificador ahora coincide con el manejador existente que solo admite hechos/hallazgos, y los metadatos de `add_intent` por lotes describen sus campos y el `summary` obligatorio. Las indicaciones del planificador/trabajador conservan la validación nueva y las repeticiones de pruebas, exigen evidencia para cada condición del objetivo y terminan las rondas solo tras un despacho satisfactorio o la prueba confirmada del objetivo final. Los recursos EN/KO/zh-Hans están emparejados.

Los prompts estándar anteriores exactos del planificador/trabajador y los esquemas `add_intent` se resuelven a los valores predeterminados corregidos en ejecución. No cambian las ediciones del usuario, vinculaciones, indicadores de activación, filas almacenadas ni historial de prompts; no hay migración de BD de producción ni cambio semántico del manejador. Consulta las [regresiones de contrato](../../agent/planner_contract_test.go), [regresiones de compatibilidad estándar](../../agent/legacy_stock_compat_test.go) y [verificación HTTP/runtime con PostgreSQL](../../server/tools_stock_schema_test.go).

La verificación con pruebas primero compiló y produjo **8 fallos de aserción de agent de nivel superior** y **9 fallos de subcasos S7 del esquema estándar** en la base. La extracción temporal de la cola sin cambios y el resolutor de paso directo usados solo para compilar las pruebas se retiraron antes de aplicar la corrección. Resultados del candidato:

| Comprobación local | Resultado observado |
| --- | --- |
| Paquete `agent` completo con PostgreSQL desechable | 549 pruebas/subpruebas pasadas, 0 fallos, 0 omisiones |
| Paquete `locale` completo | 19 pasadas, 0 fallos, 0 omisiones |
| Comprobaciones específicas del servidor de idiomas/herramientas/procedencia | 11 pasadas, 0 fallos, 0 omisiones; 1,985 claves de producción conservan las comprobaciones de integridad de tres idiomas sin recurrir a alternativas |
| `TestExplorationDirectSourceReadView` | Pasó con PostgreSQL desechable |
| `TestStockMetadataRuntimePostgres` junto con `TestWireTools`, `-count=2` | 48 pruebas/subpruebas pasadas, 0 fallos, 0 omisiones; manejadores HTTP reales de loopback, resolución en ejecución y lectura posterior de filas/historial almacenados |
| Escenarios guardados de herramientas públicas | S11: pasaron los 10 casos de callback del agente principal/planificador; S12: pasaron genealogía histórica, procedencia de fuentes y rechazo de padres no relacionados/indirectos; S3: pasaron rechazo de kind inválido/summary en blanco, errores parciales y vinculación de origen omitida/vacía/null |
| `go build ./...`, `go vet ./...`, espacios del diff | Pasaron |

`TestStockMetadataRuntimePostgres` y `TestEngineTerminalProofPersistsAuthenticReceipt` exigen un `ARTEX_TEST_DSN` desechable explícito que seleccione `scopeweaver_test_*`. El rol de prueba propietario de la base puede tener **NOCREATEDB**, igual que en `make test-db`. Cada caso crea su propio esquema `stock_*` y fija el `search_path` de cada conexión del pool exclusivamente a ese esquema, conservando la base seleccionada y las demás opciones de conexión. Las pruebas cierran los almacenes antes de eliminar solo su esquema propio con `CASCADE`; se notifican errores de limpieza, pero eliminar el esquema no detecta por sí mismo conexiones inactivas filtradas. `TestStockFixtureIsolationAndCleanup` comprueba varias conexiones del pool, invisibilidad/preservación de tablas padre y eliminación de esquema. Reproduce el aislamiento con `go test ./server -run '^Test(WireTools|StockMetadataRuntimePostgres|StockFixtureIsolationAndCleanup|EngineTerminalProofPersistsAuthenticReceipt)$' -count=2`. Los privilegios y el esquema de la BD de producción no cambian. Esta reparación de los casos sustituye el requisito anterior de base hija/CREATE DATABASE.

El primer intento del arnés S3 falló al preparar una pista con estado inválido `open`; se corrigió el caso al estado existente `active`, sin cambiar producto ni aserciones, antes de la repetición satisfactoria. La preparación final de S7 también pasó dos veces después de que otras pruebas poblaran la base padre. Estas comprobaciones establecen el comportamiento del contrato y la compatibilidad, no la causa del duplicado/timeout anterior de Flash en vivo. Siguen pendientes en esta rama `go test ./...` completo, las comprobaciones de frontend/navegador y los siete trabajos CI obligatorios del candidato exacto; no se afirma integración ni lanzamiento.

## Correcciones de ejecución compartida desde verificación CLI independiente (2026-10-17)

Rama candidata `fix/cli-discovered-runtime-bugs`, basada en el commit `dev` `3f0ef25`. Son resultados locales; las comprobaciones alojadas de la revisión exacta de la PR son independientes. Este cambio no añade la CLI independiente ni proveedores por suscripción a la aplicación original.

- Las herramientas HTTP personalizadas ahora usan la URL completa del proxy de grabación devuelta por `Manager.ProxyAddr`. Antes: `"http://" + addr` convertía `http://127.0.0.1:<port>` del caso controlado en `http://http://127.0.0.1:<port>`, por lo que fallaba antes de llegar al proxy. Después: `proxyStr = addr` conserva la URL. Pasan las rutas reales directa, explícita y automática; la automática fallaba antes de corregir.
- El inicio comprueba ahora que una ruta de transcripción existente sea un directorio antes de leerla. La regresión de archivo normal en Windows fallaba antes; ahora rechaza la ruta conservando el archivo. Los directorios ausentes siguen siendo válidos. El diagnóstico nuevo está en inglés, coreano y chino simplificado.
- El apagado cancela trabajo, vacía HTTP y espera chats admitidos, respuestas paralelas y publicación de puntos de control antes de cerrar almacenes. Antes, vaciar solo HTTP podía dejar finalizadores en segundo plano escribiendo tras cerrar la BD. Los errores finales de escritura de instantánea/contexto se propagan, conservan pendientes y hacen fallar explícitamente el apagado; registros eliminados/en conflicto siguen siendo inocuos. Este vaciado limitado no afirma cubrir cada planificador o productor del motor.
- Los datos de la prueba de uso cancelado crean ahora la tabla opcional de uso existente, como la inicialización de producción. El fallo por tabla ausente se reprodujo antes de corregir los datos; la prueba corregida con BD nueva conserva la aserción exacta de uso.

Las comprobaciones reales de apagado PostgreSQL pasaron 11 pruebas/subpruebas, incluidas persistencia tras cerrar/reabrir y fallos finales de escritura inyectados. Las del proceso principal pasaron 4 pruebas, cubriendo salida normal 0, reinicio 75, fallos limitados y limpieza ante fallo real de servicio por puerto ocupado. Pasaron regresiones específicas de proxy/inicio/idiomas. La ejecución completa del backend en Windows tuvo 1,975 aprobadas, 2 fallidas y 15 omitidas: un fallo era una limitación de privilegios sin cambios al crear enlaces simbólicos; el otro reveló el recuento esperado del catálogo, actualizado de 1,980 a 1,982 por los dos diagnósticos nuevos sin quitar comprobaciones de integridad. El catálogo corregido se verifica por separado. Las comprobaciones Linux/alojadas deben establecer los resultados restantes.

Frontend: 65 pruebas aprobadas y exportación estática correcta con Git Bash para el script npm de estilo POSIX existente. Adaptador de agentes: 7 pruebas y comprobaciones de sintaxis aprobadas. Análisis estático específico del backend correcto. No se usaron BD de usuario, objetivos reales en línea ni entregas de notificaciones; cada ejecución que escribía en BD creó y eliminó una `scopeweaver_test_*` independiente. No se afirma integración ni publicación de versión.

## Controles de chat y contexto (2026-10-16)

El autor de la [PR #29](https://github.com/cskwork/scopeweaver/pull/29) comunicó estas comprobaciones sobre `6366429` original. Son sus resultados históricos de Linux, anteriores a la revisión conjunta de PR #29/#30; no establecen el estado del candidato posterior ni la copia de adjuntos en Windows.

| Comprobación | Resultado |
| --- | --- |
| Suites de carreras con PostgreSQL 17 desechable (`db`, `server`, `agent`) | 667 pruebas de nivel superior aprobadas; sin carreras detectadas |
| Paquetes Go y análisis estático | `go test ./... -count=1` y `go vet ./...` correctos |
| Pruebas frontend, traducciones, auditoría y exportación estática | 65 aprobadas; 3,263 claves EN/KO/zh-Hans pasaron paridad; 0 hallazgos de auditoría; TypeScript y 31 páginas correctos |
| Traducciones backend | 1,978 claves de producción pasaron integridad, paridad y printf en tres idiomas |
| Control de navegador de chat | 60 escenarios aprobados en EN/KO/zh-Hans, 1440/390 px y chat independiente/de tarea; sin errores de página no capturados |
| Prueba básica y panel con aplicación integrada | E2E básica y 14 escenarios de panel aprobados; sin errores de navegador no capturados |
| Adaptador de agentes | 7 pruebas unitarias, sintaxis y 3 escenarios MCP/Pi aprobados |
| Instalador y empaquetado | 8 casos de instalador, 11 Compose, 5 archivos de empaquetado por idioma (EN/KO) y 6 pruebas de avisos de versión aprobados |

Los casos de navegador de chat cubren edición y orden de cola, reconocimiento de reinicio, reintento/edición/ramificación, inspección y compresión de contexto, corrección de herramientas y pegado de documentos. Las regresiones backend cubren persistencia del idioma aceptado, errores de comprobación previa, intercambios de herramientas interrumpidos, preparaciones de rama abandonadas y atribución de lectura de documentos.

Puedes inspeccionar el contexto actual del modelo separado del uso facturado de conversación. La compresión conserva los intercambios recientes completados que elijas; mantienes todo el historial visible y el uso de intentos anteriores.

El colaborador informó de una revisión que encontró seis defectos de integración, seguidos de correcciones de admisión de entradas generadas y recuperación de comprobaciones, rutas de adjuntos tras compresión, historial visible antiguo y controles de entrega de correcciones. La prueba de adjuntos elimina la conversación principal antes de las siguientes instrucciones de la hija; otra comprueba que la reasignación conserva el texto literal del manifiesto. Informó de una regresión fallida antes de cada corrección y de una repetición local completa. La aserción del total de razonamiento del panel se limitó a la fila de resumen porque el texto emergente usa la misma etiqueta; la repetición comunicada pasó 14 escenarios de panel.

Las pruebas de inferencia comunicadas usaron modelos loopback y BD desechables. La ejecución Windows, proveedores de pago y CI remota requieren verificación aparte. El colaborador usó Python 3 para el caso de avisos, como exige CI.

## Candidato de refuerzo nativo — 2026-10-15

Comprobaciones locales e independientes de `feat/ollama-zero-egress-hardening`, basado en main `33b131e`. Son resultados del candidato; CI remota de revisión exacta y entrega de PR siguen pendientes. Las secciones históricas siguientes conservan sus resultados originales.

| Comprobación realmente ejecutada | Resultado |
| --- | --- |
| Ejecución Go independiente, 20 paquetes con BD PostgreSQL desechables aisladas | 1,366 pruebas/subpruebas aprobadas, 0 fallos; `enrich` no tiene pruebas; 2 omisiones opcionales (`TestAgentAdapterE2E`, `TestLiveContextReview`) |
| Adaptador de agentes, comprobado aparte | 7 pruebas unitarias y 3 escenarios reales MCP/Pi aprobados; revisión de proveedor externo real no ejecutada intencionadamente |
| Suites específicas backend `-race` | Correctas sin carreras: transporte, protocolo/uso, contexto, selección/fallo de origen, búsqueda/actualización y admisión |
| `go vet ./...` | Correcto |
| Pruebas unitarias/localización frontend y exportación estática | 45 aprobadas; 3,168 claves EN/KO/zh-Hans conservan paridad, marcadores y etiquetas; TypeScript y las 31 páginas estáticas correctos |
| Catálogo backend | 1,943 claves de producción pasaron integridad/paridad directa en tres idiomas y printf |
| Instalador, Compose y avisos | 8 casos de instalador, 11 de credenciales Compose, avisos en 5 archivos de empaquetado por idioma (EN/KO) y 6 pruebas de avisos de versión bajo Python 3.12 aprobados |
| Compilaciones estáticas y dependencias | Compilación integrada macOS y cruzada Windows amd64 correctas; sin cambios de `go.mod`/`go.sum`; ejecución Windows nativa no realizada |
| Prueba básica de navegador integrado | Interfaz final recompilada: 60 aprobadas (48 básicas + 12 de panel), 0 fallos. Capturas asentadas escritorio/móvil y límites horizontales correctos en EN/KO/zh-Hans: panel [0,390] a 390px y [864,1440] a 1440px; los 20 contenidos críticos caben. Se reprodujo y corrigió el recorte previo por ancho mínimo de 420px |

La [regresión de prioridad HTTP nativa](../../server/ollama_priority_test.go) retiene al trabajador 1 dentro de `/api/chat`, pone en cola trabajadores 2/3 y después invoca el juez real conectado sin marcador de urgencia del llamante. Pasó con `-race` con orden HTTP **trabajador 1 → juez → trabajador 2 → trabajador 3**, concurrencia máxima 1, esquema/veredicto nativo válido, todas las llamadas terminadas y sin fuga de huecos. Las pruebas de cola existentes cubren roles FIFO, cancelación y reducción de límites.

La [regresión de fallo de origen nativo](../../server/provider_origin_failure_test.go) falló antes de corregir: una herramienta nativa pendiente seguida de error de transmisión podía heredar `allow` global de nube. Tras corregir, los errores de enrutamiento Stream/Complete y finales conservan el perfil nativo concreto, política del modelo de 90 segundos/`ask` y caducidad humana de 300 segundos/`deny`, sin llamar al revisor. La conmutación real a nube sí borra la preferencia nativa. La regresión y las suites de carreras afectadas pasaron.

Las comprobaciones reales de API autenticada alternaron modo estricto **false → true → false** contra un proxy local que nunca reenvía solicitudes. Los cuatro motores (`ddgs`, `brave-free`, `tavily`, `deepseek`) llegaron a ese receptor solo con modo estricto desactivado; cada uno produjo cero solicitudes adicionales en modo estricto. Se bloquearon comprobaciones y aplicación de actualizaciones estrictas conservando estado local. Las [regresiones de actualización](../../server/update_policy_test.go) también pasaron para caché, redirecciones, lectura de cuerpos en directo y limpieza/finalización de fallos tras bloquear una descarga asíncrona admitida con un cambio posterior. No demuestran control de salida de todo el host.

Ollama **0.32.15** del host, con `qwen3:4b` existente y **presupuesto de salida de 4,096 tokens**, pasó diagnósticos nativos, esquema decision/comment, repetición estructurada de llamada/resultado de herramienta, uso y contexto solicitado de 32K (`/api/ps`: **32,768**). Los fallos anteriores con presupuestos de 256/512 tokens siguen en las pruebas; no se descartaron como ejecuciones correctas. Los valores de carga/velocidad/descarga son una instantánea en caliente, no una prueba comparativa ni garantía de permanencia en memoria. No se descargaron modelos ni cambió configuración del host; no se afirma finalización autónoma ni calidad de evaluación de seguridad.

La cobertura del navegador incluye diagnósticos/avisos en tres idiomas, estado de actualización estricta, protección de carga/fallo del juez y flujos existentes de panel/uso. La revisión visual reprodujo un editor de ancho mínimo fijo de 420px que sobresalía 30px a la izquierda de una pantalla de 390px. El mínimo ahora cabe; la aplicación recompilada pasó las 60 comprobaciones y las seis capturas de escritorio/móvil muestran etiquetas legibles y contenido asentado dentro de la pantalla. Una recompilación integrada independiente coincidió byte a byte con el binario del navegador. CI remota de revisión exacta y lectura posterior de PR son controles de entrega separados. No se afirma integración, versión, llamada de proveedor de pago, escaneo externo ni notificación real.

## Seguimiento de autenticación de proveedores — 2026-10-15

Las regresiones locales en `fix/provider-env-auth`, basado en main integrado `38fde23`, reproducen y corrigen dos fallos de pasarelas locales autenticadas. La alternativa de clave al iniciar y la prueba de conexión sin clave devolvían cada una HTTP 401 con la implementación anterior para Anthropic, OpenAI Chat Completions y OpenAI Responses. Los mismos casos pasan con las correcciones.

| Comprobación | Resultado |
| --- | --- |
| [Autenticación HTTP local `FromEnv → NewProvider`](../../agent/provider_env_test.go) | 21 escenarios aprobados: 18 de formato genérico sobre claves de proveedor/archivo, sustituciones modernas/antiguas y vacías explícitas; 3 nativos sobre aislamiento de claves de nube, token Ollama y sustitución vacía explícita |
| [Controlador de prueba de conexión con casos HTTP locales](../../server/llm_gateway_test.go) | 16 escenarios aprobados: 15 genéricos sobre servicio normalizado coincidente, endpoint/formato/proxy diferentes y alternativa no local conservada; 1 nativo que rechaza heredar clave de nube activa |
| Ambos grupos específicos con `-race` | Correctos, sin carreras |

Los **37 casos** anteriores cuentan escenarios hoja, excluyendo pruebas principales. Los fallos de autenticación se reprodujeron antes de cada corrección y las mismas pruebas pasaron después. Las direcciones públicas de documentación de casos no locales usan un proxy local explícito; no se contacta a proveedores públicos.

Una prueba de conexión sin clave ni ID de perfil puede reutilizar la clave del servicio local activo solo si coinciden formato genérico, URL base normalizada y proxy, incluso en un borrador nuevo. Los servicios locales diferentes y Ollama nativo permanecen aislados. Guardar un perfil nuevo sin clave no importa credenciales del entorno.

Estas comprobaciones establecen los límites de protocolo y credenciales probados, no disponibilidad de cuentas de pago ni todas las funciones históricas. Los resultados de suite completa, navegador, CI, main integrado y artefactos de versión posteriores son controles separados. No se hicieron llamadas a proveedores de pago.

## Seguimiento de selección de modelos instalados — 2026-10-15

Verificación local de selección Ollama predeterminada de perfiles nuevos en `feat/ollama-native-flexible-providers`. Sustituye el comportamiento anterior de plantillas: elegir Ollama ahora realiza descubrimiento de modelos de solo lectura. La instantánea histórica de 37 comprobaciones de navegador siguiente es anterior; su aserción «sin solicitudes implícitas» es histórica, no el contrato actual de Ollama.

| Comprobación | Resultado |
| --- | --- |
| Pruebas unitarias y localización frontend (`npm --prefix web test`) | 45 aprobadas, 0 fallos |
| Exportación estática de producción (`npm --prefix web run build:static`) | Correcta, incluido TypeScript y las 31 páginas |
| Descubrimiento real de modelos del host mediante navegador y backend autenticado | Correcto en perfil nuevo sin guardar, Ollama en mismo equipo, proxy y clave API vacíos |
| Prueba básica completa ampliada en binario integrado y BD nueva `scopeweaver_test_default_final` | 43 aprobadas, 0 fallos (31 básicas + 12 de panel). Cubre valores instalados EN/KO en las tres plantillas, formato directo, compatibilidad de nube, selección solo de metadatos, detección fallida/vacía, edición manual/rechazo de respuestas tardías, reapertura de borradores/perfiles existentes y regresiones de configuración, acceso, localización, SSE y panel |

`/api/tags` del host real devolvió `qwen2.5-coder:7b`, `qwen3:4b` y `embeddinggemma:latest`. El navegador seleccionó **`qwen2.5-coder:7b`**, primer modelo instalado devuelto que declara `tools`. Sus metadatos indicaron `completion`, `tools`, `insert`, 7.6B parámetros y cuantización Q4_K_M. `qwen3:4b` también declaró `tools`; `embeddinggemma:latest` solo `embedding`.

El navegador envió exactamente una operación LLM, `POST /api/llm/models`, al endpoint del host sin clave ni proxy. El borrador nuevo se cerró sin guardar, activar, probar conexión ni inferir. No cambiaron la configuración de modelos ni del equipo. Esto demuestra selección y descubrimiento de metadatos, no ejecución de herramientas ni fiabilidad autónoma.

## Rama histórica de Ollama nativo — 2026-10-15 (local a la rama, no una versión)

Comprobaciones locales de `feat/ollama-native-flexible-providers`, registradas antes de abrir la PR. La CI remota se sigue por separado en GitHub Actions; nada de esto es un resultado de publicación, integración ni CI.

| Comprobación | Resultado |
| --- | --- |
| Suites Go, 20 paquetes ejecutados por separado, cada uno con BD PostgreSQL 16 desechable nueva | 1,197 pruebas/subpruebas aprobadas; 2 omisiones opcionales (`TestAgentAdapterE2E`, `TestLiveContextReview`) |
| Adaptador de agentes, ejecutado aparte | 3 escenarios reales MCP/Pi aprobados |
| `go vet ./...` | Correcto |
| Pruebas unitarias y localización frontend | 43 aprobadas; 3,159 claves únicas en inglés, coreano y chino simplificado |
| Catálogo de mensajes backend | 1,938 claves únicas en cada idioma; printf y paridad correctos |
| TypeScript y exportación estática de producción | Correctos, recompilados tras cambios finales de interfaz de estado |
| Auditoría de dependencias frontend (`npm audit`) | 0 vulnerabilidades |
| Compilaciones | Binario integrado `CGO_ENABLED=0` recompilado tras cambios finales de estado; compilación cruzada Windows amd64 `CGO_ENABLED=0` correcta (ejecución Windows nativa no realizada) |
| Prueba básica de navegador (`npm run smoke`) con binario integrado real y BD nueva `scopeweaver_test_ollama_final_health` | 37 aprobadas, 0 fallos (25 básicas + 12 de panel). Cubre tres plantillas Ollama EN/KO sin solicitudes implícitas, guardar/activar sin clave, estado de compatible local sin clave frente a nube pública sin clave, formatos de nube y GLM editable, metadatos/avisos de herramientas por modelo, comando exacto de descarga de modelo ausente, diagnósticos de herramienta/contexto/carga/velocidad y política exclusivamente local desactivada por defecto, además de regresiones EN/KO/zh-Hans de configuración, acceso, idioma, borradores, SSE y uso de aprobación |
| Credenciales Compose | 11 casos aprobados en contenedores reales `network_mode: none`, mediante entorno del contenedor y `FromEnv` |
| Instalador y empaquetado | 8 casos de instalador aprobados en total, incluida selección inglesa/coreana; 5 archivos aprobados en cada idioma |
| Avisos de versión (Python 3.12) | 6 pruebas aprobadas |
| Revisión independiente de seguridad/corrección backend | Correcta, incluidas política exclusivamente local DeepSeek, comprobaciones CIDR en directo y clasificación 404 |

### Ejecución real de Ollama

Ollama 0.32.15 del host con `qwen3:4b`, mediante API nativa en transmisión:

- En una comprobación de transmisión del SDK con una herramienta sintética, el modelo llamó a `record_fact`; ScopeWeaver la ejecutó localmente, repitió el resultado y el turno terminó. No era un `record_fact` de la BD de plataforma.
- Se solicitaron 32K de contexto; `/api/ps` indicó 32,768 asignados.
- Uso exacto de esa comprobación: turno en transmisión 186 tokens de entrada / 750 de salida; turno final 826 de entrada / 193 de salida.
- Guardar/activar sin clave mediante API real fue correcto. Los diagnósticos indicaron `tool_call_verified` y `context_verified` true, contexto 32,768, carga 2.225375 ms, 29.231 tokens/s y latencia 18,958 ms. **Fue una ejecución en caliente, no una comparación de arranque en frío ni generalización de hardware.**
- Pasaron la indicación coreana exacta de descarga para modelo ausente y el interruptor estricto que bloquea una IP pública.
- **Prueba Bash de trabajador de tarea de plataforma:** un trabajador real (tarea 1) con `qwen3:4b` (32K solicitados, 32,768 asignados; concurrencia 1; keep-alive 30m; pensamiento desactivado; máximo 4,096 tokens) hizo la llamada Bash nativa `call_219yv72x`. Las actividades 9 (llamada) y 10 (`tool_result`) comparten ese ID sin error y la salida contenía `LOCAL_TASK_OLLAMA_OK`. El registro contabilizó exactamente 6,704 tokens de entrada / 4,096 de salida y la siguiente llamada nativa repitió el mismo `tool_call_id`. La intención se creó manualmente con `db.AddIntent`, no mediante LLM. La salida también contiene un `\n` literal por el escape del caso; no se infiere un problema de saltos de línea ni del modelo. Se usó un binario compilado del código sin confirmar de la rama anterior a los cambios finales de interfaz/DTO.

### La fiabilidad de tareas autónomas no está validada

Un diagnóstico de llamadas a herramientas correcto demuestra la ruta de protocolo, no que el modelo complete tareas.

- `qwen2.5-coder:7b` declara `tools`, pero escribió JSON con forma de herramienta en `assistant.content` en vez de `tool_calls` del protocolo, por lo que no se ejecutó Bash ni herramientas de hechos. ScopeWeaver lo conservó correctamente como texto.
- `qwen3:4b` como agente principal, ante instrucciones largas, deliberó y escribió el marcador esperado como texto. Como trabajador sí ejecutó Bash (arriba), pero la tarea se pausó a los 360 s sin `record_fact` ni finalización autónoma; una llamada correcta no demuestra finalización autónoma.

**No se afirma finalización autónoma de extremo a extremo ni calidad de evaluación de seguridad para ningún modelo local, y los modelos de clase 7B no están validados como agente completo.** Los [niveles de hardware](llm-providers.md#niveles-de-hardware) son estimaciones, no comparaciones medidas.

### No probado

Sin llamadas de nube de pago, escaneos externos, notificaciones reales, publicación, despliegue, versiones ni integración. Anthropic, OpenAI Chat/Responses y pasarelas personalizadas se verificaron mediante contratos de protocolo, regresiones y flujos editables de navegador, no cuentas reales. Todas las BD fueron desechables y no cambió la configuración Ollama del host. La política opcional `local_llm_only` está desactivada por defecto; los servidores aún deben establecer `OLLAMA_NO_CLOUD` aparte.

## Ejecutar pruebas con base de datos

Las suites `db`, `server` y `agent` necesitan PostgreSQL. Resuelven su
cadena de conexión **solo** desde la variable de entorno `ARTEX_TEST_DSN` y
nunca usan como alternativa el DSN de la aplicación (`ARTEX_PG_DSN` /
`config.json`). Es intencionado: evita que `go test` modifique la base de datos
activa/de producción de la aplicación.

- **Sin `ARTEX_TEST_DSN`** → se omiten todas las pruebas con BD. No se toca ninguna base
  de datos.
- **`ARTEX_TEST_DSN` con BD desechable** → las suites se ejecutan contra ella.
- **`ARTEX_TEST_DSN` resuelve a la misma BD que la aplicación** (mismo host,
  puerto y nombre) → la ejecución falla inmediatamente con
  `refusing to run tests against the live database; point ARTEX_TEST_DSN at a
  throwaway DB`, antes de que se conecte ninguna prueba.

La forma más sencilla de ejecutarlas es:

```bash
make test-db
```

`make test-db` (mediante `scripts/test-db.sh`) crea una BD desechable llamada
`scopeweaver_test_<pid>` propiedad del rol `sw_user_test`, dirige
`ARTEX_TEST_DSN` a ella, ejecuta `go test ./db/ ./server/ ./agent/ -count=1` y
elimina la BD al salir, incluso si fallan las pruebas. Sustituye `TEST_DB_ROLE`,
`TEST_DB_HOST` o `TEST_DB_NAME` según tu PostgreSQL local. El nombre de BD
debe empezar por `scopeweaver_test_` (o `scopeweaver_adapter_` para la
prueba E2E del adaptador de agentes).

Para ejecutar un subconjunto contra una BD gestionada por ti, exporta `ARTEX_TEST_DSN`
y llama directamente a `go test`:

```bash
export ARTEX_TEST_DSN='postgres://sw_user_test@/scopeweaver_test_1?host=/tmp&sslmode=disable'
go test ./server/ -run TestSomething -count=1
```

## Candidato v0.2.1 — 2026-10-11

El candidato de versión Windows se compiló con el frontend estático real y se probó contra BD PostgreSQL 16.15 desechables. Estos resultados sustituyen las limitaciones históricas de dependencias y ejecución Windows siguientes; el registro anterior se conserva para trazabilidad.

| Comprobación | Resultado |
| --- | --- |
| Auditoría de dependencias frontend | 0 vulnerabilidades de cualquier gravedad, incluidas dependencias de desarrollo |
| Pruebas de regresión y localización frontend | 26 aprobadas; sin omisiones |
| Exportación estática de producción | TypeScript correcto; 31 páginas generadas |
| Suites Go nativas Windows, 18 paquetes | 1,016 pruebas/subpruebas aprobadas; 0 fallos; 13 omisiones existentes de plataforma/opcionales |
| Suite completa de servidor Windows nativo | 311 pruebas/subpruebas aprobadas; 3 omisiones existentes opcionales/de plataforma |
| Adaptador de agentes | 7 pruebas unitarias y sintaxis correctas; 3 escenarios MCP/Pi aprobados contra backend real autenticado y BD nueva |
| Instalador local | Los 4 casos aislados aprobados: instalación nueva, sustitución de archivo obsoleto, fallo de dependencia y fallo de compilación frontend |
| Catálogo de traducción backend | 1,868 claves de producción pasaron integridad inglés/coreano y argumentos printf |
| Aislamiento de notificaciones | La regresión determinista de trabajador obsoleto falló antes de corregir y pasó después; selección de gestión/notificación pasó 5 ejecuciones consecutivas |
| Pruebas Windows | Suite completa con PostgreSQL correcta, incluidas escrituras simultáneas, fallos de confirmación y rutas largas |
| Python y archivos de habilidades Windows | Ejecución Python real y alternativa a iniciador inutilizable correctas; rutas Unicode anidadas pasaron por listado y controladores de lectura |
| E2E de navegador en binario Windows integrado | 8 comprobaciones aprobadas: protección ante caída de configuración en ambos idiomas, configuración/acceso, SSE del mismo origen, aislamiento de API de uso, ambas tarjetas de uso localizadas y sin errores de página no capturados |

Los recuentos Go incluyen pruebas principales y subpruebas. Las 13 omisiones nativas incluyen diez casos Unix de actualización automática, un privilegio de enlace simbólico no disponible, la integración opcional del adaptador y el proveedor real. La integración se comprueba por separado y CI Linux cubre casos Unix. No se desactivaron pruebas para obtener estos resultados.

Las regresiones reproducen el fallo original Windows de sincronización de directorios, diferencia de separador de localización, iniciador Python inutilizable, fuga de trabajador de notificaciones y orden de limpieza de perfiles. El auxiliar SQL E2E ahora coloca opciones antes de la URL de conexión; una comprobación directa de caída/restauración confirmó que psql de Windows ejecuta las sentencias.

El control de versión exige además trabajos `frontend`, `windows`, `backend`, `agent-adapter` y `smoke` de navegador antes y después de integrar. Los resultados de publicación, archivos de plataforma y sumas están en la [versión v0.2.1](https://github.com/cskwork/scopeweaver/releases/tag/v0.2.1); este registro fuente no afirma que los trabajos posteriores o recursos descargados ya hayan pasado.

Se conservan las rutas existentes de intérprete Python. Si una instalación anterior guardó un alias de iniciador inutilizable, elige un intérprete ejecutable en Ajustes. Las pruebas Windows nativas no establecen actualización automática Windows ni ejecución macOS. No se usaron proveedores reales de pago, objetivos externos ni destinatarios reales de notificaciones.

## Verificación histórica — 2026-10-03

Validación registrada el **2026-10-03** para la edición inglés/coreano de ScopeWeaver, basada en el commit ARTEX original `160fe13`. Las pruebas usan BD PostgreSQL desechables y casos locales. Es entrega de código fuente; no se han publicado binarios ni imágenes de contenedor.

| Comprobación | Resultado |
| --- | --- |
| Pruebas de regresión y localización frontend | 21 aprobadas; TypeScript correcto |
| Exportación estática de producción y binario Go integrado | Compilados correctamente |
| Suites de regresión Go, 18 paquetes | 986 pruebas/subpruebas aprobadas; 1 heredada falló; 1 de proveedor externo omitida |
| Suite completa de servidor | 285 pruebas/subpruebas aprobadas; solo se omitió la prueba opcional de proveedor real |
| Catálogo de traducción backend | 1,865 claves de producción pasaron integridad inglés/coreano y argumentos printf |
| Auditoría de texto chino redactado | Sin código/documentación redactados ni comentarios Go sin traducir; excepciones deliberadas de compatibilidad/datos conservadas |
| Cobertura de navegador de escritorio | 24 rutas por idioma; sin etiquetas ausentes ni desbordamiento horizontal de página |
| Cobertura de navegador móvil | Panel, tareas, hallazgos y ajustes en ambos idiomas; sin desbordamiento horizontal de página a 390px |
| Pantallas reales de aplicación | Formularios de acceso y notificaciones comprobados en ambos idiomas; validación e inicialización verificadas |
| Cambio de idioma | Etiquetas de navegación/estado se actualizan; selección y borradores sin enviar conservados |
| Localización HTTP | Exportación Markdown/CSV, negociación de idioma, preferencia guardada y rechazo de idioma no válido comprobados |
| Contrato de proveedor GLM-5.3 | 21 pruebas/subpruebas específicas aprobadas con HTTP/SSE local; cubiertas ambas rutas de endpoint, razonamiento y repetición de llamadas a herramientas |
| Flujo de plantillas GLM en navegador | Ambas plantillas comprobadas en ambos idiomas; nombre/clave/proxy introducidos conservados; sin guardar, activar ni solicitar al proveedor |
| Scripts de empaquetado e inicio | Sintaxis shell, casos bilingües de empaquetado y 10 casos de supervisor aprobados |
| Procedencia y dependencias | Licencia original sin cambios; versiones de dependencias y módulo Go sin cambios; enlaces de documentación y diferencias correctos |

Los recuentos incluyen pruebas Go principales y subpruebas. Las ejecuciones de paquetes usan BD independientes inicializadas con las mismas tablas adicionales de medición que el inicio normal del servidor. Las pruebas y comprobaciones originales siguen activadas.

### Problemas heredados entonces

`TestGraphOverviewExpandsAssociatedCompanyScope` falló con `task scope missing: <nil>` tanto en ScopeWeaver como en el código original intacto. Esperaba alcance detallado en el resumen del grafo, que el commit original `06a43f3` hizo compacto deliberadamente. El commit `75a706d` lo sustituyó por `TestGraphOverviewKeepsAssociatedCompanyScopeCompact`, que comprueba el resumen compacto y verifica directamente el alcance almacenado de empresa. La CI completa del backend ha pasado desde entonces.

`TestTaskMetadataPatchReturnsRenameAndPin` produjo antes un fallo intermitente de limpieza de directorio temporal en la verificación original. Pasó en la ejecución final del servidor ScopeWeaver; no se hizo un cambio ajeno de ciclo de vida para ocultarlo.

La vista previa simulada original no tiene simulación de metadatos de notificación, por lo que su página Notificaciones falla. La **página real de Notificaciones del backend y el formulario de añadir canal pasan** en ambos idiomas. No se envió ninguna notificación al verificar.

La verificación v0.1.0 registró **14 hallazgos de npm audit: 1 crítico, 9 altos y 4 moderados**. Actualizaciones de framework/herramientas y eliminación de la dependencia CLI shadcn sin usar resolvieron los hallazgos en el candidato v0.2.1 anterior. Ejecuta `cd web && npm audit` para el informe actual del registro; puede cambiar al publicarse nuevos avisos.

### Alcance y límites históricos

- `TestLiveContextReview` opcional requiere configuración privada de proveedor externo y no se ejecutó. No hubo llamadas a modelos de pago, escaneos externos ni entregas reales de notificaciones.
- Las pruebas GLM demuestran el contrato local de solicitud/respuesta, no acceso a cuentas ni disponibilidad del proveedor. Coding Plan se restringe a herramientas oficialmente compatibles y ScopeWeaver no está en la lista; consulta [configuración de proveedores](llm-providers.md).
- No se verificaron ejecución de imagen Docker ni archivos por lotes Windows. Los casos de empaquetado no afirman pruebas reales de ejecución Windows.
- La configuración llegó a la ruta autenticada, pero una espera de navegador del evento de carga de ventana agotó el tiempo; después se confirmaron la inicialización y contraseña nueva mediante API real.
- El texto introducido por usuarios, pruebas guardadas, instrucciones editadas, identificadores de protocolo, marcadores antiguos de interpretación, casos Unicode y capturas originales conservan su contenido. La aplicación inglés/coreano no traduce datos arbitrarios del usuario ni salida de herramientas de terceros.
- Los documentos históricos de validación de `sidequestion/` describen trabajo original, no ejecuciones nuevas de ScopeWeaver.

Las comprobaciones remotas actuales están en [GitHub Actions](https://github.com/cskwork/scopeweaver/actions/workflows/ci.yml).

## Pegado de documentos en chat (2026-10-16)

La PR #29 registra las siguientes comprobaciones de pegado con BD PostgreSQL 17 desechable y backend local. Son resultados comunicados por el colaborador del 2026-10-16; las pruebas de integración actuales se registran aparte:

- La suite frontend pasó 52 pruebas; la exportación estática de producción pasó.
- El navegador pasó 12 combinaciones de portapapeles: chat independiente/de tarea, EN/KO/zh-Hans y 1440/390 px.
- Los casos cubrieron pegados rápidos de documentos con un propietario de borrador, texto acompañante, inserción nativa de 3,000 caracteres y deshacer, vista previa Unicode, restauración de borrador, reintento de subida, navegación, menciones, IME y ambas preferencias de envío. Pegar no envió mensajes.
- El colaborador desactivó el controlador React de pegado para confirmar que fallaba la comprobación por ausencia del segundo indicador de adjunto; lo restauró y repitió.
- La suite básica backend pasó portapapeles y casos Ollama loopback sin errores de página no capturados.

Los casos de archivos proporcionan objetos File del navegador mediante eventos de pegado. Los de texto y deshacer usan el portapapeles Chromium y pegado por teclado.
