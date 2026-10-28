# Gestión de almacenamiento

[English](../../../../../skills/playwright-cli/references/storage-state.md) · [한국어](../../../../ko/skills/playwright-cli/references/storage-state.md) · [简体中文](../../../../zh-Hans/skills/playwright-cli/references/storage-state.md) · [繁體中文](../../../../zh-Hant/skills/playwright-cli/references/storage-state.md) · [日本語](../../../../ja/skills/playwright-cli/references/storage-state.md) · Español

Gestiona cookies, localStorage, sessionStorage y estado de almacenamiento del navegador.

## Estado de almacenamiento

Guarda y restaura el estado completo, incluidas cookies y almacenamiento.

### Guardar estado

```bash
# Save to auto-generated filename (storage-state-{timestamp}.json)
playwright-cli state-save

# Save to specific filename
playwright-cli state-save my-auth-state.json
```

### Restaurar estado

```bash
# Load storage state from file
playwright-cli state-load my-auth-state.json

# Reload page to apply cookies
playwright-cli open https://example.com
```

### Formato del archivo de estado

El archivo guardado contiene:

```json
{
  "cookies": [
    {
      "name": "session_id",
      "value": "abc123",
      "domain": "example.com",
      "path": "/",
      "expires": 1893456000,
      "httpOnly": true,
      "secure": true,
      "sameSite": "Lax"
    }
  ],
  "origins": [
    {
      "origin": "https://example.com",
      "localStorage": [
        { "name": "theme", "value": "dark" },
        { "name": "user_id", "value": "12345" }
      ]
    }
  ]
}
```

## Cookies

### Listar todas las cookies

```bash
playwright-cli cookie-list
```

### Filtrar cookies por dominio

```bash
playwright-cli cookie-list --domain=example.com
```

### Filtrar cookies por ruta

```bash
playwright-cli cookie-list --path=/api
```

### Obtener una cookie concreta

```bash
playwright-cli cookie-get session_id
```

### Establecer una cookie

```bash
# Basic cookie
playwright-cli cookie-set session abc123

# Cookie with options
playwright-cli cookie-set session abc123 --domain=example.com --path=/ --httpOnly --secure --sameSite=Lax

# Cookie with expiration (Unix timestamp)
playwright-cli cookie-set remember_me token123 --expires=1893456000
```

### Eliminar una cookie

```bash
playwright-cli cookie-delete session_id
```

### Borrar todas las cookies

```bash
playwright-cli cookie-clear
```

### Avanzado: varias cookies u opciones personalizadas

Para casos complejos, como añadir varias cookies a la vez, usa `run-code`:

```bash
playwright-cli run-code "async page => {
  await page.context().addCookies([
    { name: 'session_id', value: 'sess_abc123', domain: 'example.com', path: '/', httpOnly: true },
    { name: 'preferences', value: JSON.stringify({ theme: 'dark' }), domain: 'example.com', path: '/' }
  ]);
}"
```

## Almacenamiento local

### Listar elementos localStorage

```bash
playwright-cli localstorage-list
```

### Obtener un valor

```bash
playwright-cli localstorage-get token
```

### Establecer valor

```bash
playwright-cli localstorage-set theme dark
```

### Establecer valor JSON

```bash
playwright-cli localstorage-set user_settings '{"theme":"dark","language":"en"}'
```

### Eliminar un elemento

```bash
playwright-cli localstorage-delete token
```

### Borrar localStorage

```bash
playwright-cli localstorage-clear
```

### Avanzado: varias operaciones

Para casos complejos, como establecer varios valores a la vez, usa `run-code`:

```bash
playwright-cli run-code "async page => {
  await page.evaluate(() => {
    localStorage.setItem('token', 'jwt_abc123');
    localStorage.setItem('user_id', '12345');
    localStorage.setItem('expires_at', Date.now() + 3600000);
  });
}"
```

## Almacenamiento de sesión

### Listar elementos sessionStorage

```bash
playwright-cli sessionstorage-list
```

### Obtener un valor

```bash
playwright-cli sessionstorage-get form_data
```

### Establecer valor

```bash
playwright-cli sessionstorage-set step 3
```

### Eliminar un elemento

```bash
playwright-cli sessionstorage-delete step
```

### Borrar sessionStorage

```bash
playwright-cli sessionstorage-clear
```

## IndexedDB

### Listar bases de datos

```bash
playwright-cli run-code "async page => {
  return await page.evaluate(async () => {
    const databases = await indexedDB.databases();
    return databases;
  });
}"
```

### Eliminar base de datos

```bash
playwright-cli run-code "async page => {
  await page.evaluate(() => {
    indexedDB.deleteDatabase('myDatabase');
  });
}"
```

## Patrones habituales

### Reutilizar estado de autenticación

```bash
# Step 1: Login and save state
playwright-cli open https://app.example.com/login
playwright-cli snapshot
playwright-cli fill e1 "user@example.com"
playwright-cli fill e2 "password123"
playwright-cli click e3

# Save the authenticated state
playwright-cli state-save auth.json

# Step 2: Later, restore state and skip login
playwright-cli state-load auth.json
playwright-cli open https://app.example.com/dashboard
# Already logged in!
```

### Ciclo de guardar y restaurar

```bash
# Set up authentication state
playwright-cli open https://example.com
playwright-cli eval "() => { document.cookie = 'session=abc123'; localStorage.setItem('user', 'john'); }"

# Save state to file
playwright-cli state-save my-session.json

# ... later, in a new session ...

# Restore state
playwright-cli state-load my-session.json
playwright-cli open https://example.com
# Cookies and localStorage are restored!
```

## Notas de seguridad

- Nunca confirmes en Git archivos de estado con tokens de autenticación
- Añade `*.auth-state.json` a `.gitignore`
- Elimina archivos de estado al terminar la automatización
- Usa variables de entorno para datos sensibles
- Por defecto las sesiones funcionan en memoria, más seguro para operaciones sensibles
