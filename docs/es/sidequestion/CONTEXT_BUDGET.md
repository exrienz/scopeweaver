# Conversaciones paralelas largas y presupuestos de contexto

[English](../../../sidequestion/CONTEXT_BUDGET.md) · [한국어](../../ko/sidequestion/CONTEXT_BUDGET.md) · [简体中文](../../zh-Hans/sidequestion/CONTEXT_BUDGET.md) · [繁體中文](../../zh-Hant/sidequestion/CONTEXT_BUDGET.md) · [日本語](../../ja/sidequestion/CONTEXT_BUDGET.md) · Español

Corrección histórica original, 2026-09-11. La implementación trataba caracteres JSON como tokens y heredaba la reserva de salida de 32K de la tarea principal. Por ello rechazaba antes de tiempo preguntas válidas cuando HTML, JavaScript y resultados de herramientas ocupaban mucho contexto.

## Implementaciones de código abierto revisadas

- [Contexto paralelo Grok CLI](https://github.com/superagent-ai/grok-cli/blob/fb97af83f06dca873281d60168430f06c8de6324/src/agent/agent.ts#L739): extrae texto reciente de usuario/asistente con presupuesto de unos 2000 caracteres y hasta 400 por mensaje. No mantiene historial continuo de preguntas paralelas.
- [Solicitud independiente Grok CLI](https://github.com/superagent-ai/grok-cli/blob/fb97af83f06dca873281d60168430f06c8de6324/src/utils/side-question.ts): señal de cancelación independiente, salida máxima 2048 tokens cuando se admite, sin herramientas.
- [Compresión de sesión principal Grok CLI](https://github.com/superagent-ai/grok-cli/blob/fb97af83f06dca873281d60168430f06c8de6324/src/agent/compaction.ts): estima tokens, conserva contenido reciente, actualiza resúmenes con material nuevo y gestiona truncamiento entre turnos.
- [Compresión de sesiones OpenCode](https://github.com/anomalyco/opencode/blob/b3f1a96c6dd7adeb28b36dd11add1998fc84d67b/packages/core/src/session/compaction.ts): estima la solicitud completa, reserva salida/margen, combina contenido reciente y resúmenes continuos y resume sin herramientas. Sus valores son 8000 tokens recientes y 4096 de salida de resumen.
- [Recuperación de desbordamiento OpenCode](https://github.com/anomalyco/opencode/blob/b3f1a96c6dd7adeb28b36dd11add1998fc84d67b/packages/core/src/session/runner/llm.ts): intenta recuperar solo antes de empezar salida del asistente; las llamadas recuperadas no vuelven a entrar en la misma ruta.

ARTEX tomó presupuestos separados, contenido reciente más resúmenes continuos y recuperación limitada. Conservó mensajes estructurados y emparejamiento de herramientas de norma v0.3.6 frente a fragmentos textuales de Grok. No escribió eventos de compresión al estilo OpenCode en la transcripción principal. ScopeWeaver conserva el diseño; las versiones citadas describen la implementación histórica.

## Presupuesto de solicitud y ejecución

- Los mensajes usan la estimación UTF-8 por bytes de bloques de norma con margen 4/3, más instrucciones de sistema, esquemas y sobrecarga de envoltorios. Es una estimación, no recuento exacto del modelo.
- La salida paralela usa como máximo 8192 tokens y nunca supera el límite principal. `ARTEX_BTW_MAX_OUTPUT_TOKENS` acepta límite del servidor entre 256 y 32768. No cambia modelo predeterminado ni parámetros de tarea principal.
- Presupuesto de entrada = contexto menos salida y margen. Ventanas desconocidas usan 200K. El margen es 5%, limitado a 128–8192 tokens.
- Los intercambios correctos se cargan por ordinal ascendente en lotes de hasta 20. Se conservan como máximo 20 originales, sin superar un cuarto del presupuesto de entrada ni 16K tokens.
- Los excedentes actualizan un resumen continuo con procedencia histórica y hora de contexto. Las respuestas antiguas no son pruebas nuevas; ante conflictos prevalece la última instantánea principal.
- Si el contexto principal sigue largo, solo se resumen mensajes antiguos de la copia. Se conservan hasta 8K tokens recientes. Los cortes conservan pares de herramientas; un grupo grande entra entero al resumen.
- La entrada del resumen se divide por límites seguros UTF-8 según espacio restante; salida máxima 2048. Resúmenes vacíos, truncados, con llamadas o fuera de presupuesto no se almacenan en caché. Una solicitud paralela permite hasta 12 llamadas de resumen dentro de los mismos 120 segundos. Alcanzar el límite falla explícitamente, sin bucle indefinido.
- Un primer error de desbordamiento antes de texto o llamadas permite un reintento menor. Se detiene inmediatamente si la estimación no disminuyó. Otros errores y transmisiones parciales no activan esta recuperación.
- Todo el uso devuelto, incluidos resúmenes, fallos y cancelación, se acumula en la misma solicitud. Si el proveedor no devuelve uso solo se registra cero; las estimaciones no deben presentarse como mediciones.

## Persistencia e interfaz

`side_question_sessions.memory` guarda resúmenes antiguos, ordinales cubiertos y resúmenes principales en caché por identidad de instantánea. `side_question_requests.context_info` guarda etapa de preparación, repeticiones reales, uso de resumen y estimaciones de presupuesto.

Los resúmenes se guardan solo mientras la solicitud original sigue activa y coincide su versión de borrado. Borrar también elimina cachés; escrituras tardías no restauran datos borrados. Las instantáneas nuevas no reutilizan resúmenes antiguos. Archivo v3 conserva campos; restaurar v3 antiguos sin ellos rellena objetos vacíos; v1/v2 siguen compatibles.

POST acepta y devuelve la solicitud antes de preparar/comprimir en segundo plano, sin mantener bloqueos de admisión ni transacciones durante ese trabajo. SSE/historial muestra preparación, organización, compresión y respuesta. Un fallo de compresión persiste como fallo final. El frontend conserva el error y restaura la pregunta fallida como borrador, sin aviso sobre la entrada. El sondeo del historial ya no borra errores de envío.

## Registro histórico de verificación

Son resultados originales registrados para la corrección del 2026-09-11, no resultados nuevos de ScopeWeaver.

- Repetición de 19/20/21/50 intercambios, retención de conclusiones de más de 20 intercambios y reutilización de caché tras reinicio: comprobaciones automáticas correctas.
- Respuestas chinas/contexto de código muy largos, presupuestos de fragmentos, emparejamiento de herramientas, instantáneas intactas e invalidación de caché para nuevas: comprobaciones automáticas correctas.
- Fallo/cancelación/truncamiento/exceso/herramientas en resúmenes, carreras de borrado, límite de llamadas, recuperación única y ausencia de reintentos tras transmisión parcial: comprobaciones automáticas correctas.
- Paginación PostgreSQL aislada, reinicio, archivos v1/v2/v3, restauración de metadatos, campos ausentes en v3 antiguo y 20 principales compartiendo cuatro huecos simultáneos: correctos.
- Compilación Go candidata, TypeScript, Biome de componentes cambiados y compilación Next.js de producción en directorio aislado: correctos.
- El navegador integrado con interfaz aislada a 1280×720 y 390×844 verificó preparación, indicios del rango resumido, restauración de borrador fallido, ausencia de aviso de error, desbordamiento horizontal y errores de consola. Se eliminaron los casos temporales.
- La repetición de solo lectura de instantáneas locales de trabajadores pasó el nuevo presupuesto. Por ejemplo, la instantánea de 293085 caracteres del trabajador #3 ya no se rechazaba por recuento local de caracteres. No se llamó a modelos externos.
- La revisión automática de aprobación rechazó enviar instantáneas privadas de trabajadores a Grok para una conversación real. La prueba no se ejecutó ni contó como aprobada.
- A petición del usuario, el backend local se reinició el 2026-09-11 a las 00:37 con BD, directorio de datos y acceso originales. El archivo en ejecución coincidió con SHA-256 del candidato; `/api/health` del backend y mediante proxy frontend devolvieron estado correcto.

Comandos de verificación (solo BD de prueba aisladas):

```sh
go test -race ./sidequestion ./db ./server -run 'TestSide|TestCheckpoint|TestSnapshot|TestBuildRequest|TestService|TestMainSide|TestTaskArchive' -count=1
go build ./cmd/artex
npx tsc --noEmit
npm run build -- --webpack
```

La compilación histórica frontend usó una copia separada para no sobrescribir `.next` de la vista activa. El candidato era `/private/tmp/artex-btw-budget-candidate`, copiado a `/private/tmp/artex-btw-preview/artex` e iniciado. El binario anterior se respaldó allí como `artex.before-context-budget`. Son rutas de aquella ejecución, no archivos distribuidos con ScopeWeaver.
