import { appState } from '../state.js';
import { getLanguage, localizeLastMessage, localizeNodeSource, t } from '../i18n-runtime.js';
import { escapeHtml, formatBytes, formatSpeed, getCountryFlagSVG, getCountryName, jsonAttr, renderUnlockBadges } from '../utils.js';

export function renderStatus(data) {
        appState.currentState = data;

        const pubBanner = document.getElementById('public-unauth-banner');
        if (pubBanner) {
            pubBanner.hidden = !data.ui_public_exposure;
        }

        const vpn = data.vpn || {};
        const traffic = data.traffic || {};

        if (data.version) {
            const vStr = 'v' + data.version.replace(/^v/, '');
            const vBadge = document.getElementById('app-version-badge');
            if (vBadge) vBadge.innerText = vStr;
            const sbVer = document.getElementById('sidebar-version-badge');
            if (sbVer) sbVer.innerText = vStr;
        }
        const sbPort = document.getElementById('sidebar-proxy-port');
        if (sbPort && data.proxy_addr) {
            sbPort.innerText = data.proxy_addr.split(':').pop();
        }
        const portBadge = document.getElementById('nav-port-badge');
        if (portBadge) {
            portBadge.innerText = (data.port_rules || []).length || 1;
        }

        // Speed stats
        document.getElementById('stat-down-speed').innerText = formatSpeed(traffic.download_speed_bps || 0);
        document.getElementById('stat-up-speed').innerText = formatSpeed(traffic.upload_speed_bps || 0);
        document.getElementById('stat-total-down').innerText = t('stat.total_down') + formatBytes(traffic.total_download_bytes || 0);
        document.getElementById('stat-total-up').innerText = t('stat.total_up') + formatBytes(traffic.total_upload_bytes || 0);
        document.getElementById('stat-active-conns').innerText = traffic.active_connections || 0;
        const defPort = data.proxy_addr ? data.proxy_addr.split(':').pop() : '7928';
        document.getElementById('stat-proxy-port').innerText = `${t('stat.proxy_port')}${defPort}`;
        const sbBatchPort = document.getElementById('sb-batch-port');
        if (sbBatchPort) sbBatchPort.innerText = defPort;

        // Status Badge & Buttons
        const badge = document.getElementById('conn-badge');
        const btnConnect = document.getElementById('btn-quick-connect');
        const btnConnectSstp = document.getElementById('btn-quick-connect-sstp');
        const btnDisconnect = document.getElementById('btn-disconnect');

        const isEn = getLanguage() === 'en';
        badge.className = 'badge ' + (vpn.status || 'disconnected');
        if (vpn.status === 'connected') {
            const modeTag = vpn.zero_tun ? (isEn ? 'Zero-TUN' : '免TUN') : (isEn ? 'TUN' : '内核');
            badge.innerHTML = `<span class="status-dot"></span> ${t('header.connected')} <span class="badge-mini" style="margin-left:4px;font-size:10px;opacity:0.85">${modeTag}</span>`;
            if (btnConnect) btnConnect.classList.add('hidden');
            if (btnConnectSstp) btnConnectSstp.classList.add('hidden');
            btnDisconnect.classList.remove('hidden');
        } else if (vpn.status === 'connecting' || vpn.status === 'reconnecting') {
            badge.innerHTML = `<span class="status-dot"></span> ${t('header.connecting')}`;
            if (btnConnect) btnConnect.disabled = true;
            if (btnConnectSstp) btnConnectSstp.disabled = true;
            btnDisconnect.classList.remove('hidden');
        } else {
            badge.innerHTML = `<span class="status-dot"></span> ${t('header.disconnected')}`;
            if (btnConnect) { btnConnect.classList.remove('hidden'); btnConnect.disabled = false; }
            if (btnConnectSstp) { btnConnectSstp.classList.remove('hidden'); btnConnectSstp.disabled = false; }
            btnDisconnect.classList.add('hidden');
        }

        // Info card
        let statusLabel = isEn ? ({ connected: 'Connected', connecting: 'Connecting', disconnected: 'Disconnected', reconnecting: 'Reconnecting', error: 'Error' }[vpn.status] || vpn.status || 'Disconnected') : (vpn.status_text || vpn.status || '未连接');
        if (vpn.status === 'connected' && vpn.zero_tun) {
            statusLabel += isEn ? ' · Zero-TUN (SSTP 443)' : ' · 用户态免TUN (SSTP 443)';
        }
        document.getElementById('vpn-status-text').innerText = statusLabel;
        document.getElementById('vpn-node-ip').innerText = vpn.active_node ? `${vpn.active_node.ip}:${vpn.active_node.port}` : '-';

        if (vpn.active_node) {
            let typeBadge = `<span class="badge badge-residential">${isEn ? 'Residential' : '住宅宽带'}</span>`;
            if (vpn.active_node.ip_type === 'hosting') {
                typeBadge = `<span class="badge badge-hosting">${isEn ? 'Datacenter' : '机房网络'}</span>`;
            } else if (vpn.active_node.ip_type === 'mobile') {
                typeBadge = `<span class="badge badge-mobile">${isEn ? 'Mobile' : '移动网络'}</span>`;
            }
            document.getElementById('vpn-node-type').innerHTML = `${typeBadge} ${escapeHtml(vpn.active_node.isp || '')}`;
            const loc = [getCountryName(vpn.active_node.country_short), vpn.active_node.region, vpn.active_node.city].filter(Boolean).join(' · ');
            document.getElementById('vpn-node-country').innerHTML = `
                <span class="flag-box">${getCountryFlagSVG(vpn.active_node.country_short)} ${escapeHtml(loc)} (${escapeHtml(vpn.active_node.country_short)})</span>
            `;
        } else {
            document.getElementById('vpn-node-type').innerText = '-';
            document.getElementById('vpn-node-country').innerText = '-';
        }

        document.getElementById('vpn-proxy-addr').innerText = data.proxy_addr ? `${data.proxy_addr} (HTTP/SOCKS5)` : '-';
        document.getElementById('vpn-last-msg').innerText = localizeLastMessage(vpn.last_message);

        if (vpn.uptime_seconds > 0) {
            const m = Math.floor(vpn.uptime_seconds / 60);
            const s = vpn.uptime_seconds % 60;
            document.getElementById('vpn-uptime').innerText = isEn ? `Uptime: ${m}m ${s}s` : `已连接运行: ${m}分${s}秒`;
        } else {
            document.getElementById('vpn-uptime').innerText = isEn ? 'Uptime: -' : '运行时间: -';
        }

        const totalStr = data.total_node_count && data.total_node_count > data.node_count ? (isEn ? ` (All: ${data.total_node_count})` : ` (历史全库: ${data.total_node_count})`) : '';
        document.getElementById('stat-nodes-count').innerText = `${data.node_count || 0}${totalStr} / ${isEn ? 'Blocked ' : '屏蔽 '}${data.blacklist_count || 0}`;
        document.getElementById('stat-node-source').innerText = `${isEn ? 'Source: ' : '数据源: '}${localizeNodeSource(data.node_source)}`;

        // Render active tunnels strip sorted stably by virtual interface index (tun0, tun1, tun2...)
        const tunnels = (data.tunnels || []).sort((a, b) => {
            const idxA = parseInt((a.dev_name || '').replace(/\D+/g, '')) || 0;
            const idxB = parseInt((b.dev_name || '').replace(/\D+/g, '')) || 0;
            return idxA - idxB;
        });
        document.getElementById('tunnels-count').innerText = tunnels.length;
        const tunListEl = document.getElementById('active-tunnels-list');
        if (tunListEl) {
            if (tunnels.length === 0) {
                tunListEl.innerHTML = `<div class="empty-copy">${isEn ? 'No secondary egress tunnels active. Click "+Tunnel" in node list to enable concurrent multi-egress.' : '暂无独立并发出口，在下方节点列表中点击「+并发」即可多节点同时在线'}</div>`;
            } else {
                tunListEl.innerHTML = tunnels.map(t => {
                    const cCode = t.node ? t.node.country_short : '';
                    const flag = cCode ? getCountryFlagSVG(cCode) : '';
                    const cName = cCode ? getCountryName(cCode) : '';
                    const locStr = cName ? `${cName} · ` : '';
                    const ip = t.node ? `${t.node.ip}:${t.node.port}` : '';
                    const isUp = t.status === 'connected';
                    const badgeClass = isUp ? 'connected' : (t.status === 'connecting' ? 'connecting' : 'disconnected');
                    const unlockBadges = renderUnlockBadges(t.unlock || (t.node ? appState.cachedUnlockMap[t.node.ip] : null));
                    const pingVal = (t.node && t.node.latency_ms > 0) ? t.node.latency_ms : (t.latency_ms > 0 ? t.latency_ms : 0);
                    const pingStr = pingVal > 0 ? `<span class="ping-ok">${pingVal}ms</span>` : '';
                    const statusStr = t.status === 'connected' ? (isEn ? 'Online' : '在线') : (t.status === 'connecting' ? (isEn ? 'Connecting' : '连接中') : (isEn ? 'Disconnected' : '断开'));

                    const isZeroTun = !!t.is_userland || (typeof t.dev_name === 'string' && t.dev_name.startsWith('sstp'));
                    const modeBadge = isZeroTun
                        ? `<span class="badge badge-warning badge-mini" title="${isEn ? 'Userland Zero-TUN direct connection' : '用户态免TUN直连'}">⚡ ${isEn ? 'Zero-TUN' : '免TUN'}</span>`
                        : `<span class="badge badge-info badge-mini" title="${isEn ? 'Kernel TUN virtual network device' : '内核TUN虚拟网卡'}">🔷 ${isEn ? 'TUN' : '内核TUN'}</span>`;

                    let purityBadge = '';
                    if (t.node) {
                        if (t.node.is_residential) {
                            purityBadge = `<span class="badge badge-pure badge-mini" title="${isEn ? 'Pure Residential IP' : '原生住宅家宽'}">🏡 ${isEn ? 'Res' : '住宅'}</span>`;
                        } else if (t.node.is_hosting || t.node.ip_type === 'hosting' || t.node.ip_type === 'datacenter') {
                            purityBadge = `<span class="badge badge-muted badge-mini" title="${isEn ? 'Datacenter / Hosting IP' : '机房/数据中心'}">🏢 ${isEn ? 'DC' : '机房'}</span>`;
                        }
                        if (t.node.risk_score !== undefined && t.node.risk_score >= 0) {
                            const rScore = t.node.risk_score.toFixed(1);
                            if (t.node.risk_score <= 15) {
                                purityBadge += ` <span class="badge badge-success badge-mini" title="${isEn ? 'Low Risk Pure IP' : '低风险纯净IP'}">🟢 ${rScore}</span>`;
                            } else if (t.node.risk_score <= 35) {
                                purityBadge += ` <span class="badge badge-info badge-mini" title="${isEn ? 'Clean IP' : '清洁IP'}">🔵 ${rScore}</span>`;
                            } else {
                                purityBadge += ` <span class="badge badge-danger badge-mini" title="${isEn ? 'Elevated Risk IP' : '较高风险IP'}">🟠 ${rScore}</span>`;
                            }
                        }
                    }

                    return `
                        <div class="tunnel-chip">
                            <strong class="mono text-accent">${escapeHtml(t.dev_name)}</strong>
                            ${modeBadge}
                            <span class="badge ${badgeClass} badge-mini"><span class="status-dot"></span> ${escapeHtml(statusStr)}</span>
                            <span class="flag-box text-note">${flag} ${escapeHtml(locStr)}${escapeHtml(ip)}</span>
                            ${purityBadge}
                            ${pingStr}
                            <div class="tunnel-flags">${unlockBadges}</div>
                            <button class="btn btn-outline btn-xs" data-action="probeTunnelUnlock" data-args="${jsonAttr([t.id])}" title="${isEn ? 'Probe AI & streaming unlock status for this exit' : '探测该出口的AI与流媒体解锁状态'}">${isEn ? 'Unlock' : '测解锁'}</button>
                            <button class="btn btn-danger btn-xs" data-action="stopTunnel" data-args="${jsonAttr([t.id])}">${isEn ? 'Disconnect' : '断开'}</button>
                        </div>
                    `;
                }).join('');
            }
        }
}
