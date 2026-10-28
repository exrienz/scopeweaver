# Corrección de empaquetado de avisos de licencia v0.2.1

[English](../release-notice-repair.md) · [한국어](../ko/release-notice-repair.md) · [简体中文](../zh-Hans/release-notice-repair.md) · [繁體中文](../zh-Hant/release-notice-repair.md) · [日本語](../ja/release-notice-repair.md) · Español

`repair-release-notices.yml` es una corrección manual y única de la versión v0.2.1
existente. Añade `THIRD_PARTY_NOTICES.txt` a los cinco ZIP publicados y a la
imagen Linux amd64/arm64 existente. Conserva la etiqueta Git, cada entrada original de archivo,
todas las capas originales de imagen y la configuración de ejecución. No compila
código de la aplicación. El digest de imagen de entrada debe coincidir con las etiquetas v0.2.1 y
latest existentes antes de cambiar las oficiales.

Ejecuta solo después de que el flujo original de versión por etiqueta (incluido su
trabajo Docker) termine correctamente, para que no sobrescriba la corrección.
Revisa el flujo de reparación y confirma el digest inmutable actual de imagen
multiarquitectura. El flujo usa su `GITHUB_TOKEN` existente para recursos de versión
y GHCR; no requiere cambiar permisos OAuth personales. Publica el aviso
independiente, conserva las sumas originales como `SHA256SUMS.original`
y guarda ZIP originales/corregidos, hashes de ejecutables, manifiestos/configuración
por plataforma y ambos digest en un artefacto del flujo durante 30 días.
El mantenedor debe indicar en las notas de versión las sumas y digest de imagen
cambiados. Las versiones y hashes de la aplicación no cambian.

La sustitución de recursos GitHub y actualización de etiquetas del registro son operaciones separadas. Si
la publicación falla tras cualquiera, **no vuelvas a ejecutar el flujo contra la
versión parcialmente modificada**. Descarga el artefacto conservado antes de que
caduque. Revisa `archive-verification.json`, `image-verification.json` y los
índices de imagen original/corregida para identificar qué parte ya terminó.
Con una sesión de mantenedor autorizada, termina la misma corrección validada:

```bash
# Run from the downloaded artifact root; inspect the JSON evidence first.
set -euo pipefail
assets="$(gh release view v0.2.1 --json assets --jq '.assets[].name')"
if grep -Fxq SHA256SUMS.original <<< "$assets"; then
  mkdir -p recovery-existing
  gh release download v0.2.1 --pattern SHA256SUMS.original --dir recovery-existing
  cmp repaired/SHA256SUMS.original recovery-existing/SHA256SUMS.original || exit 1
else
  gh release upload v0.2.1 repaired/SHA256SUMS.original
fi
gh release upload v0.2.1 THIRD_PARTY_NOTICES.txt --clobber
gh release upload v0.2.1 repaired/scopeweaver-0.2.1-*.zip repaired/SHA256SUMS --clobber
image=ghcr.io/cskwork/scopeweaver
digest="$(cat repaired/corrected-image-digest.txt)"
docker buildx imagetools create --tag "$image:v0.2.1" --tag "$image:latest" "$image@$digest"
```

Después descarga otra vez los archivos publicados y ejecuta `sha256sum --check SHA256SUMS`;
verifica que ambas etiquetas resuelvan al digest corregido registrado. Los ZIP
originales del artefacto y `original_image_digest` de `image-verification.json`
permiten revertir si el mantenedor elige restaurar el empaquetado
original. Conserva las pruebas de corrección más allá de la caducidad del artefacto según necesites.
