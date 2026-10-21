#!/usr/bin/env python3
"""Offline installer regressions. PTY + command mocks; never install host services."""
import errno
import hashlib
import io
import json
import os
from pathlib import Path
import pty
import select
import shutil
import shlex
import signal
import subprocess
import tarfile
import tempfile
import time
import unittest

ROOT = Path(__file__).resolve().parents[2]
INSTALLER = ROOT / "install.sh"
SOURCE = INSTALLER.read_text()


class InstallerTests(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory(prefix="nxgate-installer-test-")
        self.addCleanup(self.tmp.cleanup)
        self.dir = Path(self.tmp.name)
        self.env = dict(os.environ, TEST_DIR=str(self.dir))
        self.base = f'source {shlex.quote(str(INSTALLER))}\n'
        self.base += 'INSTALL_DIR="$TEST_DIR/install"; CONFIG_FILE="$INSTALL_DIR/config.env"\n'

    def shell(self, code, check=True):
        result = subprocess.run(["bash", "-c", self.base + code], env=self.env,
                                capture_output=True, text=True, start_new_session=True, timeout=10)
        if check:
            self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
        return result

    def terminal(self, code, replies, pipeline=False):
        """Send each answer only after its exact prompt is visible. Bounded wait."""
        fixture = self.dir / "fixture.sh"
        fixture.write_text(code)
        pid, fd = pty.fork()
        if pid == 0:
            os.environ.update(self.env)
            if pipeline:
                os.execlp("bash", "bash", "-c", f'cat {shlex.quote(str(fixture))} | bash')
            os.execlp("bash", "bash", str(fixture))
        data = b""
        sent = 0
        deadline = time.monotonic() + 8
        try:
            while time.monotonic() < deadline:
                if sent < len(replies) and replies[sent][0].encode() in data:
                    os.write(fd, replies[sent][1])
                    sent += 1
                ready, _, _ = select.select([fd], [], [], 0.05)
                if ready:
                    try:
                        chunk = os.read(fd, 65536)
                    except OSError as exc:
                        if exc.errno == errno.EIO:
                            break
                        raise
                    if not chunk:
                        break
                    data += chunk
            else:
                self.fail(f"terminal timed out after {sent} answers: {data.decode(errors='replace')}")
            _, status = os.waitpid(pid, 0)
            pid = None
            self.assertEqual(sent, len(replies), data.decode(errors="replace"))
            return os.waitstatus_to_exitcode(status), data.decode(errors="replace")
        finally:
            os.close(fd)
            if pid is not None:
                os.kill(pid, signal.SIGKILL)
                os.waitpid(pid, 0)

    def full_fixture(self):
        # Insert mocks immediately before the REAL entry guard. This tests both
        # bash file and cat | bash entry detection without any host mutations.
        mocks = '''
INSTALL_DIR="$TEST_DIR/install"; CONFIG_FILE="$INSTALL_DIR/config.env"
BIN_PATH="$INSTALL_DIR/nxgate"; SERVICE_FILE="$TEST_DIR/nxgate.service"
check_root() { :; }
require_systemd() { :; }
detect_os() { :; }
detect_arch() { :; }
install_dependencies() { echo deps >> "$TEST_DIR/events"; }
build_and_deploy() { echo build >> "$TEST_DIR/events"; }
install_service() { echo service >> "$TEST_DIR/events"; }
print_install_success() { echo done >> "$TEST_DIR/events"; }
singbox_available() { return 1; }
'''
        return SOURCE.replace('# 可 source 函数用于隔离回归', mocks + '\n# 可 source 函数用于隔离回归')

    def wizard_replies(self, confirm=b"y\n"):
        return [("1. 请输入 Web", b"\n"), ("2. 请输入 Web", b"\n"),
                ("3. 请输入 Web", b"\n"), ("4. 请输入 Web", b"SecretNoEcho123\n"),
                ("5. 请输入 本地", b"\n"), ("确认保存以上配置", confirm)]

    def test_file_and_piped_interaction(self):
        for pipeline in (False, True):
            with self.subTest(pipeline=pipeline):
                events = self.dir / "events"
                events.unlink(missing_ok=True)
                answers = self.wizard_replies() + [("是否安装可选 sing-box", b"\n")]
                rc, out = self.terminal(self.full_fixture(), answers, pipeline)
                self.assertEqual(rc, 0, out)
                self.assertNotIn("SecretNoEcho123", out)
                self.assertEqual(events.read_text().splitlines(), ["deps", "build", "service", "done"])
                config = self.dir / "install/config.env"
                self.assertIn("UI_PASSWORD=SecretNoEcho123", config.read_text())
                self.assertIn("UI_HOST=0.0.0.0", config.read_text())
                self.assertIn("LOCAL_PROXY_HOST=127.0.0.1", config.read_text())
                self.assertEqual(config.stat().st_mode & 0o777, 0o600)

    def test_explicit_installer_web_host_is_preserved(self):
        for host in ("127.0.0.1", "192.0.2.10", "::1"):
            with self.subTest(host=host):
                target = self.dir / "install"
                target.mkdir(exist_ok=True)
                config = target / "config.env"
                config.write_text("UI_HOST=" + host + "\nUI_PORT=8787\nUI_PATH=enter\nUI_USERNAME=admin\nUI_PASSWORD=original-test-password\nLOCAL_PROXY_PORT=7928\n")
                rc, out = self.terminal(self.base + "configure_install_params", self.wizard_replies())
                self.assertEqual(rc, 0, out)
                self.assertIn("UI_HOST=" + host + "\n", config.read_text())
                self.assertIn("LOCAL_PROXY_HOST=127.0.0.1", config.read_text())

    def test_fresh_installer_respects_explicit_environment_host(self):
        self.env["UI_HOST"] = "127.0.0.1"
        rc, out = self.terminal(self.base + "configure_install_params", self.wizard_replies())
        self.assertEqual(rc, 0, out)
        self.assertIn("UI_HOST=127.0.0.1", (self.dir / "install/config.env").read_text())

    def test_installer_rejects_host_environment_injection(self):
        self.env["UI_HOST"] = "0.0.0.0\nLOCAL_PROXY_HOST=0.0.0.0"
        result = self.shell('configure_install_params', check=False)
        self.assertNotEqual(result.returncode, 0)
        self.assertIn('UI_HOST 必须是单个 IP', result.stderr)
        self.assertFalse((self.dir / 'install/config.env').exists())

    def test_access_url_uses_configured_host_and_validated_public_ip(self):
        for host, want in [("0.0.0.0", "203.0.113.9"), ("::", "203.0.113.9"),
                           ("127.0.0.1", "127.0.0.1"), ("::1", "[::1]")]:
            with self.subTest(host=host):
                result = self.shell("""
mkdir -p "$INSTALL_DIR"
printf 'UI_HOST=%s\nUI_PORT=8787\nUI_PATH=/enter/\n' """ + shlex.quote(host) + """ > "$CONFIG_FILE"
curl() { echo 203.0.113.9; }
get_web_access_url
""")
                self.assertEqual(result.stdout.strip(), "http://" + want + ":8787/enter/")
        # HTML/invalid addresses are not claimed as a successfully detected server IP.
        result = self.shell("curl() { echo '<html>Error</html>'; }; get_public_ip")
        self.assertEqual(result.stdout.strip(), "<SERVER-IP>")
        result = self.shell("curl() { echo '999.1.2.3'; }; get_public_ip")
        self.assertEqual(result.stdout.strip(), "<SERVER-IP>")
        result = self.shell("curl() { return 7; }; get_public_ip")
        self.assertEqual(result.stdout.strip(), "<SERVER-IP>")
        result = self.shell("""
mkdir -p "$INSTALL_DIR"
printf 'UI_HOST=0.0.0.0\nUI_PORT=8787\nUI_PATH=enter\n' > "$CONFIG_FILE"
curl() { return 7; }
get_web_access_url
""")
        self.assertEqual(result.stdout.strip(), "http://<SERVER-IP>:8787/enter/")

    def test_panels_use_same_configured_access_url(self):
        result = self.shell("""
mkdir -p "$INSTALL_DIR"
printf 'UI_HOST=127.0.0.1\nUI_PORT=8787\nUI_PATH=enter\nLOCAL_PROXY_PORT=7928\n' > "$CONFIG_FILE"
get_app_version() { echo test; }
show_singbox_status() { echo stopped; }
show_service_status() { echo stopped; }
curl() { echo unexpected-network-call >&2; return 99; }
print_install_success
menu_status
""")
        self.assertEqual(result.stdout.count("http://127.0.0.1:8787/enter/"), 2)
        self.assertNotIn("unexpected-network-call", result.stderr)
        self.assertIn('EnvironmentFile=-/opt/nxgate/config.env', (ROOT / 'scripts/nxgate.service').read_text())
        self.assertIn('EnvironmentFile=${CONFIG_FILE}', SOURCE)
        compose = (ROOT / 'docker-compose.yml').read_text()
        self.assertIn('${NXGATE_UI_PUBLISH_HOST:-0.0.0.0}:8787:8787', compose)
        self.assertIn('127.0.0.1:7928:7928', compose)

    def test_default_confirmation_cancels_without_writes(self):
        rc, out = self.terminal(self.full_fixture(), self.wizard_replies(b"\n"), True)
        self.assertNotEqual(rc, 0, out)
        self.assertFalse((self.dir / "install").exists())
        self.assertFalse((self.dir / "events").exists())

    def test_eof_aborts_without_install(self):
        rc, out = self.terminal(self.full_fixture(), [("1. 请输入 Web", b"\x04")], True)
        self.assertNotEqual(rc, 0, out)
        self.assertIn("输入已中断", out)
        self.assertFalse((self.dir / "install").exists())
        self.assertFalse((self.dir / "events").exists())

    def test_secret_eof_restores_terminal_echo(self):
        code = self.base + '''
saved=$(stty -g)
set +e
(prompt_input "secret: " "unused" answer secret)
rc=$?
[ "$rc" -ne 0 ] || exit 99
[ "$(stty -g)" = "$saved" ] || exit 98
echo restored
'''
        rc, out = self.terminal(code, [("secret: ", b"\x04")])
        self.assertEqual(rc, 0, out)
        self.assertIn("restored", out)

    def test_corrupt_saved_settings_do_not_get_overwritten(self):
        target = self.dir / "install"
        (target / "data").mkdir(parents=True)
        config = target / "config.env"
        config.write_text("UI_PORT=8787\n")
        settings = target / "data/settings.json"
        settings.write_text("broken-json")
        result = self.shell('configure_install_params', check=False)
        self.assertNotEqual(result.returncode, 0)
        self.assertEqual(settings.read_text(), "broken-json")
        self.assertEqual(config.read_text(), "UI_PORT=8787\n")

    def test_no_tty_rejects_pipe_not_answers(self):
        fixture = self.dir / "notty.sh"
        fixture.write_text(self.full_fixture())
        result = subprocess.run(["bash"], input=fixture.read_bytes(), env=self.env,
                                start_new_session=True, capture_output=True, timeout=5)
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("需要交互终端", result.stderr.decode())
        self.assertFalse((self.dir / "install").exists())
        self.assertFalse((self.dir / "events").exists())

    def test_read_preserves_literal_backslash_and_spaces(self):
        code = self.base + 'prompt_input "literal: " "default" answer\nprintf "%s" "$answer" > "$TEST_DIR/answer"\n'
        rc, out = self.terminal(code, [("literal: ", b"  literal\\path  \n")])
        self.assertEqual(rc, 0, out)
        self.assertEqual((self.dir / "answer").read_text(), "  literal\\path  ")

    def test_systemd_preflight_before_any_install(self):
        result = self.shell('''
check_root() { :; }
systemctl() { return 1; }
configure_install_params() { touch "$TEST_DIR/bad"; }
install_dependencies() { touch "$TEST_DIR/bad"; }
main install
''', check=False)
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("systemd", result.stderr)
        self.assertFalse((self.dir / "bad").exists())

    def test_status_has_no_auto_repair_or_system_writes(self):
        self.shell('''
check_root() { :; }
menu_status() { echo status > "$TEST_DIR/events"; }
require_systemd() { return 99; }
register_shortcuts() { return 99; }
download_release_binary() { return 99; }
systemctl() { return 99; }
main status
''')
        self.assertEqual((self.dir / "events").read_text(), "status\n")

    def test_start_stop_restart_only_requested_service(self):
        for action in ("start", "stop", "restart"):
            self.shell('''
check_root() { :; }; require_systemd() { :; }
systemctl() { printf '%s\\n' "$*" > "$TEST_DIR/events"; }
main ''' + action)
            self.assertEqual((self.dir / "events").read_text(), f"{action} nxgate\n")

    def test_no_network_policy_commands_anywhere(self):
        # Includes update, port changes and status entry paths, not only main.
        self.assertNotIn("ensure_ssh_firewall_and_routing_safety", SOURCE)
        for token in ("ip rule add", "iptables -I", "ip6tables -I", "ufw allow", "firewall-cmd --"):
            self.assertNotIn(token, SOURCE)
        self.shell('''
ip() { return 99; }; iptables() { return 99; }; ip6tables() { return 99; }
ufw() { return 99; }; firewall-cmd() { return 99; }
configure_firewall 8787
PKG_MGR=apt-get
apt-get() { :; }
install_dependencies
''')

    def test_singbox_defaults_skip_and_preserve_existing(self):
        code = self.base + '''
singbox_available() { return 1; }
get_config_val() { echo 7928; }
do_install_singbox() { touch "$TEST_DIR/bad"; }
setup_singbox_integration
'''
        rc, out = self.terminal(code, [("是否安装可选 sing-box", b"\n")])
        self.assertEqual(rc, 0, out)
        self.assertFalse((self.dir / "bad").exists())
        self.shell('''
singbox_available() { return 0; }
get_config_val() { echo 7928; }
prompt_input() { return 99; }
do_install_singbox() { return 99; }
setup_singbox_integration
''')

    def singbox_fixture(self, fail=False):
        bin_dir = self.dir / "bin"
        bin_dir.mkdir(exist_ok=True)
        sb = bin_dir / "sing-box"
        sb.write_text('#!/bin/bash\necho "$*" >> "$TEST_DIR/events"\n' + ("exit 1\n" if fail else ""))
        sb.chmod(0o755)
        return self.base + '''
PATH="$TEST_DIR/bin:$PATH"
singbox_available() { return 1; }
get_config_val() { echo 7928; }
do_install_singbox() { echo install >> "$TEST_DIR/events"; }
setup_singbox_integration
'''

    def test_singbox_optin_creates_once(self):
        rc, out = self.terminal(self.singbox_fixture(), [("是否安装可选 sing-box", b"y\n"),
                                                      ("创建 VLESS-REALITY", b"y\n")])
        self.assertEqual(rc, 0, out)
        self.assertIn("监听所有地址", out)
        self.assertEqual((self.dir / "events").read_text().splitlines(),
                         ["install", "api add reality auto auto auto 127.0.0.1:7928"])
        body = SOURCE.split("do_install_singbox() {", 1)[1].split("create_initial_singbox_node()", 1)[0]
        self.assertNotIn("api add", body)

    def test_singbox_install_failure_stops_before_create(self):
        code = self.singbox_fixture().replace(
            'do_install_singbox() { echo install >> "$TEST_DIR/events"; }',
            'do_install_singbox() { return 1; }')
        rc, out = self.terminal(code, [("是否安装可选 sing-box", b"y\n")])
        self.assertNotEqual(rc, 0, out)
        self.assertFalse((self.dir / "events").exists())

    def test_singbox_component_only_and_failure_truthful(self):
        rc, out = self.terminal(self.singbox_fixture(), [("是否安装可选 sing-box", b"y\n"),
                                                      ("创建 VLESS-REALITY", b"\n")])
        self.assertEqual(rc, 0, out)
        self.assertEqual((self.dir / "events").read_text(), "install\n")
        (self.dir / "events").unlink()
        rc, out = self.terminal(self.singbox_fixture(True), [("是否安装可选 sing-box", b"y\n"),
                                                          ("创建 VLESS-REALITY", b"yes\n")])
        self.assertNotEqual(rc, 0, out)
        self.assertIn("创建入站失败", out)
        self.assertNotIn("入站已创建", out)
        self.assertEqual(len((self.dir / "events").read_text().splitlines()), 2)

    def go_fixture(self, corrupt=False, wrong_arch=False, fail_download=False, arm=False):
        # Tiny fake toolchain, not a downloaded executable. Exercises the real
        # metadata selection, hashing, staging and final move entirely in tempdir.
        script = b'#!/bin/bash\nif [ "$1" = env ]; then echo go1.26.10; else echo "go version go1.26.10 linux/amd64"; fi\n'
        archive = self.dir / "go.tar.gz"
        with tarfile.open(archive, "w:gz") as tf:
            info = tarfile.TarInfo("go/bin/go")
            info.mode = 0o755
            info.size = len(script)
            tf.addfile(info, io.BytesIO(script))
        digest = hashlib.sha256(archive.read_bytes()).hexdigest()
        arch = "armv6l" if arm else "amd64"
        metadata = [{"version": "go1.26.10", "files": [{"filename": f"go1.26.10.linux-{arch}.tar.gz",
                     "os": "linux", "arch": "arm64" if wrong_arch else arch, "kind": "archive",
                     "sha256": "0" * 64 if corrupt else digest}]}]
        (self.dir / "metadata.json").write_text(json.dumps(metadata))
        return '''
GO_ARCH=''' + ("arm" if arm else "amd64") + '''
curl() {
    printf '%s\\n' "$*" >> "$TEST_DIR/curls"
    local out="" url=""
    while [ "$#" -gt 0 ]; do
        case "$1" in -o) out="$2"; shift 2 ;; https:*) url="$1"; shift ;; *) shift ;; esac
    done
    case "$url" in
        'https://go.dev/dl/?mode=json&include=all') cp "$TEST_DIR/metadata.json" "$out" ;;
        https://go.dev/dl/go1.26.10.linux-*.tar.gz) ''' + ("return 22" if fail_download else 'cp "$TEST_DIR/go.tar.gz" "$out"') + ''' ;;
        *) return 99 ;;
    esac
}
tar() { echo extract >> "$TEST_DIR/extracts"; command tar "$@"; }
install_verified_go
'''

    def test_go_verified_staged_install_and_arm_filename(self):
        for arm in (False, True):
            with self.subTest(arm=arm):
                shutil.rmtree(self.dir / "install", ignore_errors=True)
                self.shell(self.go_fixture(arm=arm))
                target = self.dir / "install/toolchains/go1.26.10/bin/go"
                self.assertTrue(os.access(target, os.X_OK))
                urls = (self.dir / "curls").read_text()
                self.assertIn("--proto =https --proto-redir =https", urls)
                self.assertIn("linux-armv6l.tar.gz" if arm else "linux-amd64.tar.gz", urls)

    def test_go_checksum_mismatch_no_extraction_or_overwrite(self):
        target = self.dir / "install/toolchains/go1.26.10/bin/go"
        target.parent.mkdir(parents=True)
        target.write_text("existing sentinel")
        result = self.shell(self.go_fixture(corrupt=True), check=False)
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("SHA256 校验失败", result.stderr)
        self.assertFalse((self.dir / "extracts").exists())
        self.assertEqual(target.read_text(), "existing sentinel")

    def test_go_wrong_arch_or_download_failure_no_install(self):
        for kwargs in ({"wrong_arch": True}, {"fail_download": True}):
            with self.subTest(kwargs=kwargs):
                result = self.shell(self.go_fixture(**kwargs), check=False)
                self.assertNotEqual(result.returncode, 0)
                self.assertFalse((self.dir / "install").exists())
                self.assertFalse((self.dir / "extracts").exists())

    def test_unknown_arch_fails_instead_of_amd64(self):
        result = self.shell('uname() { echo riscv-unknown; }; detect_arch', check=False)
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("不支持的 CPU 架构", result.stderr)


if __name__ == "__main__":
    unittest.main(verbosity=2)
