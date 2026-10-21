#!/usr/bin/env python3
"""Download regressions: loopback HTTP fixtures only; no services installed."""
import hashlib
import http.server
import io
import tarfile
import os
from pathlib import Path
import shlex
import socket
import subprocess
import tempfile
import threading
import time
import unittest

ROOT = Path(__file__).resolve().parents[2]
DATA = b'fixture payload\n' * 1024
DIGEST = hashlib.sha256(DATA).hexdigest()
COUNTS = {}


class Handler(http.server.BaseHTTPRequestHandler):
    def log_message(self, *args):
        pass

    def do_GET(self):
        COUNTS[self.path] = COUNTS.get(self.path, 0) + 1
        try:
            if self.path == '/missing':
                self.send_error(404)
                return
            if self.path in ('/retry', '/always503') and (self.path == '/always503' or COUNTS[self.path] == 1):
                self.send_error(503)
                return
            payload = b'bad mirror archive' if self.path == '/bad' else DATA
            if self.path == '/slow-long':
                payload = DATA * 5
            self.send_response(200)
            self.send_header('Content-Length', str(len(payload)))
            self.end_headers()
            if self.path == '/stall':
                self.wfile.write(payload[:1]); self.wfile.flush()
                time.sleep(9)
                self.wfile.write(payload[1:])
            elif self.path == '/partial':
                self.wfile.write(payload[:3]); self.wfile.flush()
                self.close_connection = True
            elif self.path in ('/slow', '/slow-long'):
                for i in range(0, len(payload), 2048):
                    self.wfile.write(payload[i:i+2048]); self.wfile.flush()
                    time.sleep(1 if self.path == '/slow-long' else .3)
            else:
                self.wfile.write(payload)
        except (BrokenPipeError, ConnectionResetError):
            pass


class DownloadTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.server = http.server.ThreadingHTTPServer(('127.0.0.1', 0), Handler)
        cls.server.daemon_threads = True
        cls.thread = threading.Thread(target=cls.server.serve_forever, daemon=True)
        cls.thread.start()
        cls.url = f'http://127.0.0.1:{cls.server.server_port}'

    @classmethod
    def tearDownClass(cls):
        cls.server.shutdown(); cls.server.server_close(); cls.thread.join()

    def setUp(self):
        COUNTS.clear()
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.dir = Path(self.tmp.name)
        self.target = self.dir / 'archive'
        self.target.write_bytes(b'existing sentinel')

    def run_download(self, paths, digest=DIGEST, extra='', source='install.sh'):
        urls = [p if p.startswith('http') else self.url + p for p in paths]
        command = f'source {shlex.quote(str(ROOT/source))}\n' + extra
        command += '\nnxgate_download_file ' + ' '.join(shlex.quote(v) for v in [str(self.target), digest, *urls])
        return subprocess.run(['bash', '-c', command], env={**os.environ, 'TMPDIR': str(self.dir)}, capture_output=True, text=True, timeout=60)

    def assert_success(self, result):
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(self.target.read_bytes(), DATA)
        self.assertEqual(sorted(p.name for p in self.dir.iterdir()), ['archive'])

    def test_continuous_slow_transfer_keeps_source(self):
        start = time.monotonic()
        self.assert_success(self.run_download(['/slow', '/ok']))
        self.assertGreater(time.monotonic() - start, 1)
        self.assertNotIn('/ok', COUNTS)

    def test_download_longer_than_old_30_second_cap(self):
        start = time.monotonic()
        result = self.run_download(['/slow-long', '/ok'], digest=hashlib.sha256(DATA * 5).hexdigest())
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(self.target.read_bytes(), DATA * 5)
        self.assertGreater(time.monotonic() - start, 30)
        self.assertNotIn('/ok', COUNTS)

    def test_404_switches_without_retry(self):
        result = self.run_download(['/missing', '/ok'])
        self.assert_success(result)
        self.assertEqual(COUNTS['/missing'], 1)
        self.assertIn('HTTP=404', result.stderr)

    def test_503_one_same_source_retry(self):
        self.assert_success(self.run_download(['/retry', '/ok']))
        self.assertEqual(COUNTS['/retry'], 2)
        self.assertNotIn('/ok', COUNTS)

    def test_503_retry_is_bounded_then_switches(self):
        self.assert_success(self.run_download(['/always503', '/ok']))
        self.assertEqual(COUNTS['/always503'], 2)

    def test_connection_failure_switches(self):
        with socket.socket() as sock:
            sock.bind(('127.0.0.1', 0)); port = sock.getsockname()[1]
        result = self.run_download([f'http://127.0.0.1:{port}/refused', '/ok'])
        self.assert_success(result)
        self.assertIn('curl=7', result.stderr)

    def test_stall_switches_and_discards_partial(self):
        result = self.run_download(['/stall', '/ok'], extra='NXGATE_DOWNLOAD_STALL=5\n')
        self.assert_success(result)
        self.assertIn('curl=28', result.stderr)
        self.assertEqual(COUNTS['/stall'], 1)

    def test_partial_response_not_mixed_with_next_source(self):
        result = self.run_download(['/partial', '/ok'])
        self.assert_success(result)
        self.assertIn('curl=18', result.stderr)

    def test_bad_checksum_switches_without_installing(self):
        result = self.run_download(['/bad', '/ok'])
        self.assert_success(result)
        self.assertIn('SHA256 校验失败', result.stderr)

    def test_all_bad_preserves_existing_file(self):
        result = self.run_download(['/bad', '/missing'])
        self.assertNotEqual(result.returncode, 0)
        self.assertEqual(self.target.read_bytes(), b'existing sentinel')
        self.assertEqual(sorted(p.name for p in self.dir.iterdir()), ['archive'])

    def test_timeout_switch_no_same_source_retry(self):
        # Exercise curl timeout classification without waiting ten minutes.
        result = self.run_download(['/timeout', '/ok'], extra='''
real_curl=$(command -v curl)
curl() {
    case "${@: -1}" in
        */timeout) printf '200 3\\n'; return 28 ;;
        *) "$real_curl" "$@" ;;
    esac
}
''')
        self.assert_success(result)
        self.assertIn('curl=28 HTTP=200', result.stderr)

    def test_defaults_and_bounds(self):
        result = self.run_download(['/ok'], digest='', extra='''
curl() {
    printf '%s\\n' "$@" > "$TMPDIR/args"
    local out
    while [ "$#" -gt 0 ]; do
        if [ "$1" = --output ]; then out="$2"; fi
        shift
    done
    echo fixture > "$out"
    printf '200 7\\n'
}
''')
        self.assertEqual(result.returncode, 0, result.stderr)
        args = (self.dir/'args').read_text().splitlines()
        for key, value in [('--max-time', '600'), ('--connect-timeout', '15'), ('--speed-time', '60'), ('--speed-limit', '1024')]:
            self.assertEqual(args[args.index(key)+1], value)
        self.assertNotIn('--insecure', args)
        self.assertNotIn('--continue-at', args)
        for bad in ('0', '999999999999999', 'abc', '59', '1801'):
            result = self.run_download(['/ok'], extra=f'NXGATE_DOWNLOAD_TIMEOUT={shlex.quote(bad)}\n')
            self.assertNotEqual(result.returncode, 0)

    def test_embedded_script_uses_same_helper(self):
        a = (ROOT/'install.sh').read_text().split('# Kept identical', 1)[1].split('\ndo_install_singbox()', 1)[0].strip()
        b = (ROOT/'scripts/singbox/src/download.sh').read_text().split('# Kept identical', 1)[1].split('\nget_latest_version()', 1)[0].strip()
        self.assertEqual(a, b)
        self.assert_success(self.run_download(['/bad', '/ok'], source='scripts/singbox/src/download.sh'))

    def core_fixture(self, bad_digest=False, missing_member=False, manager=False):
        archive = self.dir / 'fixture.tar.gz'
        with tarfile.open(archive, 'w:gz') as tar:
            body = b'#!/bin/sh\necho fixture\n'
            member = tarfile.TarInfo('sing-box-1.14.0-linux-amd64/' + ('wrong' if missing_member else 'sing-box'))
            member.size = len(body)
            tar.addfile(member, io.BytesIO(body))
            library = tarfile.TarInfo('sing-box-1.14.0-linux-amd64/libcronet.so')
            library.size = 7
            tar.addfile(library, io.BytesIO(b'library'))
        digest = '0' * 64 if bad_digest else hashlib.sha256(archive.read_bytes()).hexdigest()
        # Run the exact installer core block, redirected to a temporary target.
        # No installer entrypoint, service, firewall, route or dependency action.
        source = ROOT / ('scripts/singbox/src/download.sh' if manager else 'install.sh')
        prelude = f'source {shlex.quote(str(source))}\n'
        prelude += f'singbox_core_sha256() {{ echo {digest}; }}\n'
        prelude += f'GO_ARCH=amd64; is_arch=amd64; is_core=sing-box; is_core_name=sing-box; is_core_repo=SagerNet/sing-box; is_core_bin={shlex.quote(str(self.target))}\n'
        prelude += f"""
curl() {{
    local output
    while [ "$#" -gt 0 ]; do
        if [ "$1" = --output ]; then output="$2"; fi
        shift
    done
    cp {shlex.quote(str(archive))} "$output"
    printf '200 100\\n'
}}
err() {{ echo "$*" >&2; return 1; }}
"""
        if manager:
            code = prelude + 'download core v1.14.0'
        else:
            block = (ROOT/'install.sh').read_text().split('    # 3. 部署 sing-box 官方核心二进制', 1)[1].split('    # 4. 确保基础配置', 1)[0].split('\n', 1)[1]
            block = block.replace('"/etc/sing-box/bin/sing-box"', shlex.quote(str(self.target)))
            code = prelude + 'test_core() {\n' + block + '\n}\ntest_core'
        return subprocess.run(['bash', '-c', code], env={**os.environ, 'TMPDIR': str(self.dir)}, capture_output=True, text=True, timeout=10)

    def test_verified_core_installs_only_expected_member(self):
        for manager in (False, True):
            with self.subTest(manager=manager):
                self.target.write_bytes(b'existing sentinel'); self.target.chmod(0o600)
                result = self.core_fixture(manager=manager)
                self.assertEqual(result.returncode, 0, result.stderr)
                self.assertEqual(self.target.read_bytes(), b'#!/bin/sh\necho fixture\n')
                self.assertTrue(os.access(self.target, os.X_OK))
                if manager:
                    self.assertEqual((self.dir/'libcronet.so').read_bytes(), b'library')

    def test_core_checksum_failure_never_installs(self):
        for manager in (False, True):
            with self.subTest(manager=manager):
                result = self.core_fixture(bad_digest=True, manager=manager)
                self.assertNotEqual(result.returncode, 0)
                self.assertEqual(self.target.read_bytes(), b'existing sentinel')
                self.assertIn('SHA256 校验失败', result.stderr)

    def test_core_missing_member_never_overwrites(self):
        for manager in (False, True):
            with self.subTest(manager=manager):
                result = self.core_fixture(missing_member=True, manager=manager)
                self.assertNotEqual(result.returncode, 0)
                self.assertEqual(self.target.read_bytes(), b'existing sentinel')

    def test_all_architecture_digests_are_pinned(self):
        result = subprocess.run(['bash', '-c', f'source {shlex.quote(str(ROOT/"install.sh"))}; for a in amd64 arm64 386 armv7; do singbox_core_sha256 v1.14.0 "$a"; done; ! singbox_core_sha256 v1.14.0 unknown'], capture_output=True, text=True)
        self.assertEqual(result.returncode, 0)
        digests = result.stdout.splitlines()
        self.assertEqual(len(digests), 4)
        for digest in digests:
            self.assertRegex(digest, r'^[a-f0-9]{64}$')


if __name__ == '__main__':
    unittest.main(verbosity=2)
