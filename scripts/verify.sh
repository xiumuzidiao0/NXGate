#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

MODE="changed"
BASE_REF=""
SKIP_ANDROID=0
ONLY_ANDROID=0
GRADLE_OFFLINE="--offline"

while [ "$#" -gt 0 ]; do
    case "$1" in
        --all)
            MODE="all"
            shift
            ;;
        --changed)
            MODE="changed"
            shift
            ;;
        --base)
            BASE_REF="${2:-}"
            shift 2
            ;;
        --skip-android)
            SKIP_ANDROID=1
            shift
            ;;
        --android-only)
            ONLY_ANDROID=1
            shift
            ;;
        --online)
            GRADLE_OFFLINE=""
            shift
            ;;
        -h|--help)
            cat <<'EOF'
Usage: ./scripts/verify.sh [--changed | --all] [--base <git-ref>] [--skip-android | --android-only] [--online]

Gates:
  1. Version consistency across VERSION, Go, install.sh, WebUI, Android
  2. Git diff whitespace hygiene (git diff --check)
  3. Static architecture invariants (MS-SSTP TLS, mobile CSS containment, userland health)
  4. Shell script syntax (+ shellcheck if installed)
  5. Go vet & full Go test suite
  6. WebUI (when web/ changed or --all): read-only esbuild freshness check, test:i18n, test:status, test:ui
  7. Android (when android/ changed or --all): testDebugUnitTest
EOF
            exit 0
            ;;
        *)
            echo "Unknown option: $1" >&2
            exit 2
            ;;
    esac
done

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;36m'
PLAIN='\033[0m'
BOLD='\033[1m'

collect_changed_files() {
    {
        if [ -n "$BASE_REF" ] && git rev-parse --verify "$BASE_REF" >/dev/null 2>&1; then
            git diff --name-only "$BASE_REF"...HEAD 2>/dev/null || true
        fi
        git diff --name-only HEAD 2>/dev/null || true
        git diff --cached --name-only 2>/dev/null || true
        git ls-files --others --exclude-standard 2>/dev/null || true
    } | sort -u
}

CHANGED_FILES=""
if [ "$MODE" = "changed" ]; then
    CHANGED_FILES="$(collect_changed_files)"
fi

should_check_scope() {
    local prefix="$1"
    if [ "$MODE" = "all" ]; then
        return 0
    fi
    printf '%s\n' "$CHANGED_FILES" | grep -qE "^${prefix}"
}

run_android_gate() {
    if [ "$SKIP_ANDROID" -eq 1 ]; then
        echo -e "${YELLOW}[skip] Android unit tests (--skip-android)${PLAIN}"
        return 0
    fi
    if [ ! -x "./android/gradlew" ]; then
        echo -e "${RED}[fail] ./android/gradlew not found or not executable${PLAIN}"
        return 1
    fi
    echo -e "${BLUE}==> Running Android unit tests...${PLAIN}"
    ANDROID_USER_HOME="${ANDROID_USER_HOME:-/tmp/android-home}" \
        ./android/gradlew -p android testDebugUnitTest ${GRADLE_OFFLINE} --no-daemon --max-workers=1
    echo -e "${GREEN}[ok] Android testDebugUnitTest passed${PLAIN}"
}

if [ "$ONLY_ANDROID" -eq 1 ]; then
    run_android_gate
    exit 0
fi

echo -e "\n${BOLD}${BLUE}=== NXGate Verification Gate (${MODE}) ===${PLAIN}\n"

# 1. Version consistency
if [ -x "./scripts/check-version.sh" ]; then
    echo -e "${BLUE}[1/7] Checking version consistency...${PLAIN}"
    ./scripts/check-version.sh
else
    echo -e "${YELLOW}[1/7] scripts/check-version.sh missing or not executable, skipping${PLAIN}"
fi

# 2. Whitespace hygiene
echo -e "${BLUE}[2/7] Checking git diff whitespace hygiene...${PLAIN}"
git diff --check
git diff --cached --check
echo -e "${GREEN}[ok] No whitespace errors${PLAIN}"

# 3. Static architecture invariants
echo -e "${BLUE}[3/7] Checking static architecture invariants...${PLAIN}"
if ! awk '/func sstpTLSConfig/,/^}/' pkg/sstp/tunnel.go | grep -q 'InsecureSkipVerify: true'; then
    echo -e "${RED}[fail] pkg/sstp/tunnel.go sstpTLSConfig must keep InsecureSkipVerify: true (per [MS-SSTP])${PLAIN}" >&2
    exit 1
fi
if ! awk '/func \(t \*Tunnel\) IsHealthy/,/^}/' pkg/tunnel/model.go | grep -q 'userlandHealthyLocked'; then
    echo -e "${RED}[fail] pkg/tunnel/model.go IsHealthy() must check userlandHealthyLocked()${PLAIN}" >&2
    exit 1
fi
if ! grep -q '\.mindmap-canvas' web/dist/styles.css || ! grep -q 'overflow-x: clip' web/dist/styles.css; then
    echo -e "${RED}[fail] web/dist/styles.css must retain .mindmap-canvas containment and overflow-x: clip${PLAIN}" >&2
    exit 1
fi
echo -e "${GREEN}[ok] Static architecture invariants intact${PLAIN}"

# 4. Shell script syntax & lint
echo -e "${BLUE}[4/7] Checking shell scripts...${PLAIN}"
sh_files=()
while IFS= read -r -d '' f; do
    sh_files+=("$f")
done < <(find . -maxdepth 4 \( -name "install.sh" -o -path "./scripts/*.sh" -o -path "./scripts/*/*.sh" \) -type f -print0)
sh_files+=(".githooks/commit-msg" ".githooks/pre-push")
for f in "${sh_files[@]}"; do
    if [ -f "$f" ]; then
        bash -n "$f"
    fi
done
if command -v shellcheck >/dev/null 2>&1; then
    shellcheck --severity=error install.sh scripts/verify.sh
fi
echo -e "${GREEN}[ok] Shell scripts validated (${#sh_files[@]} files)${PLAIN}"

# 5. Go vet & unit tests
if command -v go >/dev/null 2>&1; then
    echo -e "${BLUE}[5/7] Running go vet & go test -count=1 ./...${PLAIN}"
    go vet ./...
    go test -count=1 ./...
    echo -e "${GREEN}[ok] Go vet and tests passed${PLAIN}"
else
    echo -e "${RED}[fail] Go toolchain not found${PLAIN}" >&2
    exit 1
fi

# 6. WebUI checks (scoped or --all)
if should_check_scope 'web/'; then
    echo -e "${BLUE}[6/7] Running WebUI bundle freshness, i18n, transport, and Chromium smoke checks...${PLAIN}"
    if ! command -v npm >/dev/null 2>&1; then
        echo -e "${RED}[fail] npm is required when web/ files changed${PLAIN}" >&2
        exit 1
    fi
    if [ ! -d "web/node_modules" ]; then
        echo -e "${RED}[fail] web/node_modules missing; run npm --prefix web ci${PLAIN}" >&2
        exit 1
    fi

    tmp_bundle="$(mktemp)"
    trap 'rm -f "$tmp_bundle"' EXIT
    ./web/node_modules/.bin/esbuild web/src/app.js --bundle --minify-whitespace --minify-syntax --format=iife --target=es2020 "--outfile=${tmp_bundle}"
    if ! cmp -s "$tmp_bundle" "web/dist/app.js"; then
        echo -e "${RED}[fail] web/dist/app.js is out of sync with web/src/**. Run: npm --prefix web run build${PLAIN}" >&2
        exit 1
    fi

    npm --prefix web run test:i18n
    npm --prefix web run test:status

    chromium_path="$(
      cd "$ROOT_DIR/web" && node -e '
      import("node:fs").then(({existsSync}) =>
        import("playwright-core").then(({chromium}) => {
          const paths = [
            process.env.PLAYWRIGHT_CHROMIUM_EXECUTABLE,
            `${process.env.HOME || ""}/.cache/ms-playwright/chromium-1234/chrome-linux/chrome`,
            "/usr/bin/chromium",
            "/usr/bin/chromium-browser",
            "/usr/bin/google-chrome",
            chromium.executablePath(),
          ].filter(Boolean);
          process.stdout.write(paths.find(p => existsSync(p)) || "");
        })
      );
    '
    )"
    if [ -n "$chromium_path" ]; then
        npm --prefix web run test:ui
    else
        echo -e "${YELLOW}[skip] Chromium executable not found for Playwright ui-smoke; CI will verify${PLAIN}"
    fi
    echo -e "${GREEN}[ok] WebUI checks passed${PLAIN}"
else
    echo -e "${YELLOW}[6/7] No web/ changes detected; skipping WebUI checks${PLAIN}"
fi

# 7. Android checks (scoped or --all)
if should_check_scope 'android/'; then
    run_android_gate
else
    echo -e "${YELLOW}[7/7] No android/ changes detected; skipping Android tests${PLAIN}"
fi

echo -e "\n${GREEN}${BOLD}=== All selected NXGate verification gates passed ===${PLAIN}\n"
