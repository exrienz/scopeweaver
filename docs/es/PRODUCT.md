# Producto ScopeWeaver

[English](../../PRODUCT.md) · [한국어](../ko/PRODUCT.md) · [简体中文](../zh-Hans/PRODUCT.md) · [繁體中文](../zh-Hant/PRODUCT.md) · [日本語](../ja/PRODUCT.md) · Español

<!-- impeccable:product-schema 1 -->

## Plataforma
web

## Tecnologías
La aplicación usa backend Go, frontend Next.js integrado y PostgreSQL. La página GitHub Pages separada aprobada usa HTML, CSS y JavaScript simples; debe funcionar sin backend.

## Usuarios
Desarrolladores e investigadores de seguridad que evalúan flujos asistidos por IA para investigación y validación aisladas localmente. El propietario aprobó este público y el alcance de aplicación/documentación en seis idiomas.

## Propósito del producto
Coordinar planificación de objetivos, agentes, alcance de activos, cobertura, hallazgos y revisión humana en una consola autoalojada. La presentación ayuda a entender el producto, inspeccionar pruebas honestas y acceder a descargas/configuración.

## Posicionamiento
Grafos compartidos de activos y de exploración por tarea vinculan objetivos con direcciones, observaciones y pruebas. Un adaptador externo expone tareas y hallazgos a Claude Code, Codex y Pi.

## Contexto operativo
Aplicación autoalojada que requiere PostgreSQL y proveedor de modelos configurado aparte para ejecutar agentes. GitHub Pages aloja una introducción estática pública, no un servicio activo de pruebas ni backend.

## Capacidades y restricciones
Funciones y configuración se basan en README.md, adapters/agent/README.md y código. La versión v0.1.0 ofrece cinco archivos de plataforma e imagen Docker multiarquitectura. El adaptador es una interfaz de control, no sustituye agentes ni convierte suscripciones en claves API. Las restricciones permiten investigación/validación aislada local y prohíben pruebas contra objetivos en línea. La discrepancia histórica de la prueba de resumen de alcance se corrigió en `75a706d`; consulta la verificación y limitaciones actuales. No afirmes éxito de toda la suite ni preparación para producción.

## Compromisos de marca
Conservar nombre ScopeWeaver y logo de cuadrícula. Las capturas establecen una identidad precisa y neutral con azul moderado. Inglés predeterminado y opciones coreana, china simplificada/tradicional, japonesa y española. Respetar autoría ARTEX y AGPL-3.0.

## Pruebas disponibles
Las capturas del repositorio en screenshots/en y screenshots/ko muestran datos ficticios y deben etiquetarse como demostración. Logo: web/public/scopeweaver.svg. README, procedencia, verificación, adaptador y versiones son fuentes principales. Se inspeccionó una conversación real del propietario en la interfaz local autenticada: creación de tarea, grafo y trazas, y progreso en coreano. La página pública solo puede usar este resumen sin datos privados; sin objetivos, mensajes originales, afirmaciones de vulnerabilidad, tokens ni capturas privadas. Demuestra orquestación, no validación de seguridad independiente. No se aportan testimonios verificados, adopción, comparativas ni auditorías externas.

## Principios del producto
Afirmaciones verificables. Distinguir introducción pública, datos simulados y ejecución real. Poner requisitos de configuración/modelos junto a entrada. Conservar alcance/aprobaciones. Nunca publicar credenciales, objetivos ni conversaciones privadas.

## Accesibilidad e inclusión
Web adaptable escritorio/móvil, controles por teclado, contraste legible, movimiento reducido y texto completo en inglés, coreano, chino simplificado/tradicional, japonés y español.
