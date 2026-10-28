# Plantillas de proveedores LLM

[English](../llm-providers.md) · [한국어](../ko/llm-providers.md) · [简体中文](../zh-Hans/llm-providers.md) · [繁體中文](../zh-Hant/llm-providers.md) · [日本語](../ja/llm-providers.md) · Español

ScopeWeaver admite **Anthropic**, **OpenAI Chat Completions**, **OpenAI Responses**, **Ollama** nativo y el **puente OpenCode SDK** opcional. Los cuatro formatos anteriores conservan URL base, proxy, cabecera de sesión, reintentos, frecuencia y concurrencia; los proveedores de nube, pasarelas y servidores OpenAI compatibles siguen funcionando. SDK usa credenciales independientes y los límites descritos abajo; no sustituye API/OpenCode Go.

La plantilla SDK tampoco hace solicitudes; el descubrimiento requiere pulsar **Cargar modelos** explícitamente.

Una plantilla rellena el **formulario de un perfil nuevo** sin guardar credenciales, activar perfiles ni modificar los existentes. Las de Ollama también leen modelos instalados del servidor elegido; las de nube no hacen solicitudes. El formato, URL base y modelo siguen editables tras cualquier plantilla.

## Z.ai GLM-5.3

![Plantilla de perfil nuevo GLM-5.3](../../screenshots/en/llm-templates.png)

Vista previa de plantilla sin clave API introducida; no es una conexión real al proveedor.

Abre **LLM → Nuevo → Plantilla de configuración**. Elige **API general (recomendada)** para facturación API o **Coding Plan (referencia)** solo tras obtener autorización de Z.ai para ScopeWeaver.

| Campo | API general | Referencia Coding Plan |
| --- | --- | --- |
| Formato | OpenAI Chat Completions | OpenAI Chat Completions |
| URL base | `https://api.z.ai/api/paas/v4` | `https://api.z.ai/api/coding/paas/v4` |
| Modelo | `glm-5.3` | `glm-5.3` |
| Pensamiento | `enabled` | `enabled` |
| Esfuerzo de razonamiento | `low` | `low` |
| Ventana de contexto (K) | `0` (200K por defecto) | `0` (200K por defecto) |
| Límite de salida | `16384` | `16384` |

Los endpoints y la facturación son distintos: el uso de la API general es independiente de la cuota de suscripción Coding Plan. Consulta [configuración de conexión Z.ai](https://zcode.z.ai/en/docs/configuration). Confirma el acceso al modelo y facturación de tu cuenta antes de probar.

GLM-5.3 acepta texto, tiene ventana de contexto de 1M de tokens y requiere razonamiento activado. Los esfuerzos admitidos son `low`, `high` y `max`; esta plantilla usa `low`, que [midió](#efecto-medido-aproximación-glm) mucho menos de la mitad de tokens de `max` en una tarea pequeña con el mismo hallazgo. Auméntalo para tareas difíciles. El campo de modelo sigue editable; revisa razonamiento y contexto al elegir otro. Consulta la [guía oficial GLM-5.3](https://docs.z.ai/guides/llm/glm-5.3).

**Límite de compatibilidad Coding Plan:** Z.ai limita Coding Plan a herramientas oficialmente compatibles. ScopeWeaver no figura; una plantilla de endpoint no establece permiso ni compatibilidad. Obtén autorización de Z.ai antes de usarlo aquí. Consulta [integración de herramientas](https://docs.z.ai/devpack/tool/others) y [política de uso](https://docs.z.ai/devpack/usage-policy). ScopeWeaver no suplanta herramientas compatibles ni cambia endpoints de facturación automáticamente.

Introduce nombre de perfil y tu clave API. Aplicar cualquiera de las plantillas conserva nombre, clave, proxy, cabecera de sesión y preferencias ajenas ya introducidas. Guarda cuando esté listo; un perfil nuevo se activa por separado. Los controles existentes **Probar conexión** y **Cargar modelos** hacen solicitudes reales solo al pulsarlos; este cambio no realiza pruebas reales de proveedor.

Verificado con documentación oficial el **2026-10-03**. La disponibilidad, derechos de cuenta y políticas pueden cambiar.

<a id="opencode-sdk-bridge"></a>

## Puente OpenCode SDK

`opencode-sdk` es un formato de perfil opcional e independiente, no la API HTTP interna de OpenCode. El entorno Norma de ScopeWeaver conserva conversación, herramientas, aprobaciones, alcance, reintentos y compactación. El puente solo transmite un paso del modelo y captura solicitudes de herramientas del anfitrión.

| `SCOPEWEAVER_OPENCODE_RUNTIME` | Entorno fijado | Límites de argumentos e historial |
| --- | --- | --- |
| `native` (predeterminado) | CLI, cliente y SDK oficiales **2.1.4** | Conserva JSON original de argumentos, incluidos enteros de 64 bits, en respuestas e historial. Rechaza números inseguros fuera de los argumentos originales autoritativos. |
| `legacy` | SDK y CLI **1.19.4** | JSON estructuralmente válido y representable con seguridad; enteros de valor absoluto no superior a **9,007,199,254,740,991**. Puede normalizar espacios y escapes HTML/Unicode equivalentes, sin conservar bytes originales. Rechaza enteros inseguros e historial opaco/firmado de razonamiento. Usa `native` para fidelidad de argumentos originales. |

El modo nativo usa el CLI auténtico fijado con `serve --stdio` y `run --server --session`; el cliente oficial gestiona sesiones, eventos e interrupción. Cada llamada controla home/configuración/datos temporales privados y su base de datos. Conserva las definiciones e implementaciones nativas reales, con permisos ASK y sin `--auto`; un hook público `execute.before` bloquea todas las herramientas ajenas al anfitrión antes de ejecutarse. Solo devuelve las capturas a Norma para ejecución y aprobación. Conserva el contexto generado de modelo/entorno/herramientas junto con system e historial del anfitrión. Rechaza fuentes de configuración de sistema/MDM no controladas sin leerlas ni modificarlas; consulta la guía del adaptador.

Los proveedores personalizados heredados usan un mapeo explícito compatible con OpenAI. **Ninguno de los dos entornos admite Windows**, porque no está implementada la limpieza necesaria del árbol POSIX. La verificación local del entorno cubre **macOS arm64**. CI ejecuta pruebas con el entorno real en **Linux x64**; no se afirma que otras plataformas estén verificadas. Las versiones SDK no probadas requieren verificación independiente.

Sigue el [inicio del adaptador](../../adapters/opencode-sdk/README.es.md) con **Bun 1.3.14**. El puente tiene configuración propia y no importa credenciales guardadas de OpenCode ni de nube. Define un `SCOPEWEAVER_OPENCODE_TOKEN` independiente de al menos **32 caracteres no blancos** y úsalo en **LLM → Nuevo → Plantilla de configuración → Puente OpenCode SDK**. No pegues una clave API de OpenCode Go/Console.

```sh
cd adapters/opencode-sdk
bun install --frozen-lockfile --ignore-scripts
export SCOPEWEAVER_OPENCODE_TOKEN="$(openssl rand -hex 32)"
bun run start
```

Conserva el token para el campo **Token del puente**. El entorno predeterminado es `native`; define `SCOPEWEAVER_OPENCODE_RUNTIME=legacy` antes del inicio para elegir el antiguo. Escucha en `127.0.0.1:4097`. Para proveedores explícitos, usa un JSON dedicado mediante `SCOPEWEAVER_OPENCODE_PROVIDER_CONFIG`, siguiendo el esquema del adaptador, no un archivo de autenticación del usuario. Conserva las dependencias opcionales de plataforma.

`SCOPEWEAVER_OPENCODE_TIMEOUT_MS` admite **0–86,400,000 ms**, con **0** predeterminado bajo control de cancelación del anfitrión. La solicitud tiene un máximo de **16 MiB** y el exceso falla explícitamente. Los demás límites de salida y limpieza están en la guía del adaptador.

| Campo | Valor |
| --- | --- |
| Formato | `opencode-sdk` |
| URL del puente | `http://127.0.0.1:4097` |
| Modelo | `providerID/modelID`, mediante **Cargar modelos** o entrada explícita |
| Token del puente | `SCOPEWEAVER_OPENCODE_TOKEN`, no una clave API de nube |
| Streaming | Activado, con el entorno fijado seleccionado |
| Razonamiento / esfuerzo | Omitidos por la plantilla; usa solo valores admitidos por el modelo |
| Límite de salida | `0` usa el valor existente de la ruta del anfitrión; el puente aplica el límite declarado menor del modelo |
| Proxy / cabecera de sesión personalizada | No admitidos; la plantilla borra ambos |

La plantilla borra claves de nube sin guardar, deja el modelo vacío y **no descubre, infiere, guarda ni activa**. Introduce el token y pulsa **Cargar modelos** para consultar el catálogo sin inferencia. Guarda y activa por separado cuando esté listo. **Probar conexión** solo hace una solicitud al modelo al pulsarlo; descubrir metadatos no demuestra acceso de inferencia.

**Verificación gratuita anterior — 2026-10-25 KST:** si figura en la lista, selecciona `opencode/deepseek-v4.1-flash-free`. La ejecución anterior sin iniciar sesión en macOS arm64, con CLI auténtico 2.1.4, superó conexión de interfaz, escritura Bash del anfitrión tras aprobación ASK manual, archivo con `FREE_OK` exacto más salto de línea y respuesta final. Hubo **3 POST de inferencia**, reintentos desactivados, sin modelo de pago alternativo ni ejecución nativa observada. Solo demuestra un pequeño flujo aislado de esa fecha, no una repetición actual, disponibilidad permanente ni uso ilimitado. Verifica disponibilidad, elegibilidad y límites actuales. El acceso gratuito alojado heredado sigue sin verificarse.

Al editar, un token vacío conserva el guardado solo en el mismo endpoint SDK. Cambiar URL borra el token introducido; cambiar endpoint o formato con token vacío no reutiliza credenciales. `localhost`, `127.0.0.1` y `::1` son ámbitos distintos. ID y límites de salida proceden de metadatos actuales. El razonamiento depende del proveedor/modelo y falla explícitamente si no es compatible.

**Red y cuentas:** el puente loopback debe compartir el espacio de nombres de red con ScopeWeaver; localhost dentro de un contenedor es el propio contenedor. El destino puede ser nube, por lo que el modo **LLM solo local** rechaza SDK. El puente no admite proxies; configura la conectividad del proveedor en el puente. Un ID gratuito no demuestra derechos de suscripción, continuidad ni uso ilimitado; revisa términos y límites. No selecciona alternativas de pago ni importa cuentas. La política de failover configurada por el anfitrión es independiente. Los perfiles API/OpenCode Go existentes siguen separados.

**Preparar una degradación:** la restricción de formato `CHECK` de la base de datos añade `opencode-sdk`. Un binario anterior no arranca mientras queden perfiles SDK. Respalda la base y usa la versión actual para convertirlos a un formato antiguo admitido o eliminarlos antes de degradar. La verificación previa usó bases desechables, sin migrar producción.

## OpenCode Go (pasarela de suscripción opencode)

Abre **LLM → Nuevo → Plantilla de configuración** y elige **OpenCode Go · glm-5.3-flash**.

| Campo | Valor |
| --- | --- |
| Formato | OpenAI Chat Completions |
| URL base | `https://opencode.ai/zen/go/v1` |
| Modelo | `glm-5.3-flash` |
| Pensamiento | omitido (predeterminado del modelo) |
| Esfuerzo de razonamiento | `max` |
| Ventana de contexto (K) | `0` (200K por defecto) |
| Límite de salida | `16384`, enviado como `max_tokens` |
| Cabecera de sesión | `x-opencode-session` (obligatoria en la pasarela; la plantilla la rellena) |

La pasarela es compatible con OpenAI: la definición de proveedor de opencode 1.18.5 (`opencode-go`) usa `@ai-sdk/openai-compatible` en `https://opencode.ai/zen/go/v1` con la clave de entorno `OPENCODE_API_KEY` (comprobado el 2026-10-20). Los modelos GLM aceptan esfuerzo `low`, `high` y `max`; esta plantilla rellena deliberadamente `max` a petición del operador, por lo que se espera el mayor uso de tokens de razonamiento. **Cargar modelos** lee la lista de la pasarela; el campo sigue editable para otros ID como `qwen3.7-plus`. Cada solicitud debe llevar `x-opencode-session` (sin ella responde `400 MissingSessionID`); la plantilla rellena la **Cabecera de sesión** del perfil con ese nombre y ScopeWeaver envía sus propios ID de sesión como valor.

**Credenciales y condiciones:** la clave pertenece a tu suscripción opencode; opencode la guarda en su archivo de autenticación, que ScopeWeaver nunca lee. Pega la clave en el perfil o establece `OPENCODE_API_KEY`. Se rige por las condiciones de suscripción de opencode: confirma que se permite usarla desde otra herramienta antes de guardar. Como con Z.ai Coding Plan, una plantilla aquí no establece permiso.

Configuración de entorno:

```bash
SCOPEWEAVER_LLM_PROVIDER=opencode-go
SCOPEWEAVER_LLM_MODEL=glm-5.3-flash
# Key: OPENCODE_API_KEY is used when no generic key is set; SCOPEWEAVER_LLM_API_KEY overrides.
```

`SCOPEWEAVER_LLM_PROVIDER=opencode-go` se resuelve al formato de protocolo OpenAI Chat Completions; la URL base y el modelo no definidos usan los valores de la tabla anterior.

## Controlar el uso de tokens

Cada turno de agente reenvía las instrucciones de sistema, esquemas de herramientas y conversación acumulada. La entrada crece turno a turno hasta que la compresión acorta el historial, y la caché del proveedor determina cuánto se factura a la tarifa menor de caché. Esta sección responde a la [incidencia #24](https://github.com/cskwork/scopeweaver/issues/24) para perfiles de nube. Ollama nativo tiene protecciones propias descritas [abajo](#protecciones-durante-la-ejecución).

**Estado de las pruebas:** las mediciones usan Z.ai GLM-5.3 en Coding Plan como aproximación a DeepSeek. GLM tiene tokenizador, contabilidad de caché y precios distintos, por lo que **las cifras de DeepSeek no están verificadas**. La compatibilidad de DeepSeek, Claude, OpenAI, Qwen, Kimi, OpenRouter y OpenCode Go se verifica con casos construidos a partir de respuestas documentadas de cada proveedor, no llamadas reales. Incluye todo el comportamiento de solicitudes y repetición de Claude 5.5 siguiente; no se usó clave API de Claude.

### Plantillas económicas

Abre **LLM → Nuevo → Plantilla de configuración** y elige una entrada *(económica)*. Las seis fijan salida `16384` y contexto `0` (200K predeterminado de compresión). Como las plantillas GLM, conservan una clave introducida y no guardan ni activan nada.

| Plantilla | Formato | URL base | Modelo | Pensamiento | Esfuerzo | Campo de salida |
| --- | --- | --- | --- | --- | --- | --- |
| DeepSeek | OpenAI Chat Completions | `https://api.deepseek.com` | `deepseek-flash` | omitido (DeepSeek por defecto: activado) | `low` | `max_tokens` |
| Anthropic Claude | Anthropic | `https://api.anthropic.com` | `claude-haiku-5-5` | `disabled` | `low` | `max_tokens` |
| OpenAI | OpenAI Responses | `https://api.openai.com/v1` | `gpt-6-luna` | omitido | `low` | `max_output_tokens` |
| Alibaba Qwen (internacional) | OpenAI Chat Completions | `https://dashscope-intl.aliyuncs.com/compatible-mode/v1` | `qwen3.8-flash` | omitido | `low` | `max_completion_tokens` |
| Moonshot Kimi | OpenAI Chat Completions | `https://api.moonshot.ai/v1` | `kimi-k3` | omitido | `low` | `max_completion_tokens` |
| OpenRouter | OpenAI Chat Completions | `https://openrouter.ai/api/v1` | `deepseek/deepseek-v4.1-flash` | omitido | `low` | `max_tokens` |

- **El esfuerzo `low` es un valor económico sin probar** en todos los proveedores anteriores. Si los resultados se cortan o son superficiales, sube el esfuerzo hacia el predeterminado del proveedor (DeepSeek `high`, GPT-6 Luna `medium`, Qwen `xhigh`, Kimi K3 `max`) y después el límite de salida.
- Claude: la plantilla desactiva pensamiento en Haiku para ahorrar salida. Haiku solo acepta pensamiento desactivado con esfuerzo `low`, `medium` o `high`. Sonnet y Opus 5.5 lo rechazan, por lo que no tienen plantilla económica; úsalos con pensamiento **Omitir** o **Adaptativo** (consulta [Pensamiento Claude 5.5](#pensamiento-claude-55)).
- OpenAI: la plantilla usa formato Responses porque GPT-6 Luna solo acepta herramientas en Chat Completions con esfuerzo `none`.
- OpenRouter: el ID de modelo y compatibilidad de `reasoning_effort` se comprobaron en su lista pública el 2026-10-16. Claude mediante OpenRouter necesita marcadores de caché que Chat Completions no puede enviar, por lo que la plantilla usa DeepSeek Flash, que almacena prefijos automáticamente.
- Z.ai GLM-5.3: las plantillas General y Coding Plan ahora rellenan esfuerzo `low`, salida `16384` y contexto `0`, la fila económica medida abajo. Antes rellenaban `max`, `1000` y `0`. El pensamiento permanece activado porque GLM-5.3 lo requiere.

### Pensamiento Claude 5.5

Estas reglas solo se aplican a ID de modelo `claude-…-5-5` (o instantánea fechada `claude-…-5-5-YYYYMMDD`) en el host oficial (`https://api.anthropic.com` o URL base vacía).

- Pensamiento vacío (**Omitir**) ahora envía `thinking: {type: adaptive, display: summarized}`. La nueva opción **Adaptativo** envía lo mismo; **Activado** (`enabled`) también recibe `display: summarized`.
- En bucles de herramientas, los bloques de pensamiento se repiten con sus llamadas, incluidos texto vacío y `redacted_thinking`. ScopeWeaver los conserva en memoria para las 128 llamadas más recientes (máximo 4 MiB) por proveedor; los bloques con texto resumido también se guardan en el historial, pero los vacíos o redactados no sobreviven a un reinicio durante el bucle.
- Las llamadas de resumen y juez ya no envían `temperature`. También omiten su pensamiento `disabled` para Sonnet, Opus y Haiku con esfuerzo `xhigh` o `max`, que lo rechazan; se aplica en su lugar el pensamiento del perfil.
- Un perfil en **Desactivado** (`disabled`) sigue enviando `disabled` en turnos de agente; Sonnet y Opus 5.5 necesitan **Omitir** o **Adaptativo**.
- Los endpoints Anthropic compatibles de terceros (DeepSeek, Kimi, Z.ai) y otros ID de modelo no cambian: el pensamiento vacío se sigue omitiendo.
- **Verificado solo con casos de solicitudes y transmisiones**; comportamiento y facturación reales de Claude sin verificar.

### Efecto medido (aproximación GLM)

Una tarea pequeña (tres rutas en objetivo local con un secreto sin autenticación preparado), 3 ejecuciones por perfil, cada una con BD nueva, dejando terminar conversaciones del redactor. Los créditos usan multiplicadores GLM-5.3 publicados por Z.ai, (entrada sin caché × 6.9 + entrada en caché × 1.7 + salida × 24) / 10,000; se calculan, no se leen de una factura.

| Perfil | Mediana de tokens de entrada + salida | Mediana de créditos | Problema preparado encontrado |
| --- | --- | --- | --- |
| Referencia: plantilla Z.ai Coding anterior (esfuerzo `max`, contexto `1000`, salida `0`) | 209,233 (172,576–216,452) | 75.1 (64.3–88.6) | 3 de 3 |
| Económico: esfuerzo `low`, salida `16384`, contexto `0` | 96,023 (95,060–105,023) | 26.6 (25.0–27.4) | 3 de 3, cada uno documentado por el redactor |

- Económico frente a referencia: 54% menos tokens de entrada + salida y 65% menos créditos en la mediana; cada ejecución económica fue menor que la referencia más barata. La llamada mayor usó 13,287 tokens de entrada y 3,829 de salida, sin alcanzar ventana 200K ni límite de salida: el ahorro aquí viene del esfuerzo `low`. El límite y ventana acotan tareas mayores y no se midieron. Dispersión mayor/menor: 1.25× referencia, 1.10× económico.
- Reparto por función: créditos de referencia: trabajador 29.6%, planificador 29.3%, redactor 29.1%, preparación de objetivos 12.0%; económicos: redactor 38.8%, planificador 34.1%, trabajador 20.2%, preparación 6.9%. No hubo compresión ni llamadas de resumen de sesión; juez y búsqueda web desactivados.
- Proporción de entrada en caché: 85.2% referencia, 92.3% económico. Razonamiento: la salida, que lo incluye, supuso 39.3% de créditos de referencia y 23.4% de económicos; ScopeWeaver aún no registraba tokens de razonamiento por separado en esas ejecuciones.

GLM-5.3 siempre piensa; estas ejecuciones no miden el efecto de desactivar pensamiento.

Las comprobaciones mediante puentes reales (Claude con `claude -p`, OpenAI con `codex exec`) prueban comportamiento de modelos por un puente local. No prueban formato de protocolo API ni facturación.

Un puente `codex exec` ejecutó la plantilla OpenAI (gpt-6-luna, Responses) en una tarea completa: terminó el bucle de herramientas, encontró el problema preparado, el redactor lo documentó y respondió una pregunta paralela posterior (sin marcarla interrumpida). Las filas `llm_usage` incluyeron razonamiento y lectura de caché, incluida una fila `btw`. Dos ejecuciones más no terminaron dentro del tiempo del arnés por lentitud del puente; es su límite de velocidad, no un resultado del código. El puente `claude -p` se abandonó: no manejaba el protocolo de herramientas. Las solicitudes y gestión de uso Claude/OpenAI están demostradas mediante casos.

### Ajustes recomendados

1. Empieza por la plantilla económica del proveedor y conserva contexto `0` salvo que la tarea necesite más historial.
2. Mantén un límite de salida positivo. `0` no omite el campo (consulta abajo).
3. Mantén esfuerzo bajo; súbelo solo si empeoran los resultados.
4. Deja juez de aprobación, búsqueda web y compresión noa desactivados salvo necesidad (todos lo están por defecto).
5. Para tareas pequeñas, considera menos agentes de trabajo que los 3 predeterminados. El efecto no se ha medido.
6. Si no necesitas un informe detallado por hallazgo, desactiva el desencadenante del agente **Redactor de informes** (**Sistema → Agente**).
7. Tras la primera tarea, comprueba uso por función (consulta [Comprobar dónde se consumen los tokens](#comprobar-dónde-se-consumen-los-tokens)).

### Ventana de contexto, límite de salida, pensamiento y esfuerzo

| Ajuste | Comportamiento en perfiles de nube | Orientación |
| --- | --- | --- |
| Ventana de contexto (K) | Solo presupuesto local de compresión; no se envía al proveedor. `0` = 200K; valores limitados a 32–1000. | Una ventana mayor retrasa la compresión y cada llamada lleva más historial. A 200K, resultados antiguos empiezan a borrarse por encima de unos 115K tokens y el historial se resume a unos 179K. |
| Límite de salida | Límite por respuesta. **`0` no se omite**: turnos de agente envían 32768, llamadas directas Anthropic 8192 y Ollama nativo su propio valor. | No acorta respuestas; detiene las largas. Demasiado bajo las trunca. Las plantillas usan 16384. |
| Pensamiento | `thinking.type`; vacío lo omite excepto Claude 5.5 oficial, donde envía `adaptive`. El razonamiento se factura como salida. | DeepSeek piensa por defecto y documenta `disabled`. Con herramientas también reenvía razonamiento anterior como entrada. GLM-5.3 siempre piensa. La plantilla Claude lo desactiva. |
| Esfuerzo de razonamiento | De `low` a `max`; vacío lo omite. | Las plantillas usan `low`. Solo se midió en GLM-5.3; en otros es un valor económico sin probar. Los valores admitidos varían por proveedor. |

### Optimización de contexto

- **Compresión de conversación** (integrada, activada): cerca de la ventana de contexto se resumen mensajes antiguos y se conservan los últimos 8. El resumen es una llamada adicional; su salida se limita al límite de salida, máximo 20000 (20000 si el límite es `0`).
- **Borrado de salidas antiguas:** por encima del 60% de la ventana se sustituyen resultados de Read, Bash, Grep, Glob, WebFetch, WebSearch, Edit y Write anteriores a los cinco más recientes. Los resultados de grafos, pruebas y herramientas personalizadas permanecen hasta el siguiente resumen.
- **Límite de salida de herramientas:** cada bloque textual de resultado se corta a 30,000 bytes; la salida completa permanece en disco con un enlace.
- **Compresión de grafo:** las partes inactivas del grafo de exploración se agrupan en resúmenes, una llamada por grupo apto (salida máxima 1500).
- **Compresión de contexto noa** (ajustes del sistema, experimental, desactivada) sustituye a la integrada.
- El desencadenante del 72% y los límites 16/32 KiB se aplican solo a Ollama nativo.

**Caché de instrucciones:**

- Claude: ScopeWeaver ahora marca siempre las instrucciones de sistema y esquemas previos de herramientas como almacenables en caché en sesiones de planificador, trabajador, agente principal y chat (incluido redactor). Toda otra solicitud de formato Anthropic con instrucciones de sistema sin límite marcado ahora marca todo el texto: preparación de objetivos, resúmenes de conversaciones/grafos, preparación de preguntas paralelas y juez. Si el texto es menor que el mínimo del modelo no se almacena aun con marcador. El primer uso de cada texto distinto escribe caché a 1.25× el precio de entrada; las llamadas posteriores dentro de su vigencia leen al precio de caché. Las instrucciones de trabajador varían por intención, por lo que cada intención escribe una vez. La posición de marcadores se verifica con pruebas; **el ahorro real no está verificado** (sin clave Claude).
- DeepSeek, Kimi y Z.ai almacenan prefijos automáticamente. DeepSeek y Kimi documentan que sus endpoints compatibles con Anthropic ignoran marcadores de bloque. OpenAI, Qwen y OpenRouter (DeepSeek Flash) también almacenan prefijos coincidentes automáticamente.
- Las pruebas confirman que turnos consecutivos de herramientas en una sesión conservan instrucciones de sistema y esquemas iguales y solo añaden historial en los tres formatos de nube. La compresión, cierre y herramientas diferidas recién desbloqueadas cambian el prefijo. Una plantilla editada con `{{.Now}}` cambia las instrucciones en cada ejecución.

### Trabajadores, reintentos y ajustes de tarea

- **Concurrencia de trabajadores · Número de agentes trabajadores** (ajustes del sistema, 3 por tarea por defecto): cada agente se asigna una intención diferente y ejecuta su sesión con instrucciones de sistema y esquemas completos.
- **Máximo de solicitudes LLM simultáneas** (ajustes del sistema, `0` = sin límite) y **Máximo de solicitudes simultáneas** del perfil regulan llamadas. No reducen tokens por llamada.
- Los reintentos reenvían la solicitud completa. Valores predeterminados: 3 **Reintentos de conexión**, 2 **Reintentos de respuesta vacía** (solo OpenAI), 2 **Reintentos en ventana segura del mismo proveedor** (antes de cualquier salida), 2 reejecuciones de intención tras error de modelo y hasta 2 continuaciones tras respuesta de solo razonamiento. Establece `-1` en la página LLM, por perfil o política global, para desactivar una capa. La facturación de intentos fallidos depende del proveedor.
- Cuando una ejecución agota turnos o tiempo, o la tarea entra en cierre, este añade hasta 10 turnos por defecto.
- La conmutación del grupo de proveedores (desactivada por defecto) repite la solicitud completa en otro perfil.

### Causas conocidas de llamadas adicionales

- Sesiones largas: cada turno reenvía todo el historial; los turnos posteriores cuestan más que los primeros.
- Planificador: sesión nueva en cada activación (inicio, intención completada, hallazgo nuevo, latido inactivo), con varios turnos cada una.
- El desglose de objetivos se ejecuta antes de planificar, hasta 8 turnos.
- **Redactor de informes:** agente personalizado inicializado con desencadenante activo por cada llamada `report_finding`. Cada hallazgo inicia su conversación, hasta 5 en paralelo.
- **Búsqueda web DeepSeek:** cada búsqueda es una llamada independiente con la clave del perfil activo (salida máxima 1024, hasta 3 búsquedas por llamada). Las búsquedas de agentes y prueba de búsqueda en ajustes ahora se registran en `llm_usage` con función `web_search`, nombre y modelo del perfil activo; no aparecen en grabaciones LLM. DeepSeek no documenta el formato de uso de esas respuestas, por lo que ScopeWeaver lee campos de estilo Anthropic. **Una respuesta sin trama de uso se registra con 0 tokens: se cuenta la llamada, sus tokens son desconocidos.** Comprueba los tokens facturados en la consola DeepSeek.
- Juez de aprobación (desactivado por defecto): una llamada por acción sin regla coincidente. Preguntas paralelas: solo a petición, hasta 12 llamadas de resumen antes de responder.
- **Probar conexión** realiza una llamada real (salida máxima 8192) que no se registra.
- Error contable corregido: las transmisiones de formato Anthropic (Claude y endpoints compatibles) sobrecontaban salida en `llm_usage`. Una transmisión de prueba con acumulados 10 y 20 se registraba como 30 en vez de 20. Solo se inflaban cifras de ScopeWeaver, no facturación del proveedor.
- Corregido: los totales internos de sesión y transcripción también sobrecontaban salida transmitida (la misma transmisión de 10 y 20 contaba 30). Ahora cuentan 20. No son `llm_usage`; el exceso solo podía adelantar la compresión.
- Corregido: las preguntas paralelas en perfiles OpenAI Responses se marcaban interrumpidas porque la transmisión no tenía evento de parada. Ahora `response.completed` termina normalmente; `response.incomplete` sigue marcando interrupción.

### Comprobar dónde se consumen los tokens

- **Tareas → tarea → Resumen → Uso de tokens LLM (por modelo)**: llamadas, entrada, salida, **Razonamiento (en salida)** y lectura de caché por modelo, desde la tabla siempre activa `llm_usage`. Solo aparecen filas con tarea; las del redactor no la tienen. Las de búsqueda DeepSeek aparecen bajo el modelo activo.
- **Panel** y **Reglas de intercepción → Uso de tokens de aprobación** también muestran **Razonamiento (en salida)**.
- **Grabaciones LLM**: una fila por llamada con función, tokens, latencia y solicitud/respuesta original. Desactivadas por defecto; activa **Grabación** antes de la tarea. Guarda instrucciones y respuestas completas.
- SQL de `llm_usage` para una ventana temporal (la función es la columna `worker`: `planner`, `worker`, `goals`, `compactor`, `judge`, `btw` para preguntas paralelas, `mainagent`, `web_search` para búsqueda DeepSeek; vacía en conversaciones de redactor, comprobación y agentes personalizados). Los resúmenes y cierre conservan la función del agente.

```sql
SELECT COALESCE(worker, '(none)') AS lane, model, COUNT(*) AS calls,
       SUM(input_tokens) AS input, SUM(cache_read) AS cache_read,
       SUM(cache_write) AS cache_write, SUM(output_tokens) AS output,
       SUM(reasoning_tokens) AS reasoning
FROM llm_usage
WHERE ts >= '2026-10-16 09:00' AND ts < '2026-10-16 10:00'  -- your task's time range
GROUP BY 1, 2
ORDER BY SUM(input_tokens) + SUM(output_tokens) DESC;
```

`cache_read` y `cache_write` ya forman parte de `input`, y `reasoning` de `output`; no los sumes de nuevo.

Las lecturas de caché se cuentan desde `prompt_tokens_details.cached_tokens`, `prompt_cache_hit_tokens` de nivel superior de DeepSeek y `cached_tokens` antiguo de Kimi. Las escrituras se registran para OpenAI Chat, OpenAI Responses, Kimi K3 y OpenRouter (`cache_write_tokens`), Qwen (`cache_creation_input_tokens`) y Claude. Estas asignaciones solo se verifican con casos de prueba.

### Lagunas de contabilidad y seguimientos

- El razonamiento es 0 si el proveedor no informa una cifra separada; 0 no demuestra ausencia de razonamiento.
- Los nombres de campos de uso de búsqueda DeepSeek no están documentados (consulta [Causas conocidas de llamadas adicionales](#causas-conocidas-de-llamadas-adicionales)).
- Gemini aún no tiene plantilla; necesita actualizar Norma SDK a v0.4.4 (seguimiento).
- La caché de historial Claude (marcador en el último mensaje) necesita soporte SDK (seguimiento).
- Modelos OpenRouter que solo respetan `reasoning.effort` unificado pueden ignorar `reasoning_effort` de la plantilla, único formato que envía el SDK (seguimiento).

## Ollama nativo (inferencia local)

ScopeWeaver comunica con Ollama mediante su API nativa `/api/chat`. El endpoint compatible con OpenAI no permite fijar contexto ([compatibilidad OpenAI de Ollama](https://docs.ollama.com/api/openai-compatibility)), por lo que un perfil dirigido a `/v1` no garantiza `num_ctx`. El formato nativo lo envía en cada solicitud. Las URL base terminadas en `/v1`, `/api` o `/api/chat` se normalizan a la raíz del servidor.

El contexto predeterminado documentado de Ollama difiere entre páginas: las [FAQ](https://docs.ollama.com/faq) indican 4,096 tokens y la [página de contexto](https://docs.ollama.com/context-length) describe un valor según VRAM (unos 4K por debajo de 24 GiB, 32K para 24–48 GiB y 256K por encima). ScopeWeaver no depende por ello del valor del servidor.

### Plantillas

Abre **LLM → Nuevo → Plantilla de configuración → Inferencia local (Ollama)**.

| Campo | Mismo equipo | Host desde Docker | Servicio Compose incluido |
| --- | --- | --- | --- |
| Formato | Ollama (/api/chat nativa) | Ollama (/api/chat nativa) | Ollama (/api/chat nativa) |
| URL base | `http://127.0.0.1:11434` | `http://host.docker.internal:11434` | `http://ollama:11434` |
| Modelo | Modelo instalado de este servidor | Modelo instalado de este servidor | Modelo instalado de este servidor |
| Clave API | vacía | vacía | vacía |
| Ventana de contexto (K) | `32` (`num_ctx` 32,768) | `32` | `32` |
| Máximo de solicitudes simultáneas | `1` | `1` | `1` |
| Keep-alive | `30m` | `30m` | `30m` |
| Límite de salida | `4096` | `4096` | `4096` |
| Pensamiento / esfuerzo | omitido (predeterminado del modelo) | omitido | omitido |

Elegir una plantilla Ollama lee `/api/tags` y metadatos del endpoint y elige un modelo instalado, priorizando los que declaran `tools`. Elegir formato Ollama en un perfil nuevo hace lo mismo; un endpoint vacío o de plantilla de nube sin cambiar usa el host predeterminado. Un endpoint personalizado no cambia. El modelo queda vacío durante la detección y permanece vacío con error si falla o no hay modelos. Comprueba endpoint, token opcional y proxy; pulsa **Cargar modelos** o introduce una etiqueta manualmente. Editar modelo o conexión cancela la selección pendiente; respuestas tardías no sobrescriben cambios. Abrir un perfil existente conserva su modelo. Seleccionar nunca descarga ni ejecuta modelos, guarda ni activa perfiles. **Probar conexión** solo ejecuta al pulsarlo.

**Comportamiento de clave API:** las plantillas Ollama y cambiar al formato Ollama borran una clave sin guardar antes de detectar. Conservan nombre del perfil, proxy, cabecera de sesión y otras preferencias, pero **borran cualquier clave API sin guardar** para no enviar credenciales de nube a un endpoint local. GLM sigue conservando la clave introducida. Deja la clave vacía para Ollama local; introduce token solo si lo requiere un proxy inverso delante de Ollama (se envía como `Authorization: Bearer`). Los perfiles sin clave se pueden probar, guardar, activar y usar en conmutación; la tarjeta muestra **No se necesita clave** en lugar del aviso de clave ausente. Al editar un perfil nativo guardado, una clave vacía conserva el token; cambiar un perfil de nube a Ollama con el campo vacío elimina la clave anterior.

**Claves de otros formatos:** Anthropic y OpenAI requieren clave salvo URL base explícitamente local, cuyo host solo resuelve a loopback o direcciones privadas. Así, un servidor compatible local sin clave como vLLM o LM Studio funciona con clave vacía. ScopeWeaver envía entonces un marcador interno no secreto para que el SDK no herede `OPENAI_API_KEY` ni `ANTHROPIC_API_KEY` del entorno. Se rechaza un endpoint no local sin clave (*llm: API key required for non-local compatible endpoint*).

### Controles de contexto, salida y permanencia en memoria

| Ajuste | Comportamiento Ollama | Formatos de nube |
| --- | --- | --- |
| Ventana de contexto (K) | K binario enviado como `options.num_ctx` (32 = 32,768). `0` usa 32; valores menores que 16 usan 16. También fija el presupuesto de compresión. | Sin cambios: solo presupuesto de compresión, `0` = 200K |
| Límite de salida | Se envía como `options.num_predict`; `0` usa 4,096 | `0` envía 32768 en turnos de agente (Anthropic directo: 8192) |
| Keep-alive | Se envía como `keep_alive`; vacío usa `30m` (Ollama usa 5 minutos por defecto) | No se envía |
| Máximo de solicitudes simultáneas | `0` usa 1. Se comparte un control de admisión por servidor Ollama incluso entre perfiles; se aplica el límite menor configurado | `0` hereda **Configuración del sistema → Máximo de solicitudes LLM simultáneas** |

Las solicitudes en cola del juez, agente principal y preguntas paralelas ocupan el siguiente hueco libre antes que el trabajo de fondo. Cada grupo es FIFO; la inferencia activa nunca se interrumpe. Diagnósticos y precarga siguen siendo solicitudes de fondo.

Aumenta concurrencia solo si el servidor establece `OLLAMA_NUM_PARALLEL` por encima de 1. La memoria de Ollama crece con `OLLAMA_NUM_PARALLEL × context length` ([FAQ](https://docs.ollama.com/faq)).

### Protecciones durante la ejecución

- **Precarga:** antes de la primera inferencia nativa, ScopeWeaver llama a `/api/show`, rechaza modelos sin `tools` y carga el modelo con una solicitud `/api/chat` vacía usando `num_ctx` y `keep_alive` del perfil. La precarga ocupa el mismo hueco de concurrencia.
- **Compresión:** con Ollama nativo y ventana de 64K o menor, se activa al 72% y conserva pares completos de llamada/resultado. Si se activa noa experimental, esta protección tiene prioridad: se omite noa y se emite un aviso. A 32K o menos, la parte de cada salida dada al modelo se limita a 16,384 **bytes** (16 KiB, medidos como `len(s)` del SDK, no caracteres Unicode); por encima de 32K hasta 64K, el límite es 32,768 bytes (32 KiB). La salida completa se conserva en un archivo auxiliar referenciado por el resultado. Los formatos de nube conservan compresión y noa existentes.
- **Juez de aprobación:** prevalece un perfil de juez seleccionado explícitamente. Si no, se prefiere el perfil nativo real del agente/ejecución de origen (incluido vínculo, grupo o conmutación); origen de nube o ausente usa el activo. Para juez nativo, tiempo predeterminado **90 s** y fallo predeterminado **preguntar**. Prevalecen ajustes guardados; la aprobación de usuario sigue caducando a los 300 segundos con denegación por defecto. La salida nativa se restringe a esquema JSON decision/comment y se valida. El formulario no puede guardar hasta cargar correctamente su configuración del servidor.

### Servidores genéricos compatibles con OpenAI (vLLM, LM Studio)

Los formatos existentes siguen disponibles para estos servidores: elige **OpenAI (Chat Completions)** y dirige la URL base a su endpoint `/v1`. No uses el formato Ollama nativo.

- **Cargar modelos** en formato nativo lee `/api/tags`; si devuelve HTTP 404 o 405 recurre a `/v1/models`, por lo que puede aparecer la lista de un servidor genérico.
- El chat **nunca** cambia silenciosamente: un perfil nativo siempre llama a `/api/chat`, que un servidor genérico no ofrece. El error «modelo o endpoint no encontrado» y la indicación `ollama pull` no se aplican allí; descargar un modelo no arregla vLLM ni LM Studio. Cambia el formato a OpenAI.

### Cargar modelos y Probar conexión

**Cargar modelos** lee `/api/tags` y `/api/show`. Cada entrada muestra parámetros, cuantización e indicadores `tools` / `thinking` / `vision`. Si omite `tools`, muestra: *«Advertencia: este modelo no declara llamadas nativas a herramientas. Los agentes de ScopeWeaver las requieren para actualizar los grafos de activos y exploración»*. Los metadatos fallidos se muestran desconocidos, no ausentes.

**Probar conexión** comprueba el modelo con `/api/show`, ejecuta una prueba de llamada a herramienta de un turno con `num_ctx`, `keep_alive` y `num_predict`, y lee `/api/ps`. Muestra si el contexto asignado cubre la petición, si volvió una llamada, tiempo de carga en frío en milisegundos y velocidad en tokens/s. Un modelo ausente muestra el comando exacto para el servidor, por ejemplo `ollama pull qwen2.5-coder:32b`. ScopeWeaver no descarga modelos ni crea Modelfiles.

### Enrutamiento de host y contenedor

| Ejecución de ScopeWeaver | Ejecución de Ollama | URL base |
| --- | --- | --- |
| En el host | En el mismo host | `http://127.0.0.1:11434` |
| En Docker Compose | En el host | `http://host.docker.internal:11434` |
| En Docker Compose | Servicio `ollama` incluido | `http://ollama:11434` |

Dentro de un contenedor, `127.0.0.1` es el propio contenedor. Compose asigna `host.docker.internal` a la pasarela del host (`host-gateway`). Ollama escucha en `127.0.0.1:11434` por defecto; en Linux, Ollama del host debe escuchar en una dirección accesible al contenedor (por ejemplo `OLLAMA_HOST=0.0.0.0:11434`), exponiéndolo a la red. Restringe el puerto 11434 con un cortafuegos.

**Servicio Compose opcional:** `docker compose --profile local-llm up -d` inicia `ollama` con `OLLAMA_NO_CLOUD=1`, `OLLAMA_KEEP_ALIVE=30m`, `OLLAMA_NUM_PARALLEL=1`, `OLLAMA_CONTEXT_LENGTH=32768` y volumen `ollama_data`. Descarga un modelo con `docker compose exec ollama ollama pull qwen2.5-coder:14b`. Sin perfil, Compose se comporta como antes. El servicio no solicita GPU. **En macOS los contenedores Docker no pueden usar GPU Apple, por lo que solo ejecuta en CPU**; para velocidad útil, ejecuta Ollama nativamente en Mac y usa la plantilla de host.

### Política de red (`local_llm_only`)

Desactivada por defecto; los proveedores de nube siguen disponibles. Actívala en **Configuración del sistema → Red LLM exclusivamente local** o con `SCOPEWEAVER_LOCAL_LLM_ONLY=true`. Al activarla:

- HTTP de LLM (inferencia, listas, pruebas de conexión, precarga y juez) permite **loopback y CIDR de confianza explícitos** (`local_llm_allowed_cidrs`). IPv4 privado, IPv6 ULA, LAN y Docker ya no son de confianza implícita. Añade solo direcciones de servidores de inferencia que controles, preferiblemente `/32` para IPv4 o `/128` para IPv6.
- Las direcciones de enlace local, multidifusión y sin especificar siempre se rechazan, incluso en un CIDR de confianza. Se comprueban todas las direcciones resueltas de nombres DNS al abrir conexión; todas deben estar permitidas. Se rechazan todas las redirecciones y proxies de perfiles LLM.
- **Se bloquean búsqueda web de agentes (todos los motores), pruebas de conexión de búsqueda, comprobaciones y descargas de actualizaciones.** La pantalla distingue esta política intencionada de un fallo de red. Siguen disponibles estado local, progreso y reversión; no se devuelve información remota almacenada para eludir la política.
- Los cambios se aplican sin reiniciar, incluso a sesiones existentes. Una descarga admitida antes de activar el modo se detiene en la siguiente operación de red/lectura de cuerpo protegida y elimina la preparación parcial; el tráfico ya enviado no puede recuperarse.
- Los CIDR de `SCOPEWEAVER_LOCAL_LLM_ALLOWED_CIDRS` (o `ARTEX_LOCAL_LLM_ALLOWED_CIDRS` conservado) también se aplican, no se copian a la lista guardada ni se editan en la interfaz. Un `true` en entorno establece un mínimo que la interfaz no puede desactivar.
- El tráfico y herramientas de objetivos y el proxy global conservan conexión. Es una política de aplicación, no garantía criptográfica ni desconexión de todo el host.

**Migración de perfiles estrictos LAN/Docker existentes:** `http://127.0.0.1:11434` en el mismo equipo no necesita CIDR. Un endpoint LAN como `http://192.168.1.20:11434` necesita `192.168.1.20/32`; una dirección IPv6 como `fd00::20` necesita `fd00::20/128`. Para Docker resuelve el nombre **desde el contenedor ScopeWeaver**, porque host y contenedor pueden ver direcciones diferentes:

```bash
# Existing Compose service; read its actual name resolution and network addresses.
docker compose exec scopeweaver getent ahosts host.docker.internal
docker compose exec scopeweaver getent ahosts ollama
docker inspect --format '{{range .NetworkSettings.Networks}}{{.IPAddress}} {{.GlobalIPv6Address}}{{end}}' "$(docker compose ps -q ollama)"
```

Si la imagen no tiene `getent`, inspecciona la asignación host-gateway con `docker compose exec scopeweaver cat /etc/hosts` y la dirección del contenedor Ollama con `docker inspect`. Usa la IP de inferencia realmente alcanzada por ScopeWeaver, no un ejemplo copiado ni toda una subred Docker. Añádela a **CIDR de confianza adicionales** o a la lista de entorno, por ejemplo `SCOPEWEAVER_LOCAL_LLM_ALLOWED_CIDRS=192.168.1.20/32`. Ni `host.docker.internal` ni `ollama` conceden confianza implícita. Comprueba tras recrear el servicio o cambiar su dirección; actualiza la lista restringida antes de reintentar **Cargar modelos** o **Probar conexión**.

ScopeWeaver no puede ver qué hace después un servidor Ollama. **Ollama puede reenviar solicitudes a sus modelos de nube**; Compose incluido establece `OLLAMA_NO_CLOUD=1`, pero Ollama del host o remoto debe desactivar la nube aparte (`OLLAMA_NO_CLOUD=1` o `disable_ollama_cloud` en `~/.ollama/server.json`, según las [FAQ](https://docs.ollama.com/faq)).

### Entorno y configuración

`SCOPEWEAVER_LLM_*` prevalece sobre `ARTEX_LLM_*`; una variable moderna explícitamente vacía también sustituye a la antigua. Ambas prevalecen sobre el bloque `llm` de `config.json` (ruta desde `SCOPEWEAVER_CONFIG` o `ARTEX_CONFIG`).

| Sufijo | Significado |
| --- | --- |
| `PROVIDER` | `anthropic`, `openai`, `openai-responses`, `opencode-go` u `ollama` |
| `MODEL`, `BASE_URL`, `PROXY` | Modelo, endpoint y proxy LLM opcional |
| `API_KEY` | Obligatoria para Anthropic/OpenAI salvo URL base explícitamente local; token opcional para Ollama |
| `CONTEXT_WINDOW_K`, `MAX_TOKENS`, `KEEP_ALIVE`, `MAX_CONCURRENT` | Controles descritos arriba |
| `THINKING_TYPE`, `REASONING_EFFORT`, `STREAM` | Opciones existentes de razonamiento y transmisión |

Variables de política: `SCOPEWEAVER_LOCAL_LLM_ONLY` / `ARTEX_LOCAL_LLM_ONLY` y `SCOPEWEAVER_LOCAL_LLM_ALLOWED_CIDRS` / `ARTEX_LOCAL_LLM_ALLOWED_CIDRS` separados por comas.

Al iniciar, Anthropic y OpenAI usan `ANTHROPIC_API_KEY` u `OPENAI_API_KEY` si no hay clave genérica configurada, también en pasarelas locales autenticadas. `opencode-go` usa `OPENCODE_API_KEY`, nunca `OPENAI_API_KEY`, y se infiere si solo está definida `OPENCODE_API_KEY`. Para iniciar deliberadamente un perfil local sin clave estando definidas esas claves, establece explícitamente `SCOPEWEAVER_LLM_API_KEY=` (o `ARTEX_LLM_API_KEY=`). Sustituye credenciales del archivo y evita recurrir a la clave del proveedor. Ollama nativo solo recurre a `OLLAMA_API_KEY`, nunca a claves Anthropic, OpenAI ni OpenCode.

Guardar un perfil nuevo de interfaz con clave vacía no importa credenciales del entorno. **Probar conexión** con clave vacía puede reutilizar la guardada si se proporciona ID de perfil. Sin ID, una prueba local Anthropic/OpenAI reutiliza la activa solo si coinciden formato, URL base normalizada y proxy, también en borradores nuevos. Otro endpoint local, formato o proxy no hereda la activa; Ollama nativo nunca hereda clave de nube activa. Las pruebas no locales existentes conservan su alternativa de clave activa.

```bash
# .env for ScopeWeaver in Docker with Ollama on the host
# Replace your-installed-tool-model with a tool-capable tag shown by Load models.
SCOPEWEAVER_LLM_PROVIDER=ollama
SCOPEWEAVER_LLM_BASE_URL=http://host.docker.internal:11434
SCOPEWEAVER_LLM_MODEL=your-installed-tool-model
SCOPEWEAVER_LLM_CONTEXT_WINDOW_K=32
SCOPEWEAVER_LLM_MAX_CONCURRENT=1
SCOPEWEAVER_LLM_KEEP_ALIVE=30m
SCOPEWEAVER_LLM_MAX_TOKENS=4096
SCOPEWEAVER_LOCAL_LLM_ONLY=false   # add the actual host /32 or /128 before enabling strict mode
# SCOPEWEAVER_LOCAL_LLM_ALLOWED_CIDRS=192.168.1.20/32  # replace with the resolved inference IP
```

```json
{
  "llm": { "provider": "ollama", "base_url": "http://127.0.0.1:11434", "model": "your-installed-tool-model", "context_window_k": 32, "llm_max_concurrent": 1, "keep_alive": "30m", "max_tokens": 4096 },
  "local_llm_only": false,
  "local_llm_allowed_cidrs": []
}
```

### Diagnósticos y telemetría limitados

**Probar conexión** realiza un sondeo de inferencia deliberado; elegir plantilla solo descubre metadatos instalados. Además de herramienta/contexto y tiempo/velocidad, los diagnósticos nativos muestran `size` y `size_vram` de `/api/ps` como porcentaje de memoria GPU. Tamaño total ausente/cero o VRAM ausente significan **desconocido**, no 0% GPU. VRAM explícitamente cero con total positivo significa 0%. La proporción estima memoria, no reparto exacto de pesos, prueba de capacidad GPU ni garantía de permanencia del modelo.

Un porcentaje del 80% o menor avisa de posible lentitud por CPU. Otros modelos cargados y un juez explícito con modelo/contexto distinto en el mismo endpoint generan avisos separados de memoria/recarga. Usar el mismo modelo/contexto puede reducir recargas, sin garantizar permanencia. Ningún diagnóstico descarga modelos ni guarda/activa perfiles.

Los avisos de cercanía al límite son una heurística de registros del servidor basada en tokens, no un detector de truncamiento. Los tiempos Ollama originales completos solo se guardan con grabación sin procesar activada; tokens/caché se miden independientemente. Los modelos pequeños aún necesitan validación por tarea; siguen vigentes las limitaciones de fiabilidad siguientes.

### Niveles de hardware

Puntos iniciales aproximados por VRAM o memoria unificada Apple, con modelos de catálogo capaces de herramientas siguientes. **Son estimaciones, no mediciones comparativas.** El contexto es la **Ventana de contexto (K)** del perfil; debe caber en el límite del modelo y memoria (pesos + caché KV, que crece con contexto y `OLLAMA_NUM_PARALLEL`). Comprueba con **Probar conexión**, que indica si se asignó el contexto solicitado.

| Memoria | Modelo de ejemplo | Ventana de contexto (K) | Función |
| --- | --- | --- | --- |
| 8–12 GB | `qwen2.5-coder:7b`, `qwen3:8b` | `16`–`32` | Juez de aprobación o perfil auxiliar; débil como agente principal |
| 16–24 GB | `qwen2.5-coder:14b`, `qwen3:14b` | `32` | Agente principal en estación de trabajo |
| 32–48 GB | `qwen2.5-coder:32b`, `qwen3:32b` | `32` | Agente principal. Usa `64` solo si lo permite el modelo; estos indican 32K y 40K |
| 64–128 GB+ | `llama3.3:70b` | `64` (modelo indica 128K) | Agente principal en hosts de gran memoria; 64K necesita la parte alta del rango |

Las plantillas eligen modelos instalados en el servidor seleccionado y usan contexto 32K. Los ejemplos de hardware no son elecciones automáticas.

### Modelos candidatos

Solo se listan modelos cuya página de [biblioteca Ollama](https://ollama.com/library) indica capacidad `tools` (comprobado el 2026-10-15). Los tamaños descargados usan cuantización predeterminada. **La orientación de memoria es estimada**: pesos más caché KV que crece con contexto y solicitudes paralelas. ScopeWeaver no ha medido estos modelos/hardware y la calidad de llamadas varía.

| Modelo | Descarga | Notas |
| --- | --- | --- |
| `qwen2.5-coder:7b` | 4.7 GB | Candidato práctico menor; también candidato a juez |
| `qwen2.5-coder:14b` | 9.0 GB | Candidato para estación de trabajo |
| `qwen2.5-coder:32b` | 20 GB | Candidato para host mayor |
| `qwen3:8b`, `qwen3:14b`, `qwen3:32b` | 5.2 / 9.3 / 20 GB | También declaran `thinking` |
| `llama3.3:70b` | 43 GB | Hosts de gran memoria |

`qwen3:72b` y `llama3.2:8b` no aparecen en la biblioteca Ollama; `llama3.2` se distribuye en 1B y 3B. Confirma `tools` con **Cargar modelos** en tu servidor antes de usar.

**Limitación observada (2026-10-15):** declarar `tools` no garantiza llamadas en el protocolo. En pruebas de rama, `qwen2.5-coder:7b` escribió JSON de herramientas como texto plano y no ejecutó herramientas; `qwen3:4b` pasó el diagnóstico pero no completó tareas autónomas largas. Considera los modelos pequeños sin validar como agente principal; consulta [Verificación](VERIFICATION.md#la-fiabilidad-de-tareas-autónomas-no-está-validada).

Las restricciones del [README](../../README.es.md) siguen vigentes: ejecuta ScopeWeaver solo en un entorno aislado local, elijas el proveedor que elijas.

[Guía en coreano](../ko/llm-providers.md)
