# Gestión de sesiones de navegador

[English](../../../../../skills/playwright-cli/references/session-management.md) · [한국어](../../../../ko/skills/playwright-cli/references/session-management.md) · [简体中文](../../../../zh-Hans/skills/playwright-cli/references/session-management.md) · [繁體中文](../../../../zh-Hant/skills/playwright-cli/references/session-management.md) · [日本語](../../../../ja/skills/playwright-cli/references/session-management.md) · Español

Ejecuta varias sesiones aisladas simultáneas con persistencia de estado.

## Sesiones con nombre

Usa `-s` para aislar contextos:

```bash
# Browser 1: Authentication flow
playwright-cli -s=auth open https://app.example.com/login

# Browser 2: Public browsing (separate cookies, storage)
playwright-cli -s=public open https://example.com

# Commands are isolated by browser session
playwright-cli -s=auth fill e1 "user@example.com"
playwright-cli -s=public snapshot
```

## Propiedades de aislamiento

Cada sesión tiene por separado:
- Cookies
- LocalStorage / SessionStorage
- IndexedDB
- Caché
- Historial de navegación
- Pestañas abiertas

## Comandos de sesión

```bash
# List all browser sessions
playwright-cli list

# Stop a browser session (close the browser)
playwright-cli close                # stop the default browser
playwright-cli -s=mysession close   # stop a named browser

# Stop all browser sessions
playwright-cli close-all

# Forcefully kill all daemon processes (for stale/zombie processes)
playwright-cli kill-all

# Delete browser session user data (profile directory)
playwright-cli delete-data                # delete default browser data
playwright-cli -s=mysession delete-data   # delete named browser data
```

## Variable de entorno

Establece un nombre de sesión predeterminado mediante variable de entorno:

```bash
export PLAYWRIGHT_CLI_SESSION="mysession"
playwright-cli open example.com  # Uses "mysession" automatically
```

## Patrones habituales

### Extracción simultánea

```bash
#!/bin/bash
# Scrape multiple sites concurrently

# Start all browsers
playwright-cli -s=site1 open https://site1.com &
playwright-cli -s=site2 open https://site2.com &
playwright-cli -s=site3 open https://site3.com &
wait

# Take snapshots from each
playwright-cli -s=site1 snapshot
playwright-cli -s=site2 snapshot
playwright-cli -s=site3 snapshot

# Cleanup
playwright-cli close-all
```

### Sesiones de pruebas A/B

```bash
# Test different user experiences
playwright-cli -s=variant-a open "https://app.com?variant=a"
playwright-cli -s=variant-b open "https://app.com?variant=b"

# Compare
playwright-cli -s=variant-a screenshot
playwright-cli -s=variant-b screenshot
```

### Perfil persistente

Por defecto el perfil solo se mantiene en memoria. Usa `--persistent` en `open` para guardarlo en disco:

```bash
# Use persistent profile (auto-generated location)
playwright-cli open https://example.com --persistent

# Use persistent profile with custom directory
playwright-cli open https://example.com --profile=/path/to/profile
```

## Conectarse a un navegador activo

Usa `attach` para conectarte a un navegador ya activo en vez de abrir otro.

### Conectar por nombre de canal

Conecta a Chrome o Edge por canal. Debe tener depuración remota activada: abre `chrome://inspect/#remote-debugging` en él y marca «Permitir depuración remota para esta instancia de navegador».

```bash
# Attach to Chrome
playwright-cli attach --cdp=chrome

# Attach to Chrome Canary
playwright-cli attach --cdp=chrome-canary

# Attach to Microsoft Edge
playwright-cli attach --cdp=msedge

# Attach to Edge Dev
playwright-cli attach --cdp=msedge-dev
```

Canales admitidos: `chrome`, `chrome-beta`, `chrome-dev`, `chrome-canary`, `msedge`, `msedge-beta`, `msedge-dev`, `msedge-canary`.

Sin `--session`, la sesión toma el nombre del canal (p. ej., `--cdp=msedge` crea `msedge`), para que conexiones paralelas Chrome/Edge no coincidan en `default`. Pasa `--session=<name>` para cambiarlo.

### Conectar mediante endpoint CDP

Conecta a un navegador que exponga Chrome DevTools Protocol:

```bash
playwright-cli attach --cdp=http://localhost:9222
```

### Conectar mediante extensión

Conecta a un navegador con la extensión Playwright instalada:

```bash
playwright-cli attach --extension
```

### Desconectar

Desmonta una sesión conectada sin afectar al navegador externo:

```bash
# Detach the default attached session
playwright-cli detach

# Detach a specific attached session
playwright-cli -s=msedge detach
```

`detach` solo funciona en sesiones creadas con `attach`. Para las de `open`, usa `close`.

## Sesión predeterminada

Sin `-s`, los comandos usan la sesión predeterminada:

```bash
# These use the same default browser session
playwright-cli open https://example.com
playwright-cli snapshot
playwright-cli close  # Stops default browser
```

## Configuración de sesión

Configura una sesión con ajustes específicos al abrir:

```bash
# Open with config file
playwright-cli open https://example.com --config=.playwright/my-cli.json

# Open with specific browser
playwright-cli open https://example.com --browser=firefox

# Open in headed mode
playwright-cli open https://example.com --headed

# Open with persistent profile
playwright-cli open https://example.com --persistent
```

## Buenas prácticas

### 1. Nombrar sesiones según su propósito

```bash
# GOOD: Clear purpose
playwright-cli -s=github-auth open https://github.com
playwright-cli -s=docs-scrape open https://docs.example.com

# AVOID: Generic names
playwright-cli -s=s1 open https://github.com
```

### 2. Limpiar siempre

```bash
# Stop browsers when done
playwright-cli -s=auth close
playwright-cli -s=scrape close

# Or stop all at once
playwright-cli close-all

# If browsers become unresponsive or zombie processes remain
playwright-cli kill-all
```

### 3. Eliminar datos obsoletos del navegador

```bash
# Remove old browser data to free disk space
playwright-cli -s=oldsession delete-data
```
