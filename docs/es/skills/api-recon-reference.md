# Referencia de api-recon

[English](../../../skills/api-recon/reference.md) · [한국어](../../ko/skills/api-recon-reference.md) · [简体中文](../../zh-Hans/skills/api-recon-reference.md) · [繁體中文](../../zh-Hant/skills/api-recon-reference.md) · [日本語](../../ja/skills/api-recon-reference.md) · Español

Recetas grep, plantillas `config.json` y diagnóstico. Ejecuta grep en `js/`. Para paquetes de una línea, opcionalmente usa `js-beautify` o `sed 's/}/}\n/g'`; normalmente basta grep original con ventanas de contexto.

## Notas de scripts

Todos los archivos de `scripts/` son plantillas de referencia. Adáptalas al objetivo antes de ejecutar. Cambios habituales:

| Script | Adaptaciones habituales |
|---|---|
| `harvest_static.py` | Regex de endpoints, manifiestos webpack/Vite, publicPath de microfrontend, reintentos/concurrencia |
| `runtime_harvest.js` | Campos/valores de éxito neutralizados, coincidencia/estructura de simulaciones, orígenes de rutas, captura WS, `waitUntil`/`routeTimeout`/`proxy` |
| `preload.js` | `loginPathRe`, simulaciones L1, `neutralize.fields`, `apiPattern`, activar L3, `recordDetail`, `observe.*`, `neutralizeVueRouter` |
| `spider_mpa.py` | `--exclude` para enlaces destructivos, cookies, profundidad/máximo, filtros de mismo origen |
| `extract_route_map.py` | Regex `routeMap` / `routeLink`, patrones de nombres KEY |
| `build_perm_tree.py` | Interpretación de `userRouteAuth`, heurísticas jerárquicas `ROOTS`/`PREFIX_PARENT`, campos exteriores de simulaciones |
| `config.json` | Configuración central de los parámetros específicos anteriores |

Conserva las adaptaciones en el directorio de tarea, p. ej., `recon/`, e informa de cambios exactos respecto a las plantillas.

---

## A. Reconstruir los tres controles

### A1. Renderizado: ¿cómo se determina el estado de acceso?

```bash
grep -rhoaE '.{0,40}(isLogin|isAuthenticated|loggedIn|hasLogin|requireAuth)\b.{0,80}' js | head
grep -rhoaE 'function (getUser|getToken|getAuth)[0-9]?\([^)]*\)\{.{0,200}' js | head
grep -rhoaE '(localStorage|sessionStorage)\.getItem\("[^"]+"\)' js | sort -u
grep -rhoaE '(Cookies?|cookie)\.(get|load)\("[^"]+"\)' js | sort -u
grep -rhoaE '\batob\(|JSON\.parse\(|jwt|decode' js | head
```

Sigue `isLogin = f(getUser())` → `getUser = decode(storage.read(KEY))` para identificar clave, contenedor (Cookie/localStorage) y codificación:

| Codificación | Valor simulado en configuración |
|---|---|
| Cadena simple / `"1"` / token | `"value": "anything-truthy"` |
| `JSON.parse(x)` | `"value": "json:{\"id\":1,\"username\":\"admin\"}"` |
| `JSON.parse(atob(x))` | `"value": "b64json:{\"id\":1,\"username\":\"admin\"}"` |
| JWT | JWT sin firma/`alg:none`, o firmar con clave del paquete, solo para simular control del cliente |
| Cifrado (SM2/AES/RSA) | Buscar claves fijas; simular solo si el control necesita un bloque descodificable; si no, análisis estático |

Escribe en `cookies` / `localStorage`.

### A2. Interceptor: ¿qué provoca redirecciones a /login?

```bash
grep -rhoaE '.{0,60}(interceptors\.response|axios|request\.use).{0,120}' js | head
grep -rhoaE '.{0,40}(response_code|errcode|errno|\bcode\b|\bret\b|\bstatus\b)\s*[=!]==?\s*[\-0-9]{1,4}.{0,60}' js | head -20
grep -rhoaE '.{0,40}(未登录|请重新登录|登录已过期|unauthorized|登录失效|授权|token.{0,10}invalid).{0,40}' js | head
grep -rhoaE '.{0,30}(location\.href|router\.(push|replace)|navigate)\([^)]*login[^)]*\)' js | head
```

Identifica nombres de campos, valores de éxito (normalmente `0` o `200`) y fallos que redirigen. Verifica con una sesión inválida:

```bash
curl -sk -X POST -H 'Cookie: <fakekey>=junk' https://target/api/<protected> -d '{}' -H 'Content-Type: application/json'
```

Escribe en `neutralize.fields` + `neutralize.success`.

### A3. Contenido: ¿de dónde vienen menús y permisos?

```bash
grep -rhoaE '"/api[^"]*(permission|perm|role|menu|acl|resource|nav)[^"]*"' js | sort -u
grep -rhoaE '.{0,30}(menus|permissions|menuList|routeList|authList|role_permissions)\b.{0,120}' js | head
grep -rhoaE 'userRouteAuth|getResultTree|routeMap|routeLink|hasPermission|checkAuth' js | head
grep -rhoaE '([A-Z_][A-Z0-9_]*):\{name:"[^"]*",link:"/[^"]+"\}' js | head
```

Dos capas habituales en interfaces administrativas:

| API | Carga habitual | Consumidor |
|---|---|---|
| `.../role_permissions` | `{ permissions: string[], role_type }` | Controles de rutas, ACL por botón |
| `.../permissions/all` | `tree[{ code, position, children }]` | Renderizado del menú lateral |
| En paquete: `userRouteAuth` | `{ CODE: { url, name? } }` | código → ruta frontend |
| En paquete: `routeMap` | `{ KEY: { name, link } }` | Resolución de alias (webpack `o.DASHBOARD`) |

Lee el consumidor para saber cómo filtra `getResultTree(tree, permissions)` y qué campos comprueban `v-if` / `hasAuth(code)`.

En sitios pequeños, construye manualmente una carga permisiva simulada en `stubs`.

En sitios grandes con barras/módulos vacíos, reconstruye el árbol completo en sección I.

---

## B. Plantilla config.json

```json
{
  "baseUrl": "https://target/",
  "runtimeMode": "both",
  "chromium": "/usr/bin/chromium",

  "cookies": [
    { "name": "auth", "value": "b64json:{\"id\":1,\"username\":\"admin\",\"role\":\"admin\",\"func\":{},\"permissions\":[\"*\"]}" }
  ],
  "localStorage": { "token": "faketoken", "isLogin": "1" },

  "neutralize": {
    "fields": ["response_code", "code", "errno", "ret", "status"],
    "success": 0,
    "flags": { "success": true, "message": "ok" }
  },
  "forward": true,
  "loginUrlPattern": "/login",
  "apiPattern": "/api/|/rest/|/graphql",

  "mockTier": "L1+L2",
  "recordDetail": true,
  "observe": {
    "storageReads": false,
    "cookieReads": false,
    "xhrHeaders": true
  },
  "neutralizeVueRouter": true,
  "stubs": [
    {
      "match": "permissions/all|/menu|role_permissions",
      "body": {
        "response_code": 0, "code": 0,
        "data": {
          "permissions": ["*"],
          "menus": [
            { "name": "dashboard", "path": "/dashboard", "show": true, "children": [] },
            { "name": "alert", "path": "/alert", "show": true, "children": [] }
          ]
        }
      }
    }
  ],

  "explore": {
    "clickTabs": true,
    "clickTables": true,
    "pushStateFallback": true,
    "maxMenuItems": 50
  },

  "routes": ["/dashboard", "/alert", "/asset", "/device", "/report", "/config", "/system"],
  "waitMs": 1500, "perRouteMs": 900, "headless": true,
  "waitUntil": "domcontentloaded",
  "routeTimeout": 12000,
  "proxy": "",

  "captureResponses": true, "recordWs": true, "respMax": 600
}
```

Campos:
- `runtimeMode`: `depth` (Puppeteer), `coverage` (MCP de navegador), `both`
- Prefijos de `cookies[].value`: `b64json:` → base64(JSON); `json:` → JSON original; sin prefijo → literal
- `forward: true` reenvía solicitudes reales y reescribe códigos de estado; `false` usa simulaciones sin conexión
- `mockTier`: capas de precarga coverage, p. ej., `L1+L2`, `L1+L2+L3`
- `routes` viene de `routes.txt`; tras menús simulados, el arnés añade rutas `<a href>`
- `captureResponses` / `recordWs` solo se aplican a depth
- `waitUntil`: usa `domcontentloaded` en SPA grandes para evitar bloqueos de `networkidle2`
- `routeTimeout`: tiempo de `page.goto` por ruta en milisegundos
- `proxy`: `--proxy-server` de Puppeteer; también `HTTP_PROXY` / `HTTPS_PROXY`

### B1. Plantilla de dos simulaciones (role_permissions + permissions/all)

```json
"stubs": [
  {
    "match": "role_permissions",
    "body": {
      "response_code": 0,
      "data": {
        "permissions": ["MONITOR", "MONITOR_ALERT", "THREAT", "ASSETS_RISK"],
        "role_type": "SUPER_ADMIN"
      }
    }
  },
  {
    "match": "permissions/all",
    "body": {
      "response_code": 0,
      "data": [
        {
          "code": "MONITOR",
          "position": 1,
          "children": [
            { "code": "MONITOR_ALERT", "position": 1, "children": [] }
          ]
        }
      ]
    }
  }
]
```

Los campos exteriores (`response_code` / `code` / `data`) deben coincidir con A2; `permissions` debe cubrir cada código hoja del árbol.

---

## C. Configuración de precarga coverage

Edita `CONFIG` al principio de tu copia de `preload.js` o sustitúyelo antes de inyectar mediante CDP:

```javascript
const CONFIG = {
  loginPathRe: /\/(login|signin)(\/|$|\?)/i,
  mockTier: 'L1+L2',
  forward: true,
  recordDetail: true,
  extractUrlsFromResponse: true,
  neutralizeVueRouter: true,
  observe: { storageReads: false, cookieReads: false, xhrHeaders: true },
  neutralize: { fields: ['response_code', 'code'], success: 0 },
  stubs: [ /* Same as config.json stubs */ ],
  apiPattern: /\/(api|apis|v\d+|dev|internal|graphql)\//i,
};
```

Verifica `window.__API_RECON_PRELOAD__ === true` y una ruta estable.

Exportar resultados de captura:

```javascript
JSON.stringify({
  apis: [...window.__API_RECON_LOG__],
  detail: window.__API_RECON_DETAIL__,
  routes: [...(window.__API_RECON_ROUTES__ || [])],
  observe: window.__API_RECON_OBSERVE__,
}, null, 2)
```

---

## D. Cobertura de interceptores de precarga/ejecución

Interceptores de navegador integrados en preload (coverage) y runtime_harvest (depth):

| Interceptor | Valor de descubrimiento | Cobertura |
|---|---|---|
| Interceptar fetch / XHR.open | Capturar URL/método | ✅ `recordDetail` + `__API_RECON_LOG__` |
| Interceptar XHR.setRequestHeader | Descubrir Authorization y cabeceras | ✅ `observe.xhrHeaders` |
| Interceptar lecturas localStorage/cookie | Identificar claves de sesión | Opcional `observe.storageReads/cookieReads` |
| Recoger rutas Vue | Completar frontendRoutes | ✅ `__API_RECON_ROUTES__` (rutas cargadas) |
| Neutralizar controles Vue / bloquear redirecciones de acceso | Renderizar módulos para activar API | ✅ `neutralizeVueRouter` + neutralización nativa de redirecciones |
| Recoger rutas React | Completar rutas | Estático + clics; sin interceptor dedicado |
| Bloquear redirecciones de página (acceso) | Permanecer para analizar | Bloquear solo acceso, conservando navegación de negocio |
| Interceptar bibliotecas de cifrado (CryptoJS/SM, etc.) | Parámetros cifrados → cuerpo API plano | Requiere interceptor manual de entrada; registrar hallazgos en configuración |
| Eludir antidepuración | Sin ello no se capturan API dinámicas | Gestión manual; estático sigue disponible |

---

## E. Regex de endpoints con pocos resultados estáticos

Amplía `extract_endpoints` de `harvest_static.py` o inspecciona manualmente tras la recolección obligatoria:

```bash
grep -rhoaE '"/[a-z][A-Za-z0-9_/\-]{3,}"' js | sort -u
grep -rhoaE '/api/[a-zA-Z0-9_./-]+' js | sort -u
```

---

## F. Diagnóstico de problemas

| Síntoma | Causa y acción |
|---|---|
| Pocas API estáticas | Sintaxis de endpoints distinta: ampliar regex (E) |
| Muchos menos fragmentos que en manifiesto | Solo CSS o no desplegados; reintentar 404 |
| Sigue mostrando acceso | Control de renderizado: revisar clave A1, contenedor, codificación y dominio |
| Estructura visible, módulos vacíos | Control de contenido: simular menús (A3); comprobar `routes` |
| Solo inicio/idioma por ruta | Faltan códigos: reconstruir árbol (I), comprobar simulaciones `role_permissions` y `permissions/all` |
| Barra lateral visible, subpáginas vacías | Faltan nodos intermedios o códigos distintos de `userRouteAuth` |
| Cada API redirige al acceso | Interceptor: revisar `neutralize`; ampliar recorrido de campos anidados |
| Cero tramas WS | Suscripción requiere interacción; aumentar `perRouteMs` |
| Respuesta vacía | Respuestas reales requieren `forward: true` |
| Falta Chromium | Instalar o establecer `config.chromium` / `CHROMIUM` |
| Simulaciones vuelven al acceso | Interceptor tardío o setter `location.href` ausente: precarga al inicio del documento |
| Listas vacías | Matrices L3 vacías son normales; continuar pestañas/ajustes/detalles |
| Acciones Redux confundidas con rutas | Filtrar rutas internas con get/set/change/clear/toggle/upload |
| Vue sigue redirigiendo | Inyectar al inicio; si `neutralizeVueRouter: false`, quitar controles del cliente manualmente |
| URL de respuestas ausentes del registro | Activar `extractUrlsFromResponse` o extraer de `__API_RECON_DETAIL__` |
| Nombre de cabecera de autorización desconocido | Activar `observe.xhrHeaders` o inspeccionar cabeceras DevTools |
| Ejecución lenta/tiempos agotados | Usar `waitUntil: domcontentloaded`, bajar `routeTimeout`, evitar `networkidle2` |
| Falla conexión de proxy | Revisar `proxy`/entorno; alinear puertos de Puppeteer y curl |

---

## G. Objetivos reforzados

Si el servidor valida sesiones durante todo el flujo (cookies firmadas imposibles de simular, menús renderizados en servidor que no admiten simulación), la ejecución se detiene en la estructura. Comportamiento esperado:

- El análisis estático sigue enumerando endpoints porque las rutas de módulos permanecen en código
- La referencia original también describe usar una sesión real explícitamente autorizada con el mismo arnés (`forward: true`, sin neutralize) para capturar métodos/parámetros/respuestas reales. Queda fuera del modo sin credenciales de esta habilidad y no anula la prohibición de pedir credenciales o iniciar sesión realmente.

---

## H. Lista por tarea

1. Confirmar alcance autorizado
2. Leer `scripts/harvest_static.py`, adaptar, ejecutar e inspeccionar `api_static.txt` y `routes.txt`
3. Fase 1b: ventanas de anclaje de rutas + capa de vínculos → `param_candidates.json` (J)
4. Reconstruir A1/A2/A3 → `config.json` específico
5. Leer y adaptar `runtime_harvest.js` / `preload.js` antes de ejecutar
6. `runtimeMode=depth`: `npm install` → ejecutar script adaptado
7. `runtimeMode=coverage/both`: inyectar precarga adaptada al inicio → enumeración MCP de navegador + matriz de parámetros
8. Módulos sin renderizar → reconstruir permisos (I) → parchear simulaciones → repetir
9. Diferencias entre muestras + inferencia de errores → `params_merged.json`
10. Combinar → `site_map.json` + `api_merged.txt`; indicar con precisión cobertura, lagunas y cambios de scripts

---

## I. Reconstrucción del árbol de permisos (detalle de fase 4)

Úsala cuando una simulación simple `menus: [{ path, show: true }]` no monta submódulos.

### I1. Localizar módulos de autenticación

```bash
grep -l 'userRouteAuth' js/*.js
grep -l 'routeMap\|routeLink' js/*.js
grep -rhoaE 'getResultTree|role_permissions|permissions/all' js | head
```

Registra rutas API de permisos, campos de respuesta y archivos de fragmentos consumidores.

### I2. Extraer routeMap

```bash
python3 scripts/extract_route_map.py recon/js recon/
# Output recon/route_map.json
```

Ante `[!] no routeMap pattern found`, amplía regex de `extract_route_map.py` o inspecciona con grep:

```bash
grep -rhoaE '([A-Z_][A-Z0-9_]*):\{name:"[^"]*",link:"/[^"]+"\}' js | head -20
```

### I3. Construir árbol y simulaciones

```bash
python3 scripts/build_perm_tree.py recon/js recon/ --config recon/config.json
```

Lógica del script:
1. Interpretar `userRouteAuth={MONITOR:{url:...},...}`, incluidos alias webpack como `He=o.DASHBOARD`
2. Resolver alias a rutas reales mediante `route_map.json`
3. Inferir padre del prefijo (`MONITOR_ALERT` → `MONITOR`)
4. Producir `permissions_tree.json`, `permissions_all_stub.json`, `role_permissions_stub.json`
5. Con `--config`, actualizar `stubs` y ampliar `routes` en `config.json`

Adapta al principio de la copia:
- `DEFAULT_ROOTS`: códigos de módulos superiores
- `DEFAULT_PREFIX_PARENT`: asignación `PREFIX_` → padre
- `DEFAULT_EXTRA_PARENT`: nodos huérfanos sin relación de prefijo

### I4. Verificar coherencia de simulaciones

```bash
# Permission count should approximate userRouteAuth entries
wc -l recon/perm_codes_all.txt
# routes should cover all route_map links
python3 -c "import json; m=json.load(open('recon/route_map.json')); r=set(json.load(open('recon/config.json'))['routes']); print('missing', [v['link'] for v in m.values() if v['link'] not in r])"
```

### I5. Repetir ejecución y comparar

```bash
node recon/runtime_harvest.js recon/config.json
# Compare runtime_api.json counts before/after mocks; check module APIs under /attack, /asset, etc.
```

| Antes de simular | Simulación correcta |
|---|---|
| Las mismas 3–5 solicitudes de inicio por ruta | Rutas diferentes activan API de módulos diferentes |
| Solo `/api/locale/language` | Endpoints de módulos como `/api/web/...` |
| Menos de diez rutas en `routes.txt` | 80–110+ `routes` de route_map |

### I6. Si sigue fallando

- Cobertura: pulsa barra lateral y pestañas; permisos pueden requerir interacción
- Campos simulados: compara anidación con una respuesta ya autorizada. El ejemplo original usaba curl con sesión real; no obtengas credenciales ni inicies sesión con esta habilidad sin credenciales.
- Controles extra: busca comprobaciones de botones como `hasPermission|checkRole|func.`, amplía `role_permissions.permissions`
- Alternativa estática: las rutas quedan en `api_static.txt`; la ejecución añade métodos/cuerpos. Conserva `param_candidates.json` y muestras

---

## J. Reconstrucción de parámetros (fases 1b / 5b / 5c)

Un método, no un script universal. Usa regex para rutas; ventanas, vínculos UI, diferencias e inferencia de errores para parámetros.

### J1. Ventanas: de ruta a constructor de carga

```bash
# Anchor on known Phase 1 paths
grep -n '"/api/user/list"' js/*.js
grep -rhoaE '.{0,120}("/api[^"]+").{0,200}' js | head
grep -rhoaE '(params|data|body|payload)\s*:\s*\{' js | head
grep -rhoaE '(get|post|put|delete|patch)\([^,]+,\s*\{' js | head
```

### J2. Envoltorios y transportes

```bash
# axios / shared request wrapper
grep -rhoaE '(axios|request)\.(get|post|put|delete|patch)\(' js | head
grep -rhoaE 'interceptors\.(request|response)' js | head

# GraphQL
grep -rhoaE '(query|mutation)\s+\w+|gql`|graphql\(' js | head
grep -rhoaE '\$[a-zA-Z_]+\s*:\s*(Int|String|Boolean|\[)' js | head

# FormData / multipart
grep -rhoaE 'FormData|\.append\(' js | head

# Path parameters
grep -rhoaE 'path:\s*"/[^"]*:[^"]+"' js | head
grep -rhoaE 'useParams|route\.params|\$route\.params' js | head
```

### J3. Validación: campos obligatorios, formatos y enumeraciones

```bash
grep -rhoaE '(required|message|pattern|enum|validator)\s*:' js | head
grep -rhoaE 'yup\.|zod\.|async-validator|Form\.Item|a-form-item|el-form-item' js | head
grep -rhoaE 'rules\s*:\s*\[|name:\s*["\'][a-zA-Z_]+["\']' js | head
grep -rhoaE 'label.*value|options\s*:\s*\[' js | head
```

### J4. Capa de vínculos: del formulario a API

```bash
grep -rhoaE 'onFinish|handleSubmit|getFieldsValue|validateFields' js | head
grep -rhoaE '(pick|omit|transform|dayjs|moment)\(' js | head
```

Pruebas dinámicas: DevTools → Network → solicitud → Initiator (pila); sigue desde `fetch`/`send` hasta montar la carga.

### J5. Parámetros cifrados

```bash
grep -rhoaE 'encrypt|decrypt|sign|CryptoJS|sm2|sm3|sm4|RSA|AES' js | head
```

Nunca adivines campos desde texto cifrado. Intercepta entradas de cifrado, registra la carga en claro antes de cifrar y guarda conclusiones en `config.json` / `param_candidates.json`.

### J6. Matriz de activación de parámetros (obligatoria en fase 3)

Registra cada operación por módulo y compara cuerpos/consultas:

| Operación | Inspeccionar |
|---|---|
| Lista inicial | Paginación predeterminada |
| Búsqueda | keyword, filtros |
| Filtros avanzados | Campos opcionales |
| Crear/editar | Entidad completa |
| Masivo/exportación | `ids[]`, `exportType` |
| Orden/página | `sortField`, `order` |

Producir `param_samples.json`: `[{ "path", "method", "action": "search", "body", "query", "headers" }]`

### J7. Reglas de confianza

| Confianza | Condición |
|---|---|
| Alta | Punto de llamada estático y al menos dos muestras dinámicas coincidentes |
| Media | Solo estático o una muestra dinámica |
| Baja | Inferida de respuesta/error sin segunda verificación |
| Pendiente de activación | Campo estático conocido, ruta UI/permisos no alcanzada |

### J8. Recetas de escenarios

| Escenario | Orden |
|---|---|
| Lista REST | J1 objeto de carga → J6 cuatro diferencias → J3 reglas |
| Formulario crear/editar | J3 nombre → J4 cadena de envío → envío dinámico autorizado no destructivo e inspección 400 de campos vacíos |
| GraphQL | J2 declaraciones → capturar variables de cada operación |
| Cuerpo cifrado | J5 entradas interceptadas → los campos previos al cifrado son parámetros reales |

### J9. Relación con fases api-recon

| api-recon | Reconocimiento de parámetros |
|---|---|
| Fase 1 estática | J1 ventanas de anclaje |
| Fase 2 A2 interceptor | Campos globales inyectados (tenantId, sign) |
| Fase 3 dinámica | J6 matriz + `param_samples.json` |
| Fase 4 permisos | Los módulos tienen formularios distintos; permisos simulados suficientes exponen todos los campos |
| Fase 5 combinación | `params_merged.json` + confianza; no concluir obligatoriedad de una muestra |

### J10. Diagnóstico de problemas

| Síntoma | Acción |
|---|---|
| Campo estático nunca aparece | Marcar pendiente; completar permisos, filtros avanzados y opciones vinculadas |
| Misma ruta, cuerpos diferentes | Normal: registrar por `action`, no forzar esquemas juntos |
| Necesitar parámetros con respuestas simuladas | Inspeccionar solicitudes salientes, no inferir respuestas simuladas |
| 400 nombra campo anidado | Comprobar envoltorios `data`/`bizData`/`variables` |
| Solo nombre de operación GraphQL visible | Ampliar JSON `variables`; buscar `$var: Type` estático |

---
