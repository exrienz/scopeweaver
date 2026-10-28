# Contribuir a ScopeWeaver

[English](CONTRIBUTING.md) · [한국어](CONTRIBUTING.ko.md) · [简体中文](CONTRIBUTING.zh-Hans.md) · [繁體中文](CONTRIBUTING.zh-Hant.md) · [日本語](CONTRIBUTING.ja.md) · Español

Las contribuciones son bienvenidas: correcciones, adaptaciones de ARTEX original, actualizaciones de dependencias, documentación
y pruebas.

## Antes de empezar

- Lee las [restricciones de uso](README.es.md#licencia-y-exención-de-responsabilidad). Las contribuciones no deben añadir
  funciones dirigidas a sistemas reales en línea ni debilitar esas restricciones.
- Para cambios grandes, abre primero una incidencia para acordar el enfoque.
- Mantén cada solicitud de cambio centrada en un tema. Las pequeñas se revisan más rápido.

## Requisitos de una solicitud de cambio

- **Idiomas publicados y documentación completos.** El texto visible y las guías mantenidas de usuarios/colaboradores necesitan inglés, coreano, chino simplificado (`zh-Hans`), chino tradicional (`zh-Hant`), japonés (`ja`) y español (`es`):
  - mensajes del servidor: conserva la compatibilidad de `locale.Register(key, en, ko)` e incluye los idiomas adicionales mediante `locale.RegisterAll` y los recursos por idioma de `locale/`
  - interfaz web: todas las claves inglesas actuales en tablas EN/KO y `messages/{zh-hans,zh-hant,ja,es}.ts`
  - documentación: `README.md`, `CONTRIBUTING.md` y guías mantenidas con ediciones correspondientes `ko`, `zh-Hans`, `zh-Hant`, `ja` y `es`; conserva las entradas ejecutables de habilidades en inglés
- **Pruebas del comportamiento cambiado.** Añade o actualiza pruebas que fallarían sin tu cambio. No
  desactives, omitas ni debilites comprobaciones existentes para que pasen.
- **Solo bases de datos desechables.** Las pruebas que escriban en PostgreSQL deben usar una base nueva llamada
  `scopeweaver_test_*` (o `scopeweaver_adapter_*` para E2E del adaptador de agentes).
- **Atribuye el trabajo original.** Para adaptaciones de ARTEX, añade `Upstream-commit: mhtsec/ARTEX@<sha>` y una
  línea `Co-authored-by:` del autor original, y actualiza `docs/PROVENANCE.md` y
  `docs/ko/PROVENANCE.md`.
- **Licencia.** Las contribuciones se aceptan bajo la licencia [AGPL-3.0](LICENSE) del repositorio.

## Añadir o cambiar un idioma

Sigue la [guía de idiomas](docs/es/i18n.md): actualiza `locale/definitions.json` compartido
(ID canónico, nombre nativo, asignación Intl y alias), vectores de coincidencia, catálogos completos de frontend/backend,
asignación date-fns, pruebas específicas y prueba básica de navegador. `zh`, `zh-CN` y `zh-SG` corresponden a
`zh-Hans`; `zh-TW`, `zh-HK` y `zh-MO` a `zh-Hant`. Una escritura explícita prevalece sobre una región contradictoria.
Mantén el inglés como origen de claves, significado, marcadores y etiquetas enriquecidas. La alternativa inglesa/de clave del backend
es una protección de compatibilidad, no prueba de que un idioma esté listo para publicar. No traduzcas
contenido de usuarios, borradores, pruebas sin procesar, identificadores ni datos históricos. Revisa traducciones y registra
dudas; no afirmes aprobación de un hablante nativo sin pruebas. Mantén completas las guías de funciones en los seis idiomas, incluido el par inglés/coreano. Mantén sincronizados también la lista de ID canónicos del adaptador independiente y los auxiliares de mensajes shell/batch de la raíz. Ejecuta `node e2e/script-locales.mjs` y el caso nativo de Windows `node e2e/windows-launcher.mjs`/CI de Windows; los adaptadores empaquetados no deben depender durante la ejecución del JSON de la raíz del repositorio.

## Comprobaciones que ejecutar

Coinciden con los trabajos CI de [.github/workflows/ci.yml](.github/workflows/ci.yml):

```bash
# patched Norma SDK (a separate Go module)
(cd third_party/norma && go test ./... -count=1)


# backend database-backed suites (db, server, agent): against a throwaway database
# that is created and dropped for you. They require ARTEX_TEST_DSN and skip (never
# fall back to the live DSN) without it — see docs/VERIFICATION.md.
make test-db

# backend: every Go package (set ARTEX_TEST_DSN to a disposable scopeweaver_test_* database)
ARTEX_TEST_DSN=postgres://sw_user_test@/scopeweaver_test_1?host=/tmp&sslmode=disable go test ./... -count=1

# frontend
cd web && npm ci && npm test && npm run build:static

# agent adapter (and its real-backend E2E test against a scopeweaver_adapter_* database)
cd adapters/agent && npm ci && npm test && npm run check
SCOPEWEAVER_ADAPTER_E2E=1 go test ./server -run '^TestAgentAdapterE2E$' -count=1
```

El flujo `e2e` también ejecuta una prueba básica esencial de navegador contra el binario real en cada PR y
tras integrar. Para ejecutarla localmente contra un servidor activo y una base nueva `scopeweaver_test_*`
(requiere `psql` y el binario con interfaz integrada compilado desde la misma copia de trabajo):

```bash
cd e2e && npm ci && npx playwright install chromium
SCOPEWEAVER_URL=http://127.0.0.1:8787 ARTEX_PG_DSN=postgres://.../scopeweaver_test_e2e \
  SCOPEWEAVER_BINARY=/absolute/path/to/scopeweaver npm run smoke
```

Tras inicializar la contraseña con las comprobaciones básicas existentes, la regresión del panel inicia su propia
copia aislada del binario en un puerto loopback libre con la misma BD de prueba. Comprueba
pilas de tokens Anterior/Nuevo en inglés/coreano y filtros de perfiles; detiene el servidor y elimina los datos
de prueba. Las pruebas del panel y el registro del servidor hijo se guardan en `e2e/e2e-results/dashboard-token-chart/`.

La prueba básica esencial cubre los seis idiomas: restauración de preferencias, borradores de configuración/acceso/tareas/ajustes/chat,
idioma de primera API, reconexión SSE/conservación de filtros y formato de números renderizado.

Describe en la PR qué ejecutaste y qué no pudiste ejecutar.

## Revisión e integración

Crea cada rama `feat/*` o `fix/*` desde el último `dev` y abre su PR hacia `dev`.
Tras integrar, espera todas las comprobaciones y E2E en ese commit exacto de `dev`; después abre una PR de promoción
de `dev` a `main` de este repositorio. Conserva el historial de promoción con un commit de fusión para que `dev`
siga siendo ancestro de `main`. Verifica otra vez las comprobaciones tras la promoción; empieza el trabajo posterior
desde `dev`. La comprobación `branch-policy` rechaza otras rutas de PR.

1. Todas las comprobaciones obligatorias deben pasar en el candidato actual: `branch-policy`, `frontend`,
   `agent-adapter`, `backend`, `windows`, `chat-browser` y `smoke`. Tanto `dev` como `main` ejecutan
   CI y E2E tras integrar. Para nuevos colaboradores, el mantenedor aprueba la ejecución CI.
2. La PR se revisa contra sus diferencias por corrección, seguridad, traducciones completas y paridad documental en seis idiomas.
3. **El mantenedor realiza una revisión humana final de cada PR antes de integrarla.** No se integra nada
   sin ella.

El mantenedor publica versiones tras una prueba completa de extremo a extremo antes y después de integrar.
