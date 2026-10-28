# Varios registros de tráfico como pruebas de hallazgo

[English](../traffic-evidence.md) · [한국어](../ko/traffic-evidence.md) · [简体中文](../zh-Hans/traffic-evidence.md) · [繁體中文](../zh-Hant/traffic-evidence.md) · [日本語](../ja/traffic-evidence.md) · Español

Los detalles de hallazgo admiten seleccionar tráfico entre páginas, asignar funciones y notas, ordenar registros y quitar vínculos. La página de tráfico también permite seleccionar varios registros y adjuntarlos a un hallazgo existente. Las pruebas heredadas son de solo lectura; edítalas en la tarea de origen.

«Vinculación automática de tráfico por agentes» está desactivada por defecto. Lee o cambia `agent_traffic_binding` mediante `/api/settings`. Activarla añade coste de tokens por revisión de solicitudes/respuestas, llamadas e instrucciones; el siguiente turno usa el ajuste. Desactivada, se ocultan parámetros automáticos y herramienta complementaria, no se inyectan instrucciones y se rechazan nuevos envíos automáticos de sesiones ya activas. Siguen disponibles vinculación manual, captura, lectura de pruebas guardadas y exportación.

Activada, el flujo normal es: registrar hallazgo, iniciar automáticamente agente de informes, verificar y vincular tráfico y redactar contra la última versión de pruebas. El agente que comunica conserva comandos de verificación, salida importante, ID reales existentes y finalidad en `evidence`. El agente de informes comprueba hallazgo y ejecución, verifica tráfico con `traffic_search` / `traffic_get`, llama a `bind_finding_traffic` y lee la última `version` antes de guardar. Desactivada, no recibe herramientas adicionales de búsqueda/lectura de tráfico original ni vincula automáticamente; aún puede leer instantáneas manuales y redactar informes.

Por compatibilidad, aún se puede vincular explícitamente de inmediato con `report_finding.traffic_refs` / `evidence_hint_id`. Es opcional: se pueden informar hallazgos no HTTP como TCP, tráfico sin captura o sin registros exactos. Conserva comandos, registros y otras pruebas verificables y explica la ausencia cuando sea útil; no hay campo obligatorio nuevo. Cada ID debe ser válido y tener cuerpo completo. Cualquier fallo revierte toda la vinculación; un fallo explícito al informar revierte todo el envío. Volver a añadir una instantánea existente no duplica el vínculo ni sobrescribe su nota.

## Identificadores de agentes y versiones de informes

`report_finding` acepta el parámetro opcional siguiente. El orden de la matriz establece el inicial de pruebas.

```json
{
  "traffic_refs": [
    {"traffic_id": "real-traffic-id", "role": "baseline", "note": "Normal account request"},
    {"traffic_id": "another-real-traffic-id", "role": "proof", "note": "Reproduction request"}
  ]
}
```

Las funciones son `baseline` (comparación normal), `proof` (demostración), `verification` (verificación adicional) y `supporting` (respaldo predeterminado). Verifica primero registros reales con `traffic_search` / `traffic_get`. Dominio y hora solo restringen candidatos; no demuestran pertenencia a la tarea.

Las instrucciones hacen referencia a las [indicaciones de informes de CyberStrikeAI](https://github.com/RuoJi6/CyberStrikeAI/blob/54d56774b8bd285817d16d48b70a4a5e6e0963f7/internal/app/vulnerability_tools.go), adaptadas al contrato opcional de este proyecto. No es obligatorio justificar tráfico ausente. Nunca adivines ID ni repitas sondeos solo para capturar.

La primera línea de respuesta sigue siendo `finding recorded: <exploration node ID>`. El JSON posterior aporta `finding_id` del registro independiente, `finding_node_id` del nodo de exploración y un resumen de vinculación.

- `get_finding_traffic(finding_id)` recibe el ID independiente y devuelve lista ordenada y `version`. Proporciona `binding_id`, `side=request|response`, `offset` y `length` para leer fragmentos de hasta 8192 bytes.
- `update_finding_report.finding_id` sigue recibiendo el ID de nodo de exploración. Establece `evidence_version` a la versión realmente leída. Si cambian las pruebas durante la generación, se rechaza escribir con versión antigua; vuelve a leer y regenerar.
- Las llamadas antiguas sin versión no afirman cubrir pruebas de tráfico existentes. Cambios de vínculo, nota, función u orden marcan informes existentes pendientes de actualizar.

Con vinculación automática, `add_hint` / `add_task_hint` pueden guardar `traffic_refs` en un indicio o en cada elemento de una matriz `hints`. Un planificador que informa por otro agente puede usar `evidence_hint_id` para seleccionar referencias de un indicio de esta tarea, nunca heredado. No se infieren vínculos de dominios, horas ni historial. Los fallos no crean hallazgos parciales ni inician informes antes de tiempo.

Usa `bind_finding_traffic(finding_id, traffic_refs)` para añadir vínculos ausentes sin volver a registrar el hallazgo. `list_findings`, `list_task_findings`, `node_detail` y `get_task_node_detail` exponen `finding_id` y `finding_node_id`; el `id` heredado sigue significando nodo de exploración.

El inicio solo añade propiedades opcionales a esquemas antiguos. Los vínculos predeterminados originales de herramientas de tráfico se amplían al agente de informes, que recibe la herramienta complementaria por defecto. Se conservan listas personalizadas, instrucciones, descripciones y estados. Las instrucciones se añaden tras el montaje final de herramientas: el rol que comunica entrega pruebas existentes; el de informes verifica, vincula y redacta. El chat de plataforma sin contexto debe entregar un indicio estructurado a un agente de tarea en vez de informar directamente. Entrega las pruebas antes de declarar finalizada la tarea; capturas ausentes no fuerzan espera. Las llamadas fallidas a `report_finding` no inician al agente de informes.

## API

Ruta base: `/api/exploration/findings/{finding_id}/traffic`, con ID independiente. Se aplica autenticación existente; `context_task` comprueba visibilidad de tarea y acceso heredado de solo lectura.

| Método / ruta relativa | Solicitud / respuesta |
| --- | --- |
| `GET` | Resúmenes ordenados, versión de pruebas, versión de pruebas del informe |
| `POST` | Añadir todo el lote `{"traffic_refs":[...]}` |
| `PATCH /{binding_id}` | `{"version":1,"role":"proof","note":"Description"}` |
| `DELETE /{binding_id}` | `{"version":1}` |
| `PUT /order` | `{"version":1,"binding_ids":["2","1"]}`; debe incluir la lista completa |
| `GET /{binding_id}` | Metadatos de instantánea y vistas previas limitadas de cuerpos |
| `GET /{binding_id}/body` | `side`, `offset`, `length`; `download=1` descarga todos los bytes originales |

Los conflictos de versión/conjunto de orden y escrituras durante archivo devuelven `409`; escrituras heredadas `403`; vínculos inexistentes o de otro hallazgo `404`. Fallos de lectura y verificación de tráfico/adjuntos devuelven errores explícitos.

## Almacenamiento y migración

El inicio migra PostgreSQL idempotentemente: `traffic_evidence_snapshots`, `finding_traffic_bindings` y `findings.evidence_version` / `report_evidence_version` (0 por defecto). No infiere vínculos de texto histórico.

Las instantáneas guardan ID original, hora de captura, URL, método, estado, cabeceras, longitudes de cuerpos y SHA-256. Los cuerpos están en `<data>/evidence/blobs/<first-two-characters>/<hash>.bin`, separados de `data/traffic` desechable. Varios hallazgos pueden compartir instantáneas/cuerpos. No hay API para actualizar contenido; las diferencias de integridad hacen fallar lecturas y exportaciones.

Los cuerpos completos, incluidos blobs grandes y registros antiguos en directorios, se leen bajo el bloqueo de escritura de tráfico original. Primero se guardan y verifican archivos; una transacción PostgreSQL escribe nodos, relaciones de intención, hallazgos, instantáneas y vínculos. Solo se notifica al planificador tras confirmar. Un fallo puede dejar archivos sin referencia, pero no registros de negocio parciales.

El bloqueo asesor PostgreSQL `7337741004` coordina archivos y referencias SQL. Los bloqueos de fila de tarea prohíben cambios de pruebas al poner archivo en cola. Restaurar mantiene el bloqueo desde instalar cuerpos hasta confirmar metadatos. Eliminar un hallazgo elimina sus vínculos en cascada.

El limpiador se ejecuta cada hora y solo recoge contenido sin referencia fuera de operaciones activas tras al menos 24 horas. La limpieza normal de tráfico conserva el directorio de pruebas. Copia PostgreSQL y `data/evidence` juntos para respaldos de datos activos.

## Exportación y archivos

Markdown incluye lista ordenada y versión; JSON metadatos; CSV añade recuentos e ID de vínculos. `md-zip` conserva Markdown e incluye:

```text
evidence/<finding_id>/<binding_id>/
  manifest.json
  request.http
  response.http
  request.bin
  response.bin
```

Markdown enlaza mensajes con rutas relativas. Antes de descargar, el servidor copia adjuntos, verifica hashes, comprime y sincroniza el archivo y lee cada entrada ZIP para comprobar CRC. Contenido ausente o corrupto hace fallar toda la descarga. Los adjuntos completos conservan bytes binarios originales.

Archivo v3 recoge instantáneas y cuerpos por vínculos de hallazgos, independientemente de tráfico o dominios originales. Los datos activos solo se limpian tras verificar; las pruebas compartidas permanecen. Restaurar verifica e instala cuerpos antes de restaurar metadatos y vínculos en transacción y permite reintento tras fallo. v1/v2 siguen restaurables, con campos nuevos ausentes fijados explícitamente a 0.

## Verificación y límites

Usa una BD PostgreSQL de prueba separada y recién creada por paquete mediante `ARTEX_PG_DSN`, para que datos sobrantes de tareas/modelos no inicien trabajo de fondo. Ejecuta paquetes pertinentes completos y confirma que no se omitieron por falta de configuración:

```sh
# Set ARTEX_PG_DSN to the package's isolated database before each run.
# Explicit configuration failures must fail the test.
go test ./<package> -count=1
go test -race -p 1 ./evidence ./db ./agent ./server -run 'TestEvidence|TestFindingTraffic|TestFindingEvidence|TestReportFindingAtomicContract|TestTaskArchive'
```

Las comprobaciones frontend incluyen `npx tsc --noEmit`, Biome en archivos afectados, compilación Webpack y exportación `NEXT_EXPORT=1`. Usa cachés separadas para no sobrescribir un servidor de desarrollo activo.

La aceptación local de extremo a extremo usa puertos dedicados, objetivos HTTP/HTTPS por dominio controlados y directorios temporales. Cubre ambas entradas de vinculación, selección entre páginas, orden/notas, errores, herencia de solo lectura, descargas y exportación tras eliminar tráfico original, archivar, recoger cuerpos activos, restaurar y verificar hashes.

La implementación inicial usa un bloqueo global de coordinación de pruebas. Operaciones grandes o descargas lentas pueden hacer esperar a otras. No se pueden inventar capturas ausentes ni cuerpos incompletos. La función no cambia captura ni manejo de certificados HTTPS al conectar directamente a una IP.
