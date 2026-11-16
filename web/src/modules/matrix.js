import { appState } from '../state.js';
import { getLanguage, localize, localizeGroupStatus, t, tAlert, tConfirm } from '../i18n-runtime.js';
import { closeEditorDrawer, openEditorDrawer, switchView } from './shell.js';
import { fetchStatus } from '../transport.js';
import { alert, appendLog, escapeHtml, getCountryFlagSVG, getCountryName, jsonAttr } from '../utils.js';

export function switchMatrixTab(tabKey) {
    document.querySelectorAll('#port-matrix-modal .modal-tab-btn').forEach(b => b.classList.remove('active'));
    const portsEl = document.getElementById('matrix-content-ports');
    const groupsEl = document.getElementById('matrix-content-groups');
    if (portsEl) {
        const hidePorts = (tabKey !== 'ports');
        portsEl.classList.toggle('hidden', hidePorts);
        portsEl.hidden = hidePorts;
    }
    if (groupsEl) {
        const hideGroups = (tabKey !== 'groups');
        groupsEl.classList.toggle('hidden', hideGroups);
        groupsEl.hidden = hideGroups;
    }
    const activeBtn = document.getElementById('matrix-tab-' + tabKey);
    if (activeBtn) activeBtn.classList.add('active');
    if (tabKey === 'groups') renderDynamicGroups();
    if (tabKey === 'ports') renderPortRules();
}

export async function startNewTunnel(nodeId) {
    appendLog({ level: 'INFO', module: 'Tunnel', message: localize(`正在启动并发独立出口隧道: ${nodeId}...`, `Starting a secondary egress tunnel: ${nodeId}...`) });
    try {
        const res = await fetch('/api/tunnels/start', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ node_id: nodeId })
        });
        const ret = await res.json();
        if (!res.ok) {
            tAlert('启动并发隧道失败: ' + (ret.error || '未知错误'), 'Failed to launch concurrent tunnel: ' + (ret.error || 'Unknown error'));
        } else {
            appendLog({ level: 'INFO', module: 'Tunnel', message: localize(`新隧道启动成功: ${ret.tunnel.id} (${ret.tunnel.dev_name})`, `Tunnel started: ${ret.tunnel.id} (${ret.tunnel.dev_name})`) });
            fetchStatus();
        }
    } catch (err) {
        alert(localize('请求失败: ', 'Request failed: ') + err);
    }
}

export async function stopTunnel(tunnelId) {
    if (!tConfirm('确认断开并释放该并发隧道吗？', 'Are you sure you want to disconnect and release this exit tunnel?')) return;
    try {
        await fetch('/api/tunnels/stop', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ tunnel_id: tunnelId })
        });
        fetchStatus();
    } catch (err) {
        alert(localize('断开失败: ', 'Disconnect failed: ') + err);
    }
}

export async function openPortMatrixModal() {
    switchView('matrix');
}

export function closePortMatrixModal() { switchView('dashboard'); }

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

export async function fetchDynamicGroups() {
    try {
        const res = await fetch('/api/tunnel-groups');
        if (!res.ok) return;
        appState.currentDynamicGroups = await res.json();
        renderDynamicGroups();
        renderMatrixTopology();
    } catch (err) {
        console.error("fetch dynamic groups error:", err);
    }
}

export function renderDynamicGroups() {
    const container = document.getElementById('dynamic-groups-container');
    if (!container) return;
    const isEn = getLanguage() === 'en';
    if (!appState.currentDynamicGroups || appState.currentDynamicGroups.length === 0) {
        container.innerHTML = `<div class="list-empty">${isEn ? 'No dynamic exit groups. Click "+ Add Exit Group" to maintain exits automatically.' : '暂无动态出口组，点击右上角「+ 新建出口组」即可按规则自动维持出口'}</div>`;
        return;
    }

    container.innerHTML = appState.currentDynamicGroups.map(g => {
        let metricText = isEn ? 'Lowest Latency' : '延迟最低优先';
        if (g.sort_by === 'speed') metricText = isEn ? 'Highest Bandwidth' : '带宽最大优先';
        if (g.sort_by === 'score') metricText = isEn ? 'Highest Score' : '评分最高优先';

        let ipTypeText = isEn ? 'All Network Types' : '全部网络类型';
        if (g.ip_type === 'residential') ipTypeText = isEn ? 'Residential Broadband' : '住宅宽带 IP';
        if (g.ip_type === 'hosting') ipTypeText = isEn ? 'Datacenter IP' : '机房 IP';
        if (g.ip_type === 'mobile') ipTypeText = isEn ? 'Mobile Network' : '移动网络';

        let unlockText = '';
        if (g.unlock_filter === 'ai') unlockText = `<span>${isEn ? 'Unlock: ' : '解锁: '}<strong class="unlock-ai">${isEn ? 'Triple AI (GPT+Claude+Gemini)' : '三大 AI (GPT+Claude+Gemini)'}</strong></span>`;
        else if (g.unlock_filter === 'connectivity') unlockText = `<span>${isEn ? 'Unlock: ' : '解锁: '}<strong class="text-accent">${isEn ? 'Web 204 Verified' : '真实网页连通 (204)'}</strong></span>`;
        else if (g.unlock_filter === 'streaming') unlockText = `<span>${isEn ? 'Unlock: ' : '解锁: '}<strong class="unlock-stream">${isEn ? 'Streaming Only' : '仅流媒体'}</strong></span>`;
        else if (g.unlock_filter === 'full' || g.unlock_filter === 'all') unlockText = `<span>${isEn ? 'Unlock: ' : '解锁: '}<strong class="unlock-full">${isEn ? 'Full Unlock (AI+Streaming)' : '全解锁 (三大 AI+流媒体)'}</strong></span>`;

        let purityText = '';
        if (g.purity_level === 'pure_residential') purityText = isEn ? 'Pure Residential (Risk ≤ 15)' : '极度纯净住宅 (Risk ≤ 15)';
        else if (g.purity_level === 'clean') purityText = isEn ? 'Clean / Low Risk (Risk ≤ 35)' : '纯净/低风险 (Risk ≤ 35)';
        else if (g.purity_level === 'datacenter') purityText = isEn ? 'Datacenter' : '高速机房';
        const purityInfo = purityText ? `<span>${isEn ? 'Purity: ' : '纯净度: '}<strong class="text-accent">${escapeHtml(purityText)}</strong></span>` : '';

        let purityBadge = '';
        if (g.purity_level === 'pure_residential') {
            purityBadge = `<span class="badge badge-pure badge-mini" style="font-size:10px">🏡 ${isEn ? 'Pure Res' : '极纯住宅'}</span>`;
        } else if (g.purity_level === 'clean') {
            purityBadge = `<span class="badge badge-info badge-mini" style="font-size:10px">🛡️ ${isEn ? 'Clean IP' : '纯净低风险'}</span>`;
        } else if (g.purity_level === 'datacenter') {
            purityBadge = `<span class="badge badge-muted badge-mini" style="font-size:10px">🏢 ${isEn ? 'DC' : '机房'}</span>`;
        }

        const isSys = g.is_system || g.id === 'system-primary';
        let countryStr = g.country ? `${getCountryName(g.country)} (${g.country})` : (isEn ? 'All Countries / Regions' : '全部国家/地区');
        if (g.country === 'FAVORITES') {
            countryStr = isEn ? '⭐ My Favorites (Cross-Country Pool)' : '⭐ 我的收藏节点 (跨国收藏池)';
        }
        let activeCount = (g.active_tunnel_ids || []).length;
        if (isSys && appState.currentState && appState.currentState.vpn && appState.currentState.vpn.status === 'connected') {
            activeCount = 1;
        }
        const statusClass = activeCount >= g.target_count ? 'connected' : (activeCount > 0 ? 'connecting' : 'disconnected');
        const statusText = localizeGroupStatus(g.status_text);

        let fallbackInfo = '';
        if (g.fallback_policy && g.fallback_policy !== 'none') {
            let fbLabel = g.fallback_policy;
            if (g.fallback_policy === 'favorites') fbLabel = isEn ? 'Favorites' : '我的收藏';
            else if (g.fallback_policy === 'auto_low_latency') fbLabel = isEn ? 'Lowest Latency' : '最低延迟';
            else if (g.fallback_policy === 'auto_speed') fbLabel = isEn ? 'Highest Speed' : '最高带宽';
            fallbackInfo = `<span>${isEn ? 'Fallback: ' : '兜底: '}<strong class="text-strong">${fbLabel}</strong></span>`;
        }

        const fallbackBadge = g.in_fallback ? `<span class="badge badge-warning" title="${escapeHtml(g.fallback_reason || '')}">${isEn ? 'Fallback Active' : '降级运行中'}</span>` : '';
        const fallbackReasonDiv = (g.in_fallback && g.fallback_reason) ? `<div class="text-xs text-warning mt-1">${escapeHtml(g.fallback_reason)}</div>` : '';

        const favBadge = `<span class="badge badge-accent">⭐ ${isEn ? 'Favorites' : '收藏组'}</span>`;
        const isZeroTun = g.connect_mode === 'zero_tun' || g.connect_mode === 'sstp';
        const sysTitle = isZeroTun ? (isEn ? 'Primary Gateway (Zero-TUN)' : '系统主连网关 (免TUN)') : (isEn ? 'Primary Gateway (tun0)' : '系统主连网关 (tun0)');
        const sysBadge = isSys ? `<span class="badge badge-system">${sysTitle}</span>` : (g.country === 'FAVORITES' ? favBadge : `<span class="badge badge-accent">Top ${g.target_count} ${isEn ? 'Tunnels' : '隧道'}</span>`);
        const modeBadge = isZeroTun ?
            `<span class="badge badge-accent" style="font-size:10px">⚡ ${isEn ? 'Zero-TUN' : '免TUN (SSTP)'}</span>` :
            `<span class="badge badge-mini" style="font-size:10px">${isEn ? 'Kernel TUN' : '内核 TUN'}</span>`;
        const deleteBtn = isSys ? '' : `<button class="btn btn-danger btn-xs" data-action="deleteDynamicGroup" data-args="${jsonAttr([g.id])}">${isEn ? 'Delete' : '删除'}</button>`;
        const editLabel = isSys ? (isEn ? 'Configure Strategy' : '配置主连策略') : (isEn ? 'Edit' : '编辑');

        return `
            <div class="dynamic-group-card ${isSys ? 'is-system' : ''} ${g.in_fallback ? 'is-fallback' : ''}">
                <div>
                    <div class="dynamic-group-title-row">
                        <span class="badge ${statusClass}"><span class="status-dot"></span> ${escapeHtml(statusText)}</span>
                        <strong class="dynamic-group-name">${escapeHtml(g.name)}</strong>
                        ${sysBadge}
                        ${modeBadge}
                        ${purityBadge}
                        ${fallbackBadge}
                    </div>
                    <div class="dynamic-group-meta">
                        <span>${isEn ? 'Driver: ' : '连接方式: '}<strong class="text-strong">${isZeroTun ? (isEn ? 'Zero-TUN' : '免TUN用户态') : (isEn ? 'Kernel TUN' : '内核虚拟网卡')}</strong></span>
                        <span>${isEn ? 'Target: ' : '目标: '}<strong class="text-strong">${escapeHtml(countryStr)}</strong></span>
                        <span>${isEn ? 'Type: ' : '类型: '}<strong class="text-strong">${escapeHtml(ipTypeText)}</strong></span>
                        ${purityInfo}
                        ${unlockText}
                        ${fallbackInfo}
                        <span>${isEn ? 'Policy: ' : '指标: '}<strong class="dynamic-metric">${metricText}</strong></span>
                        <span>${isEn ? 'Interval: ' : '周期: '}<strong class="text-strong">${g.interval_minutes}${isEn ? ' min' : '分钟'}</strong></span>
                    </div>
                    ${fallbackReasonDiv}
                </div>
                <div class="row gap-1">
                    <button class="btn btn-outline btn-xs" data-action="editDynamicGroup" data-args="${jsonAttr([g.id])}">${editLabel}</button>
                    ${deleteBtn}
                </div>
            </div>
        `;
    }).join('');
}

export function showAddDynamicGroupForm() {
    appState.editingGroupId = null;
    document.getElementById('dynamic-group-edit-title').innerText = localize('新建动态出口组', 'New Dynamic Egress Group');
    document.getElementById('dg-name').value = '';
    const modeEl = document.getElementById('dg-connect-mode');
    if (modeEl) modeEl.value = 'tun';
    document.getElementById('dg-country').value = 'JP';
    document.getElementById('dg-iptype').value = 'residential';
    const purityEl = document.getElementById('dg-purity');
    if (purityEl) purityEl.value = 'any';
    document.getElementById('dg-unlock').value = 'none';
    document.getElementById('dg-sortby').value = 'latency';
    const fallbackEl = document.getElementById('dg-fallback');
    if (fallbackEl) fallbackEl.value = 'none';
    document.getElementById('dg-count-container').classList.remove('hidden');
    document.getElementById('dg-count').value = 3;
    document.getElementById('dg-interval').value = 15;
    openEditorDrawer('dynamic-group-edit-card');
}

export function editDynamicGroup(id) {
    const g = appState.currentDynamicGroups.find(item => item.id === id);
    if (!g) return;
    appState.editingGroupId = id;
    const isSys = g.is_system || g.id === 'system-primary';
    if (isSys) {
        document.getElementById('dynamic-group-edit-title').innerText = localize('配置主出口自动轮换策略', 'Configure Primary Exit Rotation Policy');
        document.getElementById('dg-count-container').classList.add('hidden');
    } else {
        document.getElementById('dynamic-group-edit-title').innerText = localize(`编辑出口组: ${g.name}`, `Edit Egress Group: ${g.name}`);
        document.getElementById('dg-count-container').classList.remove('hidden');
    }
    document.getElementById('dg-name').value = g.name;
    const modeEl = document.getElementById('dg-connect-mode');
    if (modeEl) modeEl.value = g.connect_mode || 'tun';
    document.getElementById('dg-country').value = g.country || '';
    document.getElementById('dg-iptype').value = g.ip_type || 'all';
    const purityEl = document.getElementById('dg-purity');
    if (purityEl) purityEl.value = g.purity_level || 'any';
    document.getElementById('dg-unlock').value = g.unlock_filter || 'none';
    document.getElementById('dg-sortby').value = g.sort_by || 'latency';
    const fallbackEl = document.getElementById('dg-fallback');
    if (fallbackEl) fallbackEl.value = g.fallback_policy || 'none';
    document.getElementById('dg-count').value = g.target_count || (isSys ? 1 : 3);
    document.getElementById('dg-interval').value = g.interval_minutes || 15;
    openEditorDrawer('dynamic-group-edit-card');
}

export function hideDynamicGroupForm() { closeEditorDrawer('dynamic-group-edit-card'); }

export async function saveDynamicGroup() {
    const name = document.getElementById('dg-name').value.trim();
    if (!name) { tAlert('请输入出口组名称', 'Please enter an exit group name'); return; }
    const connectMode = document.getElementById('dg-connect-mode')?.value || 'tun';
    const country = document.getElementById('dg-country').value;
    const ipType = document.getElementById('dg-iptype').value;
    const purityLevel = document.getElementById('dg-purity')?.value || 'any';
    const unlockFilter = document.getElementById('dg-unlock').value;
    const sortBy = document.getElementById('dg-sortby').value;
    const fallbackPolicy = document.getElementById('dg-fallback')?.value || 'none';
    const count = parseInt(document.getElementById('dg-count').value) || 3;
    const interval = parseInt(document.getElementById('dg-interval').value) || 15;

    const payload = {
        id: appState.editingGroupId || '',
        name: name,
        enabled: true,
        is_system: appState.editingGroupId === 'system-primary',
        connect_mode: connectMode,
        country: country,
        ip_type: ipType,
        purity_level: purityLevel,
        unlock_filter: unlockFilter,
        sort_by: sortBy,
        fallback_policy: fallbackPolicy,
        target_count: appState.editingGroupId === 'system-primary' ? 1 : count,
        interval_minutes: interval
    };

    try {
        const res = await fetch('/api/tunnel-groups', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(payload)
        });
        const ret = await res.json();
        if (!res.ok) { tAlert('保存失败: ' + (ret.error || '未知错误'), 'Save failed: ' + (ret.error || 'Unknown error')); return; }
        hideDynamicGroupForm();
        appendLog({ level: 'INFO', module: 'Action', message: localize(`动态出口组 [${name}] 已成功保存！`, `Dynamic egress group [${name}] saved.`) });
        await fetchDynamicGroups();
        fetchStatus();
    } catch(err) {
        alert(localize('请求异常: ', 'Request error: ') + err);
    }
}

export async function deleteDynamicGroup(id) {
    if (!tConfirm('确认删除该动态出口组吗？其维护的隧道将被安全释放。', 'Are you sure you want to delete this exit group? Maintained tunnels will be safely released.')) return;
    try {
        const res = await fetch(`/api/tunnel-groups?id=${id}`, { method: 'DELETE' });
        const ret = await res.json();
        if (!res.ok) { tAlert('删除失败: ' + (ret.error || '未知错误'), 'Delete failed: ' + (ret.error || 'Unknown error')); return; }
        await fetchDynamicGroups();
        fetchStatus();
    } catch(err) {
        tAlert('请求异常: ' + err, 'Request exception: ' + err);
    }
}

export async function evaluateDynamicGroups() {
    appendLog({ level: 'INFO', module: 'Action', message: localize('正在触发出口组重新评估轮换...', 'Re-evaluating dynamic egress groups...') });
    try {
        await fetch('/api/tunnel-groups/evaluate', { method: 'POST' });
        setTimeout(async () => {
            await fetchDynamicGroups();
            fetchStatus();
        }, 1500);
    } catch(err){}
}

export function renderMatrixTopology() {
    const container = document.getElementById('matrix-topology-flow');
    if (!container) return;
    const isEn = getLanguage() === 'en';

    const st = (typeof appState.currentState !== 'undefined' && appState.currentState) ? appState.currentState : null;
    const defaultProxyPort = (st && st.settings && st.settings.proxy_port) || (st && st.proxy_addr ? parseInt(st.proxy_addr.split(':').pop()) : 7928);
    const tunnels = (st && st.tunnels) ? st.tunnels : [];

    // Stage Header Bar
    const stageBarHtml = `
        <div class="topo-stage-bar">
            <div class="topo-stage-item stage-1">
                <span class="topo-stage-num">01</span>
                <div class="topo-stage-text">
                    <strong>${isEn ? 'Inbound Listeners' : '入站监听端口'}</strong>
                    <span>${isEn ? 'Proxy Ports & Auth' : '本地监听端口与鉴权'}</span>
                </div>
            </div>
            <span class="topo-stage-arrow">──▶</span>
            <div class="topo-stage-item stage-2">
                <span class="topo-stage-num">02</span>
                <div class="topo-stage-text">
                    <strong>${isEn ? 'Routing Dispatcher' : '分流策略与出口组'}</strong>
                    <span>${isEn ? 'Dynamic Egress Pool' : '动态选路与负载调度'}</span>
                </div>
            </div>
            <span class="topo-stage-arrow">──▶</span>
            <div class="topo-stage-item stage-3">
                <span class="topo-stage-num">03</span>
                <div class="topo-stage-text">
                    <strong>${isEn ? 'Physical Egress Exits' : '出海物理网卡与端点'}</strong>
                    <span>${isEn ? 'Active Virtual NICs & Ping' : '活跃网卡 · 属地 · 延迟'}</span>
                </div>
            </div>
        </div>
    `;

    // Prepare stream branches
    const streamsData = [];

    // Stream 0: Primary Default Egress (tun0)
    const sysGroup = (appState.currentDynamicGroups || []).find(g => g.is_system || g.id === 'system-primary');
    const sysGroupName = sysGroup ? sysGroup.name : (isEn ? 'Primary Gateway Group' : '系统主出口网关组');
    const primaryDev = (st && st.primary && st.primary.dev_name) || 'tun0';
    const primaryIp = (st && st.primary && st.primary.ip) || (st ? st.exit_ip : '');
    const primaryPort = (st && st.vpn && st.vpn.active_node && st.vpn.active_node.port) || 443;
    const primaryCountry = (st && st.primary && st.primary.country) || (st && st.vpn && st.vpn.active_node && st.vpn.active_node.country_short) || '';
    const primaryLat = (st && st.primary && st.primary.latency_ms > 0) ? st.primary.latency_ms : ((st && st.vpn && st.vpn.active_node && st.vpn.active_node.latency_ms > 0) ? st.vpn.active_node.latency_ms : 0);

    const primaryLeaves = [];
    if (primaryIp) {
        primaryLeaves.push({
            devName: primaryDev,
            ip: primaryIp,
            port: primaryPort,
            country: primaryCountry,
            latency: primaryLat,
            unlock: (st && st.vpn && st.vpn.active_node && st.vpn.active_node.unlock) || null
        });
    }

    // Only append default stream if 7928 is not already defined in currentPortRules
    const hasExplicitDefaultPort = (appState.currentPortRules || []).some(r => r.port === defaultProxyPort);
    if (!hasExplicitDefaultPort) {
        streamsData.push({
            port: defaultProxyPort,
            proto: 'SOCKS5',
            portDesc: isEn ? 'Default Auth' : '系统默认鉴权',
            isDefault: true,
            groupId: 'system-primary',
            groupName: sysGroupName,
            policyLabel: isEn ? 'Primary Route' : '系统主干路由',
            concurrencyText: isEn ? '1 Dedicated NIC' : '1 独占主网卡',
            isFallback: false,
            fallbackReason: '',
            leaves: primaryLeaves
        });
    }

    // Extra Multi-Port rules streams
    if (appState.currentPortRules && appState.currentPortRules.length > 0) {
        appState.currentPortRules.forEach(rule => {
            if (!rule.enabled) return;
            let policyLabel = isEn ? 'Round-Robin' : '单连接轮询';
            if (rule.policy === 'random') policyLabel = isEn ? 'Random' : '随机分发';
            else if (rule.policy === 'interval') policyLabel = `${rule.interval_seconds || 60}s ${isEn ? 'Rotation' : '定时轮换'}`;

            let groupTitle = '';
            let firstGroupId = '';
            let isRowFallback = false;
            let fallbackReasonText = '';
            let concurrencyText = '';
            let leaves = [];
            let countryHint = '';

            if (rule.bound_group_ids && rule.bound_group_ids.length > 0) {
                const matchedGroups = (appState.currentDynamicGroups || []).filter(g => rule.bound_group_ids.includes(g.id));
                if (matchedGroups.length > 0) {
                    groupTitle = matchedGroups.map(g => g.name).join(' / ');
                    firstGroupId = matchedGroups[0].id;
                    isRowFallback = matchedGroups.some(g => g.in_fallback);
                    if (isRowFallback) {
                        const fbGroup = matchedGroups.find(g => g.in_fallback);
                        fallbackReasonText = fbGroup ? fbGroup.fallback_reason : '';
                    }
                    const totalTarget = matchedGroups.reduce((acc, g) => acc + (g.target_count || 1), 0);
                    concurrencyText = `${totalTarget} ${isEn ? 'Target NICs' : '目标并发网卡'}`;

                    const allActiveTids = [];
                    matchedGroups.forEach(g => {
                        if (g.active_tunnel_ids) allActiveTids.push(...g.active_tunnel_ids);
                    });

                    const activeTuns = tunnels.filter(t => allActiveTids.includes(t.id) || allActiveTids.includes(t.dev_name));
                    if (activeTuns.length > 0) {
                        leaves = activeTuns.map(t => {
                            const ip = (t.node && t.node.ip) || (t.node && t.node.id) || t.dev_name;
                            const port = (t.node && t.node.port) || 0;
                            const c = (t.node && t.node.country_short) || '';
                            const latVal = (t.node && t.node.latency_ms > 0) ? t.node.latency_ms : (t.latency_ms > 0 ? t.latency_ms : 0);
                            return {
                                devName: t.dev_name,
                                ip: ip,
                                port: port,
                                country: c,
                                latency: latVal,
                                unlock: t.unlock || (t.node ? appState.cachedUnlockMap[t.node.ip] : null)
                            };
                        });
                    } else {
                        const groupCountries = matchedGroups.map(g => g.country).filter(Boolean);
                        countryHint = groupCountries.length > 0 ? groupCountries.map(c => `${getCountryFlagSVG(c)} ${getCountryName(c)}`).join(', ') : (isEn ? 'All Regions' : '全部地区');
                    }
                }
            } else if (rule.bound_tunnel_ids && rule.bound_tunnel_ids.length > 0) {
                groupTitle = isEn ? 'Bound Tunnels' : '指定隧道出口';
                concurrencyText = `${rule.bound_tunnel_ids.length} ${isEn ? 'Tunnels' : '条隧道'}`;
                const activeTuns = tunnels.filter(t => rule.bound_tunnel_ids.includes(t.id) || rule.bound_tunnel_ids.includes(t.dev_name));
                if (activeTuns.length > 0) {
                    leaves = activeTuns.map(t => {
                        const ip = (t.node && t.node.ip) || t.dev_name;
                        const port = (t.node && t.node.port) || 0;
                        const c = (t.node && t.node.country_short) || '';
                        const latVal = (t.node && t.node.latency_ms > 0) ? t.node.latency_ms : (t.latency_ms > 0 ? t.latency_ms : 0);
                        return {
                            devName: t.dev_name,
                            ip: ip,
                            port: port,
                            country: c,
                            latency: latVal,
                            unlock: t.unlock || (t.node ? appState.cachedUnlockMap[t.node.ip] : null)
                        };
                    });
                }
            } else {
                groupTitle = isEn ? 'Direct / VPS Native' : '原生直连出口';
                concurrencyText = isEn ? 'No tunnel proxy' : '无隧道包装';
                leaves = [{
                    devName: 'direct',
                    ip: isEn ? 'VPS Local Network' : 'VPS 原生网络出海',
                    port: 0,
                    country: '',
                    latency: 0,
                    unlock: null
                }];
            }

            const authText = rule.auth_mode === 'none' ? (isEn ? 'No Auth' : '免密直连') : (isEn ? 'Protected' : '独立账密');
            streamsData.push({
                port: rule.port,
                proto: 'SOCKS5',
                portDesc: authText,
                isDefault: false,
                groupId: firstGroupId,
                groupName: groupTitle || `PORT ${rule.port}`,
                policyLabel: policyLabel,
                concurrencyText: concurrencyText,
                isFallback: isRowFallback,
                fallbackReason: fallbackReasonText,
                countryHint: countryHint,
                leaves: leaves
            });
        });
    }

    // Render stream rows
    const streamsHtml = streamsData.map(s => {
        const fbBadge = s.isFallback ? `<span class="badge badge-warning badge-mini">${isEn ? 'Fallback' : '降级运行中'}</span>` : '';
        const fbReasonHtml = (s.isFallback && s.fallbackReason) ? `<div class="text-xs text-warning mt-1" style="font-size: 10px;">${escapeHtml(s.fallbackReason)}</div>` : '';

        const portAction = s.isDefault
            ? `data-action="switchView" data-args="[&quot;settings&quot;]"`
            : `data-action="editPortRule" data-args="[${s.port}]"`;
        const portTitle = s.isDefault
            ? (isEn ? 'System Default Port (Configure in Settings)' : '系统默认代理端口 (点击前往系统设置配置)')
            : (isEn ? `Configure Port ${s.port}` : `点击编辑端口 ${s.port} 规则`);

        const groupAction = s.groupId
            ? `data-action="editDynamicGroup" data-args="[&quot;${escapeHtml(s.groupId)}&quot;]"`
            : '';
        const groupTitle = s.groupId
            ? (isEn ? `Configure Exit Group: ${s.groupName}` : `点击编辑出口组: ${s.groupName}`)
            : '';

        // Calculate connector SVG dimensions
        const leafCount = Math.max(1, s.leaves.length);
        const cardHeight = 64;
        const gap = 10;
        const totalHeight = leafCount * cardHeight + (leafCount - 1) * gap;
        const centerY = totalHeight / 2;

        // Build Bezier curves for Egress
        let egressWires = '';
        for (let i = 0; i < leafCount; i++) {
            const endY = i * (cardHeight + gap) + cardHeight / 2;
            egressWires += `
                <path class="topo-wire" d="M 0,${centerY} C 25,${centerY} 25,${endY} 50,${endY}" />
                <path class="topo-wire-flow" d="M 0,${centerY} C 25,${centerY} 25,${endY} 50,${endY}" />
            `;
        }

        // Render egress leaves cards
        let leavesCardsHtml = '';
        if (s.leaves.length === 0) {
            leavesCardsHtml = `
                <div class="topo-node-card egress-card" style="min-height: 64px; justify-content: center;">
                    <div class="egress-top-row">
                        <span class="text-xs text-muted">${s.countryHint || (isEn ? 'All Regions' : '全部地区')}</span>
                        <span class="badge badge-mini connecting"><span class="status-dot"></span> ${isEn ? 'Scheduling' : '就绪等待调度'}</span>
                    </div>
                    <div class="text-xs text-muted" style="font-size: 10px;">${isEn ? 'Awaiting dynamic evaluation to allocate NICs' : '等待调度器评估分配活跃网卡'}</div>
                </div>
            `;
        } else {
            leavesCardsHtml = s.leaves.map(leaf => {
                const cCode = leaf.country || '';
                const flag = cCode ? getCountryFlagSVG(cCode) : '';
                const cName = cCode ? getCountryName(cCode) : '';
                const locText = cName ? `${cName}` : (isEn ? 'Global' : '全球');

                let latClass = 'unknown';
                let latText = isEn ? 'Ready' : '测活就绪';
                if (leaf.latency > 0) {
                    latText = `⚡ ${leaf.latency}ms`;
                    if (leaf.latency < 100) latClass = 'fast';
                    else if (leaf.latency < 200) latClass = 'medium';
                    else latClass = 'slow';
                }
                const latBadge = `<span class="egress-latency-pill ${latClass}">${latText}</span>`;

                const u = leaf.unlock;
                const gptDot = (u && u.openai === 'unlocked') ? '<span class="egress-unlock-dot ok" title="ChatGPT OK">GPT✓</span>' : '<span class="egress-unlock-dot">GPT</span>';
                const claudeDot = (u && u.claude === 'unlocked') ? '<span class="egress-unlock-dot ok" title="Claude OK">Claude✓</span>' : '<span class="egress-unlock-dot">Claude</span>';
                const geminiDot = (u && u.gemini === 'unlocked') ? '<span class="egress-unlock-dot ok" title="Gemini OK">Gemini✓</span>' : '<span class="egress-unlock-dot">Gemini</span>';
                const nfDot = (u && u.netflix === 'unlocked') ? '<span class="egress-unlock-dot ok" title="Netflix OK">NF✓</span>' : '<span class="egress-unlock-dot">NF</span>';
                const unlockHtml = `${gptDot}${claudeDot}${geminiDot}${nfDot}`;

                const portStr = leaf.port > 0 ? `:${leaf.port}` : '';

                return `
                    <div class="topo-node-card egress-card">
                        <div class="egress-top-row">
                            <div class="flex items-center gap-2">
                                <span class="egress-nic-pill">${escapeHtml(leaf.devName)}</span>
                                <span class="egress-location">${flag} ${escapeHtml(locText)}</span>
                            </div>
                            ${latBadge}
                        </div>
                        <div class="egress-ip-row">
                            <code class="egress-ip-code">${escapeHtml(leaf.ip)}${escapeHtml(portStr)}</code>
                            <div class="egress-unlock-strip">
                                ${unlockHtml}
                            </div>
                        </div>
                    </div>
                `;
            }).join('');
        }

        return `
            <div class="pipeline-stream-row ${s.isFallback ? 'is-fallback' : ''}">
                <!-- Col 1: Inbound Port Card -->
                <div class="topo-col-inbound">
                    <div class="topo-node-card inbound-card" ${portAction} title="${escapeHtml(portTitle)}" role="button" tabindex="0">
                        <div class="inbound-top-row">
                            <span class="inbound-port-title">PORT ${s.port}</span>
                            <span class="badge badge-accent badge-mini">${escapeHtml(s.proto)}</span>
                        </div>
                        <div class="inbound-meta-row">
                            <span>${escapeHtml(s.portDesc)}</span>
                            <span>·</span>
                            <span>${s.isDefault ? (isEn ? 'Primary' : '主代理') : (isEn ? 'Multi-Port' : '分流端口')}</span>
                        </div>
                    </div>
                </div>

                <!-- Connector 1 (Inbound to Routing) -->
                <svg class="topo-svg-connector" viewBox="0 0 50 ${totalHeight}" style="height: ${totalHeight}px;">
                    <path class="topo-wire" d="M 0,${centerY} L 50,${centerY}" />
                    <path class="topo-wire-flow" d="M 0,${centerY} L 50,${centerY}" />
                </svg>

                <!-- Col 2: Routing Dispatcher Card -->
                <div class="topo-col-routing">
                    <div class="topo-node-card routing-card ${s.isFallback ? 'is-fallback' : ''}" ${groupAction} title="${escapeHtml(groupTitle)}" ${s.groupId ? 'role="button" tabindex="0"' : ''}>
                        <div class="routing-top-row">
                            <div class="routing-name-wrap">
                                <strong class="routing-group-name">${escapeHtml(s.groupName)}</strong>
                            </div>
                            ${fbBadge}
                        </div>
                        <div class="routing-meta-row">
                            <span class="routing-pill">${escapeHtml(s.policyLabel)}</span>
                            <span class="text-xs text-muted">${escapeHtml(s.concurrencyText)}</span>
                        </div>
                        ${fbReasonHtml}
                    </div>
                </div>

                <!-- Connector 2 (Routing to Egress Curves) -->
                <svg class="topo-svg-connector" viewBox="0 0 50 ${totalHeight}" style="height: ${totalHeight}px;">
                    ${egressWires}
                </svg>

                <!-- Col 3: Physical Egress Cards -->
                <div class="topo-col-egress">
                    ${leavesCardsHtml}
                </div>
            </div>
        `;
    }).join('');

    container.innerHTML = `
        ${stageBarHtml}
        <div class="pipeline-streams-container" role="region" aria-label="${isEn ? 'Egress Routing Pipeline' : '分流链路实时流向拓扑'}">
            ${streamsHtml}
        </div>
    `;
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

    let updatedRules = appState.currentPortRules ? [...currentPortRules] : [];
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
    if (!tConfirm(`确认删除并停止代理端口 [${port}] 吗？`, `Are you sure you want to delete and stop proxy port [${port}]?`)) return;
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
