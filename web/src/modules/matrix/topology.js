import { appState } from '../../state.js';
import { getLanguage, localize } from '../../i18n-runtime.js';
import { closeEditorDrawer, switchView } from '../shell.js';
import { fetchStatus } from '../../transport.js';
import { alert, appendLog, escapeHtml, getCountryFlagSVG, getCountryName } from '../../utils.js';
import { renderDynamicGroups } from './groups.js';
import { renderPortRules } from './ports.js';

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
    if (!(await tConfirm('确认断开并释放该并发隧道吗？', 'Are you sure you want to disconnect and release this exit tunnel?'))) return;
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

                    const includesSystemPrimary = matchedGroups.some(g => g.is_system || g.id === 'system-primary');
                    const allActiveTids = [];
                    matchedGroups.forEach(g => {
                        if (g.active_tunnel_ids) allActiveTids.push(...g.active_tunnel_ids);
                    });

                    const activeTuns = tunnels.filter(t =>
                        allActiveTids.includes(t.id) ||
                        allActiveTids.includes(t.dev_name) ||
                        (includesSystemPrimary && (t.id === 'primary-userland' || t.dev_index === 0))
                    );
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
                    } else if (includesSystemPrimary && primaryLeaves.length > 0) {
                        leaves = primaryLeaves;
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
