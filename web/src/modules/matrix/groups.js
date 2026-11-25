import { appState } from '../../state.js';
import { getLanguage, localize, localizeGroupStatus, tAlert, tConfirm } from '../../i18n-runtime.js';
import { closeEditorDrawer, openEditorDrawer } from '../shell.js';
import { fetchStatus } from '../../transport.js';
import { fetchNodes } from '../nodes.js';
import { addNodeToBlacklist } from '../blacklist.js';
import { alert, appendLog, escapeHtml, getCountryFlagSVG, getCountryName, jsonAttr } from '../../utils.js';
import { renderMatrixTopology } from './topology.js';

const PRESET_COUNTRY_CODES = [
    'JP', 'US', 'KR', 'TW', 'HK', 'SG', 'GB', 'DE', 'FR', 'CA', 'AU',
    'NL', 'VN', 'TH', 'MY', 'IN', 'RU', 'BR', 'PH', 'ID', 'IT', 'ES',
    'SE', 'CH', 'NZ', 'PL', 'UA', 'TR', 'ZA', 'AR', 'CL', 'CO', 'MX',
    'NO', 'FI', 'DK', 'IE', 'AT', 'BE', 'CZ', 'RO', 'IL', 'AE', 'SA', 'CN'
];
const FALLBACK_STRATEGY_CODES = new Set(['', 'none', 'favorites', 'auto_low_latency', 'auto_speed']);

export function updateDynamicGroupCountryOptions(selectedCountry, selectedFallback) {
    const isEn = getLanguage() === 'en';
    const countMap = {};
    (appState.allNodes || []).forEach(n => {
        const code = (n?.country_short || '').trim().toUpperCase();
        if (code && code.length === 2) {
            countMap[code] = (countMap[code] || 0) + 1;
        }
    });

    const presetRank = new Map(PRESET_COUNTRY_CODES.map((c, idx) => [c, idx]));
    const codeSet = new Set(PRESET_COUNTRY_CODES);
    Object.keys(countMap).forEach(c => codeSet.add(c));
    (appState.currentDynamicGroups || []).forEach(g => {
        const c = (g?.country || '').trim().toUpperCase();
        if (c && c !== 'ALL' && c !== 'FAVORITES' && c.length === 2) codeSet.add(c);
        const fb = (g?.fallback_policy || '').trim().toUpperCase();
        if (fb && fb.length === 2) codeSet.add(fb);
    });
    const extraC = (selectedCountry || '').trim().toUpperCase();
    if (extraC && extraC !== 'ALL' && extraC !== 'FAVORITES' && extraC.length === 2) codeSet.add(extraC);
    const extraFb = (selectedFallback || '').trim();
    if (extraFb && !FALLBACK_STRATEGY_CODES.has(extraFb.toLowerCase()) && extraFb.length === 2) {
        codeSet.add(extraFb.toUpperCase());
    }

    const sortedCodes = Array.from(codeSet).sort((a, b) => {
        const ca = countMap[a] || 0;
        const cb = countMap[b] || 0;
        if ((ca > 0) !== (cb > 0)) return cb > 0 ? 1 : -1;
        if (ca !== cb) return cb - ca;
        const ra = presetRank.has(a) ? presetRank.get(a) : Number.MAX_SAFE_INTEGER;
        const rb = presetRank.has(b) ? presetRank.get(b) : Number.MAX_SAFE_INTEGER;
        if (ra !== rb) return ra - rb;
        return a.localeCompare(b);
    });

    const countrySelect = document.getElementById('dg-country');
    if (countrySelect) {
        const targetVal = selectedCountry !== undefined ? String(selectedCountry) : countrySelect.value;
        const opts = [
            `<option value="">${isEn ? 'All countries / regions' : '全部国家/地区'}</option>`,
            `<option value="FAVORITES">${isEn ? '⭐ My Favorites (Cross-Country)' : '⭐ 我的收藏节点 (跨国收藏池)'}</option>`,
            ...sortedCodes.map(code => {
                const countSuffix = countMap[code] > 0 ? ` · ${countMap[code]}` : '';
                return `<option value="${escapeHtml(code)}">${escapeHtml(getCountryName(code))} (${escapeHtml(code)})${countSuffix}</option>`;
            })
        ];
        countrySelect.innerHTML = opts.join('');
        const normalizedTarget = targetVal.toUpperCase() === 'ALL' ? '' : (targetVal === 'FAVORITES' ? 'FAVORITES' : (targetVal ? targetVal.toUpperCase() : ''));
        countrySelect.value = normalizedTarget;
    }

    const fallbackSelect = document.getElementById('dg-fallback');
    if (fallbackSelect) {
        const targetFb = selectedFallback !== undefined ? String(selectedFallback) : fallbackSelect.value;
        const fbOpts = [
            `<option value="none">${isEn ? 'No Fallback (Keep state / Block)' : '不启用兜底 (保持原状/无节点阻断)'}</option>`,
            `<option value="favorites">${isEn ? '⭐ Cascade to My Favorites (Recommended)' : '⭐ 级联降级至我的收藏节点 (推荐)'}</option>`,
            `<option value="auto_low_latency">${isEn ? '⚡ Cascade to Global Lowest Latency' : '⚡ 级联降级至全球最低延迟节点'}</option>`,
            `<option value="auto_speed">${isEn ? '🚀 Cascade to Global Highest Bandwidth' : '🚀 级联降级至全球最高带宽节点'}</option>`,
            ...sortedCodes.map(code => {
                const label = isEn
                    ? `Cascade to ${getCountryName(code)} (${code}) Exit`
                    : `级联降级至 ${getCountryName(code)} (${code}) 出口`;
                return `<option value="${escapeHtml(code)}">${escapeHtml(label)}</option>`;
            })
        ];
        fallbackSelect.innerHTML = fbOpts.join('');
        fallbackSelect.value = targetFb || 'none';
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

export function blacklistGroupNode(nodeId, ip, country, groupId = '') {
    return addNodeToBlacklist(nodeId, ip, country, groupId);
}

function resolveGroupActiveExits(g) {
    const st = appState.currentState || {};
    const tunnels = Array.isArray(st.tunnels) ? st.tunnels : [];
    const allNodes = Array.isArray(appState.allNodes) ? appState.allNodes : [];
    const isSys = g.is_system || g.id === 'system-primary';
    const activeIds = Array.isArray(g.active_tunnel_ids) ? g.active_tunnel_ids : [];

    const matched = [];
    const seen = new Set();
    const pushTunnel = (t) => {
        if (!t || !t.node) return;
        const key = t.id || t.node.id;
        if (!key || seen.has(key)) return;
        seen.add(key);
        const liveNode = allNodes.find(n => n.id === t.node.id);
        const isFav = liveNode ? !!liveNode.is_favorite : !!t.node.is_favorite;
        const lat = (t.node.latency_ms > 0) ? t.node.latency_ms : (t.latency_ms > 0 ? t.latency_ms : 0);
        const unlock = t.unlock || (t.node.ip && appState.cachedUnlockMap ? appState.cachedUnlockMap[t.node.ip] : null) || t.node.unlock;
        matched.push({
            tunnelId: t.id || '',
            devName: t.dev_name || (t.is_userland ? 'sstp' : 'tun0'),
            nodeId: t.node.id || '',
            ip: t.node.ip || '',
            port: t.node.port || 0,
            country: t.node.country_short || '',
            latency: lat,
            isFavorite: isFav,
            unlock
        });
    };

    tunnels.forEach(t => {
        if (!t) return;
        if (activeIds.includes(t.id) || activeIds.includes(t.dev_name) || (isSys && (t.id === 'primary-userland' || t.dev_index === 0))) {
            pushTunnel(t);
        }
    });

    if (isSys && matched.length === 0 && st.vpn && st.vpn.active_node && st.vpn.status === 'connected') {
        const n = st.vpn.active_node;
        const liveNode = allNodes.find(item => item.id === n.id);
        matched.push({
            tunnelId: st.vpn.zero_tun ? 'primary-userland' : 'tun0',
            devName: st.vpn.zero_tun ? 'sstp' : 'tun0',
            nodeId: n.id || '',
            ip: n.ip || '',
            port: n.port || 0,
            country: n.country_short || '',
            latency: n.latency_ms > 0 ? n.latency_ms : 0,
            isFavorite: liveNode ? !!liveNode.is_favorite : !!n.is_favorite,
            unlock: (n.ip && appState.cachedUnlockMap ? appState.cachedUnlockMap[n.ip] : null) || n.unlock
        });
    }
    return matched;
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

        const activeExits = resolveGroupActiveExits(g);
        const exitsHtml = activeExits.length > 0 ? `
            <div class="group-nodes-section">
                <div class="group-nodes-title">${isEn ? 'Active Exits' : '当前活跃出口节点'} (${activeExits.length})</div>
                <div class="group-nodes-list">
                    ${activeExits.map(ex => {
                        const cName = ex.country ? getCountryName(ex.country) : (isEn ? 'Global' : '未知');
                        const flag = ex.country ? getCountryFlagSVG(ex.country) : '';
                        const portStr = ex.port > 0 ? `:${ex.port}` : '';
                        const latStr = ex.latency > 0 ? `<span class="badge latency-badge latency-available badge-mini">${ex.latency}ms</span>` : '';
                        const u = ex.unlock;
                        const unlockHint = u ? `${u.openai === 'unlocked' ? '<span class="egress-unlock-dot ok">GPT✓</span>' : ''}${u.claude === 'unlocked' ? '<span class="egress-unlock-dot ok">Claude✓</span>' : ''}${u.gemini === 'unlocked' ? '<span class="egress-unlock-dot ok">Gemini✓</span>' : ''}` : '';
                        const starSvg = ex.isFavorite
                            ? '<svg viewBox="0 0 24 24" width="13" height="13" fill="#f59e0b" stroke="#f59e0b" stroke-width="1.6"><polygon points="12 2 15.09 8.26 22 9.27 17 14.14 18.18 21.02 12 17.77 5.82 21.02 7 14.14 2 9.27 8.91 8.26 12 2"/></svg>'
                            : '<svg viewBox="0 0 24 24" width="13" height="13" fill="none" stroke="currentColor" stroke-width="1.6"><polygon points="12 2 15.09 8.26 22 9.27 17 14.14 18.18 21.02 12 17.77 5.82 21.02 7 14.14 2 9.27 8.91 8.26 12 2"/></svg>';
                        return `
                            <div class="group-node-row">
                                <div class="group-node-info">
                                    <span class="egress-nic-pill">${escapeHtml(ex.devName)}</span>
                                    <span class="flag-box text-xs">${flag} ${escapeHtml(cName)}</span>
                                    <code class="egress-ip-code">${escapeHtml(ex.ip)}${escapeHtml(portStr)}</code>
                                    ${latStr}
                                    ${unlockHint}
                                </div>
                                ${ex.nodeId ? `<div class="group-node-actions">
                                    <button type="button" class="btn btn-outline btn-xs group-node-star ${ex.isFavorite ? 'is-fav' : ''}" data-action="toggleFavorite" data-args="${jsonAttr([ex.nodeId])}" title="${ex.isFavorite ? (isEn ? 'Remove from favorites' : '取消收藏') : (isEn ? 'Add to favorites' : '加入收藏')}">
                                        ${starSvg} <span>${ex.isFavorite ? (isEn ? 'Fav' : '已收藏') : (isEn ? 'Fav' : '收藏')}</span>
                                    </button>
                                    <button type="button" class="btn btn-outline btn-xs btn-icon-danger" data-action="blacklistGroupNode" data-args="${jsonAttr([ex.nodeId, ex.ip, ex.country, g.id])}" title="${isEn ? 'Block node & auto-rotate' : '拉黑此节点并自动换线'}">
                                        🚫 <span>${isEn ? 'Block' : '拉黑'}</span>
                                    </button>
                                </div>` : ''}
                            </div>
                        `;
                    }).join('')}
                </div>
            </div>
        ` : '';

        return `
            <div class="dynamic-group-card ${isSys ? 'is-system' : ''} ${g.in_fallback ? 'is-fallback' : ''}">
                <div class="dynamic-group-main">
                    <div class="dynamic-group-header-bar">
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
                        <div class="row gap-1 shrink-0">
                            <button class="btn btn-outline btn-xs" data-action="editDynamicGroup" data-args="${jsonAttr([g.id])}">${editLabel}</button>
                            ${deleteBtn}
                        </div>
                    </div>
                    ${exitsHtml}
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
    updateDynamicGroupCountryOptions('JP', 'none');
    if (!appState.allNodes || appState.allNodes.length === 0) fetchNodes();
    document.getElementById('dg-iptype').value = 'residential';
    const purityEl = document.getElementById('dg-purity');
    if (purityEl) purityEl.value = 'any';
    document.getElementById('dg-unlock').value = 'none';
    document.getElementById('dg-sortby').value = 'latency';
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
    updateDynamicGroupCountryOptions(g.country || '', g.fallback_policy || 'none');
    if (!appState.allNodes || appState.allNodes.length === 0) fetchNodes();
    document.getElementById('dg-iptype').value = g.ip_type || 'all';
    const purityEl = document.getElementById('dg-purity');
    if (purityEl) purityEl.value = g.purity_level || 'any';
    document.getElementById('dg-unlock').value = g.unlock_filter || 'none';
    document.getElementById('dg-sortby').value = g.sort_by || 'latency';
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
    if (!(await tConfirm('确认删除该动态出口组吗？其维护的隧道将被安全释放。', 'Are you sure you want to delete this exit group? Maintained tunnels will be safely released.'))) return;
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
