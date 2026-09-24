# NXGate (Adaptive Multi-Egress Intelligent Routing Gateway)

<div align="center">

**Multi-Egress Traffic Scheduling, Global Residential Broadband Auto-Discovery, and Edge Anti-Censorship Gateway for Linux VPS**

[English](README_EN.md) | [简体中文](README.md)

[![Release](https://img.shields.io/github/v/release/xiumuzidiao0/NXGate?style=flat-square&label=Release&color=16a34a)](https://github.com/xiumuzidiao0/NXGate/releases/latest)
[![Go Version](https://img.shields.io/badge/Go-1.25.13+-00ADD8?style=flat-square&logo=go)](https://go.dev/)
[![Platform](https://img.shields.io/badge/Platform-amd64%20%7C%20arm64%20%7C%20386%20%7C%20arm-6366f1?style=flat-square)](https://github.com/xiumuzidiao0/NXGate/releases/latest)
[![License](https://img.shields.io/badge/License-GPL--3.0-334155?style=flat-square)](LICENSE)

</div>

NXGate is a high-performance adaptive multi-egress intelligent routing gateway built on **Go 1.25.13+ native concurrency models and the Linux kernel policy routing stack**. Packaged as a single static executable (with an embedded 5-view responsive Single Page Application Web Console and a companion Material 3 native Android management client), NXGate operates with a resident memory footprint of less than 15MB. It is engineered specifically for Linux VPS and edge instances, featuring high concurrency, zero host route contamination, and kernel-level isolated policy routing.

---

## Table of Contents

- [System Architecture](#system-architecture)
- [Core Subsystem Engineering Design](#core-subsystem-engineering-design)
  - [1. L3/L4 Policy Routing & Host Safe-Kernel Model](#1-l3l4-policy-routing--host-safe-kernel-model)
  - [2. Multi-Protocol Proxy Relay & SOCKS5 UDP Associate](#2-multi-protocol-proxy-relay--socks5-udp-associate)
  - [3. M:N Multi-Port Scheduling & Dynamic Adaptive Egress Matrix](#3-mn-multi-port-scheduling--dynamic-adaptive-egress-matrix)
  - [4. Four-Phase Full-Lifecycle Node Discovery & Filtering Pipeline](#4-four-phase-full-lifecycle-node-discovery--filtering-pipeline)
  - [5. Edge Anti-Censorship Ingress & 7×24h Watchdog Supervisor](#5-edge-anti-censorship-ingress--724h-watchdog-supervisor)
  - [6. Client Matrix (SPA Web Console & Android Native App)](#6-client-matrix-spa-web-console--android-native-app)
- [Quick Deployment & Operations](#quick-deployment--operations)
  - [One-Line Installation & Deployment](#one-line-installation--deployment)
  - [CLI Quick Reference](#cli-quick-reference)
  - [Systemd Service Lifecycle](#systemd-service-lifecycle)
- [Configuration Specification](#configuration-specification)
- [Source Build & Quality Gates](#source-build--quality-gates)
  - [Local Compilation & Execution](#local-compilation--execution)
  - [Static Cross-Compilation for All Architectures](#static-cross-compilation-for-all-architectures)
  - [Automated Quality Admission Audit](#automated-quality-admission-audit)
- [Open Source License](#open-source-license)

---

## System Architecture

NXGate establishes a multi-tier decoupled pipeline spanning from edge anti-censorship ingress to global native residential broadband egress:

```text
                                       ┌──────────────────────────────────────────────────────────┐
                                       │                    Client Applications                   │
                                       │   Android Client / Clash Meta / Web / Headless Scraper   │
                                       └────────────────────────────┬─────────────────────────────┘
                                                                    │
                                                 VLESS / Hysteria2 / TUIC / Direct SOCKS5
                                                                    ▼
┌─────────────────────────────────────────────────────────────────────────────────────────────────────────────────┐
│ NXGate Ingress & Core Gateway                                                                                   │
│                                                                                                                 │
│   ┌────────────────────────────────────────┐       ┌────────────────────────────────────────────────────────┐   │
│   │ Edge Anti-Censorship Ingress (sing-box)│       │ Unified Proxy Demuxer (RFC 1928)                       │   │
│   │ 22 Inbound Protocols (Reality/Hy2/TUIC)├──────►│ Dual-Stack Sniffer: HTTP CONNECT / SOCKS5 TCP / UDP  │   │
│   │ 30s Heartbeat Watchdog Supervisor      │       │ Isolated Auth Provider (Random Base64 Credential)      │   │
│   └────────────────────────────────────────┘       └──────────────────────────┬─────────────────────────────┘   │
│                                                                               │                                 │
│                                    M:N Port Scheduler (Round-Robin / Random / Interval / Sticky)                │
│                                                                               │                                 │
│                                                                               ▼                                 │
│   ┌─────────────────────────────────────────────────────────────────────────────────────────────────────────┐   │
│   │ Dynamic Egress Pool & Circuit Breaker                                                                   │   │
│   │   • System Primary Group (tun0, Priority System Egress)                                                 │   │
│   │   • Dynamic Groups (tun1..tunN, Auto-Balanced by Country / Residential / Physical AI Unlock)           │   │
│   │   • Sliding-Window Failure Counter & 15s Sub-Second Failover Engine                                     │   │
│   └───────────────────────────────────────────────────┬─────────────────────────────────────────────────────┘   │
└───────────────────────────────────────────────────────┼─────────────────────────────────────────────────────────┘
                                                        │
                         Linux Policy Routing (SO_BINDTODEVICE + Isolated Table IDs)
                                                        │
┌───────────────────────────────────────────────────────▼─────────────────────────────────────────────────────────┐
│ Linux Kernel & Network Stack                                                                                    │
│                                                                                                                 │
│   [Priority 50]  sport/dport 22 ───────────────────► Table main (Host eth0 Default Gateway, 100% Locked)       │
│   [Priority 1000] oif tun0      ───────────────────► Table 100  (Default via tun0 dev, route-noexec)           │
│   [Priority 1000] oif tun1      ───────────────────► Table 101  (Default via tun1 dev, route-noexec)           │
│   [Priority 1000] oif tunN      ───────────────────► Table 100+N(Default via tunN dev, route-noexec)           │
└───────────────────────────────────────────────────────┬─────────────────────────────────────────────────────────┘
                                                        │
                                                        ▼
                                       ┌──────────────────────────────────┐
                                       │     Global VPNGate Exit Matrix   │
                                       │  Residential Broadband & Datacenter│
                                       └──────────────────────────────────┘
```

---

## Core Subsystem Engineering Design

### 1. L3/L4 Policy Routing & Host Safe-Kernel Model

To eliminate the common vulnerability where OpenVPN hijacks default host routing and causes SSH lockouts, NXGate constructs a kernel-level barrier:

- **Complete Route Execution Disabling (`route-noexec` & `route-nopull`)**:
  OpenVPN sub-processes are launched with low-level flags that physically strip them of permission to modify kernel routing tables or invoke `ip route`.
- **Dedicated Independent Routing Tables**:
  Each egress tunnel interface is allocated an isolated routing table ID (`Table 100` for `tun0`, `Table 101` for `tun1`, etc.).
- **Host Interface Pinning**:
  SSH daemon ports and administrative traffic are explicitly locked to priority 50 routing rules (`Table main`), guaranteeing SSH connectivity under all circumstances.

---

### 2. Multi-Protocol Proxy Relay & SOCKS5 UDP Associate

- **Dual-Stack Single-Port Auto-Sensing**:
  A single listener port dynamically demuxes HTTP CONNECT tunneling and RFC 1928 SOCKS5 TCP handshakes without requiring separate ports.
- **SOCKS5 UDP Associate Full Cone Relay**:
  Implements full UDP encapsulation and socket lifecycle forwarding, enabling DNS resolution, VoIP, and gaming over tunnels.
- **High-Performance Memory Pool (`sync.Pool`)**:
  Relay buffers use 32KB pre-allocated slices recycled via `sync.Pool`, reducing memory allocation churn by 80%+ under sustained gigabit workloads.

---

### 3. M:N Multi-Port Scheduling & Dynamic Adaptive Egress Matrix

- **Port Routing Rules**:
  Exposes arbitrary custom ports that route traffic to specific dynamic groups, dedicated tunnels, or round-robin rotation pools.
- **Dynamic Group Evaluator**:
  Periodically evaluates candidates matching target countries, residential ISP requirements, and AI unlock criteria, ensuring warm pool elasticity.
- **Sub-Second Failover**:
  Sliding-window error tracking automatically quarantines degraded exits and seamlessly promotes candidate tunnels in less than 15 seconds.

---

### 4. Four-Phase Full-Lifecycle Node Discovery & Filtering Pipeline

Out of thousands of global candidate endpoints, NXGate filters for high-reliability connections through four sequential gates:

1. **Phase 1: Millisecond TCP Port Knock**:
   Concurrent 48-worker pool tests SYN handshake latency within 2.5 seconds, pruning offline endpoints and unreachable ports.
2. **Phase 2: L4 Throughput Probing**:
   Physical slice download tests verify sustained downstream bandwidth of $\ge 70\text{ KB/s}$, eliminating zero-throughput zombie nodes.
3. **Phase 3: Triple AI Physical Verification**:
   Validates official endpoints for OpenAI (`ios.chat.openai.com`), Claude (`api.anthropic.com`), and Google Gemini.
4. **Phase 4: Residential Broadband Heuristic Classification**:
   Multi-signal analysis of ASN, ISP prefixes, reverse DNS, and hosting CIDR lists tags nodes as native residential or datacenter.

---

### 5. Edge Anti-Censorship Ingress & 7×24h Watchdog Supervisor

- **22 Inbound Protocols**:
  Integrated sing-box core manages VLESS-REALITY, Hysteria2, TUIC v5, Shadowsocks 2022, and AnyTLS inbounds, multiplexing traffic into NXGate's residential exit matrix.
- **Native Universal & Clash Subscriptions with age Encryption**:
  - **Single WebUI Port Delivery**: Distributes Universal (Base64/Raw) and Clash Meta (YAML) subscriptions directly through the WebUI port without secondary Python or Caddy listeners.
  - **Random SubToken Protection**: Dedicated 16-character random security token protects subscription endpoints (`/<token>/api/singbox/subscription` and `/sub/<token>`), returning hidden 404s to unauthorized scanners.
  - **age End-to-End Encryption (RFC 5234 / RFC 7405)**: Supports modern `X25519` and post-quantum `MLKEM768-X25519` recipients. When enabled, subscriptions are emitted as armored age ciphertext, preventing intermediate interception and caching leaks.
- **7×24h Watchdog**:
  Monitors daemon health every 30 seconds and automatically initiates cold-start recovery if an unrecoverable state or OOM occurs.

---

### 6. Client Matrix (SPA Web Console & Android Native App)

- **Responsive 5-View SPA Console**:
  Embedded in the binary via `go:embed`. Features deep space palette, real-time SVG waveform charts, interactive port matrix, node square with multi-attribute filtering, and comprehensive system settings.
- **Material 3 Android Management App (`com.nxgate.app`)**:
  - Native Jetpack Compose architecture with Monet wallpaper dynamic color support;
  - Adaptive 5-destination navigation (Dashboard, Monitoring, Routing, Nodes, System);
  - CameraX + ZXing QR scanner for one-click gateway profile import (`nxgate://server`);
  - Biometric hardware security lock (BiometricPrompt) with customizable background timeout grace periods;
  - Dual subscription manager with system share and deep-link one-click import into external clients (`clash://`, `sing-box://`, `v2rayng://`);
  - Quick Settings Tile (`TileService`) and Android Home Screen Desktop Widget (`AppWidgetProvider`) for glanceable status and one-tap exit failover;
  - Resilient SSE real-time streaming with exponential backoff and non-intrusive status micro-capsules;
  - Complete English and Simplified Chinese internationalization (i18n).

---

## Quick Deployment & Operations

### One-Line Installation & Deployment

Execute with `root` privileges on any modern Linux distribution (Debian, Ubuntu, CentOS, RHEL, Rocky, AlmaLinux, Alpine):

```bash
curl -sSL https://raw.githubusercontent.com/xiumuzidiao0/NXGate/main/install.sh | bash
```

The script automatically matches CPU architecture, downloads precompiled binaries, configures systemd daemon units, and links the global `nx` CLI helper.

### CLI Quick Reference

Manage the gateway from any terminal session with `nx`:

```bash
nx               # Launch interactive terminal control center
nx status        # Print gateway status, ingress protocols, and active exits
nx update        # Check and pull latest GitHub Release binary with SHA-256 verification
nx restart       # Gracefully restart the gateway service
nx start         # Start the gateway service
nx stop          # Stop the gateway service
nx logs          # View real-time journal logs (journalctl -u nxgate -f)
```

### Systemd Service Lifecycle

```bash
# Check service status
systemctl status nxgate

# Restart service
systemctl restart nxgate

# Check autostart status
systemctl is-enabled nxgate
```

---

## Configuration Specification

Configuration file is located at `/opt/nxgate/config.env` (compatible with legacy `/opt/aimilivpn/config.env`, environment variables override file settings):

| Environment Variable | Default | Valid Range / Format | Description |
| :--- | :--- | :--- | :--- |
| `UI_HOST` | `::` | IP string | Web console bind address |
| `UI_PORT` | `8787` | `1-65535` | Web console listening port |
| `UI_PATH` | `enter` | Alphanumeric string | Secret URL path prefix (anti-scanning, e.g. `/enter/`) |
| `UI_USERNAME` | `admin` | String | Web console administrative username |
| `UI_PASSWORD` | *(random)* | String | Web console administrative password |
| `SUB_TOKEN` | *(random)* | 16-char string | Random security token for secret subscription delivery |
| `AGE_ENCRYPT_ENABLED` | `false` | `true/false` | Enable age end-to-end subscription encryption |
| `AGE_PUBLIC_KEY` | *(empty)* | `age1...` / `age1pq...` | age recipient public key for subscription encryption |
| `LOCAL_PROXY_HOST`| `127.0.0.1` | IP string | Local proxy listener address |
| `LOCAL_PROXY_PORT`| `7928` | `1-65535` | Local proxy listener port (HTTP/SOCKS5 auto-sensing) |
| `LOCAL_PROXY_MAX_CONNECTIONS` | `512` | `16-4096` | Max concurrent proxy connections |
| `FETCH_INTERVAL_SECONDS` | `900` | `60-86400` | Full mirror feed refresh interval (seconds) |
| `CHECK_INTERVAL_SECONDS` | `20` | `5-300` | Active tunnel health check heartbeat interval (seconds) |
| `TARGET_VALID_NODES` | `5` | `1-50` | Minimum prechecked candidates maintained in memory pool |
| `MAX_SCAN_ROWS` | `1000` | `10-5000` | Maximum rows parsed per feed refresh |
| `DISCOVERY_COUNTRIES` | *(empty)* | Comma-separated ISO (e.g. `JP,US`) | Country filter whitelist (empty = global) |
| `DATA_DIR` | `/opt/nxgate/data` | Absolute path | Runtime certificates, routes, and data directory |

---

## Source Build & Quality Gates

### Local Compilation & Execution

Requires Go 1.25.13+:

```bash
# 1. Clone repository
git clone https://github.com/xiumuzidiao0/NXGate.git
cd NXGate

# 2. Compile binary for current platform
CGO_ENABLED=0 go build -ldflags="-s -w" -o bin/nxgate ./cmd/nxgate

# 3. Run gateway service (requires root for TUN interface management)
sudo ./bin/nxgate
```

### Static Cross-Compilation for All Architectures

Execute the build script to output zero-dependency static binaries for 4 CPU architectures:

```bash
chmod +x scripts/build.sh
./scripts/build.sh
```

Artifacts are emitted into `dist/`:
- `nxgate_linux_amd64` (x86_64 servers, uncompressed ELF and `.gz` archives)
- `nxgate_linux_arm64` (aarch64 Raspberry Pi / Ampere ARM servers)
- `nxgate_linux_386` (32-bit x86)
- `nxgate_linux_arm` (32-bit ARMv7)
- `SHA256SUMS.txt` (Cryptographic verification checksums)

### Automated Quality Admission Audit

Codebase is gated by local Git Pre-Push hooks and GitHub Actions CI/CD workflows:

```bash
# 1. Static code analysis and CVE vulnerability scan
bash scripts/audit.sh

# 2. Comprehensive unit and integration test suite
go test -count=1 -v ./...

# 3. Full-link version consistency check
bash scripts/check-version.sh

# 4. Multi-viewport Playwright regression tests
cd web && npm test
```

---

## Open Source License

NXGate is licensed under the [GNU General Public License v3.0 (GPL-3.0)](LICENSE).
Derivative works must retain author attribution in accordance with the license terms.
