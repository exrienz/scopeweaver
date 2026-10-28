# Procedencia y modificaciones de ScopeWeaver

[English](../PROVENANCE.md) · [한국어](../ko/PROVENANCE.md) · [简体中文](../zh-Hans/PROVENANCE.md) · [繁體中文](../zh-Hant/PROVENANCE.md) · [日本語](../ja/PROVENANCE.md) · Español

ScopeWeaver es una obra derivada independiente de [Autumn-27/ARTEX](https://github.com/Autumn-27/ARTEX), basada en el commit `160fe13`. Se mantiene por separado en [cskwork/scopeweaver](https://github.com/cskwork/scopeweaver), sin relación en la red de bifurcaciones de GitHub.

Las modificaciones de ScopeWeaver del **2026-10-02** abarcan el nombre del producto, interfaz y mensajes con inglés predeterminado/coreano seleccionable, documentación e indicaciones de agentes incluidas, mensajes de configuración/compilación/inicio/actualización/restablecimiento de contraseña, gestión de idioma de informes y agentes y nombres de distribución. El ejecutable es `scopeweaver` (`scopeweaver.exe` en Windows); los archivos siguen `scopeweaver-<version>-<os>-<arch>.zip`. Docker Compose compila localmente el código extraído. El flujo de versiones configurado apunta a este repositorio y GHCR.

Se conserva `LICENSE` original sin cambios. La autoría original, agradecimientos a colaboradores e historial de cambios siguen atribuidos a ARTEX. Se conservan por compatibilidad el módulo Go `github.com/Autumn-27/artex`, entrada fuente `./cmd/artex`, contratos de configuración `ARTEX_*`, valores de BD `artex` y usuario de acceso `ARTEX`.

Esta fecha de modificación no es una etiqueta de versión. No afirma que se hayan publicado binarios ni imágenes de contenedor. Los documentos históricos de validación de `sidequestion/` describen ejecuciones originales; no son comprobantes nuevos de ScopeWeaver. El JSON histórico traducido conserva identificadores y mediciones técnicas y etiqueta la narrativa traducida.

Las modificaciones del **2026-10-15** añaden formato opcional de perfil Ollama nativo (`/api/chat` con `num_ctx`, `keep_alive` y `num_predict`), tres plantillas Ollama del formulario nuevo, perfiles locales sin clave, concurrencia por perfil, valores predeterminados de precarga, compresión y juez de aprobación adaptados a Ollama y política opcional de red LLM `local_llm_only`. Se añaden variables `SCOPEWEAVER_LLM_*` y `SCOPEWEAVER_LOCAL_LLM_*` como alias con prioridad sobre los contratos `ARTEX_*` conservados. Se conserva el comportamiento de Anthropic, OpenAI, OpenAI Responses, URL personalizadas, proxies y plantillas GLM; la política está desactivada por defecto.

El candidato posterior de refuerzo nativo del **2026-10-15** añade admisión urgente al siguiente hueco sin interrupción, selección del juez según origen nativo real con salida restringida por esquema y valores de 90 segundos/preguntar, y compresión al 72% con límites de salida de herramientas de 16/32 KiB hasta 64K. Restringe los destinos estrictos opcionales a loopback y CIDR de inferencia explícitos (incluida migración LAN/Docker), rechaza redirecciones/proxies y bloquea búsqueda de agentes y acciones de red de actualización conservando estado/reversión locales. Acompañan estimaciones de memoria GPU de solo lectura, avisos de modelos compartidos, carga segura de ajustes del juez y texto inglés/coreano/chino simplificado. Son controles de aplicación, no garantías de desconexión total del host ni calidad de modelo; no introduce dependencias, migraciones, descarga automática de modelos ni versiones. Consulta la [guía de proveedores](llm-providers.md) y [verificación del candidato](VERIFICATION.md#candidato-de-refuerzo-nativo--2026-10-15).

Modificaciones adicionales del **2026-10-03** completan la auditoría de traducción del servidor y añaden plantillas GLM-5.3 con compatibilidad de razonamiento siempre activado obligatoria. Las plantillas usan el adaptador existente y no crean ni activan perfiles automáticamente.

Las correcciones posteriores originales de ARTEX de los commits `e2c8a23`, `86729b6` y
`a951e4a` se adaptan aquí: SSE de producción del mismo origen, contabilidad independiente
de tokens del modelo de aprobación y autenticación que bloquea ante fallos con creación atómica de contraseña
inicial y agrupación de conexiones de BD limitada. Los mensajes y guías de despliegue nuevos están
disponibles en inglés y coreano. La autoría original se atribuye en los commits de adaptación;
los demás cambios de ScopeWeaver permanecen.

<a id="norma-execution-and-compaction-fixes--2026-10-25"></a>
## Correcciones de ejecución y compresión de Norma — 2026-10-25

El reemplazo mantenido por el repositorio en `third_party/norma` parte de
[`52f17dd52d95b087265e5d5b306fe85c1ea05599`](https://github.com/Autumn-27/Norma/commit/52f17dd52d95b087265e5d5b306fe85c1ea05599).
Ese commit añade únicamente la licencia Apache-2.0 a v0.4.3
(`320ae32989da97dd91469bb89164cbfe87d83d5d`): 211 archivos / 1,293,448 bytes antes de los cambios locales. Esta instantánea no contiene un archivo NOTICE del proyecto original.

Las correcciones locales conservan el contexto de timeout original del ejecutor hasta la resolución final, reparan el cliente ausente en la prueba de reintento, ajustan el ritmo predeterminado de compresión para ventanas pequeñas y evitan que la consolidación útil de resúmenes quede postergada indefinidamente. También aprenden solo límites de contexto menores explícitos, impiden que aumenten los límites aprendidos y distinguen los errores de longitud de salida del desbordamiento de contexto. El historial sin procesar y los archivos históricos permanecen intactos; las vistas irreducibles siguen fallando explícitamente. La prueba obsoleta de limitación de ventanas pequeñas se convierte en una prueba positiva de consolidación.

`third_party/norma/README.scopeweaver.md` enumera los diez archivos originales modificados y las cuatro adiciones. Cada archivo original modificado lleva un aviso de modificación destacado. Los demás archivos originales, la licencia original y los metadatos del módulo no cambian. No cambian las firmas exportadas ni los formatos serializados; las correcciones de comportamiento enumeradas son intencionadas.

El reemplazo del módulo raíz y el orden de copia de dependencias de Docker seleccionan este código. La licencia Apache completa también se incluye en `THIRD_PARTY_NOTICES.txt` para el empaquetado existente de archivos/contenedores. El trabajo CI de backend ejecuta por separado la batería completa del SDK anidado; el recorrido de paquetes raíz no la incluye. No se incluye publicación en el repositorio original ni migración de base de datos.

La corrección de `enableNoa`, responsabilidad del host, transmite la ventana de contexto ya configurada, incluidas 250K y 1M, en lugar de sustituirla silenciosamente por la alternativa de 200K del SDK. Los límites de tokens de salida, Noa desactivado y las políticas de Ollama nativo son independientes y se conservan. Consulta la [guía de verificación](VERIFICATION.md) correspondiente.
