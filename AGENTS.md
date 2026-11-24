# NXGate Project Engineering Rules

These rules apply across the entire NXGate repository (`Go` backend, `web/` WebUI, `android/` Compose app, `install.sh`, and `scripts/`). They exist to prevent recurring regressions observed across multi-driver networking, cross-surface state, and mobile responsive rendering.

---

## 1. Dual-Track Symmetry (Primary / Secondary x Kernel TUN / Userland SSTP)

Any change to tunnel lifecycle, health probing, blacklisting, failover, node deduplication, or topology status **MUST** audit all 4 cells of the matrix:

| Scope | Kernel (`OpenVPN tunN`) | Userland (`SSTP sstpN`) |
| :--- | :--- | :--- |
| **Primary (`DevIndex == 0`)** | `vpn.Manager` + `Pool.StartPrimaryTunnel` | `vpn.Manager.sstpTunnel` + `primary-userland` |
| **Secondary / Dynamic (`DevIndex >= 1`)** | `Pool.StartTunnel` + `DynamicGroupManager` | `Pool.StartUserlandTunnel` + `DynamicGroupManager` |

- **Userland health parity**: Never treat `StatusConnected` or an open TCP socket as healthy for userland SSTP. `Tunnel.IsHealthy()` and `Tunnel.IsAvailable()` must check underlying session health (`Closer.(interface{ IsHealthy() bool })` / `HealthCheck`), dead sessions must transition and reap via `<-st.Done()` / `ReapStaleTunnels()`, and both `EvaluateGroup` and `healthBatch` must probe real outbound forwarding (`CheckDialerConnectivityContext` / `probeDialerConnectivity`).
- **Probe budget vs. per-target dial timeout**: When `healthBatch` probes a black-holed tunnel sequentially across multiple targets, ensure the probe context deadline exceeds `(targetCount * perTargetTimeout)` while staying below `healthRoundBudget`, so a hung relay returns explicit failure instead of `context.DeadlineExceeded` (which hysteresis treats as unknown).
- **Primary/secondary node exclusion**: `DynamicGroupManager.EvaluateGroup` must keep `primaryConnector.GetActiveNode()` inside `usedNodeIDs` so secondary groups never dial the same VPNGate relay simultaneously as the primary connection (even when primary runs in userland mode without a kernel `Tunnel` entry).
- **Verified forwarding failures block immediately**: Functional forwarding failures (handshake succeeded, IP packets blocked) must use `Blacklist().MarkManual(...)` with an explicit duration — never 3-strike `Mark(...)` that lets broken relays re-enter selection immediately.
- **MS-SSTP TLS invariant**: `sstpTLSConfig` in `pkg/sstp/tunnel.go` must retain `InsecureSkipVerify: true` (per `[MS-SSTP]`, session authentication lives at the PPP/MS-CHAPv2 layer; enabling TLS hostname/chain verification breaks VPNGate relays).
- **Port listener failover stays bounded**: `PortListener` failover retries must stay strictly inside `SelectTunnelExcluding(...)` for the rule's bound groups/tunnels and fail closed (`repGeneralFailure`) when exhausted; never escape explicit bindings. `PortListener.Start` must check `l.closed` after `Listen` returns.

---

## 2. Cross-Surface Single Source of Truth (Go / WebUI / Android)

- **No narrow hardcoded backend enums**: UI selectors for dynamic backend pools (countries, active tunnels, dynamic groups, sing-box outbounds) must dynamically merge live API data (`/api/nodes`, `/api/tunnels`, `/api/tunnel-groups`, etc.) with fallback presets and `Intl.DisplayNames` (Web) / `java.util.Locale` (Android) instead of short hardcoded lists.
- **Explicit empty/disabled config semantics**: When supporting disabling features via empty config (e.g., `UI_PATH=""`, disabled UI auth), trace the full chain: environment parsing -> file persistence/reload -> WebUI form serialization & post-save redirect URL.
- **WebUI bundle parity**: Every edit under `web/src/**` requires rebuilding `web/dist/app.js` with `npm --prefix web run build` before committing or testing Go embed outputs.
- **Bilingual i18n parity**: Keep `web/src/i18n.js` (`zh` and `en`), `web/dist/index.html` (`data-i18n*`), and Android `AppStrings.kt` / `LocalizedDescriptions.kt` synchronized, and verify via `npm --prefix web run test:i18n` and Android unit tests.

---

## 3. Mobile & Responsive Layout Containment

- **Contain wide widgets inside scroll wrappers**: Topology graphs, wide tables, and `<pre>` blocks must reside inside containers with `min-width: 0; max-width: 100%; overflow-x: auto`. Never rely on child `min-width` to expand a card or `.view-page`.
- **Use `overflow-x: clip`, not `hidden`, on page roots**: Keep `overflow-x: clip` on `html`, `body`, and `.app-layout` so `.sidebar`'s `position: sticky` stays functional while preventing mobile browsers (e.g., Android Firefox) from permanently zooming out the visual viewport.
- **Test realistic populated views across viewports**: `web/tests/ui-smoke.mjs` must populate multi-tunnel/group/multi-country mock data and assert `document.documentElement.scrollWidth <= document.documentElement.clientWidth + 1` on every active view and tab at `390px` width.

---

## 4. Verification & Release Hygiene

- **Run unified verification before committing or pushing**: Execute `./scripts/verify.sh --changed` (or `--all` before releases). Do not swallow failing UI or Android checks silently.
- **Batch version bumps**: Never bump versions mid-debugging per single fix commit. Batch verified fixes and synchronize all version locations once via `./scripts/check-version.sh --set <version> --bump-code`.
- **Host / environment notes**:
  - Go tests on ARM64 Linux with 39-bit VMA must omit `-race` locally (TSan requires >= 47-bit VMA); CI runs `-race` on `amd64`.
  - Android unit tests use `ANDROID_USER_HOME=/tmp/android-home ./android/gradlew -p android testDebugUnitTest --offline --no-daemon --max-workers=1`.
