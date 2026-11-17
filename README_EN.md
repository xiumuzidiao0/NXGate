# NXGate

[简体中文](README.md) | [English](README_EN.md)

[![Release](https://img.shields.io/github/v/release/xiumuzidiao0/NXGate?style=flat-square&label=Release)](https://github.com/xiumuzidiao0/NXGate/releases/latest)
[![CI](https://img.shields.io/github/actions/workflow/status/xiumuzidiao0/NXGate/ci.yml?branch=main&style=flat-square&label=CI)](https://github.com/xiumuzidiao0/NXGate/actions/workflows/ci.yml)
[![Go](https://img.shields.io/badge/Go-1.26+-00ADD8?style=flat-square)](go.mod)

NXGate is a Go-based multi-egress proxy gateway for Linux servers. It maintains concurrent VPN tunnels or custom upstream proxies, routes local HTTP and SOCKS5 listeners across filtered exits, optionally manages `sing-box` inbounds and encrypted subscriptions, and ships with an embedded Web UI and a companion Android app.

---

## Features

- **Dual Drivers with Host Route Isolation**
  - **Kernel TUN (OpenVPN)**: Uses `route-noexec` / `route-nopull` and isolated policy routing tables (`100 + devIndex`) so tunnel traffic stays separated from host SSH and default routes.
  - **User-space SSTP (Zero-TUN)**: Operates without `/dev/net/tun` or `CAP_NET_ADMIN`, suitable for unprivileged containers or non-Linux hosts.
- **Node Filtering & Dynamic Exit Groups**
  - Pulls feeds from mirrors, local snapshots, and user-imported HTTP/SOCKS5 proxies, filtering by country/region, network type (residential/datacenter), risk score, and reachability.
  - Maintains primary and secondary dynamic exit groups with periodic health checks, weighted quarantine, and automatic replacement.
- **Multi-Port Proxy Relay**
  - Single-port HTTP CONNECT and SOCKS5 (TCP + UDP Associate) listeners bound to all healthy tunnels, specific tunnels, or dynamic groups, supporting `round_robin`, `random`, `interval`, and `least_rtt` policies.
- **Optional `sing-box` Edge Inbounds & Subscriptions**
  - Manages multi-protocol inbounds chained outbound to `direct` or any local proxy port.
  - Serves Raw/Base64 and Clash YAML subscriptions through the Web port with `SUB_TOKEN` protection and optional end-to-end `age` encryption (`X25519` / `MLKEM768-X25519`).
- **Control Plane**
  - Single static Go binary embedding a bilingual (Chinese/English) Web console via `go:embed` with SSE live updates.
  - Companion Kotlin / Jetpack Compose Android client supporting QR profile import, biometric lock, and remote routing controls.

---

## Quick Start

### Linux (`systemd`)

Review the installer first, then run on a `systemd`-based Linux host (Debian, Ubuntu, CentOS, RHEL, Rocky, AlmaLinux, Fedora):

```bash
curl -fL https://raw.githubusercontent.com/xiumuzidiao0/NXGate/main/install.sh -o install.sh
less install.sh
bash install.sh
```

- Requires an interactive terminal (`/dev/tty`) before installing dependencies; skips `sing-box` by default and does not modify firewall, SSH, or policy routing rules automatically (pre-existing rules from older scripts are left intact).

### Docker Compose

[`docker-compose.yml`](docker-compose.yml) runs in user-space mode by default (no `--privileged` or TUN device required):

```bash
docker compose up -d
```

By default, Web is published on `0.0.0.0:8787` and proxy listeners bind to `127.0.0.1:7928`. To restrict Web to loopback, set `NXGATE_UI_PUBLISH_HOST=127.0.0.1` in `.env` before starting Compose.

### Service CLI

The installer registers the `nx` helper and `nxgate` systemd unit:

```bash
nx               # Interactive terminal menu
nx status        # Print service status and access URL
nx start         # Start service
nx stop          # Stop service
nx restart       # Restart service
nx update        # Update release binary
journalctl -u nxgate -f
```

---

## Configuration

- **Precedence**: `systemd` installs load `/opt/nxgate/config.env`. Settings saved from the Web console or API persist in `${DATA_DIR}/settings.json` and override corresponding startup environment variables on restart. Standalone binaries read process environment variables directly.
- **Security Defaults**:
  - When unset, initial startup generates a random path prefix (`${DATA_DIR}/admin_path.txt`) and random password (`${DATA_DIR}/ui_password.txt`) for user `admin`.
  - Both `ui_path` and Web authentication can be explicitly disabled in settings; if exposed on wildcard binds such as `0.0.0.0` without auth, both server logs and Web UI display an active security warning. Place public deployments behind HTTPS and source restrictions.
  - Proxy listeners bind to `127.0.0.1` by default.

| Environment Variable | Default | Description |
| :--- | :--- | :--- |
| `UI_HOST` | `0.0.0.0` | Web console bind address |
| `UI_PORT` | `8787` | Web console port (`1-65535`) |
| `UI_PATH` | Random & persisted | Secret URL path prefix |
| `UI_USERNAME` / `UI_PASSWORD` | `admin` / Random | Web Basic Auth credentials |
| `SUB_TOKEN` | Random & persisted | Subscription access token |
| `AGE_ENCRYPT_ENABLED` | `false` | Enable `age` subscription encryption |
| `AGE_PUBLIC_KEY` | Empty | `age1...` or `age1pq...` recipient public key |
| `LOCAL_PROXY_HOST` | `127.0.0.1` | Default proxy bind address |
| `LOCAL_PROXY_PORT` | `7928` | Default proxy port (`1-65535`) |
| `LOCAL_PROXY_GLOBAL_MAX_CONNECTIONS` | `1024` | Shared global proxy connection cap (`1-65536`) |
| `LOCAL_PROXY_MAX_CONNECTIONS` | `512` | Per-port connection cap (`16-4096`) |
| `DISCOVERY_COUNTRIES` | Empty | Comma-separated ISO country filter (e.g. `JP,US`) |
| `DATA_DIR` | `data` | Runtime data and config persistence directory |

---

## Build & Verification

Requirements: Go `1.26+`, Node.js `20+` (for Web UI assets), JDK `21` and Android SDK (for Android builds/tests).

```bash
# 1. Build Web UI assets (after editing web/src/*)
npm --prefix web install
npm --prefix web run build

# 2. Build server binary or multi-arch release artifacts
CGO_ENABLED=0 go build -ldflags="-s -w" -o bin/nxgate ./cmd/nxgate
./scripts/build.sh

# 3. Run verification checks
go vet ./...
go test -count=1 ./...
npm --prefix web run test:i18n
npm --prefix web run test:status
npm --prefix web run test:ui
./scripts/check-version.sh
```

Android unit tests can be run inside `android/` via `./gradlew testDebugUnitTest --no-daemon`.
