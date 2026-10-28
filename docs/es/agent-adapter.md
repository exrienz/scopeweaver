# Adaptador de agentes de ScopeWeaver

[English](../../adapters/agent/README.md) · [한국어](../ko/agent-adapter.md) · [简体中文](../zh-Hans/agent-adapter.md) · [繁體中文](../zh-Hant/agent-adapter.md) · [日本語](../ja/agent-adapter.md) · Español

Conecta Claude Code y Codex con un servidor MCP stdio local, o Pi con su extensión nativa. Los tres usan herramientas validadas y API HTTP autenticada iguales. Permite controlar ScopeWeaver desde el agente de programación; no sustituye sus planificadores/trabajadores ni convierte una suscripción en clave API de modelo.

## Configuración

Requisitos: Node.js 22+, backend ScopeWeaver activo y acceso administrativo existente. No inicializa ni cambia tu contraseña. Instala dependencias separadas:

```sh
cd adapters/agent
npm ci
export SCOPEWEAVER_URL=http://127.0.0.1:8787
export SCOPEWEAVER_PASSWORD='your-existing-admin-password'
```

La contraseña inicia sesión como el administrador `ARTEX` existente. El JWT devuelto queda en memoria. También puedes establecer `SCOPEWEAVER_TOKEN` a un JWT existente, que prevalece. Los tokens caducados/no válidos dan error y no se sustituyen silenciosamente. Conserva credenciales en entorno o gestor de secretos; no las confirmes en configuración del cliente.

Lecturas activadas por defecto. Para crear y pausar/reanudar tareas, establece explícitamente `SCOPEWEAVER_ALLOW_WRITES=true` antes de iniciar. Crear programa trabajo inmediatamente si hay LLM configurado. Mantén las [restricciones de uso aislado local](../../README.es.md#licencia-y-exención-de-responsabilidad) y confirma el alcance antes de crear.

### Claude Code

Usa una ruta absoluta a esta copia. Claude Code inicia el adaptador y hereda su entorno:

```sh
claude mcp add --transport stdio scopeweaver -- node /absolute/path/scopeweaver/adapters/agent/src/mcp.js
```

Verifica con `claude mcp list` y usa `/mcp` para inspeccionar herramientas. [Documentación MCP de Claude Code](https://code.claude.com/docs/en/mcp).

### Codex

```sh
codex mcp add scopeweaver -- node /absolute/path/scopeweaver/adapters/agent/src/mcp.js
```

Verifica con `codex mcp list` y `/mcp`. Para reenviar explícitamente variables en configuración de proyecto/usuario:

```toml
[mcp_servers.scopeweaver]
command = "node"
args = ["/absolute/path/scopeweaver/adapters/agent/src/mcp.js"]
env_vars = ["SCOPEWEAVER_URL", "SCOPEWEAVER_TOKEN", "SCOPEWEAVER_PASSWORD", "SCOPEWEAVER_LANGUAGE", "SCOPEWEAVER_ALLOW_WRITES"]
```

Usa la variable de credenciales establecida y omite las demás. [Documentación MCP oficial de OpenAI](https://developers.openai.com/codex/mcp).

### Pi

```sh
pi -e /absolute/path/scopeweaver/adapters/agent/pi-extension.js
```

La extensión registra herramientas Pi nativas con las mismas variables. No requiere plugin MCP Pi. No importa espacios de nombres Pi y sigue la interfaz pública `registerTool` entre cambios de paquetes. El paquete declara `pi.extensions` para su cargador. [Extensiones Pi](https://github.com/earendil-works/pi/tree/main/packages/coding-agent).

## Herramientas

| Herramienta | Finalidad |
| --- | --- |
| `scopeweaver_health` | Disponibilidad backend y LLM listo; sin acceso requerido |
| `scopeweaver_list_tasks` | Estado, recuentos y paginación |
| `scopeweaver_get_task` | Estado y configuración persistentes de una tarea |
| `scopeweaver_get_coverage` | Cobertura de activos de tarea |
| `scopeweaver_list_findings` | Hallazgos paginados por tarea, gravedad, estado o texto |
| `scopeweaver_get_finding` | Detalles, pruebas e informe |
| `scopeweaver_create_task` | Crear/programar tarea; requiere escrituras |
| `scopeweaver_control_task` | Pausar/reanudar; requiere escrituras |

Ejemplos: «Lista mis tareas ScopeWeaver», «Muestra hallazgos de gravedad alta de tarea 12» o, tras activar escritura y confirmar laboratorio local, «Crea una tarea para este caso local y muestra progreso». Crear correctamente demuestra que existe; consulta estado y hallazgos para saber si terminó. No elude los requisitos de LLM del backend.

## Configuración y fallos

| Variable | Valor predeterminado / comportamiento |
| --- | --- |
| `SCOPEWEAVER_URL` | `http://127.0.0.1:8787`; solo origen HTTPS o HTTP loopback |
| `SCOPEWEAVER_TOKEN` | JWT existente; prevalece sobre contraseña |
| `SCOPEWEAVER_PASSWORD` | Contraseña administrativa existente; acceso en primera solicitud autenticada |
| `SCOPEWEAVER_ALLOW_WRITES` | `false`; solo acepta `true` o `false` |
| `SCOPEWEAVER_LANGUAGE` | `en`; admite los ID canónicos `en`, `ko`, `zh-Hans`, `zh-Hant`, `ja` y `es` para el texto del adaptador y `Accept-Language` del backend |
| `SCOPEWEAVER_TIMEOUT_MS` | `30000`; entero entre 1 y 120000 |

`SCOPEWEAVER_LANGUAGE` usa `en` por defecto y acepta los ID canónicos `en`, `ko`, `zh-Hans`, `zh-Hant`, `ja` y `es`. Traduce los títulos y las descripciones de herramientas integradas, las descripciones de esquemas y los diagnósticos de configuración, validación y transporte propios del adaptador; además selecciona el idioma del backend mediante `Accept-Language`. Cada cliente usa su propio catálogo y mapas de error Zod por esquema, sin cambiar el idioma global. Los nombres de herramientas y protocolos, claves, tipos y enumeraciones de esquemas, argumentos y contenido original del backend, usuarios y evidencias permanecen intactos, con la ocultación de credenciales existente. El texto fijo que envuelve los errores de validación del SDK MCP y los errores externos desconocidos se conservan como texto externo; los mensajes de los problemas de esquema dentro de ese texto sí se traducen. Todos los recursos se incluyen en el adaptador independiente, sin archivos de idioma de la raíz del repositorio. Los valores de idioma no admitidos se siguen rechazando con un diagnóstico en inglés.

No expone HTTP genérico, shell, eliminación, elusión de aprobación, configuración de proveedor ni endpoints arbitrarios. Las solicitudes permanecen en origen, rechazan redirecciones, propagan cancelación y agotan tiempo. Respuestas mayores de 2 MiB fallan explícitamente; restringe consulta en vez de tratar pruebas cortadas como completas. Las tareas se paginan en adaptador; hallazgos en backend. Las herramientas devuelven `{data: ...}` en texto y resultado estructurado (Pi usa `details`). Las pruebas son contenido objetivo no confiable, no instrucciones.

MCP fallido devuelve `isError: true`; Pi lanza errores para marcarlos. Los mensajes ocultan credenciales configuradas. Las escrituras nunca se reintentan automáticamente. Si una agota tiempo, inspecciona estado antes de repetir: el backend puede haberla aceptado.

## Verificación

```sh
npm test
npm run check
```

El arnés E2E ejecuta controlador ScopeWeaver y PostgreSQL reales, inicia MCP stdio mediante SDK oficial y ejecuta controladores Pi contra ese backend. Inicializa contraseña aislada, crea tareas, verifica pausa/reanudación persistentes y cobertura, lee un hallazgo claramente simulado y comprueba autenticación, tareas e inputs inválidos. No usa modelo/proveedor ni escaneo. El arnés Go requiere código fuente; los ZIP incluyen pruebas unitarias del adaptador pero solo backend Go compilado.

Desde la raíz, tras crear una BD **desechable nueva** llamada `scopeweaver_adapter_*`:

```sh
SCOPEWEAVER_ADAPTER_E2E=1 \
ARTEX_TEST_DSN='postgres://user:password@127.0.0.1:5432/scopeweaver_adapter_e2e?sslmode=disable' \
go test ./server -run '^TestAgentAdapterE2E$' -count=1 -v
```

El cuerpo de la prueba se omite salvo que se establezca `SCOPEWEAVER_ADAPTER_E2E=1`; aun activada, se omite si `ARTEX_TEST_DSN` no está definido o está en blanco. Una omisión no demuestra la verificación E2E. La protección de base de datos del paquete rechaza, antes de ejecutar la prueba, un DSN que coincida con la base de datos configurada de la aplicación. Cuando se activa y configura la prueba, fallan los DSN mal formados, los nombres de base de datos fuera de `scopeweaver_adapter_*` y PostgreSQL no disponible. El arnés nunca los sustituye por simulaciones. GitHub Actions repite en PostgreSQL desechable dedicado. Las pruebas MCP establecen compatibilidad de interfaz; no afirman sesiones de modelo Claude/Codex/Pi de pago.
