# Registro de verificación de `/btw`

[English](../../../sidequestion/VALIDATION.md) · [한국어](../../ko/sidequestion/VALIDATION.md) · [简体中文](../../zh-Hans/sidequestion/VALIDATION.md) · [繁體中文](../../zh-Hant/sidequestion/VALIDATION.md) · [日本語](../../ja/sidequestion/VALIDATION.md) · Español

Registro histórico original. Fecha: 2026-09-10. Rama: `codex/btw-side-question`. Base: `8dae851b9b622f2ff2631f332fde9719d0b16fba`. No son resultados nuevos de ScopeWeaver.

Se usaron BD y directorio de datos aislados. Las credenciales de modelos reales se inyectaron solo en ese entorno, nunca en código ni este registro. No cambió el modelo predeterminado. Go 1.26.3, norma v0.3.6, Next.js 16.2.9.

Conversaciones, objetos devueltos, aserciones y revisión Qwen están en [validation-2026-09-10.json](../../../sidequestion/validation-2026-09-10.json), sin credenciales API. Se traducen textos narrativos; se conservan identificadores y mediciones.

## Comprobaciones de ingeniería

| Alcance | Resultado | Pruebas |
| --- | --- | --- |
| Copias profundas de mensajes y argumentos | Correcto | `TestCheckpointDeepCopyAndBoundaries` |
| Resúmenes/compresión no sobrescriben; respuestas completas y estados finales publican; parciales excluidas | Correcto | `TestCheckpointDeepCopyAndBoundaries`, `TestSnapshotExcludesPartialStreamAndSelectsPoolMember` |
| Identidad real del miembro del grupo de modelos | Correcto | `TestSnapshotExcludesPartialStreamAndSelectsPoolMember` |
| Pares de herramientas, repetición de 20 intercambios, recorte y desbordamiento | Correcto | `TestBuildRequestCompactionToolPairingAndBudget` |
| Concurrencia principal/paralela y aislamiento bidireccional de cancelación | Correcto | Proveedor bloqueante, `TestMainSideConcurrencyAndIndependentCancellation` |
| Sin ejecución de herramientas, con/sin transmisión, uso disponible al fallar | Correcto | `TestServiceNoToolsAndUsageOnFailure` |
| ChatAgent norma real + Read local; aislamiento de transcripción/actividad principal | Correcto | `TestSideActualChatCheckpointToolResultAndTranscriptIsolation`, subpruebas con/sin transmisión |
| Persistencia, paginación, idempotencia, respuestas parciales tras reinicio | Correcto | `TestSideHistoryIdempotencyPagingAndRecovery` |
| Carreras borrado/escritura tardía, eliminación principal, comparación de versiones | Correcto | `TestSideClearLateWritersAndDeletedParent` |
| Archivo/restauración MainAgent/Worker, v1/v2/v3 | Correcto | `TestSideTaskArchiveVersions` |
| Tres API principales, autenticación, propiedad, eliminación lógica de trabajador | Correcto | `TestSideHTTPGlobalLimitTaskWorkerAndDeletion`, `TestSideCheckpointPersistsBeforeAdmissionAndRestart` |
| Preguntas durante sesión principal ocupada; reconexión/desconexión SSE independiente, cancelar y borrar | Correcto | `TestSideHTTPBusyIsolationClearAndReconnect` |
| Una solicitud simultánea por principal / cuatro globales | Correcto | Dos pruebas `TestSideHTTP…` |
| Instantánea guardada antes de admisión, seguimiento tras reinicio, sin inventar instantáneas antiguas | Correcto | `TestSideCheckpointPersistsBeforeAdmissionAndRestart` |
| Rechazo de perfiles en caché eliminados o modelos cambiados | Correcto | `TestSideRejectsDeletedOrChangedCachedProfile` |
| Cancelar y esperar persistencia final de respuesta/uso antes de archivar | Correcto | `TestSideTaskDrainPersistsBeforeArchive` |
| Registrar uso y atribución paralela una vez al cancelar temprano el consumidor | Correcto | `TestSideUsageRecordedOnceOnConsumerCancellation` |
| Contextos restaurados de trabajador/plazo publican instantáneas nuevas | Correcto | `TestSideRestoredWorkerRuntimePublishesNewCheckpoint` |
| Detección de carreras en paquetes relacionados | Correcto | Comandos siguientes |
| TypeScript y compilación de producción | Correcto | `npx tsc --noEmit`, `npm run build` |
| Biome de módulos frontend nuevos | Correcto | `biome check`, tres módulos nuevos |

Reproduce las comprobaciones con `ARTEX_PG_DSN` dirigido a una BD desechable separada, nunca producción:

```sh
go test -race ./agent ./db ./server ./sidequestion ./llmrec ./llmpool \
  -run 'Test(Side|Checkpoint|Snapshot|BuildRequest|Service|MainSide|CaptureRun|TaskArchive|CompleteForwards|StopIntent|CancelIntent)' -count=1
cd web
npx tsc --noEmit
npx biome check src/lib/side-questions.ts src/hooks/use-side-questions.ts src/components/side-question-workspace.tsx
npm run build
```

La regresión Go completa no pasó. Dos pruebas existentes de `server` fallaron al limpiar directorios temporales con `TempDir RemoveAll … directory not empty`:

- `TestInheritedActivityDetailAndRelationDeletion`
- `TestTaskMetadataPatchReturnsRenameAndPin`

Exportar la base intacta y repetir `server` en el mismo entorno aislado reprodujo ambos fallos. La base también falló el recuento de nodos objetivo de `TestCoreTaskLifecyclePG`; la regresión final modificada no. Los demás paquetes, casos paralelos y carreras pasaron. Los problemas de base no contaron como aceptados ni se cambiaron aserciones para ocultarlos.

Next.js emitió el aviso existente de varios archivos de bloqueo/inferencia de raíz; la compilación y generación de páginas terminaron.

## Comprobaciones de navegador

Codex In-app Browser se conectó a servidores locales Go/Next.js aislados. Las comprobaciones de escritorio y pantalla estrecha 390 × 844 usaron automatización y revisión de capturas/registros:

- En chat normal, `/btw` mostró contenido principal y paralelo juntos en la barra lateral de escritorio.
- Funcionaron seguimientos; detener conservó salida parcial mientras continuaba el flujo principal.
- Las solicitudes continuaron con panel cerrado; reabrir restauró respuestas completadas y `/btw` vacío el historial tras actualizar.
- Entrada, botones, historial y cierre del panel móvil funcionaron sin desbordamiento horizontal.
- Borrar exigió confirmación, eliminó historial y conservó transcripción/instantánea principal.
- Cambiar preguntas entre MainAgent y dos trabajadores mantuvo etiquetas e historiales separados.
- Un modelo local bloqueante mantuvo al trabajador activo. Tras enviar `/btw` desde su entrada y detener la solicitud paralela, siguió mostrando ejecución en directo y su botón de pausa, conservando respuestas parciales.
- Los registros de errores/avisos del navegador estaban vacíos.

Los casos controlados probaron tiempos exactos de concurrencia independientemente de velocidad del modelo. Dos comprobaciones iniciales de trabajador no tenían ventana válida porque tarea/respuesta ya habían terminado. Tras corregir el caso, las repeticiones pasaron; los intentos iniciales no contaron como aprobados.

## Conversaciones con modelos reales

El sondeo preferido `grok-4.6` en `http://127.0.0.1:12580/tingly/openai` compatible con OpenAI devolvió HTTP 200, modelo `grok-4.6` y `READY` en 2.82 segundos. Al estar disponible, no se usó ni verificó la alternativa Tingly `glm` ni Zhipu `glm-5.3`.

| Escenario | Resultado real |
| --- | --- |
| Preguntar activo, objetivo y marcador durante ejecución principal | Devolvió `redhaze.top`, objetivo de leer/resumir la página principal y `BTW-REAL-0910`; terminó en 16.97 segundos |
| Preguntar pruebas de herramientas tras leer página | WebFetch 200 correcto, redirecciones curl 301 → 302 → 200 y título; 7.24 segundos |
| Pedir crear un archivo con Bash en pregunta paralela | Rechazado; archivo no creado; 7.74 segundos |
| Solicitud paralela completada deja intacto el contexto principal | SHA-256 de transcripción y actividad intactos; cero herramientas paralelas ejecutadas |
| Seguimiento tras detener/reiniciar Go realmente | Conservó tres registros previos; respondió activo, marcador y título desde instantánea persistente sin reejecutar el agente |
| Conversación nueva con Grok sin transmisión | Activo y `ATOMIC-0910` correctos; uso devuelto/guardado: entrada 11734, salida 138, cache_read 11520 |

La conversación principal usó WebFetch y Bash/curl para leer la página pública y llegó a `https://id.redhaze.top/home`, titulada «RedHaze Technology RedHaze Group · Portal global integrado del grupo» (traducido). Bash guardó una respuesta temporal local; no hubo escritura remota. Se verificó aparte de la ausencia de herramientas paralelas.

Suma de transcripción principal: `e7e61f135a4a120954b539f357e8c4205d7d5cd7460dcaf3dc0fd066463e1d00`.

Limitación de uso: Grok de Tingly en transmisión no devolvió uso. Un sondeo directo `stream_options.include_usage=true` devolvió HTTP 200, 12 tramas de datos y cero de uso. Los ceros significan ausencia de datos del endpoint, no ausencia de facturación. El uso sin transmisión y el de fallos/cancelaciones de casos persistió correctamente.

## Revisión Qwen

El modelo `qwen-flash` en `https://dashscope.aliyuncs.com/compatible-mode/v1` compatible con OpenAI devolvió HTTP 200. Recibió las tres primeras conversaciones reales, pruebas de herramientas principales y aserciones, y devolvió `verdict: accept`, `concerns: []`. Juzgó respuestas coherentes con activo, marcador y página, y rechazo de herramientas coherente con restricciones. Uso: prompt 6625, completion 312, total 6937.

La revisión excluyó pruebas posteriores de reinicio y sin transmisión. Qwen describió «sin escrituras» con demasiada amplitud: curl principal sí creó un archivo local, como se registra arriba. Las aserciones establecen concurrencia, cero herramientas paralelas y aislamiento; la revisión de modelo solo ayuda a evaluar calidad de respuesta.
