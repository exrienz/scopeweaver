# Trazas

[English](../../../../../skills/playwright-cli/references/tracing.md) · [한국어](../../../../ko/skills/playwright-cli/references/tracing.md) · [简体中文](../../../../zh-Hans/skills/playwright-cli/references/tracing.md) · [繁體中文](../../../../zh-Hant/skills/playwright-cli/references/tracing.md) · [日本語](../../../../ja/skills/playwright-cli/references/tracing.md) · Español

Captura trazas detalladas para depuración y análisis. Incluyen instantáneas DOM, capturas, red y registros de consola.

## Uso básico

```bash
# Start trace recording
playwright-cli tracing-start

# Perform actions
playwright-cli open https://example.com
playwright-cli click e1
playwright-cli fill e2 "test"

# Stop trace recording
playwright-cli tracing-stop
```

## Archivos de salida de trazas

Al iniciar trazas, Playwright crea `traces/` con varios archivos:

### `trace-{timestamp}.trace`

**Registro de acciones**: archivo principal con:
- Cada acción (clics, rellenado, navegación)
- Instantáneas DOM antes y después
- Capturas en cada paso
- Información temporal
- Mensajes de consola
- Ubicaciones de código

### `trace-{timestamp}.network`

**Registro de red**: actividad completa:
- Todas las solicitudes y respuestas HTTP
- Cabeceras y cuerpos de solicitudes
- Cabeceras y cuerpos de respuestas
- Tiempos (DNS, conexión, TLS, TTFB, descarga)
- Tamaños de recursos
- Solicitudes fallidas y errores

### `resources/`

**Directorio de recursos**: caché de:
- Imágenes, fuentes, estilos y scripts
- Cuerpos de respuesta para repetición
- Recursos para reconstruir estado

## Qué capturan las trazas

| Categoría | Detalles |
|----------|---------|
| **Acciones** | Clics, rellenado, puntero, teclado, navegación |
| **DOM** | Instantánea completa antes/después de cada acción |
| **Capturas** | Estado visual por paso |
| **Red** | Solicitudes, respuestas, cabeceras, cuerpos y tiempos |
| **Consola** | Todos los mensajes console.log, warn y error |
| **Tiempos** | Medición precisa por operación |

## Casos de uso

### Depurar acciones fallidas

```bash
playwright-cli tracing-start
playwright-cli open https://app.example.com

# This click fails - why?
playwright-cli click e5

playwright-cli tracing-stop
# Open trace to see DOM state when click was attempted
```

### Analizar rendimiento

```bash
playwright-cli tracing-start
playwright-cli open https://slow-site.com
playwright-cli tracing-stop

# View network waterfall to identify slow resources
```

### Capturar pruebas

```bash
# Record a complete user flow for documentation
playwright-cli tracing-start

playwright-cli open https://app.example.com/checkout
playwright-cli fill e1 "4111111111111111"
playwright-cli fill e2 "12/25"
playwright-cli fill e3 "123"
playwright-cli click e4

playwright-cli tracing-stop
# Trace shows exact sequence of events
```

## Traza frente a vídeo y captura

| Función | Traza | Vídeo | Captura |
|---------|-------|-------|------------|
| **Formato** | Archivo .trace | Vídeo .webm | Imagen .png/.jpeg |
| **Inspección DOM** | Sí | No | No |
| **Detalles de red** | Sí | No | No |
| **Repetición paso a paso** | Sí | Continua | Un fotograma |
| **Tamaño** | Mediano | Grande | Pequeño |
| **Mejor para** | Depurar | Demostraciones | Captura rápida |

## Buenas prácticas

### 1. Iniciar trazas antes del problema

```bash
# Trace the entire flow, not just the failing step
playwright-cli tracing-start
playwright-cli open https://example.com
# ... all steps leading to the issue ...
playwright-cli tracing-stop
```

### 2. Limpiar trazas antiguas

Las trazas pueden ocupar mucho disco:

```bash
# Remove traces older than 7 days
find .playwright-cli/traces -mtime +7 -delete
```

## Limitaciones

- Añaden sobrecarga a la automatización
- Las grandes pueden ocupar mucho disco
- Algún contenido dinámico puede no reproducirse perfectamente
