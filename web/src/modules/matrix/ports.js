import { appState } from '../../state.js';
import { getLanguage, localize, tAlert, tConfirm } from '../../i18n-runtime.js';
import { closeEditorDrawer, openEditorDrawer } from '../shell.js';
import { fetchStatus } from '../../transport.js';
import { alert, escapeHtml, getCountryFlagSVG, getCountryName, jsonAttr } from '../../utils.js';
import { renderMatrixTopology } from './topology.js';

export async function fetchPortRules() {
    try {
        const res = await fetch('/api/proxy/ports');
        if (!res.ok) return;
        appState.currentPortRules = await res.json();
        renderPortRules();
        renderMatrixTopology();
    } catch (err) {
        console.error("fetch port rules error:", err);
    }
}

export function renderPortRules() {
    const container = document.getElementById('port-rules-cards');
    if (!container) return;
    const isEn = getLanguage() === 'en';
    if (!appState.currentPortRules || appState.currentPortRules.length === 0) {
        container.innerHTML = `<div class="list-empty">${isEn ? 'No port routing rules. Click "+ Add Port Rule" above to specify port exits.' : '暂未配置自定义端口，点击上方「+ 新增代理端口」添加'}</div>`;
        return;
    }

    const tunnels = (appState.currentState && appState.currentState.tunnels) ? appState.currentState.tunnels : [];

    container.innerHTML = appState.currentPortRules.map(r => {
        let policyBadge = `<span class="badge policy-round"><svg aria-hidden="true" viewBox="0 0 24 24" class="icon-xs icon-stroke"><polyline points="23 4 23 10 17 10"/><path d="M20.49 15a9 9 0 1 1-2.12-9.36L23 10"/></svg> ${isEn ? 'Round-Robin' : '单连接轮询'}</span>`;
        if (r.policy === 'random') {
            policyBadge = `<span class="badge policy-random"><svg aria-hidden="true" viewBox="0 0 24 24" class="icon-xs icon-stroke"><polygon points="13 2 3 14 12 14 11 22 21 10 12 10 13 2"/></svg> ${isEn ? 'Random' : '随机分发'}</span>`;
        } else if (r.policy === 'interval') {
            policyBadge = `<span class="badge policy-interval"><svg aria-hidden="true" viewBox="0 0 24 24" class="icon-xs icon-stroke"><circle cx="12" cy="12" r="10"/><polyline points="12 6 12 12 16 14"/></svg> ${isEn ? 'Timed Rotation' : '定时轮换'} (${r.interval_seconds || 300}s)</span>`;
        }

        let authBadge = `<span class="auth-note">${isEn ? 'Random Credentials' : '系统随机账密'}</span>`;
        if (r.auth_mode === 'none') {
            authBadge = `<span class="auth-note auth-open">${isEn ? 'No Authentication' : '免密直连'}</span>`;
        } else if (r.auth_mode === 'custom') {
            authBadge = `<span class="auth-note text-accent">${isEn ? 'Custom: ' : '独立账号: '}${escapeHtml(r.auth_user || (isEn ? 'Not Set' : '未设'))}</span>`;
        }

        let boundBadges = [];
        if (r.bound_group_ids && r.bound_group_ids.length > 0) {
            r.bound_group_ids.forEach(gid => {
                const g = appState.currentDynamicGroups.find(item => item.id === gid);
                const gName = g ? g.name : gid;
                boundBadges.push(`<span class="badge badge-system">${isEn ? 'Group: ' : '动态池: '}${escapeHtml(gName)}</span>`);
            });
        }
        if (r.bound_tunnel_ids && r.bound_tunnel_ids.length > 0) {
            r.bound_tunnel_ids.forEach(id => {
                const t = tunnels.find(item => item.id === id);
                if (t && t.node) {
                    const flag = getCountryFlagSVG(t.node.country_short);
                    boundBadges.push(`<span class="badge badge-proto text-xs" >${flag} ${escapeHtml(t.dev_name)} (${escapeHtml(t.node.ip)})</span>`);
                } else {
                    boundBadges.push(`<span class="badge badge-proto text-xs" >${escapeHtml(id)}</span>`);
                }
            });
        }

        let boundHtml = boundBadges.join(' ');
        if (boundBadges.length === 0) {
            boundHtml = `<span class="auth-note text-accent">${isEn ? 'All Online Tunnels (Dynamic Load Balancing)' : '全部在线隧道 (动态负载均衡)'}</span>`;
        }

        const httpUrl = `http://127.0.0.1:${r.port}`;
        const socksUrl = `socks5://127.0.0.1:${r.port}`;

        return `
            <div class="port-card">
                <div class="port-card-top">
                    <div class="port-card-badge">
                        <span class="status-dot status-dot-success"></span>
                        PORT ${r.port}
                        <span class="badge badge-proto badge-mini">HTTP / SOCKS5</span>
                    </div>
                    <div  class="row items-center gap-2 wrap">
                        <span class="copy-pill" data-action="copyText" data-args="${jsonAttr([httpUrl, isEn ? 'HTTP Proxy URL' : 'HTTP 代理地址'])}">
                            <svg aria-hidden="true"  viewBox="0 0 24 24" class="icon-xs icon-stroke"><rect x="9" y="9" width="13" height="13" rx="2" ry="2"/><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/></svg>
                            http://127.0.0.1:${r.port}
                        </span>
                        <span class="copy-pill" data-action="copyText" data-args="${jsonAttr([socksUrl, isEn ? 'SOCKS5 Proxy URL' : 'SOCKS5 代理地址'])}">
                            <svg aria-hidden="true"  viewBox="0 0 24 24" class="icon-xs icon-stroke"><rect x="9" y="9" width="13" height="13" rx="2" ry="2"/><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/></svg>
                            socks5://127.0.0.1:${r.port}
                        </span>
                        <button class="btn btn-outline btn-xs" data-action="editPortRule" data-args="${jsonAttr([r.port])}">${isEn ? 'Edit' : '编辑'}</button>
                        <button class="btn btn-danger btn-xs" data-action="deletePortRule" data-args="${jsonAttr([r.port])}">${isEn ? 'Delete' : '删除'}</button>
                    </div>
                </div>
                <div class="port-card-body">
                    <div class="port-card-col">
                        <span class="port-card-label">${isEn ? 'Routing Policy' : '分流调度策略'}</span>
                        <div class="port-card-val">${policyBadge}</div>
                    </div>
                    <div class="port-card-col">
                        <span class="port-card-label">${isEn ? 'Bound Egress / Exit Groups' : '绑定出口 / 动态出口组'}</span>
                        <div class="port-card-val">${boundHtml}</div>
                    </div>
                    <div class="port-card-col">
                        <span class="port-card-label">${isEn ? 'Authentication' : '代理鉴权状态'}</span>
                        <div class="port-card-val">${authBadge}</div>
                    </div>
                </div>
            </div>
        `;
    }).join('');
}

export function showAddPortForm() {
    appState.editingPort = null;
    document.getElementById('port-edit-title').textContent = localize(`新增代理监听端口与分流绑定`, 'Add proxy port and egress binding');
    document.getElementById('rule-port').value = '';
    document.getElementById('rule-port').disabled = false;
    selectPolicy('round_robin');
    document.getElementById('rule-interval').value = '300';
    selectAuthMode('random');
    document.getElementById('rule-auth-user').value = '';
    document.getElementById('rule-auth-pass').value = '';
    renderTunnelCheckboxes([], []);
    suggestNextPort();
    openEditorDrawer('port-edit-card');
}

export function editPortRule(port) {
    const rule = appState.currentPortRules.find(r => r.port === port);
    if (!rule) return;
    appState.editingPort = port;
    document.getElementById('port-edit-title').textContent = localize(`编辑端口 [${port}] 绑定与调度规则`, `Edit port [${port}] bindings and routing`);
    document.getElementById('rule-port').value = rule.port;
    document.getElementById('rule-port').disabled = true;
    selectPolicy(rule.policy || 'round_robin');
    document.getElementById('rule-interval').value = rule.interval_seconds || 300;
    let aMode = rule.auth_mode || 'random';
    if (aMode === 'default_web') aMode = 'random';
    selectAuthMode(aMode);
    document.getElementById('rule-auth-user').value = rule.auth_user || '';
    document.getElementById('rule-auth-pass').value = rule.auth_pass || '';
    renderTunnelCheckboxes(rule.bound_tunnel_ids || [], rule.bound_group_ids || []);
    openEditorDrawer('port-edit-card');
}

export function hideEditPortForm() { closeEditorDrawer('port-edit-card'); }

export function selectPolicy(pol) {
    document.getElementById('rule-policy').value = pol;
    ['round_robin', 'random', 'interval'].forEach(p => {
        const el = document.getElementById('card-policy-' + p);
        if (el) {
            if (p === pol) el.classList.add('selected');
            else el.classList.remove('selected');
        }
    });
    document.getElementById('rule-interval-group').classList.toggle('hidden', pol !== 'interval');
}

export function selectAuthMode(mode) {
    if (mode === 'default_web') mode = 'random';
    document.getElementById('rule-auth-mode').value = mode;
    ['random', 'default_web', 'none', 'custom'].forEach(m => {
        const el = document.getElementById('card-auth-' + m);
        if (el) {
            if (m === mode || (mode === 'random' && m === 'default_web')) el.classList.add('selected');
            else el.classList.remove('selected');
        }
    });
    document.getElementById('rule-custom-auth-group').classList.toggle('hidden', mode !== 'custom');
}

export function generateRandomPortAuth() {
    const chars = 'abcdefghijklmnopqrstuvwxyz0123456789';
    let user = 'u_';
    for (let i = 0; i < 6; i++) user += chars[Math.floor(Math.random() * chars.length)];
    let pass = '';
    const passChars = 'abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789';
    for (let i = 0; i < 16; i++) pass += passChars[Math.floor(Math.random() * passChars.length)];
    document.getElementById('rule-auth-user').value = user;
    document.getElementById('rule-auth-pass').value = pass;
}

export function suggestNextPort() {
    let max = 7927;
    if (appState.currentPortRules && appState.currentPortRules.length > 0) {
        appState.currentPortRules.forEach(r => { if (r.port > max) max = r.port; });
    }
    document.getElementById('rule-port').value = max + 1;
}

export function setPortVal(val) { document.getElementById('rule-port').value = val; }
export function setIntervalVal(sec) { document.getElementById('rule-interval').value = sec; }

export function renderTunnelCheckboxes(selectedTunnelIds, selectedGroupIds) {
    const container = document.getElementById('rule-tunnels-checkboxes');
    const tunnels = (appState.currentState && appState.currentState.tunnels) ? appState.currentState.tunnels : [];
    selectedTunnelIds = selectedTunnelIds || [];
    selectedGroupIds = selectedGroupIds || [];
    const selTunMap = {};
    selectedTunnelIds.forEach(id => selTunMap[id] = true);
    const selGrpMap = {};
    selectedGroupIds.forEach(id => selGrpMap[id] = true);

    const isAllChecked = (selectedTunnelIds.length === 0 && selectedGroupIds.length === 0);

    let html = `
        <label class="check-row check-row-all">
            <input type="checkbox" id="chk-tunnel-all" ${isAllChecked ? 'checked' : ''} data-change-action="onAllTunnelsCheckChanged">
            <span><strong>${localize('全部在线隧道 (默认)', 'All healthy tunnels (default)')}</strong> - ${localize('自动根据当前所有运行的隧道动态负载均衡', 'Load balance across all eligible healthy tunnels')}</span>
        </label>
    `;

    if (appState.currentDynamicGroups && appState.currentDynamicGroups.length > 0) {
        html += `<div class="check-section-label text-accent">${localize('动态出口组 (按规则自动维持并定期轮换):', 'Dynamic egress groups (maintained and rotated automatically):')}</div>`;
        appState.currentDynamicGroups.forEach(g => {
            const isChecked = !isAllChecked && selGrpMap[g.id];
            let metricText = g.sort_by === 'speed' ? localize('最大带宽', 'Highest bandwidth') : (g.sort_by === 'score' ? localize('最高评分', 'Highest score') : localize('最低延迟', 'Lowest latency'));
            html += `
                <label  class="check-row">
                    <input type="checkbox" class="chk-dynamic-group" value="${escapeHtml(g.id)}" ${isChecked ? 'checked' : ''} data-change-action="onSpecificTunnelCheckChanged">
                    <span class="badge badge-system">${localize('出口组', 'Egress group')}</span>
                    <strong class="check-name">${escapeHtml(g.name)}</strong>
                    <span  class="text-xs text-muted">(${escapeHtml(g.country || localize('全部', 'All'))} · ${escapeHtml(g.ip_type === 'residential' ? localize('家宽', 'Residential') : (g.ip_type === 'hosting' ? localize('机房', 'Datacenter') : localize('不限', 'Any')))} · ${escapeHtml(metricText)} Top${escapeHtml(g.target_count)})</span>
                </label>
            `;
        });
    }

    if (tunnels.length > 0) {
        html += `<div class="check-section-label text-muted">${localize('固定独立在线隧道:', 'Specific active tunnels:')}</div>`;
        tunnels.forEach(t => {
            const ip = t.node ? t.node.ip : '';
            const cName = t.node ? getCountryName(t.node.country_short) : '';
            const flag = t.node ? getCountryFlagSVG(t.node.country_short) : '';
            const isChecked = !isAllChecked && selTunMap[t.id];
            const pingStr = t.node && t.node.latency_ms > 0 ? `(${t.node.latency_ms}ms)` : '';
            html += `
                <label  class="check-row">
                    <input type="checkbox" class="chk-single-tunnel" value="${escapeHtml(t.id)}" ${isChecked ? 'checked' : ''} data-change-action="onSpecificTunnelCheckChanged">
                    <span class="flag-box">${flag}</span>
                    <strong  class="mono text-accent">${escapeHtml(t.dev_name)}</strong>
                    <span>${escapeHtml(cName)} (${escapeHtml(ip)})</span>
                    ${pingStr ? `<span class="text-xs ping-ok">${pingStr}</span>` : ''}
                </label>
            `;
        });
    }

    container.innerHTML = html;
}

export function onAllTunnelsCheckChanged(event, element) {
    const el = element || (event?.target?.closest ? event.target.closest('input') : null) || event?.target || this;
    if (el && el.checked) {
        document.querySelectorAll('.chk-single-tunnel').forEach(c => c.checked = false);
        document.querySelectorAll('.chk-dynamic-group').forEach(c => c.checked = false);
    }
}

export function onSpecificTunnelCheckChanged() {
    const anyTunChecked = Array.from(document.querySelectorAll('.chk-single-tunnel')).some(c => c.checked);
    const anyGrpChecked = Array.from(document.querySelectorAll('.chk-dynamic-group')).some(c => c.checked);
    const allChk = document.getElementById('chk-tunnel-all');
    if (anyTunChecked || anyGrpChecked) {
        allChk.checked = false;
    } else {
        allChk.checked = true;
    }
}

export async function savePortRule() {
    const port = parseInt(document.getElementById('rule-port').value);
    if (!port || port < 1 || port > 65535) {
        tAlert('请输入有效的端口号 (1-65535)', 'Please enter a valid port number (1-65535)');
        return;
    }

    const policy = document.getElementById('rule-policy').value;
    const interval = parseInt(document.getElementById('rule-interval').value) || 300;
    const authMode = document.getElementById('rule-auth-mode').value;
    const authUser = document.getElementById('rule-auth-user').value.trim();
    const authPass = document.getElementById('rule-auth-pass').value.trim();

    if (authMode === 'custom') {
        if (!authUser || !authPass) {
            tAlert('自定义认证模式必须同时填写用户名和密码，或选择免密模式！', 'Custom auth mode requires both username and password, or choose No Auth mode!');
            return;
        }
    }

    let boundTunnels = [];
    let boundGroups = [];
    const allChk = document.getElementById('chk-tunnel-all');
    if (!allChk || !allChk.checked) {
        document.querySelectorAll('.chk-single-tunnel:checked').forEach(c => boundTunnels.push(c.value));
        document.querySelectorAll('.chk-dynamic-group:checked').forEach(c => boundGroups.push(c.value));
    }

    const newRule = {
        port: port,
        enabled: true,
        bound_tunnel_ids: boundTunnels,
        bound_group_ids: boundGroups,
        policy: policy,
        interval_seconds: interval,
        auth_mode: authMode,
        auth_user: authUser,
        auth_pass: authPass
    };

    let updatedRules = appState.currentPortRules ? [...appState.currentPortRules] : [];
    const idx = updatedRules.findIndex(r => r.port === port);
    if (idx >= 0) updatedRules[idx] = newRule;
    else updatedRules.push(newRule);

    try {
        const res = await fetch('/api/proxy/ports', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ rules: updatedRules })
        });
        const ret = await res.json();
        if (!res.ok) {
            tAlert('保存端口规则失败: ' + (ret.error || '未知错误'), 'Failed to save port rule: ' + (ret.error || 'Unknown error'));
            return;
        }
        appState.currentPortRules = ret.rules || updatedRules;
        renderPortRules();
        hideEditPortForm();
        tAlert(`端口 [${port}] 规则已保存并实时生效！`, `Port [${port}] rule saved and active!`);
        fetchStatus();
    } catch (err) {
        alert(localize('请求异常: ', 'Request error: ') + err);
    }
}

export async function deletePortRule(port) {
    if (!(await tConfirm(`确认删除并停止代理端口 [${port}] 吗？`, `Are you sure you want to delete and stop proxy port [${port}]?`))) return;
    const updatedRules = appState.currentPortRules.filter(r => r.port !== port);
    try {
        const res = await fetch('/api/proxy/ports', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ rules: updatedRules })
        });
        const ret = await res.json();
        if (!res.ok) { tAlert('删除失败: ' + (ret.error || '未知错误'), 'Delete failed: ' + (ret.error || 'Unknown error')); return; }
        appState.currentPortRules = ret.rules || updatedRules;
        renderPortRules();
        fetchStatus();
    } catch (err) {
        tAlert('请求异常: ' + err, 'Request exception: ' + err);
    }
}
