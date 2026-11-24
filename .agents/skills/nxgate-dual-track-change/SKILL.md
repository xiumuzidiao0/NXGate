---
name: nxgate-dual-track-change
description: >-
  Use this skill whenever modifying NXGate tunnel lifecycle, OpenVPN or userland
  SSTP drivers, health probing, blacklisting, dynamic egress groups, or multi-port
  proxy scheduling/failover.
---

# NXGate Dual-Track Networking Change Checklist

Use this procedure before and after editing `pkg/sstp/`, `pkg/tunnel/`, `pkg/vpn/`, or `pkg/proxy/` to prevent dual-track regressions between primary vs. secondary tunnels and Kernel TUN vs. Userland SSTP.

## 1. Audit All 4 Cells of the Matrix

Before writing code, check how your change behaves across every cell:

| Scope | Kernel (`OpenVPN tunN`) | Userland (`SSTP sstpN`) |
| :--- | :--- | :--- |
| **Primary (`DevIndex == 0`)** | `vpn.Manager` + `Pool.StartPrimaryTunnel` | `vpn.Manager.sstpTunnel` + `primary-userland` |
| **Secondary (`DevIndex >= 1`)** | `Pool.StartTunnel` + `DynamicGroupManager` | `Pool.StartUserlandTunnel` + `DynamicGroupManager` |

Never fix a failure mode on only the primary path or only the kernel TUN path.

## 2. Verify Protocol & Lifecycle Invariants

1. **Underlying SSTP health & reap**:
   - Confirm `Tunnel.IsHealthy()` and `Tunnel.IsAvailable()` inspect the underlying userland session health (`userlandHealthyLocked()`), not merely `Status == StatusConnected`.
   - Confirm `StartUserlandTunnel` monitors `<-st.Done()`, handles `StatusStopped` before returning, and calls `p.cleanupTunnel(...)`, and `Pool.ReapStaleTunnels()` cleans up dead connected userland tunnels.
2. **Real egress forwarding verification**:
   - Confirm both admission (`DynamicGroupManager.EvaluateGroup`) and periodic巡检 (`healthBatch`) verify actual IP forwarding through `CheckDialerConnectivityContext` / `probeDialerConnectivity` for `IsUserland` tunnels.
   - Check timeout math: `probeCtx` timeout (`7s`) must exceed `len(userlandConnectivityTargets) * perTargetTimeout` (`3 * 2s = 6s`) while staying under `healthRoundBudget` (`8s`), so black-holed relays return `(false, nil)` instead of `context.DeadlineExceeded`.
3. **Node deduplication & blacklisting**:
   - Confirm `EvaluateGroup` adds `primaryConnector.GetActiveNode()` to `usedNodeIDs` so secondary groups never collide with the primary relay.
   - Use `Blacklist().MarkManual(...)` for verified forwarding failures rather than 3-strike `Mark(...)`.
4. **SSTP TLS & PPP keepalive**:
   - Preserve `sstpTLSConfig` `InsecureSkipVerify: true` in `pkg/sstp/tunnel.go` (`[MS-SSTP]` authenticates at PPP).
   - Preserve PPP LCP Echo replies alongside SSTP echoes in `pkg/sstp/tunnel.go`.
5. **Bounded port failover & listener close safety**:
   - Ensure `PortListener` failover uses `SelectTunnelExcluding(...)` within `BoundTunnelIDs` / `BoundGroupIDs` only (never escaping explicit bindings) and `PortListener.Start` checks `l.closed` after `Listen`.

## 3. Verify With Targeted Tests

Run targeted tests with race detection when supported by the host (`amd64`), or `-count=1` on ARM64 39-bit VMA:

```bash
go vet ./...
go test -count=1 ./pkg/sstp/... ./pkg/tunnel/... ./pkg/proxy/... ./pkg/vpn/...
```

Before committing or pushing, run `./scripts/verify.sh --changed`.
