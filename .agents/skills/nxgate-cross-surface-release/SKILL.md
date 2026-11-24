---
name: nxgate-cross-surface-release
description: >-
  Use this skill whenever making cross-surface changes spanning the Go backend,
  WebUI (`web/`), and Android app (`android/`), modifying i18n/mobile layouts, or
  preparing commits and version releases in NXGate.
---

# NXGate Cross-Surface Change & Release Runbook

Use this workflow when a feature or bugfix spans backend APIs, WebUI, Android, mobile layouts, i18n, or release preparation.

## 1. Cross-Surface Parity Checklist

- **Dynamic selectors instead of narrow subsets**: Never hardcode short lists of countries, tunnels, groups, or outbounds in WebUI (`web/`) or Android (`android/`). Always merge live API data (e.g., `/api/nodes`, `/api/tunnels`, `/api/tunnel-groups`) with preset codes and localization fallbacks (`Intl.DisplayNames` / `Locale`).
- **Config / empty-value round-trip**: Verify environment variable parsing, JSON persistence on restart, and WebUI form serialization/redirects together when changing auth or path settings.
- **Mobile responsive containment (`390px`)**:
  - Wide visual blocks (topology mindmaps, tables, code containers) must have `min-width: 0; max-width: 100%; overflow-x: auto` inside their own wrapper.
  - Keep `overflow-x: clip` (never `hidden`) on `html`, `body`, and `.app-layout` so `.sidebar` `position: sticky` continues working.
  - Ensure `web/tests/ui-smoke.mjs` tests populated realistic states and checks `scrollWidth <= clientWidth + 1` on **every active view and tab** at `390px`.
- **i18n symmetry**: Keep `web/src/i18n.js` (`zh`/`en`), `web/dist/index.html` (`data-i18n*`), and Android `AppStrings.kt` / `LocalizedDescriptions.kt` aligned.

## 2. Rebuild & Validate Locally

1. If `web/src/**` changed, rebuild the bundle before testing:
   ```bash
   npm --prefix web run build
   ```
2. Run the unified verification gate:
   ```bash
   ./scripts/verify.sh --changed
   ```
   *(If running inside a read-only sandbox where `~/.gradle` cannot be locked, run `./scripts/verify.sh --changed --skip-android` first, then run `./scripts/verify.sh --android-only` with elevated filesystem permission.)*

## 3. Release & Version Sync Discipline

- Batch all related bugfixes cleanly; never bump versions mid-debugging for individual fixes.
- When cutting a release, run:
  ```bash
  ./scripts/check-version.sh --set <new-version> --bump-code
  ./scripts/verify.sh --all
  ```
- Inspect `.agents/` rules if unsure whether a networking invariant also applies (`nxgate-dual-track-change`).
