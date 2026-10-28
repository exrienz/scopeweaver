# Grabación de vídeo

[English](../../../../../skills/playwright-cli/references/video-recording.md) · [한국어](../../../../ko/skills/playwright-cli/references/video-recording.md) · [简体中文](../../../../zh-Hans/skills/playwright-cli/references/video-recording.md) · [繁體中文](../../../../zh-Hant/skills/playwright-cli/references/video-recording.md) · [日本語](../../../../ja/skills/playwright-cli/references/video-recording.md) · Español

Graba sesiones de automatización para depuración, documentación o verificación. Produce WebM (códec VP8/VP9).

## Grabación básica

```bash
# Open browser first
playwright-cli open

# Start recording
playwright-cli video-start demo.webm

# Add a chapter marker for section transitions
playwright-cli video-chapter "Getting Started" --description="Opening the homepage" --duration=2000

# Navigate and perform actions
playwright-cli goto https://example.com
playwright-cli snapshot
playwright-cli click e1

# Add another chapter
playwright-cli video-chapter "Filling Form" --description="Entering test data" --duration=2000
playwright-cli fill e2 "test input"

# Stop and save
playwright-cli video-stop
```

## Buenas prácticas

### 1. Usar nombres descriptivos

```bash
# Include context in filename
playwright-cli video-start recordings/login-flow-2024-01-15.webm
playwright-cli video-start recordings/checkout-test-run-42.webm
```

### 2. Grabar secuencias completas.

Para vídeo de usuario o prueba de trabajo, conviene crear un fragmento de código y ejecutarlo con run-code.
Permite pausas adecuadas y anotaciones. Hay API nuevas de Playwright para ello.

1) Realiza el escenario con CLI y anota localizadores y acciones. Necesitarás los localizadores para obtener rectángulos de resaltado.
2) Crea un archivo con el guion de vídeo (abajo). Usa pressSequentially con retraso para escribir de forma legible y pausas razonables.
3) Usa playwright-cli run-code --filename your-script.js

**Importante**: las superposiciones tienen `pointer-events: none`; no interfieren. Puedes mantenerlas visibles al pulsar, rellenar o actuar.

```js
async page => {
  await page.screencast.start({ path: 'video.webm', size: { width: 1280, height: 800 } });
  await page.goto('https://demo.playwright.dev/todomvc');

  // Show a chapter card — blurs the page and shows a dialog.
  // Blocks until duration expires, then auto-removes.
  // Use this for simple use cases, but always feel free to hand-craft your own beautiful
  // overlay via await page.screencast.showOverlay().
  await page.screencast.showChapter('Adding Todo Items', {
    description: 'We will add several items to the todo list.',
    duration: 2000,
  });

  // Perform action
  await page.getByRole('textbox', { name: 'What needs to be done?' }).pressSequentially('Walk the dog', { delay: 60 });
  await page.getByRole('textbox', { name: 'What needs to be done?' }).press('Enter');
  await page.waitForTimeout(1000);

  // Show next chapter
  await page.screencast.showChapter('Verifying Results', {
    description: 'Checking the item appeared in the list.',
    duration: 2000,
  });

  // Add a sticky annotation that stays while you perform actions.
  // Overlays are pointer-events: none, so they won't block clicks.
  const annotation = await page.screencast.showOverlay(`
    <div style="position: absolute; top: 8px; right: 8px;
      padding: 6px 12px; background: rgba(0,0,0,0.7);
      border-radius: 8px; font-size: 13px; color: white;">
      ✓ Item added successfully
    </div>
  `);

  // Perform more actions while the annotation is visible
  await page.getByRole('textbox', { name: 'What needs to be done?' }).pressSequentially('Buy groceries', { delay: 60 });
  await page.getByRole('textbox', { name: 'What needs to be done?' }).press('Enter');
  await page.waitForTimeout(1500);

  // Remove the annotation when done
  await annotation.dispose();

  // You can also highlight relevant locators and provide contextual annotations.
  const bounds = await page.getByText('Walk the dog').boundingBox();
  await page.screencast.showOverlay(`
    <div style="position: absolute;
      top: ${bounds.y}px;
      left: ${bounds.x}px;
      width: ${bounds.width}px;
      height: ${bounds.height}px;
      border: 1px solid red;">
    </div>
    <div style="position: absolute;
      top: ${bounds.y + bounds.height + 5}px;
      left: ${bounds.x + bounds.width / 2}px;
      transform: translateX(-50%);
      padding: 6px;
      background: #808080;
      border-radius: 10px;
      font-size: 14px;
      color: white;">Check it out, it is right above this text
    </div>
  `, { duration: 2000 });

  await page.screencast.stop();
}
```

Usa creatividad; las superposiciones son potentes.

### Resumen de API de superposiciones

| Método | Uso |
|--------|----------|
| `page.screencast.showChapter(title, { description?, duration?, styleSheet? })` | Tarjeta de capítulo a pantalla completa con fondo desenfocado; ideal para transiciones |
| `page.screencast.showOverlay(html, { duration? })` | HTML personalizado para avisos, etiquetas y resaltados |
| `disposable.dispose()` | Quitar superposición persistente añadida sin duración |
| `page.screencast.hideOverlays()` / `page.screencast.showOverlays()` | Ocultar/mostrar todas temporalmente |

## Trazas frente a vídeo

| Función | Vídeo | Trazas |
|---------|-------|---------|
| Salida | WebM | Archivo de trazas (visible en Trace Viewer) |
| Muestra | Grabación visual | DOM, red, consola y acciones |
| Uso | Demostraciones, documentación | Depuración, análisis |
| Tamaño | Mayor | Menor |

## Limitaciones

- Grabar añade una pequeña sobrecarga
- Las grabaciones grandes pueden ocupar mucho disco
