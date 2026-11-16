import { appState } from '../state.js';
import { getLanguage, localize, t, tAlert } from '../i18n-runtime.js';
import { fetchStatus, fetchUnlockCache } from '../transport.js';
import { appendLog, escapeHtml, getCountryFlagSVG, getCountryName, jsonAttr, renderUnlockBadges } from '../utils.js';

export async function fetchNodes() {
    try {
        const res = await fetch('/api/nodes');
        if (!res.ok) return;
        appState.allNodes = await res.json();
        const nBadge = document.getElementById('nav-node-badge');
        if (nBadge) {
            nBadge.innerText = appState.allNodes.length;
        }
        updateCountryFilter();
        renderNodes();
        document.getElementById('nodes-table-wrap')?.setAttribute('aria-busy', 'false');
    } catch (err) {
        console.error("fetch nodes error:", err);
        document.getElementById('nodes-table-wrap')?.setAttribute('aria-busy', 'false');
    }
}

export function updateCountryFilter() {
    const select = document.getElementById('country-filter');
    const curr = select.value;
    const countries = [...new Set(appState.allNodes.map(n => n.country_short))].filter(Boolean).sort();

    select.innerHTML = `<option value="">${localize('全部国家/地区', 'All countries / regions')}</option>` +
        countries.map(c => `<option value="${escapeHtml(c)}">${escapeHtml(getCountryName(c))} (${escapeHtml(c)})</option>`).join('');
    select.value = curr;
}


export function toggleSort(field) {
    if (field === 'latency') {
        appState.currentSort = appState.currentSort === 'latency_asc' ? 'score_desc' : 'latency_asc';
    } else if (field === 'speed') {
        appState.currentSort = appState.currentSort === 'speed_desc' ? 'latency_asc' : 'speed_desc';
    } else if (field === 'score') {
        appState.currentSort = appState.currentSort === 'score_desc' ? 'latency_asc' : 'score_desc';
    }
    document.getElementById('sort-filter').value = appState.currentSort;
    renderNodes();
}

export function selectQuickFilter(filter) {
    appState.activeQuickFilter = filter;
    document.querySelectorAll('.chip').forEach(c => c.classList.remove('active'));
    const el = document.getElementById('chip-' + filter);
    if (el) el.classList.add('active');
    renderNodes();
}

export async function toggleFavorite(nodeId) {
    try {
        const res = await fetch('/api/nodes/favorite', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ node_id: nodeId })
        });
        const ret = await res.json();
        if (res.ok) {
            const node = appState.allNodes.find(n => n.id === nodeId);
            if (node) {
                node.is_favorite = ret.is_favorite;
                renderNodes();
            }
        }
    } catch (err) {}
}


export const COUNTRY_COORDS = {
    JP: [36.2048, 138.2529],
    US: [37.0902, -95.7129],
    KR: [35.9078, 127.7669],
    TW: [23.6978, 120.9605],
    HK: [22.3193, 114.1694],
    SG: [1.3521, 103.8198],
    GB: [55.3781, -3.4360],
    DE: [51.1657, 10.4515],
    FR: [46.2276, 2.2137],
    CA: [56.1304, -106.3468],
    AU: [-25.2744, 133.7751],
    NL: [52.1326, 5.2913],
    RU: [61.5240, 105.3188],
    VN: [14.0583, 108.2772],
    TH: [15.8700, 100.9925],
    MY: [4.2105, 101.9758],
    IN: [20.5937, 78.9629],
    BR: [-14.2350, -51.9253],
    UA: [48.3794, 31.1656],
    PL: [51.9194, 19.1451],
    SE: [60.1282, 18.6435],
    NO: [60.4720, 8.4689],
    FI: [61.9241, 25.7482],
    IT: [41.8719, 12.5674],
    ES: [40.4637, -3.7492],
    CH: [46.8182, 8.2275],
    AT: [47.5162, 14.5501],
    NZ: [-40.9006, 174.8860],
    ID: [-0.7893, 113.9213],
    PH: [12.8797, 121.7740],
    TR: [38.9637, 35.2433],
    ZA: [-30.5595, 22.9375],
    MX: [23.6345, -102.5528],
    AR: [-38.4161, -63.6167],
    CL: [-35.6751, -71.5430]
};

export function getNodeCoordinates(node) {
    if (node && Number.isFinite(node.latitude) && Number.isFinite(node.longitude) && (node.latitude !== 0 || node.longitude !== 0)) {
        return [node.latitude, node.longitude];
    }
    const c = String(node?.country_short || '').toUpperCase();
    if (COUNTRY_COORDS[c]) {
        const [baseLat, baseLon] = COUNTRY_COORDS[c];
        const ipStr = String(node?.ip || '0.0.0.0');
        let hash = 0;
        for (let i = 0; i < ipStr.length; i++) {
            hash = (hash * 31 + ipStr.charCodeAt(i)) & 0xffffff;
        }
        const jitterLat = (((hash % 200) - 100) / 100) * 1.5;
        const jitterLon = ((((hash >> 8) % 200) - 100) / 100) * 2.0;
        return [baseLat + jitterLat, baseLon + jitterLon];
    }
    return null;
}

export function initNodesMap() {
    if (appState.nodesMap) return;
    const container = document.getElementById('global-nodes-map');
    if (!container || !window.L) return;

    try {
        appState.nodesMap = L.map('global-nodes-map', {
            zoomControl: false,
            attributionControl: false,
            minZoom: 1,
            maxZoom: 10,
            maxBounds: [[-85, -180], [85, 180]],
            maxBoundsViscosity: 1.0,
            worldCopyJump: true
        }).setView([25, 0], 2);

        // 1. Add 100% offline local vector landmass (completely avoids any external network or API key requirement)
        if (window.WORLD_LAND) {
            L.geoJSON(window.WORLD_LAND, {
                style: {
                    fillColor: '#172554',
                    fillOpacity: 0.95,
                    color: '#2563eb',
                    weight: 1.2,
                    lineJoin: 'round'
                }
            }).addTo(appState.nodesMap);
        }

        appState.nodesMarkerGroup = L.featureGroup().addTo(appState.nodesMap);
    } catch (err) {
        console.warn('Map initialization deferred or failed:', err);
    }
}

export function updateNodesMapMarkers(nodesToRender) {
    initNodesMap();
    if (!appState.nodesMap || !appState.nodesMarkerGroup || !window.L) return;
    appState.nodesMarkerGroup.clearLayers();

    const nodesList = nodesToRender || appState.allNodes || [];
    const isEn = getLanguage() === 'en';
    let activeExitCount = 0;
    let pinnedCount = 0;

    if (appState.currentState?.vpn?.status === 'connected') {
        activeExitCount++;
    }
    if (Array.isArray(appState.currentState?.tunnels)) {
        activeExitCount += appState.currentState.tunnels.filter(t => t && t.status === 'connected').length;
    }

    const primaryId = appState.currentState?.vpn?.active_node_id;
    const primaryIp = appState.currentState?.vpn?.active_node?.ip;
    const dynamicTunnels = (appState.currentState?.tunnels || []);

    nodesList.forEach(node => {
        const coords = getNodeCoordinates(node);
        if (!coords) return;

        pinnedCount++;
        const isPrimary = primaryId === node.id || (primaryIp && primaryIp === node.ip);
        const isDynamic = dynamicTunnels.some(t => t.node && (t.node.id === node.id || t.node.ip === node.ip));

        let pinClass = 'pin-clean';
        if (isPrimary) {
            pinClass = 'pin-primary';
        } else if (isDynamic) {
            pinClass = 'pin-dynamic';
        } else if (node.risk_level === 'pure') {
            pinClass = 'pin-pure';
        } else if (node.risk_level === 'elevated') {
            pinClass = 'pin-elevated';
        } else if (node.risk_level === 'high') {
            pinClass = 'pin-high';
        } else if (node.risk_level === 'critical') {
            pinClass = 'pin-critical';
        }

        const icon = L.divIcon({
            className: 'node-pin-wrapper',
            html: `<div class="node-pin ${pinClass}" title="${escapeHtml(node.ip || '')} (${escapeHtml(node.city || node.country_long || '')})"></div>`,
            iconSize: [16, 16],
            iconAnchor: [8, 8],
            popupAnchor: [0, -8]
        });

        const marker = L.marker(coords, { icon });

        const latencyText = node.latency_ms > 0 ? `${node.latency_ms} ms` : (node.ping > 0 ? `${isEn ? "Source " : "来源 "}${node.ping} ms` : (isEn ? 'Unknown' : '未知'));
        const riskLabel = node.risk_label || (isEn ? 'Clean' : '纯净住宅');
        const riskLevel = node.risk_level || 'clean';

        const popupHtml = `
            <div class="map-popup-header">
                <span class="map-popup-title">
                    ${getCountryFlagSVG(node.country_short)}
                    <span>${escapeHtml(getCountryName(node.country_short))} · ${escapeHtml(node.city || '')}</span>
                </span>
                <span class="risk-badge ${escapeHtml(riskLevel)}">${escapeHtml(riskLabel)}</span>
            </div>
            <div class="map-popup-body">
                <div class="map-popup-row">
                    <span class="map-popup-label">${isEn ? 'Endpoint' : '出口端点'}</span>
                    <span class="map-popup-val">${escapeHtml(node.ip || '')}:${node.port} (${escapeHtml(String(node.proto || 'UDP').toUpperCase())})</span>
                </div>
                <div class="map-popup-row">
                    <span class="map-popup-label">${isEn ? 'Network' : '运营商/ISP'}</span>
                    <span class="map-popup-val" style="font-size:10px">${escapeHtml(node.isp || node.operator || (isEn ? 'Residential' : '住宅宽带'))}</span>
                </div>
                <div class="map-popup-row">
                    <span class="map-popup-label">${isEn ? 'Latency / Speed' : '实测延迟 / 带宽'}</span>
                    <span class="map-popup-val">${latencyText} · ${(Number(node.speed || 0) / 1000000).toFixed(1)}M</span>
                </div>
                <div style="display:flex;gap:6px;margin-top:6px">
                    <button type="button" class="btn btn-sm ${isPrimary && !appState.currentState?.vpn?.zero_tun ? 'btn-outline' : ''} map-popup-btn" style="flex:1" data-action="connectToNode" data-args="${jsonAttr([node.id, 'kernel'])}">
                        ${isPrimary && !appState.currentState?.vpn?.zero_tun ? (isEn ? 'Kernel Exit' : '当前内核连') : (isEn ? 'Kernel' : '内核直连')}
                    </button>
                    <button type="button" class="btn btn-sm ${isPrimary && appState.currentState?.vpn?.zero_tun ? 'btn-outline' : ''} map-popup-btn" style="flex:1" data-action="connectToNode" data-args="${jsonAttr([node.id, 'sstp'])}">
                        ⚡${isPrimary && appState.currentState?.vpn?.zero_tun ? (isEn ? 'Zero-TUN Exit' : '当前免TUN') : (isEn ? 'Zero-TUN' : '免TUN直连')}
                    </button>
                </div>
            </div>
        `;

        marker.bindPopup(popupHtml, { minWidth: 220, maxWidth: 290 });
        appState.nodesMarkerGroup.addLayer(marker);
    });

    const badgeEl = document.getElementById('map-active-badge');
    if (badgeEl) {
        badgeEl.innerText = `${activeExitCount} ${isEn ? 'Active' : '活跃出口'} · ${pinnedCount} ${isEn ? 'Mapped' : '打点'}`;
    }
}

export function resetMapView() {
    if (!appState.nodesMap) return;
    if (appState.nodesMarkerGroup && appState.nodesMarkerGroup.getLayers().length > 0) {
        try {
            appState.nodesMap.fitBounds(appState.nodesMarkerGroup.getBounds().pad(0.15));
        } catch (e) {
            appState.nodesMap.setView([25, 10], 2);
        }
    } else {
        appState.nodesMap.setView([25, 10], 2);
    }
}

export function toggleMapCollapse() {
    const wrap = document.getElementById('map-container-wrap');
    const icon = document.getElementById('map-collapse-icon');
    const text = document.getElementById('map-collapse-text');
    if (!wrap) return;

    appState.isMapCollapsed = !appState.isMapCollapsed;
    if (appState.isMapCollapsed) {
        wrap.classList.add('is-collapsed');
        if (text) text.innerText = getLanguage() === 'en' ? 'Expand' : '展开';
        if (icon) icon.innerHTML = '<polyline points="6 9 12 15 18 9"/>';
    } else {
        wrap.classList.remove('is-collapsed');
        if (text) text.innerText = getLanguage() === 'en' ? 'Collapse' : '收起';
        if (icon) icon.innerHTML = '<polyline points="18 15 12 9 6 15"/>';
        setTimeout(() => { if (appState.nodesMap) appState.nodesMap.invalidateSize(); }, 250);
    }
}

export function renderNodes() {
    const tbody = document.getElementById('nodes-tbody');
    const search = document.getElementById('search-filter').value.toLowerCase();
    const country = document.getElementById('country-filter').value;
    const ipType = document.getElementById('ip-type-filter').value;
    const sortMode = document.getElementById('sort-filter').value;

    const favCount = appState.allNodes.filter(n => n.is_favorite).length;
    const favCountEl = document.getElementById('fav-count');
    if (favCountEl) favCountEl.innerText = favCount;

    const filtered = appState.allNodes.filter(n => {
        // Quick Chips Filter Logic
        if (appState.activeQuickFilter === 'fav' && !n.is_favorite) return false;
        if (appState.activeQuickFilter === 'res' && n.ip_type !== 'residential') return false;
        if (appState.activeQuickFilter === 'pure' && n.risk_level !== 'pure') return false;
        if (appState.activeQuickFilter === 'host' && n.ip_type !== 'hosting') return false;
        if (appState.activeQuickFilter === 'fast' && (n.latency_ms <= 0 || n.latency_ms >= 150)) return false;
        if (appState.activeQuickFilter === 'ai') {
            const u = (appState.cachedUnlockMap && appState.cachedUnlockMap[n.ip]) || n.unlock;
            if (!u || u.openai !== 'unlocked' || u.claude !== 'unlocked' || u.gemini !== 'unlocked') return false;
        }
        if (appState.activeQuickFilter === 'gpt') {
            const u = (appState.cachedUnlockMap && appState.cachedUnlockMap[n.ip]) || n.unlock;
            if (!u || u.openai !== 'unlocked') return false;
        }
        if (appState.activeQuickFilter === 'claude') {
            const u = (appState.cachedUnlockMap && appState.cachedUnlockMap[n.ip]) || n.unlock;
            if (!u || u.claude !== 'unlocked') return false;
        }
        if (appState.activeQuickFilter === 'gemini') {
            const u = (appState.cachedUnlockMap && appState.cachedUnlockMap[n.ip]) || n.unlock;
            if (!u || u.gemini !== 'unlocked') return false;
        }
        if (appState.activeQuickFilter === 'nf') {
            const u = (appState.cachedUnlockMap && appState.cachedUnlockMap[n.ip]) || n.unlock;
            if (!u || u.netflix !== 'unlocked') return false;
        }

        // Standard Filters
        if (country && n.country_short !== country) return false;
        if (ipType && n.ip_type !== ipType) return false;
        if (search) {
            const cName = getCountryName(n.country_short).toLowerCase();
            const str = `${n.ip} ${n.country_long} ${n.country_short} ${cName} ${n.hostname} ${n.isp || ''} ${n.city || ''} ${n.region || ''}`.toLowerCase();
            if (!str.includes(search)) return false;
        }
        return true;
    });

    filtered.sort((a, b) => {
        if (sortMode === 'latency_asc') {
            const aAvail = (a.latency_ms > 0 || a.probe_status === 'connected') ? 1 : (a.latency_ms === -1 ? -1 : 0);
            const bAvail = (b.latency_ms > 0 || b.probe_status === 'connected') ? 1 : (b.latency_ms === -1 ? -1 : 0);
            if (aAvail !== bAvail) return bAvail - aAvail;
            if ((a.latency_ms > 0) !== (b.latency_ms > 0)) return a.latency_ms > 0 ? -1 : 1;
            if (a.latency_ms > 0 && b.latency_ms > 0) return a.latency_ms - b.latency_ms;
            return b.score - a.score;
        } else if (sortMode === 'speed_desc') {
            return b.speed - a.speed;
        } else if (sortMode === 'score_desc') {
            return b.score - a.score;
        } else if (sortMode === 'ping_asc') {
            return a.ping - b.ping;
        }
        return 0;
    });

    const isEn = getLanguage() === 'en';
    document.getElementById('filtered-count').innerText = `${isEn ? 'Showing ' : '已筛选出 '}${filtered.length} / ${appState.allNodes.length} ${isEn ? 'nodes' : '个节点'}`;

    updateNodesMapMarkers(filtered);

    if (filtered.length === 0) {
        tbody.innerHTML = `<tr><td colspan="9" class="empty-state">${isEn ? 'No nodes match the selected criteria' : '未找到匹配条件的节点'}</td></tr>`;
        return;
    }

    tbody.innerHTML = filtered.map(n => {
        const isCurrent = appState.currentState && appState.currentState.vpn && appState.currentState.vpn.active_node_id === n.id;
        const activeTun = (appState.currentState && appState.currentState.tunnels) ? appState.currentState.tunnels.find(t => t.node && (t.node.id === n.id || t.node.ip === n.ip)) : null;

        let latencyBadge = '';
        if (n.probe_status === 'connected' && n.latency_ms <= 0) {
            latencyBadge = `<span class="badge latency-badge latency-available">${isEn ? 'Connection succeeded' : '曾连接成功'}</span>`;
        } else if (n.latency_ms > 0) {
            latencyBadge = `<span class="badge latency-badge latency-available"><span class="status-dot"></span> ${isEn ? 'TCP ' : 'TCP可达 '}${n.latency_ms} ms</span>`;
        } else if (n.latency_ms === -1) {
            latencyBadge = `<span class="badge latency-badge latency-timeout"><span class="status-dot"></span> ${isEn ? 'Timeout' : '超时不可达'}</span>`;
        } else {
            latencyBadge = `<span class="badge latency-badge latency-unknown"><span class="status-dot"></span> ${isEn ? 'Unverified' : '未验证'}</span>`;
        }

        const speedMbps = (n.speed / 1000000).toFixed(1) + ' Mbps';

        let typeTag = `<span class="badge badge-residential">${isEn ? 'Residential' : '住宅宽带'}</span>`;
        if (n.ip_type === 'hosting') {
            typeTag = `<span class="badge badge-hosting">${isEn ? 'Datacenter' : '机房网络'}</span>`;
        } else if (n.ip_type === 'mobile') {
            typeTag = `<span class="badge badge-mobile">${isEn ? 'Mobile' : '移动网络'}</span>`;
        }

        const ispInfo = n.isp ? `<div class="node-isp" title="${escapeHtml(n.isp)}">${escapeHtml(n.isp)}</div>` : '';
        const cityInfo = n.city ? `<span class="node-city"> · ${escapeHtml(n.city)}</span>` : '';

        const repVal = n.reputation_score !== undefined ? n.reputation_score : 60;
        const repClass = repVal < 45 ? 'rep-bad' : (repVal < 70 ? 'rep-warn' : 'rep-good');
        const repBadge = `<span class="badge reputation-badge ${repClass}">${repVal}${isEn ? ' pts' : '分'}</span>`;

        const unlockData = (appState.cachedUnlockMap && appState.cachedUnlockMap[n.ip]) || n.unlock;
        const unlockInfo = unlockData ? renderUnlockBadges(unlockData) : `<span class="text-xs text-muted">${isEn ? 'Not Probed' : '未检测'}</span>`;

        let riskBadge = '';
        if (n.risk_level) {
            const riskLabel = (!isEn && n.risk_label) || (n.risk_level === 'pure' ? (isEn ? 'Pure' : '极度纯净') : (isEn ? 'Clean' : '纯净住宅'));
            riskBadge = `<span class="risk-badge ${escapeHtml(n.risk_level)}" title="${isEn ? 'Risk Score' : '风险评分'}: ${n.risk_score || 0}%">${escapeHtml(riskLabel)}</span>`;
        }

        return `
            <tr class="${isCurrent ? 'node-row-current' : (activeTun ? 'node-row-active' : '')}">
                <td data-label="" class="text-center">
                    <button class="star-btn ${n.is_favorite ? 'active' : ''}" data-action="toggleFavorite" data-args="${jsonAttr([n.id])}" title="${n.is_favorite ? (isEn ? 'Remove from favorites' : '取消收藏') : (isEn ? 'Add to favorites' : '加入收藏')}">
                        ${n.is_favorite ? '<svg viewBox="0 0 24 24" width="14" height="14" fill="#f59e0b" stroke="#f59e0b" stroke-width="1.5"><polygon points="12 2 15.09 8.26 22 9.27 17 14.14 18.18 21.02 12 17.77 5.82 21.02 7 14.14 2 9.27 8.91 8.26 12 2"/></svg>' : '<svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="#64748b" stroke-width="1.5"><polygon points="12 2 15.09 8.26 22 9.27 17 14.14 18.18 21.02 12 17.77 5.82 21.02 7 14.14 2 9.27 8.91 8.26 12 2"/></svg>'}
                    </button>
                </td>
                <td data-label="${isEn ? 'Region' : '地区'}">
                    <div class="flag-box">
                        ${getCountryFlagSVG(n.country_short)}
                        <span>${escapeHtml(getCountryName(n.country_short))}</span>
                        ${cityInfo}
                    </div>
                </td>
                <td data-label="${isEn ? 'Endpoint' : '出口端点'}">
                    <div class="node-endpoint">${escapeHtml(n.ip)}:${escapeHtml(n.port)}</div>
                    <span class="badge badge-proto">${escapeHtml(String(n.proto || '').toUpperCase())}</span>
                </td>
                <td data-label="${isEn ? 'Type' : '网络类型'}">${typeTag} ${riskBadge}${ispInfo}</td>
                <td data-label="${isEn ? 'Latency' : '延迟'}">${latencyBadge}</td>
                <td data-label="${isEn ? 'Reputation' : '信誉'}">${repBadge}</td>
                <td data-label="${isEn ? 'Unlocks' : '解锁能力'}">${unlockInfo}</td>
                <td data-label="${isEn ? 'Bandwidth' : '带宽'}">
                    <div class="node-speed">${speedMbps}</div>
                    <div class="text-xs text-muted">${isEn ? 'Score: ' : '评分: '}${n.score}</div>
                </td>
                <td data-label="${isEn ? 'Actions' : '操作'}" class="text-right">
                    ${isCurrent ?
                        `<div class="action-group">
                            <span class="badge connected"><span class="status-dot"></span> ${isEn ? (appState.currentState?.vpn?.zero_tun ? 'Zero-TUN' : 'Primary') : (appState.currentState?.vpn?.zero_tun ? '当前免TUN' : '当前主连')}</span>
                            <button class="btn btn-danger btn-xs" data-action="disconnectVPN">${isEn ? 'Disconnect' : '断开'}</button>
                        </div>` :
                        (activeTun ?
                            `<div class="action-group">
                                <span class="badge badge-current"><span class="status-dot"></span> ${escapeHtml(activeTun.dev_name)}</span>
                                <button class="btn btn-danger btn-xs" data-action="stopTunnel" data-args="${jsonAttr([activeTun.id])}">${isEn ? 'Disconnect' : '断开'}</button>
                            </div>` :
                            `<div class="action-group">
                                <button class="btn btn-xs" data-action="connectToNode" data-args="${jsonAttr([n.id, 'kernel'])}" title="${isEn ? 'Connect via OpenVPN (kernel TUN)' : '内核 OpenVPN 直连'}">${isEn ? 'Kernel' : '内核连'}</button>
                                <button class="btn btn-outline btn-xs" data-action="connectToNode" data-args="${jsonAttr([n.id, 'sstp'])}" title="${isEn ? 'Zero-TUN userland direct connect via SSTP 443' : '纯用户态免虚拟网卡直连 (SSTP 443)'}">⚡${isEn ? 'ZeroTUN' : '免TUN'}</button>
                                <button class="btn btn-outline btn-xs" data-action="startNewTunnel" data-args="${jsonAttr([n.id])}" title="${isEn ? 'Launch as new independent tunnel exit' : '启动为新的独立并发出口'}">${isEn ? '+Tunnel' : '+并发'}</button>
                                <button class="btn btn-outline btn-xs btn-icon-danger" data-action="addNodeToBlacklist" data-args="${jsonAttr([n.id, n.ip, n.country_short])}" title="${isEn ? 'Blacklist node for 24h' : '屏蔽/拉黑此节点24小时'}">×</button>
                            </div>`
                        )
                    }
                </td>
            </tr>
        `;
    }).join('');
}

export async function probeCurrentNodes() {
    const search = document.getElementById('search-filter').value.toLowerCase();
    const country = document.getElementById('country-filter').value;
    const ipType = document.getElementById('ip-type-filter').value;

    const filtered = appState.allNodes.filter(n => {
        if (country && n.country_short !== country) return false;
        if (ipType && n.ip_type !== ipType) return false;
        if (search) {
            const cName = getCountryName(n.country_short).toLowerCase();
            const str = `${n.ip} ${n.country_long} ${n.country_short} ${cName} ${n.hostname} ${n.isp || ''} ${n.city || ''} ${n.region || ''}`.toLowerCase();
            if (!str.includes(search)) return false;
        }
        return true;
    });

    const ids = filtered.map(n => n.id);
    if (ids.length === 0) {
        tAlert('当前筛选条件下没有节点', 'No nodes match the selected criteria');
        return;
    }

    const btn = document.getElementById('btn-probe-nodes');
    const oldHtml = btn.innerHTML;
    btn.disabled = true;
    btn.innerHTML = `<svg aria-hidden="true" viewBox="0 0 24 24" class="spin"><path d="M21 12a9 9 0 1 1-6.219-8.56"/></svg> ${localize("测速与解锁检测中", "Probing connectivity and service access")} (${ids.length})...`;
    appendLog({ level: 'INFO', module: 'Action', message: localize(`正在对当前筛选出的 ${ids.length} 个节点并发执行可用性测速与 AI/流媒体解锁同步检测...`, `Checking connectivity and service access for ${ids.length} selected nodes...`) });

    await fetch('/api/nodes/probe', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ node_ids: ids })
    });

    let count = 0;
    const timer = setInterval(async () => {
        count++;
        await fetchNodes();
        await fetchUnlockCache();
        if (count >= 5) {
            clearInterval(timer);
            btn.disabled = false;
            btn.innerHTML = oldHtml;
            appendLog({ level: 'INFO', module: 'Action', message: localize('当前筛选节点可用性与 AI/流媒体解锁检测完成！', 'Node connectivity and service-access checks complete.') });
        }
    }, 700);
}

export async function quickConnect(mode = 'kernel') {
    const isUserland = mode === 'sstp' || mode === 'userland';
    appendLog({ level: 'INFO', module: 'Action', message: isUserland ? localize('正在请求快速免 TUN 直连最优节点 (SSTP 443)...', 'Requesting the best user-space SSTP node (port 443)...') : localize('正在请求快速连接最优节点 (内核模式)...', 'Requesting the best node (kernel mode)...') });
    await fetch('/api/connect', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ mode: isUserland ? 'sstp' : 'kernel' })
    });
    fetchStatus();
}

export async function connectToNode(nodeId, mode = 'kernel') {
    const isUserland = mode === 'sstp' || mode === 'userland';
    appendLog({ level: 'INFO', module: 'Action', message: localize(`正在请求切换至节点: ${nodeId} [${isUserland ? '免 TUN 用户态 SSTP' : '内核 OpenVPN'}]...`, `Switching exit node to ${nodeId} [${isUserland ? 'Zero-TUN SSTP' : 'Kernel OpenVPN'}]...`) });
    await fetch('/api/connect', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ node_id: nodeId, mode: isUserland ? 'sstp' : 'kernel' })
    });
    fetchStatus();
}

export async function disconnectVPN() {
    appendLog({ level: 'INFO', module: 'Action', message: localize('正在断开当前 VPN 连接...', 'Disconnecting VPN...') });
    await fetch('/api/disconnect', { method: 'POST' });
    fetchStatus();
}

export async function refreshNodes() {
    appendLog({ level: 'INFO', module: 'Action', message: localize('正在刷新远端节点列表...', 'Refreshing node list...') });
    await fetch('/api/refresh', { method: 'POST' });
    setTimeout(fetchNodes, 1500);
    fetchStatus();
}
