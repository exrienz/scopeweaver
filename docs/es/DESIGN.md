# Presentación ScopeWeaver — DESIGN.md

[English](../../DESIGN.md) · [한국어](../ko/DESIGN.md) · [简体中文](../zh-Hans/DESIGN.md) · [繁體中文](../zh-Hant/DESIGN.md) · [日本語](../ja/DESIGN.md) · Español

<!-- impeccable:design-schema 1 -->

Página estática GitHub Pages de ScopeWeaver. Inglés predeterminado
(`landing/index.html`); coreano, chino simplificado, tradicional, japonés y español son equivalentes
(`landing/{ko,zh-Hans,zh-Hant,ja,es}/index.html`). HTML/CSS/JS
simples, sin framework, CDN, seguimiento ni backend. Servida bajo la
base `/scopeweaver/` (y subdirectorios de idioma) con rutas relativas de recursos.

## Modo

Persuadir. La superficie es el escaparate: un desarrollador o investigador de seguridad
debe entender qué es ScopeWeaver, verlo funcionando, comprender cómo fluye una
tarea y cómo la dirige un agente de programación, confiar en los límites y acceder a la
versión o comandos de configuración.

## Mundo visual

La consola de la aplicación ampliada a escala de página.

- **Fondo** `#fafafa`, **superficie** `#ffffff`, hundida `#f4f4f3`.
- **Tinta** `#171717` (también logo y franja de agentes de ancho completo), con
  `#3f3f3f` / `#595959` para texto secundario/terciario. En franja oscura:
  texto `#fafafa`, secundario `#adb0b8` (neutral matizado, ≥ 6:1 sobre `#171717`).
- **Un azul moderado**, reservado a enlaces, acentos y foco: `#2563eb`,
  hover `#1b45c9` en claro; `#7cafff` / `#a8c8ff` en oscuro. Los colores de enlace
  usan `--link` / `--link-hover`, que la franja oscura reasigna para mantener
  legibilidad al pasar el puntero.
- **Líneas finas**: reglas de 1px `#e6e6e4` / `#d4d4d2` separan secciones; sin
  divisores pesados.
- **Motivo de cuadrícula de alcance**: cuadrícula tenue de dos ejes que refleja el logo 3×3
  detrás de la cabecera (máscara radial) y la franja de agentes. Es el motivo de marca
  acordado; el detector señala cuadrículas decorativas por defecto, y esta dirección
  visual prevalece aquí.
- **Profundidad**: sombras moderadas con desplazamiento real y desenfoque suave; sin
  halos de color centrados. Capturas en un marco ligero de ventana.

## Tipografía

- **Schibsted Grotesk** (woff2 variable autoalojada, pesos 400–900, OFL) para
  toda la voz: títulos 640–660, espaciado −0.028em y ajuste equilibrado. El tamaño
  de portada se limita a ≈3.7rem.
- **JetBrains Mono** (woff2 variable autoalojada, OFL) para identificadores,
  comandos, procedencia e indicadores, con funciones `tnum`/`zero`.
- Solo se autoalojan fuentes latinas; la edición coreana añade una pila Hangul del sistema
  (`Apple SD Gothic Neo`, `Pretendard`, `Malgun Gothic`, `Noto Sans KR`)
  tras la fuente latina para mantener el mismo aspecto de ASCII.
  El coreano usa `word-break: keep-all` + `overflow-wrap: break-word`;
  identificadores `<code>` en texto usan `overflow-wrap: anywhere` para que nombres largos
  se ajusten sin ensanchar cuadrículas (las apiladas usan `minmax(0, 1fr)`).
- Longitud de línea ~33–68ch; texto funcional ≥ 11px.

## Estructura

Una columna separada por líneas finas, con densidad variable deliberada:

1. **Cabecera**: reparto 5/7, propuesta y acciones a la izquierda; una captura real
   grande del panel enmarcada a la derecha.
2. **Recorrido**: reparto 7/5. Lista ARIA vertical (cinco pantallas reales:
   Panel, Tareas, Hallazgos, Configuración del sistema, Plantillas LLM) cambia una
   captura fija y expande una explicación por pantalla. El mecanismo
   (dos grafos vinculados, planificador → trabajadores de una intención → hechos/hallazgos, revisión
   humana y comprobación) se explica en texto, sin diagrama generado.
3. **Puente de agentes**: franja de tinta completa. Lista ARIA horizontal cambia
   comandos de instalación Claude Code / Codex / Pi, con botón de copia; las ocho
   herramientas y la descripción honesta «interfaz de control, no sustituto»
   acompañan una captura móvil enmarcada.
4. **Instalar**: tres rutas concretas (script, Docker Compose, archivo de versión)
   con comandos copiables y los cinco indicadores de plataforma + GHCR.
5. **Requisitos**: cuatro celdas de líneas finas: PostgreSQL, proveedor de modelos configurado
   aparte, Node.js 22+ para adaptador y objetivo local de laboratorio.
6. **Límites y créditos**: lista simple (solo aislamiento local; la discrepancia histórica heredada del resumen de alcance se corrigió en `75a706d`
   (consulta [verificación](VERIFICATION.md), sin afirmar éxito actual de toda la suite); sin afirmaciones de producción/eficacia; Docker/Windows no verificados;
   hallazgos npm audit; adaptador con casos locales y ejecución real de agente a proveedor
   no verificada independientemente, con enlace a VERIFICATION) junto a la
   nota anonimizada del flujo del propietario y procedencia ARTEX/AGPL-3.0,
   fuentes y capturas
   correspondientes.
7. **Pie**: marca, recursos, licencia y nota «introducción, no servicio
   activo».

## Interacción distintiva

La lista del recorrido. Seleccionar pantalla funde la captura (opacidad)
y revela su explicación mediante expansión animada de `grid-template-rows`:
un momento diseñado, no una entrada por sección. Tabindex móvil con
Up/Down/Home/End; la lista horizontal usa Left/Right.

## Accesibilidad y estados

- Enlace de salto, regiones, navegación etiquetada y anillo de foco visible (azul en
  claro, azul claro en oscuro).
- Pestañas ARIA con `aria-selected` / `tabindex` / `hidden` correctos y teclado;
  botones anuncian estado localizado (`Copy`/`Copied` ·
  `복사`/`복사됨`) y permiten selección manual sin portapapeles.
- Menú móvil con `aria-expanded`. Los elementos de cabecera no se encogen; un
  paso ≤420px reduce espacios hasta 320px. El enlace GitHub solo con icono conserva
  «GitHub» visualmente oculto como nombre accesible.
- Superficies del navegador tematizadas: selección, desplazamiento, foco, distancia
  del subrayado y números monoespaciados tabulares.
- `prefers-reduced-motion` anula transiciones y desplazamiento suave.
- Imágenes con ancho/alto y `aspect-ratio` evitan saltos; la principal
  usa `fetchpriority="high"`, las demás carga diferida bajo el primer pliegue.

## Honestidad y procedencia

Cada captura se etiqueta como datos ficticios. El párrafo del propietario
solo demuestra orquestación anonimizada, sin objetivos, hallazgos, mensajes,
uso de modelos ni credenciales. Las afirmaciones se rastrean a README, adaptador,
PROVENANCE, VERIFICATION y cambios v0.1.0. ScopeWeaver deriva de
ARTEX bajo AGPL-3.0; uso restringido a investigación aislada local.

## Verificación realizada (al compilar, sin navegador)

- `tools/validate-landing.mjs` (fuera de `landing/`): existencia de recursos, seguridad de
  rutas relativas, paridad EN/KO de ID/pestañas/controles, enlaces de idioma, pies de demo,
  sin metadatos de planificación publicados, `.nojekyll` presente: 17/17.
- `node --check` en JavaScript y script de validación.
- Detector Impeccable: 19 → 4 residuales (hover de acento único emparejado estáticamente
  contra franja oscura; franja completa acordada con margen de su envoltorio interior):
  ambos renderizan bien. El motivo de cuadrícula es la marca acordada.
- El coordinador inspeccionó inglés/coreano en Chrome a 1440px y 390px,
  aplicó un lote de correcciones y confirmó que ambos también caben a 320px
  sin desbordamiento horizontal. Pasaron pestañas de capturas/agentes, teclado,
  menú móvil, enlaces e indicaciones de copia localizadas. No se observaron errores
  de consola. Las capturas diferidas inferiores cargaron al navegar.
- Pages solo despliega `landing/`; el flujo ejecuta integridad y sintaxis
  JavaScript antes de subir. Los documentos de desarrollo y datos privados
  permanecen fuera del artefacto público.

## Atribución de implementación

Se solicitó Claude Code con `claude-opus-5-5` a esfuerzo medio. La sesión principal
recurrió automáticamente a `claude-opus-4-8` tras una interrupción de protección
del proveedor. La sesión limitada de correcciones de navegador se ejecutó
íntegramente con `claude-opus-5-5` a esfuerzo medio. Por ello, la página no se atribuye
exclusivamente a Opus 5.5.
