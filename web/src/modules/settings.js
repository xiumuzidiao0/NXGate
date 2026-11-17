import { appState } from '../state.js';
import { getLanguage, localize, t, tAlert, tConfirm } from '../i18n-runtime.js';
import { editDynamicGroup, fetchDynamicGroups, fetchPortRules, switchMatrixTab } from './matrix.js';
import { switchView } from './shell.js';
import { fetchStatus } from '../transport.js';
import { alert, escapeHtml, showToast } from '../utils.js';

export async function testTelegramAlert() {
    try {
        const res = await fetch('/api/telegram/test', { method: 'POST' });
        const ret = await res.json();
        if (res.ok) tAlert(ret.message || '测试消息发送成功！', ret.message || 'Test alert sent successfully!');
        else tAlert('测试失败: ' + (ret.error || '未知错误'), 'Test alert failed: ' + (ret.error || 'Unknown error'));
    } catch(err) {
        alert(localize('请求失败: ', 'Request failed: ') + err);
    }
}

export function switchSettingsTab(tabKey) {
    document.querySelectorAll('#settings-modal .modal-tab-btn').forEach(b => b.classList.remove('active'));
    ['base', 'tg', 'app', 'update', 'backup'].forEach(k => {
        const el = document.getElementById('tab-content-' + k);
        if (el) {
            const isHidden = (k !== tabKey);
            el.classList.toggle('hidden', isHidden);
            el.hidden = isHidden;
        }
    });
    const activeBtn = document.getElementById('tab-btn-' + tabKey);
    if (activeBtn) activeBtn.classList.add('active');
    if (tabKey === 'update') {
        checkForUpdates(true);
    }
}

export function onDisableAuthChanged() {
    const disableChk = document.getElementById('cfg-disable-auth');
    const userInput = document.getElementById('cfg-username');
    const passInput = document.getElementById('cfg-password');
    const randBtn = document.querySelector('[data-action="randomPassword"]');
    const disabled = !!disableChk?.checked;
    if (userInput) userInput.disabled = disabled;
    if (passInput) passInput.disabled = disabled;
    if (randBtn) randBtn.disabled = disabled;
}

export async function loadSettingsForm() {
    try {
        const res = await fetch('/api/settings');
        if (!res.ok) return;
        const data = await res.json();
        document.getElementById('cfg-web-port').value = data.ui_port || 8787;
        document.getElementById('cfg-web-path').value = data.ui_path ?? '';
        const userInput = document.getElementById('cfg-username');
        userInput.value = data.ui_username ?? '';
        userInput.dataset.prevUser = data.ui_username ?? '';
        document.getElementById('cfg-password').value = '';
        const disableChk = document.getElementById('cfg-disable-auth');
        if (disableChk) {
            disableChk.checked = !data.ui_username && appState.currentState?.ui_auth_enabled !== true;
        }
        onDisableAuthChanged();
        document.getElementById('cfg-proxy-port').value = data.proxy_port || 7928;
 document.getElementById('cfg-proxy-global-max').value = data.proxy_global_max_connections ?? 1024;
        document.getElementById('cfg-tg-token').value = data.telegram_bot_token || '';
        document.getElementById('cfg-tg-chatid').value = data.telegram_chat_id || '';
        const agePubEl = document.getElementById('cfg-age-pubkey');
        if (agePubEl) agePubEl.value = data.age_public_key || '';
        switchSettingsTab('base');
    } catch (err) {
        console.warn('获取系统配置失败:', err);
    }
}

export function closeSettingsModal() { switchView('dashboard'); }

export function openSystemPrimaryConfig() {
    switchView('matrix');
    switchMatrixTab('groups');
    editDynamicGroup('system-primary');
}

export function randomPath() {
    const chars = 'abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789';
    let res = '';
    for (let i = 0; i < 8; i++) res += chars.charAt(Math.floor(Math.random() * chars.length));
    document.getElementById('cfg-web-path').value = res;
}

export function randomPassword() {
    const chars = 'abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*';
    let res = '';
    for (let i = 0; i < 14; i++) res += chars.charAt(Math.floor(Math.random() * chars.length));
    document.getElementById('cfg-password').value = res;
}

export async function saveSettings(e) {
    e.preventDefault();
    const webPort = parseInt(document.getElementById('cfg-web-port').value);
    const webPath = document.getElementById('cfg-web-path').value.trim().replace(/^\/+|\/+$/g, '');
    const disableAuth = !!document.getElementById('cfg-disable-auth')?.checked;
    const userInput = document.getElementById('cfg-username');
    const prevUser = userInput?.dataset?.prevUser || '';
    const user = userInput.value.trim();
    const pass = document.getElementById('cfg-password').value.trim();
    const proxyPort = parseInt(document.getElementById('cfg-proxy-port').value);

    if (webPort === proxyPort) {
        tAlert('错误: Web 管理端口不能与本地代理端口相同！', 'Error: Web console port cannot be identical to proxy port!');
        return;
    }

    if (!disableAuth) {
        if (!user) {
            tAlert('请输入 Web 管理员账号，或勾选关闭认证！', 'Please enter an admin username or enable Disable Authentication.');
            return;
        }
        if (!pass && !prevUser) {
            tAlert('开启控制台认证时必须同时填写管理员密码！', 'A password is required when enabling console authentication.');
            return;
        }
    }

    const tgToken = document.getElementById('cfg-tg-token').value.trim();
    const tgChatID = document.getElementById('cfg-tg-chatid').value.trim();
    const agePub = (document.getElementById('cfg-age-pubkey')?.value || '').trim();

    const payload = {
        ui_port: webPort,
        ui_path: webPath,
        disable_ui_auth: disableAuth,
        ui_username: disableAuth ? '' : user,
        proxy_port: proxyPort,
        proxy_global_max_connections: Number(document.getElementById('cfg-proxy-global-max').value),
        telegram_bot_token: tgToken,
        telegram_chat_id: tgChatID,
        age_public_key: agePub,
        age_encrypt_enabled: !!agePub
    };
    if (disableAuth) {
        payload.ui_password = '';
    } else if (pass) {
        payload.ui_password = pass;
    }

    try {
        const res = await fetch('/api/settings', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(payload)
        });
        const ret = await res.json();
        if (!res.ok) {
            tAlert('保存失败: ' + (ret.error || '未知错误'), 'Save failed: ' + (ret.error || 'Unknown error'));
            return;
        }

        closeSettingsModal();

        const curPort = window.location.port || (window.location.protocol === 'https:' ? '443' : '80');
        const curPath = window.location.pathname.replace(/^\/+|\/+$/g, '');
        const portSuffix = window.location.port ? `:${window.location.port}` : '';
        if (ret.restart_required) {
            tAlert(
                `端口配置已保存，新端口将在重启 NXGate 服务后生效（当前仍运行在端口 ${curPort}）。`,
                `Port settings saved. Restart NXGate for the new ports to take effect (currently listening on ${curPort}).`
            );
            if (curPath !== webPath) {
                window.location.href = `${window.location.protocol}//${window.location.hostname}${portSuffix}/${webPath}`;
                return;
            }
            fetchStatus();
            return;
        }

        tAlert('配置修改成功并已生效！', 'Configuration updated and applied successfully!');
        if (curPath !== webPath) {
            const newUrl = `${window.location.protocol}//${window.location.hostname}${portSuffix}/${webPath}`;
            tAlert(`Web 访问入口已变更，即将跳转至新地址:\n${newUrl}`, `Web console URL changed. Redirecting to new address:\n${newUrl}`);
            window.location.href = newUrl;
        } else {
            fetchStatus();
        }
    } catch (err) {
        alert(localize('网络请求失败: ', 'Network request failed: ') + err);
    }
}

// Setup SSE connection

export function downloadBackupPackage() {
    const a = document.createElement('a');
    a.href = '/api/system/backup/export';
    a.download = `nxgate-backup-${new Date().toISOString().slice(0,10)}.json`;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    showToast(getLanguage() === 'en' ? 'Downloaded configuration backup' : '已触发下载全量备份文件 nxgate-backup.json');
}

export function triggerBackupFileSelect() {
    document.getElementById('backup-file-input')?.click();
}

export async function onBackupFileSelected(event) {
    const file = event.target?.files?.[0];
    if (!file) return;

    try {
        const text = await file.text();
        const pkg = JSON.parse(text);
        const isConfirmed = await tConfirm(
            `确认从备份文件 [${file.name}] 导入恢复配置吗？\n将恢复 ${pkg.favorites?.length || 0} 个收藏、${pkg.blacklist?.length || 0} 条黑名单、${pkg.dynamic_groups?.length || 0} 个出口组及端口分流规则。`,
            `Restore configuration from [${file.name}]?\nThis will restore ${pkg.favorites?.length || 0} favorites, ${pkg.blacklist?.length || 0} blacklist rules, and ${pkg.dynamic_groups?.length || 0} dynamic groups.`
        );
        if (!isConfirmed) {
            event.target.value = '';
            return;
        }

        const res = await fetch('/api/system/backup/import', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: text
        });
        const ret = await res.json();
        if (res.ok) {
            tAlert(`配置恢复成功！已恢复 ${ret.restored_items} 项系统配置规则并立即生效。`, `Configuration successfully restored! ${ret.restored_items} rule sets restored and applied.`);
            await Promise.all([fetchStatus(), fetchPortRules(), fetchDynamicGroups(), loadSettingsForm()]);
        } else {
            tAlert('导入配置失败: ' + (ret.error || '解析错误'), 'Import failed: ' + (ret.error || 'Parsing error'));
        }
    } catch (err) {
        tAlert('无效的 JSON 备份文件: ' + err.message, 'Invalid JSON backup file: ' + err.message);
    }
    event.target.value = '';
}

export async function openMobileAppModal() {
    const host = window.location.hostname || '127.0.0.1';
    const port = window.location.port || (window.location.protocol === 'https:' ? '443' : '80');
    const proto = window.location.protocol === 'https:' ? 'https' : 'http';

    document.getElementById('app-server-name').value = `NXGate (${host})`;
    document.getElementById('app-server-protocol').value = proto;
    document.getElementById('app-server-host').value = host;
    document.getElementById('app-server-port').value = port;
    const testEl = document.getElementById('app-api-test-result');
    if (testEl) {
        testEl.classList.add('hidden');
        testEl.innerHTML = '';
    }

    try {
        const res = await fetch('/api/app/profile');
        if (res.ok) {
            const data = await res.json();
            if (data && data.profile) {
                appState.currentAppProfile = data.profile;
            }
        }
    } catch (e) {
        console.warn('获取服务器导入配置失败:', e);
    }

    renderAppProfileQRCode();
    document.getElementById('mobile-app-modal').classList.add('open');
}

export function closeMobileAppModal() {
    document.getElementById('mobile-app-modal').classList.remove('open');
}

export function renderAppProfileQRCode() {
    const name = (document.getElementById('app-server-name')?.value || '').trim() || 'NXGate';
    const proto = document.getElementById('app-server-protocol')?.value || 'http';
    const host = (document.getElementById('app-server-host')?.value || '').trim() || window.location.hostname || '127.0.0.1';
    const port = parseInt(document.getElementById('app-server-port')?.value) || (window.location.port ? parseInt(window.location.port) : 8787);
    const path = appState.currentAppProfile ? (appState.currentAppProfile.path ?? '') : ((appState.currentState && appState.currentState.admin_path) ?? '');
    const user = appState.currentAppProfile ? (appState.currentAppProfile.username ?? '') : '';
    const pass = (appState.currentAppProfile && appState.currentAppProfile.password) || '';
    const isTls = proto === 'https';

    const uri = `nxgate://server?host=${encodeURIComponent(host)}&port=${port}&path=${encodeURIComponent(path)}&user=${encodeURIComponent(user)}&pass=${encodeURIComponent(pass)}&name=${encodeURIComponent(name)}&tls=${isTls ? '1' : '0'}`;

    const jsonProfile = {
        type: 'nxgate_server',
        version: 1,
        name: name,
        host: host,
        port: port,
        path: path,
        username: user,
        password: pass,
        proxy_port: (appState.currentAppProfile && appState.currentAppProfile.proxy_port) || 7928,
        tls: isTls
    };

    const uriEl = document.getElementById('app-connect-uri');
    if (uriEl) uriEl.value = uri;
    const jsonEl = document.getElementById('app-connect-json');
    if (jsonEl) jsonEl.value = JSON.stringify(jsonProfile, null, 2);

    const qrImg = document.getElementById('app-qr-img');
    if (qrImg) {
        qrImg.src = `https://api.qrserver.com/v1/create-qr-code/?size=220x220&data=${encodeURIComponent(uri)}`;
    }
}

export function onAppProfileFieldChanged() {
    renderAppProfileQRCode();
}

export async function testMobileApiConnectivity() {
    const testEl = document.getElementById('app-api-test-result');
    if (!testEl) return;
    testEl.classList.remove('hidden');
    testEl.innerHTML = `<span class="text-muted">${localize('正在测试连接', 'Testing')} /api/app/info ...</span>`;
    const start = Date.now();
    try {
        const res = await fetch('/api/app/info');
        const elapsed = Date.now() - start;
        if (res.ok) {
            const data = await res.json();
            testEl.innerHTML = `<span class="text-success">${localize('API 连通正常！', 'API reachable.')} (${elapsed} ms, ${localize('版本', 'version')}: v${escapeHtml(data.version || '')}, ${localize('状态', 'status')}: ${escapeHtml(data.status || localize('就绪', 'Ready'))})</span>`;
        } else {
            testEl.innerHTML = `<span class="text-danger">${localize('API 测试返回异常', 'API test failed')} (HTTP ${res.status})</span>`;
        }
    } catch (err) {
        testEl.innerHTML = `<span class="text-danger">${localize('无法连接到 API', 'Cannot reach API')}: ${escapeHtml(err.message)}</span>`;
    }
}

export function openUpdateModal() {
    const modal = document.getElementById('update-modal');
    if (modal) {
        modal.classList.add('open');
        modal.hidden = false;
    }
    checkForUpdates(true);
}

export function closeUpdateModal() {
    const modal = document.getElementById('update-modal');
    if (modal) {
        modal.classList.remove('open');
        modal.hidden = true;
    }
}

export async function checkForUpdates(silent = false) {
    const curVerElements = [
        document.getElementById('update-cur-ver'),
        document.getElementById('modal-update-cur-ver')
    ];
    const latestVerElements = [
        document.getElementById('update-latest-ver'),
        document.getElementById('modal-update-latest-ver')
    ];
    const statusTags = [
        document.getElementById('update-status-tag'),
        document.getElementById('modal-update-status-tag')
    ];
    const updateBtns = [
        document.getElementById('btn-do-update'),
        document.getElementById('modal-btn-do-update')
    ];
    const releaseBoxes = [
        document.getElementById('update-info-box'),
        document.getElementById('modal-update-info-box')
    ];

    statusTags.forEach(t => {
        if (t) {
            t.className = 'badge badge-proto';
            t.textContent = localize('检查中...', 'Checking...');
        }
    });
    latestVerElements.forEach(el => {
        if (el) el.textContent = localize('查询中...', 'Checking...');
    });

    try {
        const res = await fetch('/api/update/check');
        if (!res.ok) throw new Error(`HTTP ${res.status}`);
        const data = await res.json();

        curVerElements.forEach(el => {
            if (el) el.textContent = 'v' + String(data.current_version || '2.5.6').replace(/^v/, '');
        });

        if (!data.ok) {
            statusTags.forEach(t => {
                if (t) {
                    t.className = 'badge badge-danger';
                    t.textContent = localize('检查失败', 'Check failed');
                }
            });
            latestVerElements.forEach(el => {
                if (el) el.textContent = localize('无法获取', 'Unavailable');
            });
            if (!silent) showToast(data.error || localize('检查更新失败', 'Update check failed'), 'error');
            return;
        }

        const latestVer = 'v' + String(data.latest_version || '').replace(/^v/, '');
        latestVerElements.forEach(el => {
            if (el) el.textContent = latestVer;
        });

        if (data.release_notes || data.release_name) {
            const title = data.release_name || localize(`${latestVer} 发行说明`, `${latestVer} release notes`);
            const dateStr = data.published_at ? new Date(data.published_at).toLocaleDateString(getLanguage() === 'en' ? 'en-US' : 'zh-CN') : '';
            const notesFormatted = escapeHtml(data.release_notes || localize('已检测到远端主干最新稳定版本。', 'The latest stable release is available.')).replace(/\n/g, '<br>');

            const elTitle = document.getElementById('update-release-title');
            if (elTitle) elTitle.textContent = title;
            const elMTitle = document.getElementById('modal-update-release-title');
            if (elMTitle) elMTitle.textContent = title;

            const elDate = document.getElementById('update-release-date');
            if (elDate) elDate.textContent = dateStr;
            const elMDate = document.getElementById('modal-update-release-date');
            if (elMDate) elMDate.textContent = dateStr;

            const elNotes = document.getElementById('update-release-notes');
            if (elNotes) elNotes.innerHTML = notesFormatted;
            const elMNotes = document.getElementById('modal-update-release-notes');
            if (elMNotes) elMNotes.innerHTML = notesFormatted;

            releaseBoxes.forEach(box => {
                if (box) box.classList.remove('hidden');
            });
        }

        if (data.has_update) {
            statusTags.forEach(t => {
                if (t) {
                    t.className = 'badge badge-success';
                    t.textContent = localize('可更新', 'Update available');
                }
            });
            updateBtns.forEach(b => {
                if (b) {
                    b.classList.remove('hidden');
                    b.hidden = false;
                }
            });
            const badgeNew = document.getElementById('settings-update-badge');
            if (badgeNew) badgeNew.classList.remove('hidden');
            const dotNew = document.getElementById('header-update-dot');
            if (dotNew) dotNew.classList.remove('hidden');

            if (!silent) showToast(localize(`发现新版本 ${latestVer}，可点击立即升级！`, `Version ${latestVer} is available.`), 'info');
        } else {
            statusTags.forEach(t => {
                if (t) {
                    t.className = 'badge badge-proto';
                    t.textContent = localize('已是最新', 'Up to date');
                }
            });
            updateBtns.forEach(b => {
                if (b) {
                    b.classList.add('hidden');
                    b.hidden = true;
                }
            });
            const badgeNew = document.getElementById('settings-update-badge');
            if (badgeNew) badgeNew.classList.add('hidden');
            const dotNew = document.getElementById('header-update-dot');
            if (dotNew) dotNew.classList.add('hidden');

            if (!silent) showToast(localize('当前已是最新稳定版本 (v', 'Already on the latest stable version (v') + String(data.current_version || '').replace(/^v/, '') + ')', 'success');
        }
    } catch (err) {
        console.error('检查更新失败:', err);
        statusTags.forEach(t => {
            if (t) {
                t.className = 'badge badge-danger';
                t.textContent = localize('网络异常', 'Network error');
            }
        });
        if (!silent) showToast(localize('检查更新失败，请检查网络连接', 'Update check failed. Check your connection.'), 'error');
    }
}

export async function triggerSystemUpdate(force = false) {
    if (appState.isUpdating) return;
    const actionDesc = force ? localize('强制重新安装当前最新构建', 'reinstall the latest build') : localize('升级至最新版本', 'upgrade to the latest version');
    if (!(await tConfirm(`确认立即执行系统更新（${actionDesc}）并重启服务吗？`, `Proceed with system update (${actionDesc}) and restart service?`))) return;

    appState.isUpdating = true;
    const progressBoxes = [
        document.getElementById('update-progress-box'),
        document.getElementById('modal-update-progress-box')
    ];
    const progressSteps = [
        document.getElementById('update-progress-step'),
        document.getElementById('modal-update-progress-step')
    ];
    const updateBtns = [
        document.getElementById('btn-do-update'),
        document.getElementById('modal-btn-do-update'),
        document.getElementById('btn-check-update'),
        document.getElementById('btn-force-update')
    ];

    progressBoxes.forEach(b => b && b.classList.remove('hidden'));
    progressSteps.forEach(s => s && (s.textContent = localize('正在下载官方二进制包并比对 SHA-256 校验和...', 'Downloading the official binary and verifying its SHA-256 checksum...')));
    updateBtns.forEach(b => b && (b.disabled = true));

    try {
        const res = await fetch('/api/update/trigger', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ force: !!force })
        });
        const ret = await res.json();

        if (!res.ok || !ret.ok) {
            appState.isUpdating = false;
            updateBtns.forEach(b => b && (b.disabled = false));
            progressSteps.forEach(s => s && (s.textContent = `${ret.error || localize('更新失败', 'Update failed')}`));
            tAlert('系统更新失败: ' + (ret.error || '未知错误'), 'System update failed: ' + (ret.error || 'Unknown error'));
            return;
        }

        // Success! Service is restarting
        let countdown = 5;
        const updateMsg = escapeHtml(ret.message || localize('更新完成！', 'Update complete!'));
        progressSteps.forEach(s => s && (s.innerHTML = `<strong>${updateMsg}</strong> ${localize("页面将在", "Reloading in")} <strong>${countdown}</strong> ${localize("秒后自动刷新...", "seconds...")}`));

        const timer = setInterval(() => {
            countdown--;
            if (countdown <= 0) {
                clearInterval(timer);
                window.location.reload();
            } else {
                progressSteps.forEach(s => s && (s.innerHTML = `<strong>${updateMsg}</strong> ${localize("页面将在", "Reloading in")} <strong>${countdown}</strong> ${localize("秒后自动刷新...", "seconds...")}`));
            }
        }, 1000);

    } catch (err) {
        appState.isUpdating = false;
        updateBtns.forEach(b => b && (b.disabled = false));
        progressSteps.forEach(s => s && (s.textContent = localize(`网络异常: ${err}`, `Network error: ${err}`)));
        alert(localize('请求更新失败: ', 'Update request failed: ') + err);
    }
}

export function triggerForceUpdate() {
    triggerSystemUpdate(true);
}
