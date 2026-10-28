# Puente OpenCode SDK

[English](README.md) · [한국어](README.ko.md) · [简体中文](README.zh-Hans.md) · [繁體中文](README.zh-Hant.md) · [日本語](README.ja.md) · [Español](README.es.md)

Puente opcional limitado a loopback para **CLI/cliente/SDK nativo de OpenCode 2.1.4** y la pareja **SDK/CLI heredada 1.19.4**, probada por separado. ScopeWeaver conserva la conversación, ejecución de herramientas, aprobaciones, controles de alcance, reintentos y compactación. El puente transmite un solo paso del modelo y captura solicitudes de herramientas; no ejecuta herramientas de ScopeWeaver ni herramientas internas de OpenCode. Los proveedores API y OpenCode Go existentes funcionan sin este puente.

## Inicio

El entorno de CI probado es **Bun 1.3.14**. No se han verificado versiones posteriores.

```sh
cd adapters/opencode-sdk
bun install --frozen-lockfile --ignore-scripts
export SCOPEWEAVER_OPENCODE_TOKEN="$(openssl rand -hex 32)"
bun run start
```

El token del puente debe contener al menos **32 caracteres no blancos**. El servidor CLI nativo privado usa una contraseña independiente generada por llamada (`OPENCODE_PASSWORD`) y autenticación HTTP Basic. No es el token del puente ni una clave del proveedor.

En el perfil de proveedor de ScopeWeaver, elige `opencode-sdk`, configura `http://127.0.0.1:4097` e introduce ese **token del puente**. Es distinto de una clave API de nube. No reutilices un token de OpenCode Go/Console. Carga los modelos y selecciona el `providerID/modelID` exacto.

El entorno predeterminado, sin iniciar sesión, usa un catálogo aislado y el descubrimiento público. Solo realiza solicitudes GET al catálogo: **nunca inferencia**. Muestra únicamente modelos OpenCode de precio cero presentes en la lista pública y reconocidos por la versión fijada. Los fallos del catálogo producen un error explícito. Figurar en la lista no acredita elegibilidad, cuota disponible ni éxito de inferencia. Los modelos gratuitos pueden cambiar o desaparecer; no hay sustitución predeterminada por modelos de pago ni por otras credenciales.

**Verificación anterior del 25 de octubre de 2026 (KST):** `opencode/deepseek-v4.1-flash-free` completó una conversación real de ScopeWeaver mediante el puente de producción CLI/cliente nativo 2.1.4. Tres inferencias reales cubrieron la prueba de conexión de la interfaz, una solicitud de herramienta del anfitrión revisada y aprobada antes de afectar al archivo y la respuesta final visible `FREE_OK`. El modelo procedía del catálogo predeterminado de precio cero. No se usaron claves API, importaciones de cuentas, simulaciones de modelos/respuestas, identidades falsificadas ni modelos de pago alternativos. Las herramientas nativas siguieron bloqueadas y ScopeWeaver ejecutó la acción del anfitrión. Es evidencia previa de esa fecha, no una repetición realizada ahora.

La inferencia nativa usa el CLI auténtico fijado y las API oficiales del cliente; el descubrimiento sin inferencia usa el SDK integrado. La ruta anterior del SDK integrado dedicada a capturar herramientas recibió HTTP 403 `FreeTierError`. El acceso gratuito alojado de la versión heredada sigue sin verificarse. Los modelos y la elegibilidad pueden cambiar; la prueba fechada no establece una cuota numérica ni disponibilidad futura. Un rechazo estructurado se devuelve como `free_tier_restricted`, sin elegir un modelo de pago.

`SCOPEWEAVER_OPENCODE_PORT` modifica el puerto predeterminado 4097. `SCOPEWEAVER_OPENCODE_HOST` debe ser `127.0.0.1`. Se rechazan solicitudes de navegador con `Origin` y URL que no sean de loopback. Todas las rutas requieren el token: protégelo como una contraseña.

`SCOPEWEAVER_OPENCODE_TIMEOUT_MS` limita la llamada completa entre **0 y 86,400,000 ms**. El valor predeterminado **0** deja la duración bajo el control de cancelación del anfitrión; no hay un límite oculto de dos minutos. La cancelación y el cierre tienen un plazo independiente de **1.5 segundos** para terminar por la fuerza. La consulta del catálogo público tiene **10 segundos**. Los límites son **16 MiB** por solicitud, **2,000,000 caracteres** por JSON de argumentos, **8,000,000 caracteres** para la suma de texto/razonamiento/argumentos del modelo nativo y **20,000,000 bytes** de salida del trabajador. Superarlos causa un error explícito, sin truncar datos.

## Configuración dedicada del proveedor

Para un modelo explícito compatible con OpenAI o una prueba local, indica en `SCOPEWEAVER_OPENCODE_PROVIDER_CONFIG` un archivo JSON administrado por el operador del puente. Guárdalo fuera del checkout o con el nombre ignorado por Git `provider-config.local.json`:

```json
{
  "providers": {
    "local": {
      "name": "Local model",
      "package": "@opencode/ai/providers/openai-compatible",
      "settings": {
        "baseURL": "http://127.0.0.1:8000/v1",
        "apiKey": "explicit-provider-key-if-needed"
      },
      "models": {
        "my-model": {
          "name": "My model",
          "capabilities": { "tools": true },
          "limit": { "context": 200000, "output": 32000 },
          "compatibility": { "maxTokensField": "max_tokens" }
        }
      }
    }
  }
}
```

Selecciona `local/my-model`. El archivo sustituye el descubrimiento predeterminado de ese puente. No carga configuración OpenCode de usuario/proyecto, archivos `.env`, complementos, servidores MCP ni credenciales guardadas. Solo entrega al proveedor su `apiKey` explícita. Un destino remoto puede cobrar la inferencia; configurarlo no lo convierte en gratuito. Se rechazan paquetes SDK arbitrarios y claves de configuración adicionales.

## Límites de ejecución

Cada llamada usa un trabajador Bun separado, un entorno limitado a una lista permitida y rutas temporales propias de home/configuración/datos. El modo nativo controla un servidor privado del CLI oficial, una ejecución real `run`, su base de datos temporal y un transporte de salida loopback autenticado de un solo uso. El cliente oficial gestiona descubrimiento, eventos de sesión e interrupción. No se heredan credenciales de nube, archivos de autenticación del usuario, proxies ni precargas. Se usan directamente los binarios opcionales de plataforma fijados; `--ignore-scripts` sigue siendo válido y no hace falta instalación global. **No omitas las dependencias opcionales**.

Se conservan las definiciones y las implementaciones originales de las herramientas nativas de OpenCode. Todos sus permisos son ASK y no se usa `--auto`. Un hook público `execute.before` rechaza de manera independiente, antes de ejecutarse, cualquier herramienta ajena al anfitrión, incluida la generada `execute`. Una solicitud nativa hace fallar la llamada del puente y no llega a ScopeWeaver. Solo las funciones registradas de captura del anfitrión pueden esperar y convertirse en solicitudes de herramientas del anfitrión; se eliminan los demás agentes. Los alias sin colisiones `sw_tool_N` evitan reemplazar herramientas nativas. Las descripciones conservan el nombre y texto originales, y las respuestas restauran el nombre del anfitrión.

Los segmentos system del anfitrión se unen con dos saltos de línea y se pasan sin cambios mediante la configuración pública del agente personalizado. También se conserva el contexto generado por OpenCode sobre modelo, entorno y herramientas. La política final informa de que solo se permiten funciones marcadas `Host tool:`. El hook público context mantiene roles, ID/resultados de herramientas y límites/configuración por llamada. No se garantiza que system contenga solo los segmentos del anfitrión. Se transmiten campos thinking/effort específicos del modelo; las opciones no admitidas fallan explícitamente. Un límite de salida declarado menor reduce el máximo solicitado. Entrada, salida y caché se contabilizan por separado.

Se bloquea una segunda solicitud al modelo. El transporte privado reenvía la solicitud auténtica al URL original, rechaza redirecciones y no sustituye destino, modelo, identidad ni credenciales. Las herramientas capturadas solo se entregan después de interrumpir, borrar la sesión y limpiar los procesos propios. Desconexión, DELETE, vencimiento y cierre terminan el grupo POSIX del trabajador y lo fuerzan después de **1.5 segundos**. Esa limpieza del árbol de procesos no está implementada en Windows: **ambos modos, nativo y heredado, rechazan inferencia en Windows**.

La configuración pública del CLI no permite desactivar por completo la búsqueda de configuración de sistema/MDM. El modo nativo rechaza el inicio con `configuration_error` si existe `/Library/Application Support/opencode` o `ai.opencode.managed.plist` bajo `/Library/Managed Preferences` global/del usuario en macOS, o `/etc/opencode` en Linux. También rechaza rutas inaccesibles y enlaces simbólicos rotos. No lee ni modifica esas fuentes.

El modo nativo conserva el JSON original de argumentos, incluidos enteros de 64 bits, en respuestas completas, flujos e historial de herramientas del anfitrión. La proyección analizada por JavaScript nunca se ejecuta. Rechaza números inseguros fuera de los argumentos originales autoritativos, contenido no admitido, herramientas nativas desconocidas, ID duplicados y respuestas malformadas. Los esquemas admiten definiciones en línea, JSON Pointer local `$ref`/`$defs` y uniones; rechazan referencias remotas, `$id`, `$dynamicRef` y referencias sin resolver. La inicialización aún carga código propio de proveedores/catálogo: es una frontera de la aplicación, **no un aislamiento del sistema operativo**. Un puente loopback puede usar un modelo remoto y no cumple la política de inferencia solo local de ScopeWeaver.

## Versiones y límites del modo heredado

| `SCOPEWEAVER_OPENCODE_RUNTIME` | Pareja fijada | Capacidad `runtime` |
| --- | --- | --- |
| `native` (predeterminado) | CLI oficial, `@opencode/client` y SDK **2.1.4** | `native` |
| `legacy` | `@opencode-ai/sdk/v2` y CLI **1.19.4** | `legacy` |

Define `SCOPEWEAVER_OPENCODE_RUNTIME=legacy` antes de iniciar para usar la API antigua. La verificación local previa de ambos entornos reales cubrió **macOS arm64**. CI está configurada para pruebas reales en Linux, pero su resultado alojado sigue pendiente. Otras versiones SDK se rechazan antes de inferir; no se garantiza compatibilidad universal entre versiones y plataformas.

El modo heredado usa directamente el binario oficial opcional correspondiente. Por eso admite `--ignore-scripts`: no llama a un envoltorio sin inicializar ni a una instalación global de `opencode`. No omitas las dependencias opcionales. Su CLI y el servidor MCP loopback de captura terminan con el grupo POSIX del trabajador. Windows no dispone de esa limpieza. Cada llamada usa una copia privada del paquete del complemento y de su archivo de bloqueo, sin enlaces escribibles a dependencias compartidas.

El modo heredado admite JSON/historial estructuralmente válido y representable con seguridad. Puede normalizar espacios y escapes HTML/Unicode equivalentes; no conserva los bytes léxicos originales. Rechaza explícitamente enteros inseguros cuyo valor absoluto supere **9,007,199,254,740,991**, historial de razonamiento opaco/firmado y sobrescrituras de proveedor no admitidas. Usa el modo nativo para conservar argumentos originales de 64 bits. Los proveedores heredados personalizados usan la configuración explícita compatible con OpenAI anterior, pero solo se mapean la compatibilidad `max_tokens`, las capacidades de herramientas admitidas y los límites de contexto/salida. Se rechazan `reasoningField` y sobrescrituras de capacidades no admitidos. Los perfiles API/Go siguen disponibles para entradas fuera de los límites probados de ambos modos SDK.

## Protocolo y comprobaciones

Protocolo **`scopeweaver-opencode-sdk`, versión 1**, con rutas autenticadas: `GET /v1/capabilities`, `GET /v1/models`, `POST /v1/complete`, `POST /v1/stream` y `DELETE /v1/requests/{request_id}`. El flujo NDJSON transmite deltas reales de texto/razonamiento, solicitudes capturadas y uso final; en caso de éxito hay exactamente un `message_stop`. Los errores usan códigos acotados y mensajes con datos del proveedor ocultos. Solo un agotamiento estructurado explícito de facturación/cuota se clasifica como `quota_exhausted` (HTTP 402 al completar); límites genéricos de frecuencia, autenticación y 5xx no se reclasifican. Un exceso estructurado de contexto devuelve `context_length_exceeded` (HTTP 400), manteniendo la señal de compactación del anfitrión; no basta un 400 genérico ni palabras en el mensaje. `FreeTierError` estructurado produce `free_tier_restricted` (HTTP 403), separado de cuota y prohibiciones genéricas. El puente no elige un modelo alternativo; el anfitrión decide según su política configurada.

Ejemplo de solicitud completa; el endpoint de flujo acepta el mismo cuerpo:

```json
{
  "request_id": "unique-call-uuid",
  "session_id": "stable-host-conversation-uuid",
  "model": "local/my-model",
  "system": ["Host system instructions"],
  "messages": [{ "role": "user", "content": [{ "type": "text", "text": "Hello" }] }],
  "tools": [],
  "max_tokens": 64,
  "thinking": "disabled"
}
```

Campos opcionales: `session_id`, `temperature`, `stop` (lista de cadenas), `thinking`, `thinking_type` y `reasoning_effort`. Los bloques admitidos son `text`, `thinking`/`tool_use` del asistente y `tool_result` del usuario con texto. Las herramientas declaran `name`, `description` e `input_schema`. La respuesta completa contiene `message`, `stop_reason` (`end_turn`, `tool_use` o `max_tokens`) y `usage` con entrada/salida/lectura de caché/escritura de caché separadas. Consulta el contrato y límites en [protocol.mjs](src/protocol.mjs) y el transporte en [bridge.mjs](src/bridge.mjs).

En `opencode-go`, `session_id` mantiene estable `x-opencode-session` entre sesiones SDK nuevas por llamada; si falta se usa `request_id`. `opencode` conserva las cabeceras genuinas de sesión e identidad generadas por el entorno. Los proveedores genéricos no reciben cabeceras de sesión específicas de la pasarela. Se conserva el User-Agent auténtico del CLI.

```sh
bun run check
bun test --timeout 30000
```

Las pruebas usan la pareja real fijada de CLI/cliente/SDK con modelos SSE locales. Verifican descubrimiento sin inferencia, historial/configuración, captura segura, flujo/uso, argumentos numéricos originales, llamadas nativas hostiles, aislamiento concurrente y cancelación. No prueban la elegibilidad real de modelos gratuitos alojados.

## Perfiles de ScopeWeaver y límite de degradación

Un token vacío conserva el guardado solo para el mismo endpoint SDK. Al cambiar URL o formato no se reutiliza; `localhost`, `127.0.0.1` y `::1` son ámbitos distintos de credenciales. La plantilla SDK borra claves de nube sin guardar, proxy y cabecera de sesión personalizada, deja el modelo vacío y no descubre, infiere, guarda ni activa. Usa **Cargar modelos** de forma explícita; **Probar conexión** solo hace inferencia al pulsarlo.

La restricción `CHECK` de formatos de perfil añade `opencode-sdk`. Los binarios antiguos no arrancan mientras queden perfiles SDK. Antes de degradar, respalda la base de datos y usa la versión actual para convertirlos a un formato antiguo compatible o eliminarlos. La verificación previa de migración usó bases desechables, sin migrar una base de producción. Consulta la [guía de proveedores](../../docs/es/llm-providers.md#opencode-sdk-bridge).
