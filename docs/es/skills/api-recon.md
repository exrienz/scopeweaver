---
name: api-recon
description: Usar para inventariar las interfaces API de un sitio dentro del alcance autorizado.
---

# Reconocimiento de API (descubrimiento de interfaces frontend)

[English](../../../skills/api-recon/SKILL.md) · [한국어](../../ko/skills/api-recon.md) · [简体中文](../../zh-Hans/skills/api-recon.md) · [繁體中文](../../zh-Hant/skills/api-recon.md) · [日本語](../../ja/skills/api-recon.md) · Español

Con autorización, descubre lo más completamente posible API backend (rutas, métodos, parámetros, respuestas), rutas frontend y desencadenantes de interfaz (pestañas, diálogos, acciones de tabla).

---

## Límites y prohibiciones (lectura obligatoria)

Esta habilidad solo inventaría API y parámetros. No descubre ni explota vulnerabilidades.

### Límites de tarea

| Alcance | Permitido | Prohibido |
|---|---|---|
| Objetivo | Enumerar rutas, métodos, parámetros y desencadenantes de interfaz | SQLi/XSS, ataques de control de acceso, fuerza bruta, fuzzing de vulnerabilidades, solicitudes manipuladas, operaciones destructivas |
| Autenticación | Interceptar y simular controles de acceso del cliente | Pedir/adivinar credenciales; enviar formularios reales de acceso |
| Ejecución | Interceptar sin credenciales; renderizar estructura SPA posterior al acceso mediante simulaciones | Flujos que requieran sesión backend real |

### Análisis dinámico sin credenciales (predeterminado en fase 3)

1. Intercepta y simula API de acceso, permisos, menús e inicio con `preload.js` / `runtime_harvest.js`.
2. Devuelve cuerpos simulados con estructura y código de éxito correctos; los datos pueden estar vacíos.
3. Renderiza páginas posteriores al acceso sin backend o pese a 401 para provocar más solicitudes XHR/fetch/WebSocket.
4. Se esperan datos, tablas vacías e interfaz provisional. No cambies a acceso real ni pruebas de vulnerabilidades.

Usa simulaciones solo para montar rutas/componentes y registrar solicitudes salientes. El objetivo es descubrir qué más envía el frontend, no obtener datos reales del backend.

### Prohibiciones del flujo

| Prohibido | Alternativa obligatoria |
|---|---|
| grep/curl/Read del `index-*.js` principal para buscar API antes de terminar fase 1 | Ejecutar `OUTDIR/harvest_static.py` |
| Sustitutos escritos a mano como `extract_apis.py` | Adaptar y repetir `OUTDIR/harvest_static.py` |
| Repetir grep/comando tras al menos dos fallos | Cambiar estrategia: leer tool_logs, adaptar harvest, consultar referencia |
| Omitir controles A/B y ejecutar plantillas originales de `scripts/` | Copiar a OUTDIR y adaptar al objetivo |
| Credenciales reales, OTP o OAuth | Usar simulaciones como arriba |
| Omitir simulaciones para obtener datos reales mediante ataques de acceso/inyección | Registrar solo salidas dentro del reconocimiento |
| Eliminar, exportar datos sensibles, escrituras masivas u otras acciones irreversibles | También se aplica a clics de cobertura |
| Afirmar todas las páginas/API sin enumeración dinámica y de ejecución | Cumplir definición de finalización o indicar límites |
| Afirmar todos los parámetros sin matriz de activación ni diferencias | Matriz fase 3b + diferencias fase 5 |
| Inferir obligatorio/opcional de una muestra | Comparar muestras o inspeccionar validaciones/errores |

---

## Dos capas y modos de ejecución

| Capa | Salida | Límites |
|---|---|---|
| Estática (paquete JS) | Rutas de endpoint, borradores de rutas y campos candidatos donde se montan cargas | Sin métodos HTTP; parámetros requieren fase 1b; pierde URL construidas dinámicamente |
| Dinámica (sesión activa) | Métodos, cuerpos, respuestas, URL dinámicas, WS/SSE; diferencias de muestras completan parámetros | Las páginas deben renderizar para emitir solicitudes; una muestra no establece obligatorio/opcional |

| Modo | Motor | Uso |
|---|---|---|
| **depth** | `runtime_harvest.js` (Puppeteer) | Inventario API, métodos/parámetros/respuestas, WS/SSE, lotes repetibles |
| **coverage** | navegador + `preload.js` | Clics en pestañas/diálogos/tablas para cobertura más profunda |
| **both** | depth y después coverage | Más completo; tarda más |

Método de parámetros (sin script universal): harvest/regex para rutas; ampliar ventanas de anclaje, cadenas de vínculos UI, diferencias de muestras e inferencia de errores para parámetros. Consulta recetas grep en sección J de [reference.md](api-recon-reference.md).

---

## Definición de finalización

Afirma finalización solo al cumplir todas las condiciones:

- [ ] Estático: fase 1 produce `api_static.txt`, `routes.txt`, `js/`
- [ ] Dinámico: al menos depth o coverage; coverage/both requiere interceptores y bucle de enumeración dinámica operativos
- [ ] Estructura: las rutas de negocio permanecen fuera de `/login` (comprueba enrutamiento hash)
- [ ] Parámetros: coverage/both completa matriz y `param_samples.json`; fase 5 combina `params_merged.json`
- [ ] Profundidad con módulos vacíos: restaurar árbol de permisos en fase 4 y repetir hasta que aparezcan API de módulos, más allá de idioma/inicio
- [ ] Entrega: todas las salidas fase 5; `insert_assets` guarda activos de servicio y endpoint

---

## Scripts y controles

`scripts/` contiene plantillas de referencia. Nunca ejecutes un original sin adaptar y lo trates como final.

Lee, adapta al objetivo, escribe en `OUTDIR` (p. ej., `recon/`) y registra `CHANGES.md`. Si no sirve, reescribe con el método y toma solo su estructura.

| Control | Cuándo | Plantilla → copia OUTDIR | Cambios habituales |
|---|---|---|---|
| A (estático) | Tras fase 0, antes del primer harvest/spider | `harvest_static.py` / `spider_mpa.py` | La regex predeterminada sirve en la mayoría; cambiar regex de endpoints, `publicPath` webpack/Vite o exclusiones/cookie MPA solo por incompatibilidad de manifiesto/sintaxis |
| B (dinámico) | Tras fase 2, antes de depth/coverage | `runtime_harvest.js` / `preload.js` + `config.json` | Claves cookie/localStorage, valores de éxito neutralizados, simulaciones, regex de acceso, prefijos API, hash/history |

Orden SPA obligatorio: los números de fase prevalecen sobre navegar exploratoriamente antes de programar. No reordenes:

| Paso | Obligatorio | Prohibido |
|---|---|---|
| Tras fase 0 | Siguiente Bash = `python3 OUTDIR/harvest_static.py <URL> OUTDIR` | curl/grep/Read del `index-*.js` principal (normalmente >500KB) |
| Control A | Copiar script, cambios mínimos necesarios, ejecutar inmediatamente | Extraer API manualmente antes de decidir recolectar |
| Antes de terminar fase 1 | Comprobar salidas con `wc -l`; adaptar y reintentar ante 404 | Escribir extractores; repetir grep de URL no descargadas |
| Desde fase 1b | grep solo en `OUTDIR/js/*.js` | Usar paquete principal en vez de harvest |

- Correcto: copiar `harvest_static.py`, ajustar regex opcionalmente, ejecutar inmediatamente
- Incorrecto: curl al paquete principal, grep repetido, extractor temporal, harvest al final
- MPA: tras fase 0, siguiente Bash = `python3 OUTDIR/spider_mpa.py ...`

---

## Límites de herramientas y salida

| Restricción | Regla |
|---|---|
| Archivos grandes | Nunca introducir en contexto con Read/grep `index-*.js` >100KB; procesar por lotes con scripts OUTDIR |
| Salida grep | Exigir `\| head -20` o `-m 5`; conservar resúmenes de rutas, no fragmentos de paquetes |
| Validación | Usar `wc -l`, `ls \| wc -l`; no leer directorios completos |
| Vista previa regex | Opcional, máximo una vez, solo fragmentos/HTML ≤50KB; harvest es la fuente estática de verdad |
| reference | Recetas/plantillas/diagnóstico en [reference.md](api-recon-reference.md); no repetirlo aquí |

---

## Hoja de ruta de ejecución

```
Phase 0 classification + OUTDIR
  → Gate A → Phase 1 harvest (run immediately)
  → Phase 1b parameter reconstruction
  → Phase 2 three authentication gates → config.json
  → Gate B → Phase 3 runtime + parameter matrix
  → Phase 4 permission tree if needed → rerun Phase 3
  → Phase 5 merge report + insert_assets for every discovered service/endpoint asset; omit none
```

Comprueba en orden; no avances hasta completar el paso anterior.

1. [ ] **Fase 0**: clasificar SPA/MPA; crear `OUTDIR` → [Fase 0](#fase-0-clasificación)
2. [ ] **Control A + fase 1**: copiar scripts, recolectar inmediatamente, verificar con `wc -l` → [Fase 1](#fase-1-descubrimiento-estático)
3. [ ] **Fase 1b**: ventanas de anclaje + capa de vínculos → `param_candidates.json` → [Fase 1b](#fase-1b-reconstrucción-de-parámetros)
4. [ ] **Fase 2**: tres controles de autenticación → `config.json` → [Fase 2](#fase-2-tres-controles-de-autenticación)
5. [ ] **Control B**: adaptar scripts dinámicos → [Fase 3](#fase-3-ejecución-dinámica)
6. [ ] **Fase 3**: depth / coverage / both; verificar entrada a estructura; matriz de parámetros → `param_samples.json`
7. [ ] **Fase 4**, si hace falta: árbol de permisos → parchear simulaciones → repetir fase 3 → [Fase 4](#fase-4-reconstrucción-del-árbol-de-permisos)
8. [ ] **Fase 5**: combinar salidas + informe + `insert_assets` → [Fase 5](#fase-5-combinar-e-informar)

---

## Fase 0: clasificación

Obtén el HTML de entrada y crea `OUTDIR`; no modifiques `scripts/` de la habilidad:

- **SPA**: estructura vacía + `<div id=app>` + fragmentos → fases 1–5
- **MPA**: SSR + `<form>`, sin paquete de endpoints; tras control A:

```bash
python3 recon/spider_mpa.py <BASE_URL> <OUTDIR> [--cookie "session=..."] [--max 300] [--depth 5] [--exclude "logout|delete|destroy"]
```

Salidas: `forms.txt`, `links.txt`, `api_inline.txt`. Para SPA con casi cero formularios, pasa a fase 1.

---

## Fase 1: descubrimiento estático

Sigue [scripts y controles](#scripts-y-controles) y [límites de herramientas/salida](#límites-de-herramientas-y-salida).

```bash
python3 recon/harvest_static.py <BASE_URL> <OUTDIR>
```

Harvest interpreta scripts HTML y manifiestos webpack/Vite, descarga todos los fragmentos diferidos y produce `js/`, `api_static.txt`, `routes.txt`, `chunkmap.txt`.

```bash
wc -l OUTDIR/api_static.txt OUTDIR/routes.txt
ls OUTDIR/js | wc -l
```

- Compara recuento con manifiesto; ante 404 corrige harvest y reintenta, no curl uno a uno
- Pocas entradas en `api_static.txt`: amplía regex de endpoints de OUTDIR y repite (consulta referencia)

### Fase 1b: reconstrucción de parámetros

Las rutas vienen de fase 1; investiga campos por separado. Sigue [límites de grep](#límites-de-herramientas-y-salida).

Finalización: en API importantes, identifica nombre del campo, ubicación de transporte, tipo inferido, obligatoriedad, ejemplo y confianza.

#### 1b.0: formas de transporte

| Forma | Ubicación de parámetros | Prueba estática |
|---|---|---|
| REST JSON | cuerpo + consulta | Cerca del anclaje de ruta `(params\|data\|body)\s*:\s*\{` |
| GraphQL | `variables` | plantillas gql, `$page: Int` |
| Formulario tradicional | urlencoded | `<form>`, `FormData` |
| Subida de archivos | multipart | `FormData.append` |
| Parámetros de ruta | `/user/:id` | Tabla de rutas + `useParams` / `$route.params` |
| Cifrado/firma | Envuelto en `sign`/`data` | Interceptar entradas de la función de cifrado (referencia D) |

Salida: anotar cada API con `transport: query|json|form|graphql|encrypted`.

#### 1b.1: ampliar ventanas de anclaje

Amplía alrededor de rutas conocidas para encontrar el montaje de cargas:

```bash
grep -n '"/api/user/list"' OUTDIR/js/*.js | head -20
grep -rhoaE '.{0,120}("/api[^"]+").{0,200}' OUTDIR/js/*.js | head -20
grep -rhoaE '(params|data|body|payload)\s*:\s*\{' OUTDIR/js/*.js | head -20
```

| Envoltorio | Indicios de parámetros |
|---|---|
| Instancia axios | `data` / `params` |
| Envoltorio de solicitud compartido | Campos globales inyectados por interceptor |
| Cliente OpenAPI | Firmas de métodos generadas |
| React Query / SWR | Segundo argumento del hook |
| Composable Vue | Argumentos del composable |

Indicios de tipo: `yup`/`zod`/rules, `Form.Item name=`, Swagger integrado.

→ `param_candidates.json`: `{ path, fields[], source: "static-callsite", confidence }`

#### 1b.2: capa de vínculos

```
Form field → onFinish/handleSubmit → transform → API payload
```

| Origen del vínculo | Método |
|---|---|
| Envío de formulario | Seguir envío → transformación → API |
| Búsqueda de tabla | `getFieldsValue()` → `params` |
| Rutas | `:id` / `?tab=` |
| Interceptor | `tenantId` global, paginación, firma |
| Selección de enumeración | `options` → valores de enumeración API |

Sigue la pila DevTools desde `fetch`/`XHR.send` hacia el constructor de carga.

#### 1b.3: tres preguntas de carga (distintas de controles de autenticación fase 2)

| Pregunta | Respuesta requerida |
|---|---|
| Montaje | Dónde se construye y transforma la carga |
| Validación | required, pattern, enum |
| Transporte | path / query / body / multipart / headers |

En el control del interceptor (fase 2), inspecciona también campos globales inyectados (Authorization, `X-Tenant-Id`, sign).

#### 1b.4: conectar con fase 3

Los candidatos vienen del análisis estático/de vínculos. Obligatoriedad, opcionalidad y dependencias condicionales necesitan matrices/diferencias de fase 3 e inferencia de errores de fase 5.

---

## Fase 2: tres controles de autenticación

Usa grep de `OUTDIR/js/` con `head`; registra hallazgos en `config.json` (recetas de referencia):

| Control | Pregunta | Palabras clave |
|---|---|---|
| Renderizado | ¿Cómo se determina el estado de acceso? | `isLogin`, `getToken`, Cookie/localStorage |
| Interceptor | ¿Qué provoca redirecciones a `/login`? | `response_code`, `errno`, interceptor axios |
| Contenido | ¿De dónde vienen menús/permisos? | `menu`, `permission`, `role`, `acl`, `routes` |

No trates nombres de claves localStorage como credenciales; verifica mediante fragmentos/cadenas de solicitudes.

Salida = control B: escribe conclusiones en `config.json` y adapta `OUTDIR/runtime_harvest.js` / `preload.js`.

### Fase 2b: observación de API (opcional)

Usa `preload.js` de OUTDIR para identificar claves de sesión, Authorization y URL API anidadas:

| Configuración | Salida |
|---|---|
| `recordDetail: true` | `__API_RECON_DETAIL__` |
| `observe.xhrHeaders: true` | Observaciones de cabeceras |
| `extractUrlsFromResponse: true` | API anidadas en respuestas |
| `observe.storageReads/cookieReads: true` | Actualizar configuración |
| `neutralizeVueRouter: true` | `__API_RECON_ROUTES__` |

Exporta en cada ronda de cobertura: `__API_RECON_LOG__`, `__API_RECON_DETAIL__`, `__API_RECON_ROUTES__`, `__API_RECON_OBSERVE__`.

---

## Fase 3: ejecución dinámica

Supera antes el control B. Sigue los [límites](#límites-y-prohibiciones-lectura-obligatoria) y la estrategia de simulación sin credenciales.

Establece `"runtimeMode": "depth" | "coverage" | "both"` en `config.json` (plantilla en referencia).

### Interceptores y simulaciones (compartidos por depth y coverage)

| Capa | Alcance | Finalidad |
|---|---|---|
| L1 exacta | Simulación de auth/permisos/inicio | Superar autenticación de renderizado inicial |
| L2 corrección de resultados negativos | Todas las respuestas JSON | Código sin sesión → éxito |
| L3 alternativa | `/api`, etc., sin coincidencia L1 | Cuerpo vacío correcto para renderizar |

- **depth**: autenticación falsa + parche de código de negocio `forward` + `stubs`; recorrer `routes` (hash/history); producir `runtime_api.json`
- **coverage**: inyectar `preload.js` al inicio del documento (CDP `addScriptToEvaluateOnNewDocument` o userscript)

Verifica que exista `window.__API_RECON_PRELOAD__` y que las rutas de negocio no vuelvan a `/login`.

```bash
cd recon && npm install
node runtime_harvest.js config.json
```

### 3b: enumeración dinámica de cobertura (obligatoria)

1. Navegación principal/lateral: pulsar cada elemento, esperar 1–3 segundos de red
2. Pestañas: `role=tab`, `.ant-tabs-tab`
3. Tablas: ver/editar/detalles de primera fila
4. Barra de herramientas: exportar, filtrar, crear (evitar eliminación irreversible; siguen todas las prohibiciones)
5. Cada módulo: combinar API/rutas
6. SPA: `pushState` controlado para rutas sin cubrir de `routes.txt` (prohibido para MPA)

Matriz de activación obligatoria: registra cada tipo de operación por módulo y compara varias muestras:

| Operación | Parámetros adicionales habituales |
|---|---|
| Lista inicial | Paginación + filtros predeterminados |
| Búsqueda | keyword, filter |
| Filtros avanzados | Más campos opcionales |
| Crear/editar | Entidad completa |
| Masivo/exportar/ordenar | `ids[]`, `exportType`, `sortField` |

Los cuerpos/cabeceras salientes siguen siendo reales con simulaciones; usa solicitudes como pruebas. Registra `scan_raw.json`, `param_samples.json`, `api_detail.json`.

- **Vue**: `neutralizeVueRouter: true` + precarga al inicio del documento
- **React**: `routes.txt` + clics laterales + `pushState`
- **both**: primero 3a depth, después 3b coverage

---

## Fase 4: reconstrucción del árbol de permisos

Desencadenante: módulos vacíos o solo inicio (p. ej., idioma) en todas las rutas indica que no se superó el control de contenido.

| Síntoma | Significado |
|---|---|
| Estructura abierta | Controles de renderizado e interceptor superados |
| Faltan elementos laterales/clics vacíos | Estructura de simulación o códigos de permiso incompletos |
| Pocas API idénticas en cada ruta | Falló `v-if permission` |
| `routes.txt` mucho menor que rutas del paquete | Completar desde módulos de autenticación |

```bash
grep -rhoaE '"/api[^"]*(permission|perm|role|menu|acl)[^"]*"' OUTDIR/js/*.js | sort -u | head -30
grep -rhoaE 'userRouteAuth|getResultTree|routeMap|routeLink|menuList|authList' OUTDIR/js/*.js | head -20
```

Cadena típica: `role_permissions` (códigos planos) + `permissions/all` (árbol) → `getResultTree` → `userRouteAuth[CODE].url`.

```bash
python3 recon/extract_route_map.py recon/js recon/
python3 recon/build_perm_tree.py recon/js recon/ --config recon/config.json
```

Salidas intermedias: `route_map.json`, `userRouteAuth.json`, `permissions_tree.json`, `*_stub.json`, `perm_codes_all.txt`.

Comprueba simulaciones: `response_code` exterior coincide con el interceptor; códigos planos coinciden con árbol; `routes` cubre cada enlace `route_map`.

Tras actualizar `config.json`, repite fase 3. Las SPA grandes pueden ajustar `waitUntil`, `routeTimeout`, `perRouteMs` (referencia A3/I).

---

## Fase 5: combinar e informar

### Salidas

| Archivo | Fase | Contenido |
|---|---|---|
| `js/`, `api_static.txt`, `routes.txt`, `chunkmap.txt` | 1 | Paquetes y rutas estáticos |
| `param_candidates.json` | 1b | Parámetros candidatos estáticos |
| `config.json` | 2 | Tres controles + configuración dinámica |
| `runtime_api.json` | 3a | Captura depth detallada, incluidos WS/SSE |
| `param_samples.json`, `scan_raw.json`, `api_detail.json` | 3b | Varias muestras, registros de clics y detalles |
| `route_map.json` y relacionados | 4 | Intermedios del árbol de permisos, si se usa |
| `params_merged.json` | 5 | Campos combinados + confianza |
| `api_merged.txt` | 5 | `METHOD /path [params] [static\|runtime\|both]` |
| `site_map.json` | 5 | Rutas, API, parámetros, funciones, límites |
| **insert_assets** | 5 | Guardar todos los activos de servicio y endpoint |

### 5b: combinar parámetros

Compara `param_samples.json`; no hay script universal. Las reglas de confianza están en referencia J7 (alta/media/baja/pendiente de activación).

### 5c: inferir de errores

Con autorización, solicitudes incompletas pueden inspeccionar errores 400 para descubrir parámetros, no probar vulnerabilidades: `field 'x' is required`, errores de enumeración, etc. Considera envoltorios `data`, `variables` y `bizData` previo al cifrado.

Informa runtimeMode, recuentos API estáticos/dinámicos, confianza de parámetros, módulos sin cubrir y resumen `CHANGES.md` respecto a plantillas.

`site_map.json` sugerido:

```json
{
  "site": "https://example.com",
  "runtimeMode": "both",
  "appType": "vue-spa",
  "routeGuardStrategy": ["nav-neutralize", "L1-auth", "L2-patch", "forward"],
  "apisFromStatic": [],
  "apisFromRuntime": [],
  "apis": [],
  "params": [{ "method": "POST", "path": "/api/user/list", "transport": "json", "fields": [] }],
  "frontendRoutes": [],
  "routesVerifiedByClick": [],
  "featuresTriggered": [],
  "limitations": ""
}
```

Más campos y recetas grep: [reference.md](api-recon-reference.md).

---

## Notas generales

- Independiente del framework: mismo método para carga diferida webpack/Vite/Angular
- Transporte: REST/JSON, GraphQL, WebSocket, SSE; gRPC-web excluido
- SSR: las consultas del cliente se capturan; RSC/Server Actions no se enumeran completamente
- Puntos ciegos: JSVMP, WASM, HMAC/mTLS estricto → análisis estático y límites explícitos
- Parámetros no visibles: interacciones condicionales, parámetros ocultos, montaje WASM → pendiente de activar/inaccesible
- Alternativa estática: se pueden descubrir endpoints con ejecución bloqueada

---

## Recursos adicionales

- Recetas grep, plantillas `config.json`, diagnóstico, interceptores, reconstrucción de parámetros (J), plantilla site_map: **[reference.md](api-recon-reference.md)**
- Rutas de plantillas: [scripts y controles](#scripts-y-controles)
