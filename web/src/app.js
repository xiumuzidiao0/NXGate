// Auto-detect secret path prefix from current browser location (e.g. /mysecret or /enter)
        const apiPrefix = (() => {
            const parts = window.location.pathname.split('/').filter(Boolean);
            if (parts.length > 0 && parts[0] !== 'api' && parts[0] !== 'metrics') {
                return '/' + parts[0];
            }
            return '';
        })();
        window.__apiPrefix = apiPrefix;

        // Automatically route all /api/ and /metrics requests through the current secret path prefix
        // so that the browser always sends Basic Auth credentials and avoids cross-path 404s
        const originalFetch = window.fetch;
        window.fetch = async function(input, init) {
            if (typeof input === 'string' && apiPrefix) {
                if (input.startsWith('/api/') || input === '/api' || input.startsWith('/metrics')) {
                    input = apiPrefix + input;
                }
            }
            try {
                const response = await originalFetch(input, init);
                if (response.status >= 500) {
                    showAppAlert(`服务返回 ${response.status}，部分数据可能未更新。`);
                } else if (response.ok) {
                    clearAppAlert();
                }
                return response;
            } catch (error) {
                showAppAlert('无法连接管理服务，请检查网络或服务状态。');
                throw error;
            }
        };

        let allNodes = [];
        let currentState = null;
        let cachedUnlockMap = {};
        let activeQuickFilter = 'all';

import { I18N } from './i18n.js';

        function getLanguage() {
            try {
                const saved = localStorage.getItem('nxgate_lang');
                if (saved === 'en' || saved === 'zh') return saved;
            } catch(e) {}
            return 'zh';
        }

        function t(key, fallback) {
            const lang = getLanguage();
            if (I18N[lang] && I18N[lang][key] !== undefined) {
                return I18N[lang][key];
            }
            if (I18N.zh && I18N.zh[key] !== undefined) {
                return I18N.zh[key];
            }
            return fallback !== undefined ? fallback : key;
        }

        function tAlert(zhMsg, enMsg) {
            alert(getLanguage() === 'en' ? enMsg : zhMsg);
        }

        function tConfirm(zhMsg, enMsg) {
            return confirm(getLanguage() === 'en' ? enMsg : zhMsg);
        }

        function tPrompt(zhMsg, enMsg, defaultVal) {
            return prompt(getLanguage() === 'en' ? enMsg : zhMsg, defaultVal);
        }

        function applyLanguage(lang = getLanguage(), container = document) {
            container.querySelectorAll('[data-i18n]').forEach(el => {
                const text = t(el.dataset.i18n);
                if (text) el.textContent = text;
            });
            container.querySelectorAll('[data-i18n-placeholder]').forEach(el => {
                const text = t(el.dataset.i18nPlaceholder);
                if (text) el.placeholder = text;
            });
            container.querySelectorAll('[data-i18n-title]').forEach(el => {
                const text = t(el.dataset.i18nTitle);
                if (text) el.title = text;
            });
            const langBtnText = document.getElementById('lang-btn-text');
            if (langBtnText) {
                langBtnText.textContent = lang === 'zh' ? 'English' : '中文';
            }
            const langSelect = document.getElementById('cfg-ui-lang');
            if (langSelect) {
                langSelect.value = lang;
            }
            document.documentElement.lang = (lang === 'zh' ? 'zh-CN' : 'en');
        }

        function toggleLanguage() {
            const nextLang = getLanguage() === 'zh' ? 'en' : 'zh';
            setLanguage(nextLang);
        }

        function onUiLanguageChanged(event) {
            const newLang = event?.target?.value || 'zh';
            setLanguage(newLang);
        }

        function setLanguage(lang) {
            try {
                localStorage.setItem('nxgate_lang', lang);
            } catch(e) {}
            applyLanguage(lang);
            const hash = (window.location.hash || '#dashboard').replace(/^#/, '');
            switchView(hash);
            if (currentState) {
                const traffic = currentState.traffic || {};
                const downEl = document.getElementById('stat-total-down');
                if (downEl) downEl.innerText = t('stat.total_down') + formatBytes(traffic.total_download_bytes || 0);
                const upEl = document.getElementById('stat-total-up');
                if (upEl) upEl.innerText = t('stat.total_up') + formatBytes(traffic.total_upload_bytes || 0);
                const portEl = document.getElementById('stat-proxy-port');
                if (portEl) portEl.innerText = `${t('stat.proxy_port')}${currentState.proxy_addr ? currentState.proxy_addr.split(':').pop() : '7928'}`;
                const vpn = currentState.vpn || {};
                const badge = document.getElementById('conn-badge');
                if (badge) {
                    if (vpn.status === 'connected') badge.innerHTML = `<span class="status-dot"></span> ${t('header.connected')}`;
                    else if (vpn.status === 'connecting' || vpn.status === 'reconnecting') badge.innerHTML = `<span class="status-dot"></span> ${t('header.connecting')}`;
                    else badge.innerHTML = `<span class="status-dot"></span> ${t('header.disconnected')}`;
                }
            }
            if (allNodes.length > 0) renderNodes(allNodes);
            if (singBoxOverview) renderSingBox(singBoxOverview);
            if (currentDynamicGroups) renderDynamicGroups();
            if (currentPortRules) renderPortRules();
            if (currentState) fetchStatus();
            showToast(lang === 'en' ? 'Language switched to English' : '已切换至简体中文');
        }

        const countryNames = {
            JP: '日本', US: '美国', KR: '韩国', TW: '台湾', HK: '香港',
            SG: '新加坡', GB: '英国', DE: '德国', FR: '法国', CA: '加拿大',
            AU: '澳大利亚', VN: '越南', TH: '泰国', MY: '马来西亚', IN: '印度',
            RU: '俄罗斯', NL: '荷兰', BR: '巴西', PH: '菲律宾', ID: '印尼'
        };

        const countryNamesEn = {
            JP: 'Japan', US: 'United States', KR: 'South Korea', TW: 'Taiwan', HK: 'Hong Kong',
            SG: 'Singapore', GB: 'United Kingdom', DE: 'Germany', FR: 'France', CA: 'Canada',
            AU: 'Australia', VN: 'Vietnam', TH: 'Thailand', MY: 'Malaysia', IN: 'India',
            RU: 'Russia', NL: 'Netherlands', BR: 'Brazil', PH: 'Philippines', ID: 'Indonesia'
        };

        const flagSVGs = {
            JP: '<svg aria-hidden="true" class="flag-img" viewBox="0 0 900 600"><rect fill="#fff" width="900" height="600"/><circle fill="#bc002d" cx="450" cy="300" r="180"/></svg>',
            US: '<svg aria-hidden="true" class="flag-img" viewBox="0 0 7410 3900"><rect fill="#b22234" width="7410" height="3900"/><path stroke="#fff" stroke-width="300" d="M0,450H7410M0,1050H7410M0,1650H7410M0,2250H7410M0,2850H7410M0,3450H7410"/><rect fill="#3c3b6e" width="2964" height="2100"/></svg>',
            KR: '<svg aria-hidden="true" class="flag-img" viewBox="0 0 900 600"><rect fill="#fff" width="900" height="600"/><path fill="#cd2e3a" d="M450,150a150,150 0 0,1 0,300a75,75 0 0,1 0,-150z"/><path fill="#0047a0" d="M450,450a150,150 0 0,1 0,-300a75,75 0 0,1 0,150z"/></svg>',
            TW: '<svg aria-hidden="true" class="flag-img" viewBox="0 0 900 600"><rect fill="#fe0000" width="900" height="600"/><rect fill="#000095" width="450" height="300"/><circle fill="#fff" cx="225" cy="150" r="75"/></svg>',
            HK: '<svg aria-hidden="true" class="flag-img" viewBox="0 0 900 600"><rect fill="#c8102e" width="900" height="600"/><circle fill="#fff" cx="450" cy="300" r="110" opacity="0.9"/></svg>',
            SG: '<svg aria-hidden="true" class="flag-img" viewBox="0 0 900 600"><rect fill="#ed2939" width="900" height="300"/><rect fill="#fff" y="300" width="900" height="300"/><circle fill="#fff" cx="225" cy="150" r="100"/><circle fill="#ed2939" cx="265" cy="150" r="100"/></svg>',
            DE: '<svg aria-hidden="true" class="flag-img" viewBox="0 0 5 3"><rect width="5" height="1" y="0" fill="#000"/><rect width="5" height="1" y="1" fill="#D00"/><rect width="5" height="1" y="2" fill="#FFCE00"/></svg>',
            FR: '<svg aria-hidden="true" class="flag-img" viewBox="0 0 900 600"><rect fill="#002395" width="300" height="600"/><rect fill="#fff" x="300" width="300" height="600"/><rect fill="#ed2939" x="600" width="300" height="600"/></svg>',
            GB: '<svg aria-hidden="true" class="flag-img" viewBox="0 0 60 30"><rect fill="#012169" width="60" height="30"/><path d="M0,0 L60,30 M60,0 L0,30" stroke="#fff" stroke-width="6"/><path d="M0,0 L60,30 M60,0 L0,30" stroke="#C8102E" stroke-width="4"/><path d="M30,0 v30 M0,15 h60" stroke="#fff" stroke-width="10"/><path d="M30,0 v30 M0,15 h60" stroke="#C8102E" stroke-width="6"/></svg>',
            CA: '<svg aria-hidden="true" class="flag-img" viewBox="0 0 900 450"><rect fill="#f00" width="225" height="450"/><rect fill="#fff" x="225" width="450" height="450"/><rect fill="#f00" x="675" width="225" height="450"/><polygon fill="#f00" points="450,112 470,170 515,160 480,200 500,240 450,225 400,240 420,200 385,160 430,170"/></svg>',
            VN: '<svg aria-hidden="true" class="flag-img" viewBox="0 0 900 600"><rect fill="#da251d" width="900" height="600"/><polygon fill="#ffff00" points="450,140 487,254 607,254 510,324 547,438 450,368 353,438 390,324 293,254 413,254"/></svg>',
            TH: '<svg aria-hidden="true" class="flag-img" viewBox="0 0 900 600"><rect fill="#a51931" width="900" height="600"/><rect fill="#f4f5f8" y="100" width="900" height="400"/><rect fill="#2d2a4a" y="200" width="900" height="200"/></svg>'
        };

        function getCountryFlagSVG(code) {
            code = (code || '').toUpperCase();
            if (flagSVGs[code]) return flagSVGs[code];
            return '<svg aria-hidden="true" class="flag-img flag-fallback" viewBox="0 0 24 24"><circle cx="12" cy="12" r="10" stroke="#64748b" fill="none"/><line x1="2" y1="12" x2="22" y2="12" stroke="#64748b"/></svg>';
        }

        function getCountryName(code) {
            code = (code || '').toUpperCase();
            if (getLanguage() === 'en') {
                return countryNamesEn[code] || code;
            }
            return countryNames[code] || code;
        }

        function escapeHtml(value) {
            return String(value ?? '').replace(/[&<>"']/g, c => ({
                '&': '&amp;',
                '<': '&lt;',
                '>': '&gt;',
                '"': '&quot;',
                "'": '&#39;'
            })[c]);
        }

        function jsonAttr(value) {
            return escapeHtml(JSON.stringify(value));
        }

        function safeLogLevel(level) {
            const value = String(level || 'info').toLowerCase();
            return ['info', 'warning', 'error'].includes(value) ? value : 'info';
        }

        function formatBytes(bytes) {
            if (!bytes || bytes === 0) return '0 B';
            const k = 1024;
            const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
            const i = Math.floor(Math.log(bytes) / Math.log(k));
            return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
        }

        function formatSpeed(bps) { return formatBytes(bps) + '/s'; }

        function appendLog(entry) {
            const term = document.getElementById('terminal');
            const row = document.createElement('div');
            row.className = 'log-line';
            const timeStr = entry.timestamp ? new Date(entry.timestamp).toLocaleTimeString() : new Date().toLocaleTimeString();
            row.innerHTML = `<span class="log-time">[${escapeHtml(timeStr)}]</span> <span class="log-level-${safeLogLevel(entry.level)}">[${escapeHtml(entry.module || 'System')}]</span> ${escapeHtml(entry.message)}`;
            term.appendChild(row);
            term.scrollTop = term.scrollHeight;
        }

        function clearLogs() { document.getElementById('terminal').innerHTML = ''; }

        function renderUnlockBadges(u) {
            if (!u) return '<span  class="text-xs text-muted">-</span>';
            const isEn = getLanguage() === 'en';
            const badges = [];
            if (u.openai === 'unlocked') badges.push(`<span class="badge unlock-badge unlock-open" title="${isEn ? 'OpenAI / ChatGPT Available' : 'OpenAI / ChatGPT 解锁正常'}">${isEn ? 'GPT OK' : 'GPT 可用'}</span>`);
            else if (u.openai === 'blocked') badges.push(`<span class="badge unlock-badge unlock-blocked" title="${isEn ? 'OpenAI Blocked' : 'OpenAI 阻断拦截'}">${isEn ? 'GPT Block' : 'GPT 阻断'}</span>`);

            if (u.claude === 'unlocked') badges.push(`<span class="badge unlock-badge unlock-open" title="${isEn ? 'Claude / Anthropic Available' : 'Claude / Anthropic 访问正常'}">${isEn ? 'Claude OK' : 'Claude 可用'}</span>`);
            else if (u.claude === 'blocked') badges.push(`<span class="badge unlock-badge unlock-blocked" title="${isEn ? 'Claude Blocked' : 'Claude 风控拦截'}">${isEn ? 'Claude Block' : 'Claude 阻断'}</span>`);

            if (u.gemini === 'unlocked') badges.push(`<span class="badge unlock-badge unlock-open" title="${isEn ? 'Google Gemini Available' : 'Google Gemini 访问正常'}">${isEn ? 'Gemini OK' : 'Gemini 可用'}</span>`);
            else if (u.gemini === 'blocked') badges.push(`<span class="badge unlock-badge unlock-blocked" title="${isEn ? 'Google Gemini Blocked' : 'Google Gemini 限制访问'}">${isEn ? 'Gemini Block' : 'Gemini 阻断'}</span>`);

            if (u.netflix === 'unlocked') badges.push(`<span class="badge unlock-badge unlock-warn" title="${isEn ? 'Netflix Native Streaming Available' : 'Netflix 原生流媒体解锁'}">${isEn ? 'NF OK' : 'NF 可用'}</span>`);
            else if (u.netflix === 'blocked') badges.push(`<span class="badge unlock-badge unlock-blocked" title="${isEn ? 'Netflix Restricted' : 'Netflix 限制访问'}">${isEn ? 'NF Restrict' : 'NF 限制'}</span>`);

            if (u.google === 'unlocked') badges.push(`<span class="badge unlock-badge text-accent" title="${isEn ? 'Google Clean Search' : 'Google Search 干净无验证码'}">${isEn ? 'Google OK' : 'Google 可用'}</span>`);
            else if (u.google === 'blocked') badges.push(`<span class="badge unlock-badge unlock-blocked" title="${isEn ? 'Google Captcha Required' : 'Google 出现验证码异常'}">${isEn ? 'Google Captcha' : 'Google 验证'}</span>`);

            if (u.cloudflare === 'unlocked') badges.push(`<span class="badge unlock-badge text-accent" title="${isEn ? 'Cloudflare 204 OK' : 'Cloudflare 204 出海连通正常'}">${isEn ? 'CF OK' : 'CF 连通'}</span>`);
            else if (u.cloudflare === 'blocked') badges.push(`<span class="badge unlock-badge unlock-blocked" title="${isEn ? 'Cloudflare WAF Blocked' : 'Cloudflare 遭遇风控阻断'}">${isEn ? 'CF Block' : 'CF 阻断'}</span>`);

            if (badges.length === 0) return `<span class="text-xs text-muted">${isEn ? 'Not Probed' : '未检测'}</span>`;
            const tag = u.is_probed ? `<span class="badge badge-system badge-mini" title="${isEn ? 'Probed via virtual NIC' : '经虚拟网卡物理流量实测'}">${isEn ? 'Probed' : '实测'}</span> ` : '';
            return `<div class="unlock-badges">${tag}${badges.join('')}</div>`;
        }

        async function fetchUnlockCache() {
            try {
                const res = await fetch('/api/unlock');
                if (res.ok) {
                    cachedUnlockMap = await res.json() || {};
                    renderNodes();
                }
            } catch(e) {}
        }

        async function probeTunnelUnlock(tunnelId) {
            appendLog({ level: 'INFO', module: 'Unlock', message: `正在对隧道 ${tunnelId} 启动流媒体与 AI 解锁探测...` });
            try {
                await fetch('/api/unlock/probe', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ tunnel_id: tunnelId })
                });
                setTimeout(async () => {
                    await fetchUnlockCache();
                    fetchStatus();
                }, 3500);
            } catch(err) {
                tAlert('探测请求失败: ' + err, 'Probe request failed: ' + err);
            }
        }

        async function testTelegramAlert() {
            try {
                const res = await fetch('/api/telegram/test', { method: 'POST' });
                const ret = await res.json();
                if (res.ok) tAlert(ret.message || '测试消息发送成功！', ret.message || 'Test alert sent successfully!');
                else tAlert('测试失败: ' + (ret.error || '未知错误'), 'Test alert failed: ' + (ret.error || 'Unknown error'));
            } catch(err) {
                alert('请求失败: ' + err);
            }
        }

        async function fetchStatus() {
            try {
                const res = await fetch('/api/status');
                if (!res.ok) return;
                const data = await res.json();
                currentState = data;

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
                document.getElementById('stat-proxy-port').innerText = `${t('stat.proxy_port')}${data.proxy_addr ? data.proxy_addr.split(':').pop() : '7928'}`;

                // Status Badge & Buttons
                const badge = document.getElementById('conn-badge');
                const btnConnect = document.getElementById('btn-quick-connect');
                const btnDisconnect = document.getElementById('btn-disconnect');

                badge.className = 'badge ' + (vpn.status || 'disconnected');
                if (vpn.status === 'connected') {
                    badge.innerHTML = `<span class="status-dot"></span> ${t('header.connected')}`;
                    btnConnect.classList.add('hidden');
                    btnDisconnect.classList.remove('hidden');
                } else if (vpn.status === 'connecting' || vpn.status === 'reconnecting') {
                    badge.innerHTML = `<span class="status-dot"></span> ${t('header.connecting')}`;
                    btnConnect.disabled = true;
                    btnDisconnect.classList.remove('hidden');
                } else {
                    badge.innerHTML = `<span class="status-dot"></span> ${t('header.disconnected')}`;
                    btnConnect.classList.remove('hidden');
                    btnConnect.disabled = false;
                    btnDisconnect.classList.add('hidden');
                }

                // Info card
                const isEn = getLanguage() === 'en';
                document.getElementById('vpn-status-text').innerText = vpn.status === 'connected' ? (isEn ? 'Connected' : '已连接') : (vpn.status_text || vpn.status || (isEn ? 'Disconnected' : '未连接'));
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
                document.getElementById('vpn-last-msg').innerText = vpn.last_message || (isEn ? 'Service running normally' : '服务正常运行中');

                if (vpn.uptime_seconds > 0) {
                    const m = Math.floor(vpn.uptime_seconds / 60);
                    const s = vpn.uptime_seconds % 60;
                    document.getElementById('vpn-uptime').innerText = isEn ? `Uptime: ${m}m ${s}s` : `已连接运行: ${m}分${s}秒`;
                } else {
                    document.getElementById('vpn-uptime').innerText = isEn ? 'Uptime: -' : '运行时间: -';
                }

                const totalStr = data.total_node_count && data.total_node_count > data.node_count ? (isEn ? ` (All: ${data.total_node_count})` : ` (历史全库: ${data.total_node_count})`) : '';
                document.getElementById('stat-nodes-count').innerText = `${data.node_count || 0}${totalStr} / ${isEn ? 'Blocked ' : '屏蔽 '}${data.blacklist_count || 0}`;
                document.getElementById('stat-node-source').innerText = `${isEn ? 'Source: ' : '数据源: '}${data.node_source || (isEn ? 'Unknown' : '未知')}`;

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
                            const unlockBadges = renderUnlockBadges(t.unlock || (t.node ? cachedUnlockMap[t.node.ip] : null));
                            const pingVal = (t.node && t.node.latency_ms > 0) ? t.node.latency_ms : (t.latency_ms > 0 ? t.latency_ms : 0);
                            const pingStr = pingVal > 0 ? `<span class="ping-ok">${pingVal}ms</span>` : '';
                            const statusStr = t.status === 'connected' ? (isEn ? 'Online' : '在线') : (t.status === 'connecting' ? (isEn ? 'Connecting' : '连接中') : (isEn ? 'Disconnected' : '断开'));
                            return `
                                <div class="tunnel-chip">
                                    <strong class="mono text-accent">${escapeHtml(t.dev_name)}</strong>
                                    <span class="badge ${badgeClass} badge-mini"><span class="status-dot"></span> ${escapeHtml(statusStr)}</span>
                                    <span class="flag-box text-note">${flag} ${escapeHtml(locStr)}${escapeHtml(ip)}</span>
                                    ${pingStr}
                                    <div class="tunnel-flags">${unlockBadges}</div>
                                    <button class="btn btn-outline btn-xs" data-action="probeTunnelUnlock" data-args="${jsonAttr([t.id])}" title="${isEn ? 'Probe AI & streaming unlock status for this exit' : '探测该出口的AI与流媒体解锁状态'}">${isEn ? 'Unlock' : '测解锁'}</button>
                                    <button class="btn btn-danger btn-xs" data-action="stopTunnel" data-args="${jsonAttr([t.id])}">${isEn ? 'Disconnect' : '断开'}</button>
                                </div>
                            `;
                        }).join('');
                    }
                }

            } catch (err) {
                console.error("fetch status error:", err);
            }
        }

        async function fetchNodes() {
            try {
                const res = await fetch('/api/nodes');
                if (!res.ok) return;
                allNodes = await res.json();
                const nBadge = document.getElementById('nav-node-badge');
                if (nBadge) {
                    nBadge.innerText = allNodes.length;
                }
                updateCountryFilter();
                renderNodes();
                document.getElementById('nodes-table-wrap')?.setAttribute('aria-busy', 'false');
            } catch (err) {
                console.error("fetch nodes error:", err);
                document.getElementById('nodes-table-wrap')?.setAttribute('aria-busy', 'false');
            }
        }

        function updateCountryFilter() {
            const select = document.getElementById('country-filter');
            const curr = select.value;
            const countries = [...new Set(allNodes.map(n => n.country_short))].filter(Boolean).sort();

            select.innerHTML = '<option value="">全部国家/地区</option>' +
                countries.map(c => `<option value="${escapeHtml(c)}">${escapeHtml(getCountryName(c))} (${escapeHtml(c)})</option>`).join('');
            select.value = curr;
        }

        let currentSort = 'latency_asc';

        function toggleSort(field) {
            if (field === 'latency') {
                currentSort = currentSort === 'latency_asc' ? 'score_desc' : 'latency_asc';
            } else if (field === 'speed') {
                currentSort = currentSort === 'speed_desc' ? 'latency_asc' : 'speed_desc';
            } else if (field === 'score') {
                currentSort = currentSort === 'score_desc' ? 'latency_asc' : 'score_desc';
            }
            document.getElementById('sort-filter').value = currentSort;
            renderNodes();
        }

        function selectQuickFilter(filter) {
            activeQuickFilter = filter;
            document.querySelectorAll('.chip').forEach(c => c.classList.remove('active'));
            const el = document.getElementById('chip-' + filter);
            if (el) el.classList.add('active');
            renderNodes();
        }

        async function toggleFavorite(nodeId) {
            try {
                const res = await fetch('/api/nodes/favorite', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ node_id: nodeId })
                });
                const ret = await res.json();
                if (res.ok) {
                    const node = allNodes.find(n => n.id === nodeId);
                    if (node) {
                        node.is_favorite = ret.is_favorite;
                        renderNodes();
                    }
                }
            } catch (err) {}
        }

        function renderNodes() {
            const tbody = document.getElementById('nodes-tbody');
            const search = document.getElementById('search-filter').value.toLowerCase();
            const country = document.getElementById('country-filter').value;
            const ipType = document.getElementById('ip-type-filter').value;
            const sortMode = document.getElementById('sort-filter').value;

            const favCount = allNodes.filter(n => n.is_favorite).length;
            const favCountEl = document.getElementById('fav-count');
            if (favCountEl) favCountEl.innerText = favCount;

            const filtered = allNodes.filter(n => {
                // Quick Chips Filter Logic
                if (activeQuickFilter === 'fav' && !n.is_favorite) return false;
                if (activeQuickFilter === 'res' && n.ip_type !== 'residential') return false;
                if (activeQuickFilter === 'host' && n.ip_type !== 'hosting') return false;
                if (activeQuickFilter === 'fast' && (n.latency_ms <= 0 || n.latency_ms >= 150)) return false;
                if (activeQuickFilter === 'ai') {
                    const u = (cachedUnlockMap && cachedUnlockMap[n.ip]) || n.unlock;
                    if (!u || u.openai !== 'unlocked' || u.claude !== 'unlocked' || u.gemini !== 'unlocked') return false;
                }
                if (activeQuickFilter === 'gpt') {
                    const u = (cachedUnlockMap && cachedUnlockMap[n.ip]) || n.unlock;
                    if (!u || u.openai !== 'unlocked') return false;
                }
                if (activeQuickFilter === 'claude') {
                    const u = (cachedUnlockMap && cachedUnlockMap[n.ip]) || n.unlock;
                    if (!u || u.claude !== 'unlocked') return false;
                }
                if (activeQuickFilter === 'gemini') {
                    const u = (cachedUnlockMap && cachedUnlockMap[n.ip]) || n.unlock;
                    if (!u || u.gemini !== 'unlocked') return false;
                }
                if (activeQuickFilter === 'nf') {
                    const u = (cachedUnlockMap && cachedUnlockMap[n.ip]) || n.unlock;
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
                    const aAvail = a.latency_ms > 0 ? 1 : (a.latency_ms === -1 ? -1 : 0);
                    const bAvail = b.latency_ms > 0 ? 1 : (b.latency_ms === -1 ? -1 : 0);
                    if (aAvail !== bAvail) return bAvail - aAvail;
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
            document.getElementById('filtered-count').innerText = `${isEn ? 'Showing ' : '已筛选出 '}${filtered.length} / ${allNodes.length} ${isEn ? 'nodes' : '个节点'}`;

            if (filtered.length === 0) {
                tbody.innerHTML = `<tr><td colspan="9" class="empty-state">${isEn ? 'No nodes match the selected criteria' : '未找到匹配条件的节点'}</td></tr>`;
                return;
            }

            tbody.innerHTML = filtered.map(n => {
                const isCurrent = currentState && currentState.vpn && currentState.vpn.active_node_id === n.id;
                const activeTun = (currentState && currentState.tunnels) ? currentState.tunnels.find(t => t.node && (t.node.id === n.id || t.node.ip === n.ip)) : null;

                let latencyBadge = '';
                if (n.latency_ms > 0) {
                    latencyBadge = `<span class="badge latency-badge latency-available"><span class="status-dot"></span> ${isEn ? 'OK ' : '可用 '}${n.latency_ms} ms</span>`;
                } else if (n.latency_ms === -1) {
                    latencyBadge = `<span class="badge latency-badge latency-timeout"><span class="status-dot"></span> ${isEn ? 'Timeout' : '超时不可达'}</span>`;
                } else {
                    latencyBadge = `<span class="badge latency-badge latency-unknown"><span class="status-dot"></span> ${isEn ? 'Untested' : '未测速'}</span>`;
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

                const unlockData = (cachedUnlockMap && cachedUnlockMap[n.ip]) || n.unlock;
                const unlockInfo = unlockData ? renderUnlockBadges(unlockData) : `<span class="text-xs text-muted">${isEn ? 'Not Probed' : '未检测'}</span>`;

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
                        <td data-label="${isEn ? 'Type' : '网络类型'}">${typeTag}${ispInfo}</td>
                        <td data-label="${isEn ? 'Latency' : '延迟'}">${latencyBadge}</td>
                        <td data-label="${isEn ? 'Reputation' : '信誉'}">${repBadge}</td>
                        <td data-label="${isEn ? 'Unlocks' : '解锁能力'}">${unlockInfo}</td>
                        <td data-label="${isEn ? 'Bandwidth' : '带宽'}">
                            <div class="node-speed">${speedMbps}</div>
                            <div class="text-xs text-muted">${isEn ? 'Score: ' : '评分: '}${n.score}</div>
                        </td>
                        <td data-label="${isEn ? 'Actions' : '操作'}" class="text-right">
                            ${isCurrent ?
                                `<span class="badge connected"><span class="status-dot"></span> ${isEn ? 'Primary' : '当前主连'}</span>` :
                                (activeTun ?
                                    `<div class="action-group">
                                        <span class="badge badge-current"><span class="status-dot"></span> ${escapeHtml(activeTun.dev_name)}</span>
                                        <button class="btn btn-danger btn-xs" data-action="stopTunnel" data-args="${jsonAttr([activeTun.id])}">${isEn ? 'Disconnect' : '断开'}</button>
                                    </div>` :
                                    `<div class="action-group">
                                        <button class="btn btn-xs" data-action="connectToNode" data-args="${jsonAttr([n.id])}" title="${isEn ? 'Set as primary gateway exit' : '设置为主网关出口'}">${isEn ? 'Primary' : '主连'}</button>
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

        async function probeCurrentNodes() {
            const search = document.getElementById('search-filter').value.toLowerCase();
            const country = document.getElementById('country-filter').value;
            const ipType = document.getElementById('ip-type-filter').value;

            const filtered = allNodes.filter(n => {
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
            btn.innerHTML = `<svg aria-hidden="true" viewBox="0 0 24 24" class="spin"><path d="M21 12a9 9 0 1 1-6.219-8.56"/></svg> 测速与解锁检测中 (${ids.length}个)...`;
            appendLog({ level: 'INFO', module: 'Action', message: `正在对当前筛选出的 ${ids.length} 个节点并发执行可用性测速与 AI/流媒体解锁同步检测...` });

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
                    appendLog({ level: 'INFO', module: 'Action', message: '当前筛选节点可用性与 AI/流媒体解锁检测完成！' });
                }
            }, 700);
        }

        async function quickConnect() {
            appendLog({ level: 'INFO', module: 'Action', message: '正在请求快速连接最优节点...' });
            await fetch('/api/connect', { method: 'POST' });
            fetchStatus();
        }

        async function connectToNode(nodeId) {
            appendLog({ level: 'INFO', module: 'Action', message: `正在请求切换至节点: ${nodeId}...` });
            await fetch('/api/connect', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ node_id: nodeId })
            });
            fetchStatus();
        }

        async function disconnectVPN() {
            appendLog({ level: 'INFO', module: 'Action', message: '正在断开当前 VPN 连接...' });
            await fetch('/api/disconnect', { method: 'POST' });
            fetchStatus();
        }

        async function refreshNodes() {
            appendLog({ level: 'INFO', module: 'Action', message: '正在刷新远端节点列表...' });
            await fetch('/api/refresh', { method: 'POST' });
            setTimeout(fetchNodes, 1500);
            fetchStatus();
        }

        // Settings Modal & Tabs
        function switchSettingsTab(tabKey) {
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

        async function loadSettingsForm() {
            try {
                const res = await fetch('/api/settings');
                if (!res.ok) return;
                const data = await res.json();
                document.getElementById('cfg-web-port').value = data.ui_port || 8787;
                document.getElementById('cfg-web-path').value = data.ui_path || 'nxgate';
                document.getElementById('cfg-username').value = data.ui_username || 'admin';
                document.getElementById('cfg-password').value = '';
                document.getElementById('cfg-proxy-port').value = data.proxy_port || 7928;
                document.getElementById('cfg-tg-token').value = data.telegram_bot_token || '';
                document.getElementById('cfg-tg-chatid').value = data.telegram_chat_id || '';
                const agePubEl = document.getElementById('cfg-age-pubkey');
                if (agePubEl) agePubEl.value = data.age_public_key || '';
                switchSettingsTab('base');
            } catch (err) {
                console.warn('获取系统配置失败:', err);
            }
        }

        async function openSettingsModal() {
            switchView('settings');
        }

        function closeSettingsModal() { switchView('dashboard'); }

        function openSystemPrimaryConfig() {
            switchView('matrix');
            switchMatrixTab('groups');
            editDynamicGroup('system-primary');
        }

        function randomPath() {
            const chars = 'abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789';
            let res = '';
            for (let i = 0; i < 8; i++) res += chars.charAt(Math.floor(Math.random() * chars.length));
            document.getElementById('cfg-web-path').value = res;
        }

        function randomPassword() {
            const chars = 'abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*';
            let res = '';
            for (let i = 0; i < 14; i++) res += chars.charAt(Math.floor(Math.random() * chars.length));
            document.getElementById('cfg-password').value = res;
        }

        async function saveSettings(e) {
            e.preventDefault();
            const webPort = parseInt(document.getElementById('cfg-web-port').value);
            const webPath = document.getElementById('cfg-web-path').value.trim().replace(/^\/+|\/+$/g, '');
            const user = document.getElementById('cfg-username').value.trim();
            const pass = document.getElementById('cfg-password').value.trim();
            const proxyPort = parseInt(document.getElementById('cfg-proxy-port').value);

            if (webPort === proxyPort) {
                tAlert('错误: Web 管理端口不能与本地代理端口相同！', 'Error: Web console port cannot be identical to proxy port!');
                return;
            }

            const tgToken = document.getElementById('cfg-tg-token').value.trim();
            const tgChatID = document.getElementById('cfg-tg-chatid').value.trim();
            const agePub = (document.getElementById('cfg-age-pubkey')?.value || '').trim();

            const payload = {
                ui_port: webPort,
                ui_path: webPath,
                ui_username: user,
                proxy_port: proxyPort,
                telegram_bot_token: tgToken,
                telegram_chat_id: tgChatID,
                age_public_key: agePub,
                age_encrypt_enabled: !!agePub
            };
            if (pass) payload.ui_password = pass;

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

                tAlert('配置修改成功并已生效！', 'Configuration updated and applied successfully!');
                closeSettingsModal();

                const curPort = window.location.port || (window.location.protocol === 'https:' ? '443' : '80');
                if (webPort.toString() !== curPort || !window.location.pathname.includes(webPath)) {
                    const newUrl = `${window.location.protocol}//${window.location.hostname}:${webPort}/${webPath}`;
                    tAlert(`Web 访问入口已变更，即将跳转至新地址:\n${newUrl}`, `Web console URL changed. Redirecting to new address:\n${newUrl}`);
                    window.location.href = newUrl;
                } else {
                    fetchStatus();
                }
            } catch (err) {
                alert('网络请求失败: ' + err);
            }
        }

        // Setup SSE connection
        function setupSSE() {
            const es = new EventSource((apiPrefix || '') + '/api/events');
            es.onopen = () => {
                appendLog({ level: 'INFO', module: 'SSE', message: '已成功连通服务器实时推流总线' });
            };
            es.addEventListener('log', (e) => {
                try {
                    const entry = JSON.parse(e.data);
                    appendLog(entry);
                } catch(err){}
            });
            es.addEventListener('status', (e) => {
                try {
                    const data = JSON.parse(e.data);
                    currentState = data;
                    fetchStatus();
                } catch(err){}
            });
            es.onerror = () => {};
        }

        // Port Matrix Modal & Dynamic Groups
        let currentPortRules = [];
        let currentDynamicGroups = [];
        let editingPort = null;
        let editingGroupId = null;

        function switchMatrixTab(tabKey) {
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

        async function startNewTunnel(nodeId) {
            appendLog({ level: 'INFO', module: 'Tunnel', message: `正在启动并发独立出口隧道: ${nodeId}...` });
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
                    appendLog({ level: 'INFO', module: 'Tunnel', message: `新隧道启动成功: ${ret.tunnel.id} (${ret.tunnel.dev_name})` });
                    fetchStatus();
                }
            } catch (err) {
                alert('请求失败: ' + err);
            }
        }

        async function stopTunnel(tunnelId) {
            if (!tConfirm('确认断开并释放该并发隧道吗？', 'Are you sure you want to disconnect and release this exit tunnel?')) return;
            try {
                await fetch('/api/tunnels/stop', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ tunnel_id: tunnelId })
                });
                fetchStatus();
            } catch (err) {
                alert('断开失败: ' + err);
            }
        }

        async function openPortMatrixModal() {
            switchView('matrix');
        }

        function closePortMatrixModal() { switchView('dashboard'); }

        async function fetchPortRules() {
            try {
                const res = await fetch('/api/proxy/ports');
                if (!res.ok) return;
                currentPortRules = await res.json();
                renderPortRules();
                renderMatrixTopology();
            } catch (err) {
                console.error("fetch port rules error:", err);
            }
        }

        async function fetchDynamicGroups() {
            try {
                const res = await fetch('/api/tunnel-groups');
                if (!res.ok) return;
                currentDynamicGroups = await res.json();
                renderDynamicGroups();
                renderMatrixTopology();
            } catch (err) {
                console.error("fetch dynamic groups error:", err);
            }
        }

        function renderDynamicGroups() {
            const container = document.getElementById('dynamic-groups-container');
            if (!container) return;
            const isEn = getLanguage() === 'en';
            if (!currentDynamicGroups || currentDynamicGroups.length === 0) {
                container.innerHTML = `<div class="list-empty">${isEn ? 'No dynamic exit groups. Click "+ Add Exit Group" to maintain exits automatically.' : '暂无动态出口组，点击右上角「+ 新建出口组」即可按规则自动维持出口'}</div>`;
                return;
            }

            container.innerHTML = currentDynamicGroups.map(g => {
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

                const isSys = g.is_system || g.id === 'system-primary';
                let countryStr = g.country ? `${getCountryName(g.country)} (${g.country})` : (isEn ? 'All Countries / Regions' : '全部国家/地区');
                if (g.country === 'FAVORITES') {
                    countryStr = isEn ? '⭐ My Favorites (Cross-Country Pool)' : '⭐ 我的收藏节点 (跨国收藏池)';
                }
                const activeCount = (g.active_tunnel_ids || []).length;
                const statusClass = activeCount >= g.target_count ? 'connected' : (activeCount > 0 ? 'connecting' : 'disconnected');
                const statusText = g.status_text || (isEn ? 'Normal' : '正常');

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
                const sysBadge = isSys ? `<span class="badge badge-system">${isEn ? 'Primary Gateway (tun0)' : '系统主连网关 (tun0)'}</span>` : (g.country === 'FAVORITES' ? favBadge : `<span class="badge badge-accent">Top ${g.target_count} ${isEn ? 'Tunnels' : '隧道'}</span>`);
                const deleteBtn = isSys ? '' : `<button class="btn btn-danger btn-xs" data-action="deleteDynamicGroup" data-args="${jsonAttr([g.id])}">${isEn ? 'Delete' : '删除'}</button>`;
                const editLabel = isSys ? (isEn ? 'Configure Strategy' : '配置主连策略') : (isEn ? 'Edit' : '编辑');

                return `
                    <div class="dynamic-group-card ${isSys ? 'is-system' : ''} ${g.in_fallback ? 'is-fallback' : ''}">
                        <div>
                            <div class="dynamic-group-title-row">
                                <span class="badge ${statusClass}"><span class="status-dot"></span> ${escapeHtml(statusText)}</span>
                                <strong class="dynamic-group-name">${escapeHtml(g.name)}</strong>
                                ${sysBadge}
                                ${fallbackBadge}
                            </div>
                            <div class="dynamic-group-meta">
                                <span>${isEn ? 'Target: ' : '目标: '}<strong class="text-strong">${escapeHtml(countryStr)}</strong></span>
                                <span>${isEn ? 'Type: ' : '类型: '}<strong class="text-strong">${escapeHtml(ipTypeText)}</strong></span>
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

        function showAddDynamicGroupForm() {
            editingGroupId = null;
            document.getElementById('dynamic-group-edit-title').innerText = '新建动态出口组';
            document.getElementById('dg-name').value = '';
            document.getElementById('dg-country').value = 'JP';
            document.getElementById('dg-iptype').value = 'residential';
            document.getElementById('dg-unlock').value = 'none';
            document.getElementById('dg-sortby').value = 'latency';
            const fallbackEl = document.getElementById('dg-fallback');
            if (fallbackEl) fallbackEl.value = 'none';
            document.getElementById('dg-count-container').classList.remove('hidden');
            document.getElementById('dg-count').value = 3;
            document.getElementById('dg-interval').value = 15;
            openEditorDrawer('dynamic-group-edit-card');
        }

        function editDynamicGroup(id) {
            const g = currentDynamicGroups.find(item => item.id === id);
            if (!g) return;
            editingGroupId = id;
            const isSys = g.is_system || g.id === 'system-primary';
            if (isSys) {
                document.getElementById('dynamic-group-edit-title').innerText = '配置主出口自动轮换策略 (tun0)';
                document.getElementById('dg-count-container').classList.add('hidden');
            } else {
                document.getElementById('dynamic-group-edit-title').innerText = `编辑出口组: ${g.name}`;
                document.getElementById('dg-count-container').classList.remove('hidden');
            }
            document.getElementById('dg-name').value = g.name;
            document.getElementById('dg-country').value = g.country || '';
            document.getElementById('dg-iptype').value = g.ip_type || 'all';
            document.getElementById('dg-unlock').value = g.unlock_filter || 'none';
            document.getElementById('dg-sortby').value = g.sort_by || 'latency';
            const fallbackEl = document.getElementById('dg-fallback');
            if (fallbackEl) fallbackEl.value = g.fallback_policy || 'none';
            document.getElementById('dg-count').value = g.target_count || (isSys ? 1 : 3);
            document.getElementById('dg-interval').value = g.interval_minutes || 15;
            openEditorDrawer('dynamic-group-edit-card');
        }

        function hideDynamicGroupForm() { closeEditorDrawer('dynamic-group-edit-card'); }

        async function saveDynamicGroup() {
            const name = document.getElementById('dg-name').value.trim();
            if (!name) { tAlert('请输入出口组名称', 'Please enter an exit group name'); return; }
            const country = document.getElementById('dg-country').value;
            const ipType = document.getElementById('dg-iptype').value;
            const unlockFilter = document.getElementById('dg-unlock').value;
            const sortBy = document.getElementById('dg-sortby').value;
            const fallbackPolicy = document.getElementById('dg-fallback')?.value || 'none';
            const count = parseInt(document.getElementById('dg-count').value) || 3;
            const interval = parseInt(document.getElementById('dg-interval').value) || 15;

            const payload = {
                id: editingGroupId || '',
                name: name,
                enabled: true,
                is_system: editingGroupId === 'system-primary',
                country: country,
                ip_type: ipType,
                unlock_filter: unlockFilter,
                sort_by: sortBy,
                fallback_policy: fallbackPolicy,
                target_count: editingGroupId === 'system-primary' ? 1 : count,
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
                appendLog({ level: 'INFO', module: 'Action', message: `动态出口组 [${name}] 已成功保存！` });
                await fetchDynamicGroups();
                updateSystemPrimaryRotateBanner();
                fetchStatus();
            } catch(err) {
                alert('请求异常: ' + err);
            }
        }

        async function deleteDynamicGroup(id) {
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

        async function evaluateDynamicGroups() {
            appendLog({ level: 'INFO', module: 'Action', message: '正在触发出口组重新评估轮换...' });
            try {
                await fetch('/api/tunnel-groups/evaluate', { method: 'POST' });
                setTimeout(async () => {
                    await fetchDynamicGroups();
                    fetchStatus();
                }, 1500);
            } catch(err){}
        }

        function showToast(text, type = 'success', duration = 2500) {
            const t = document.getElementById('toast');
            const msg = document.getElementById('toast-msg');
            if (t && msg) {
                msg.innerText = text;
                t.dataset.type = type;
                t.classList.add('open');
                clearTimeout(t._timer);
                t._timer = setTimeout(() => { t.classList.remove('open'); }, duration);
            }
        }

        function alert(message) {
            const text = String(message ?? (getLanguage() === 'en' ? 'Action incomplete' : '操作未完成'));
            const type = /失败|错误|异常|无法|不能|不存在|请输入|未安装|尚未|暂无|fail|error|cannot|invalid|not|missing/i.test(text) ? 'error' : 'info';
            showToast(text, type, type === 'error' ? 5000 : 3200);
        }

        function showAppAlert(message) {
            const el = document.getElementById('app-alert');
            if (!el) return;
            el.textContent = message;
            el.hidden = false;
        }

        function clearAppAlert() {
            const el = document.getElementById('app-alert');
            if (!el) return;
            el.hidden = true;
            el.textContent = '';
        }

        function copyText(text, label) {
            if (navigator.clipboard && navigator.clipboard.writeText) {
                navigator.clipboard.writeText(text);
            } else {
                const ta = document.createElement('textarea');
                ta.value = text;
                document.body.appendChild(ta);
                ta.select();
                document.execCommand('copy');
                document.body.removeChild(ta);
            }
            showToast(`已复制: ${label || text}`);
        }

        function renderMatrixTopology() {
            const container = document.getElementById('matrix-topology-flow');
            if (!container) return;
            const isEn = getLanguage() === 'en';

            const st = (typeof currentState !== 'undefined' && currentState) ? currentState : null;
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
            const sysGroup = (currentDynamicGroups || []).find(g => g.is_system || g.id === 'system-primary');
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
            const hasExplicitDefaultPort = (currentPortRules || []).some(r => r.port === defaultProxyPort);
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
            if (currentPortRules && currentPortRules.length > 0) {
                currentPortRules.forEach(rule => {
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
                        const matchedGroups = (currentDynamicGroups || []).filter(g => rule.bound_group_ids.includes(g.id));
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
                                        unlock: t.unlock || (t.node ? cachedUnlockMap[t.node.ip] : null)
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
                                    unlock: t.unlock || (t.node ? cachedUnlockMap[t.node.ip] : null)
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

        function downloadBackupPackage() {
            const a = document.createElement('a');
            a.href = '/api/system/backup/export';
            a.download = `nxgate-backup-${new Date().toISOString().slice(0,10)}.json`;
            document.body.appendChild(a);
            a.click();
            document.body.removeChild(a);
            showToast(getLanguage() === 'en' ? 'Downloaded configuration backup' : '已触发下载全量备份文件 nxgate-backup.json');
        }

        function triggerBackupFileSelect() {
            document.getElementById('backup-file-input')?.click();
        }

        async function onBackupFileSelected(event) {
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

        // ==========================================
        //  屏蔽库与故障熔断隔离交互 (Blacklist Manager)
        // ==========================================
        let currentBlacklist = [];

        async function openBlacklistModal() {
            document.getElementById('blacklist-modal').classList.add('open');
            await fetchBlacklist();
        }

        function closeBlacklistModal() {
            document.getElementById('blacklist-modal').classList.remove('open');
        }

        async function fetchBlacklist(showNotice = false) {
            try {
                const res = await fetch('/api/blacklist');
                if (!res.ok) return;
                currentBlacklist = await res.json() || [];
                renderBlacklist(currentBlacklist);
                if (showNotice) {
                    showToast(getLanguage() === 'en' ? 'Blacklist updated' : '屏蔽库列表已刷新');
                }
            } catch (err) {
                console.error("fetch blacklist error:", err);
            }
        }

        let activeBlacklistTab = 'temp';

        function switchBlacklistTab(tab) {
            activeBlacklistTab = tab;
            const btnTemp = document.getElementById('bl-tab-temp');
            const btnPerm = document.getElementById('bl-tab-perm');
            const contentTemp = document.getElementById('bl-content-temp');
            const contentPerm = document.getElementById('bl-content-perm');
            if (btnTemp && btnPerm && contentTemp && contentPerm) {
                if (tab === 'temp') {
                    btnTemp.classList.add('active');
                    btnPerm.classList.remove('active');
                    contentTemp.classList.remove('hidden');
                    contentPerm.classList.add('hidden');
                } else {
                    btnPerm.classList.add('active');
                    btnTemp.classList.remove('active');
                    contentPerm.classList.remove('hidden');
                    contentTemp.classList.add('hidden');
                }
            }
        }

        function renderBlacklist(items) {
            const isEn = getLanguage() === 'en';
            const tempContainer = document.getElementById('blacklist-temp-container');
            const permContainer = document.getElementById('blacklist-perm-container');
            const tempCountEl = document.getElementById('bl-temp-count');
            const permCountEl = document.getElementById('bl-perm-count');

            const allItems = items || [];
            const tempItems = allItems.filter(i => !i.is_permanent);
            const permItems = allItems.filter(i => i.is_permanent);

            if (tempCountEl) tempCountEl.innerText = tempItems.length;
            if (permCountEl) permCountEl.innerText = permItems.length;

            const now = Date.now();

            // 1. Render Temporary Quarantine
            if (tempContainer) {
                if (tempItems.length === 0) {
                    tempContainer.innerHTML = `
                        <div class="blacklist-empty">
                            <div class="blacklist-empty-title">${isEn ? 'No Temporary Quarantined Nodes' : '当前无临时故障隔离节点'}</div>
                            <div class="blacklist-empty-copy">
                                ${isEn ? 'Nodes that fail handshakes or drop throughput are automatically quarantined temporarily. All nodes healthy.' : '当节点在连接时发生多次超时、认证拒绝或异常断线时，系统会自动临时隔离；目前所有节点运行正常。'}
                            </div>
                        </div>
                    `;
                } else {
                    tempContainer.innerHTML = `
                        <div class="blacklist-list">
                            ${tempItems.map(item => {
                                const untilTime = new Date(item.until).getTime();
                                const diffSec = Math.max(0, Math.floor((untilTime - now) / 1000));
                                let leftStr = isEn ? 'Expiring soon' : '即将解封';
                                if (diffSec > 3600) {
                                    leftStr = isEn ? `${Math.floor(diffSec / 3600)}h ${Math.floor((diffSec % 3600) / 60)}m left` : `${Math.floor(diffSec / 3600)}小时${Math.floor((diffSec % 3600) / 60)}分后解封`;
                                } else if (diffSec > 0) {
                                    leftStr = isEn ? `${Math.floor(diffSec / 60)}m ${diffSec % 60}s left` : `${Math.floor(diffSec / 60)}分${diffSec % 60}秒后解封`;
                                }

                                const cCode = item.country || '';
                                const flag = cCode ? getCountryFlagSVG(cCode) : '';
                                const scopeBadge = item.scope === 'ip' ? `<span class="badge badge-accent badge-mini">${isEn ? 'Entire IP' : '整机IP隔离'}</span>` : '';
                                const failBadge = item.fail_count > 1 ? `<span class="badge unlock-blocked badge-mini">${isEn ? 'Failed ' : '失败 '}${item.fail_count}${isEn ? 'x' : ' 次'}</span>` : '';

                                return `
                                    <div class="blacklist-item">
                                        <div class="blacklist-main">
                                            <span class="flag-box">${flag}</span>
                                            <div>
                                                <div class="blacklist-id">
                                                    ${escapeHtml(item.id || item.ip)} ${scopeBadge}
                                                </div>
                                                <div class="blacklist-meta">
                                                    <span>${isEn ? 'Reason: ' : '原因: '}<strong class="blacklist-reason">${escapeHtml(item.reason || (isEn ? 'Connection Failed' : '故障断线'))}</strong></span>
                                                    <span>·</span>
                                                    <span>${isEn ? 'Status: ' : '状态: '}<span class="blacklist-expiry">${leftStr}</span></span>
                                                    ${failBadge}
                                                </div>
                                            </div>
                                        </div>
                                        <div>
                                            <button class="btn btn-outline btn-xs" data-action="removeNodeFromBlacklist" data-args="${jsonAttr([item.id, item.ip])}">
                                                ${isEn ? 'Unblock' : '解除隔离'}
                                            </button>
                                        </div>
                                    </div>
                                `;
                            }).join('')}
                        </div>
                    `;
                }
            }

            // 2. Render Permanent Blacklist
            if (permContainer) {
                if (permItems.length === 0) {
                    permContainer.innerHTML = `
                        <div class="blacklist-empty">
                            <div class="blacklist-empty-title">${isEn ? 'No Permanently Blocked Nodes' : '当前无用户永久屏蔽节点'}</div>
                            <div class="blacklist-empty-copy">
                                ${isEn ? 'Nodes permanently blocked from the node list appear here. Completely hidden and never connected.' : '在节点列表中手动选择「永久屏蔽」的节点在此存放。永久屏蔽的节点在列表中彻底隐藏，严禁建立连接，且免疫探活复活。'}
                            </div>
                        </div>
                    `;
                } else {
                    permContainer.innerHTML = `
                        <div class="blacklist-list">
                            ${permItems.map(item => {
                                const cCode = item.country || '';
                                const flag = cCode ? getCountryFlagSVG(cCode) : '';
                                const isCIDR = item.scope === 'cidr' || (item.ip && item.ip.includes('/'));
                                const scopeBadge = isCIDR
                                    ? `<span class="badge badge-accent badge-mini" style="background: rgba(220, 38, 38, 0.15); color: #ef4444; border: 1px solid rgba(220, 38, 38, 0.3);">${isEn ? 'Subnet (CIDR)' : '整网段动态拦截'}</span>`
                                    : (item.scope === 'ip' ? `<span class="badge badge-accent badge-mini">${isEn ? 'Entire IP' : '整机IP永久拉黑'}</span>` : `<span class="badge unlock-blocked badge-mini">${isEn ? 'Permanent' : '永久屏蔽'}</span>`);

                                return `
                                    <div class="blacklist-item">
                                        <div class="blacklist-main">
                                            <span class="flag-box">${flag}</span>
                                            <div>
                                                <div class="blacklist-id">
                                                    ${escapeHtml(item.id || item.ip)} ${scopeBadge}
                                                </div>
                                                <div class="blacklist-meta">
                                                    <span>${isEn ? 'Reason: ' : '原因: '}<strong class="blacklist-reason">${escapeHtml(item.reason || (isEn ? 'Manual Permanent Block' : '用户手动永久屏蔽'))}</strong></span>
                                                    <span>·</span>
                                                    <span><strong class="text-danger">${isEn ? 'Tombstone (Never Connect)' : '永久封禁 (永不连接)'}</strong></span>
                                                </div>
                                            </div>
                                        </div>
                                        <div>
                                            <button class="btn btn-outline btn-xs" data-action="removeNodeFromBlacklist" data-args="${jsonAttr([item.id, item.ip])}">
                                                ${isEn ? 'Remove from Blacklist' : '移出黑名单/解封'}
                                            </button>
                                        </div>
                                    </div>
                                `;
                            }).join('')}
                        </div>
                    `;
                }
            }
        }

        async function removeNodeFromBlacklist(nodeId, ip) {
            try {
                const res = await fetch('/api/blacklist/remove', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ node_id: nodeId, ip: ip })
                });
                const ret = await res.json();
                if (!res.ok) {
                    tAlert('解除失败: ' + (ret.error || '未知错误'), 'Unblock failed: ' + (ret.error || 'Unknown error'));
                    return;
                }
                showToast(getLanguage() === 'en' ? `Unblocked [${nodeId}]` : `已成功解除屏蔽 [${nodeId}]`);
                await fetchBlacklist();
                fetchStatus();
                fetchNodes();
            } catch (err) {
                tAlert('请求失败: ' + err, 'Request failed: ' + err);
            }
        }

        async function clearTempBlacklist() {
            if (!tConfirm('确定清空临时故障隔离池吗？隔离节点将立即放回可用候选池重测。', 'Clear temporary quarantine? Nodes will return to candidate pool for probing.')) return;
            try {
                const res = await fetch('/api/blacklist/clear?type=temporary', { method: 'POST' });
                const ret = await res.json();
                if (!res.ok) {
                    tAlert('清空失败: ' + (ret.error || '未知错误'), 'Clear failed: ' + (ret.error || 'Unknown error'));
                    return;
                }
                showToast(getLanguage() === 'en' ? 'Cleared temporary quarantine' : '已清空临时故障隔离库');
                await fetchBlacklist();
                fetchStatus();
                fetchNodes();
            } catch (err) {
                tAlert('请求失败: ' + err, 'Request failed: ' + err);
            }
        }

        async function clearPermBlacklist() {
            if (!tConfirm('确定清空所有用户永久黑名单吗？被拉黑的节点将重新在节点列表中显示。', 'Are you sure you want to clear all permanent blacklisted nodes? They will reappear in the node list.')) return;
            try {
                const res = await fetch('/api/blacklist/clear?type=permanent', { method: 'POST' });
                const ret = await res.json();
                if (!res.ok) {
                    tAlert('清空失败: ' + (ret.error || '未知错误'), 'Clear failed: ' + (ret.error || 'Unknown error'));
                    return;
                }
                showToast(getLanguage() === 'en' ? 'Cleared permanent blacklist' : '已清空用户永久黑名单');
                await fetchBlacklist();
                fetchStatus();
                fetchNodes();
            } catch (err) {
                tAlert('请求失败: ' + err, 'Request failed: ' + err);
            }
        }

        async function clearAllBlacklist() {
            await clearTempBlacklist();
        }

        async function resurrectBlacklist() {
            const btn = document.getElementById('btn-resurrect-bl');
            if (btn) {
                btn.disabled = true;
                btn.innerText = getLanguage() === 'en' ? 'Probing...' : '正在探活复活...';
            }
            try {
                const res = await fetch('/api/blacklist/resurrect', { method: 'POST' });
                const ret = await res.json();
                if (!res.ok) {
                    tAlert('探活检测失败: ' + (ret.error || '未知错误'), 'Probe & resurrect failed: ' + (ret.error || 'Unknown error'));
                    return;
                }
                const revivedCount = ret.revived_count || 0;
                if (revivedCount > 0) {
                    showToast(getLanguage() === 'en' ? `Resurrected ${revivedCount} nodes!` : `探活成功！已复活并释放 ${revivedCount} 个节点`);
                } else {
                    showToast(getLanguage() === 'en' ? 'Probing complete: no temporary nodes revived' : '探活完成：当前临时隔离节点均未响应，暂无复活');
                }
                await fetchBlacklist();
                fetchStatus();
                fetchNodes();
            } catch (err) {
                tAlert('请求失败: ' + err, 'Request failed: ' + err);
            } finally {
                if (btn) {
                    btn.disabled = false;
                    btn.innerText = getLanguage() === 'en' ? 'Probe & Resurrect' : '立即探活复活节点';
                }
            }
        }

        async function addNodeToBlacklist(nodeId, ip, country) {
            let derivedSubnet = '';
            if (ip) {
                const parts = ip.split('.');
                if (parts.length === 4) {
                    derivedSubnet = `${parts[0]}.${parts[1]}.${parts[2]}.0/24`;
                }
            }

            const promptZh = `请选择对节点 [${nodeId}] 的屏蔽方式：\n\n1 = 永久屏蔽此节点 (单机，永不连接，从列表彻底隐藏) [推荐]\n2 = 永久屏蔽整个 /24 网段 (${derivedSubnet || ip + '/24'} 动态规则，未来该网段新节点均自动屏蔽)\n3 = 永久屏蔽整机 IP (${ip} 所有端口)\n4 = 临时隔离 24 小时 (仅在故障隔离池中观察)\n\n请输入 1, 2, 3 或 4:`;
            const promptEn = `Choose blocking method for [${nodeId}]:\n\n1 = Permanent block (Single node, never connect, hidden) [Recommended]\n2 = Permanent /24 subnet block (${derivedSubnet || ip + '/24'} dynamic rule, auto-blocks future nodes)\n3 = Permanent IP block (${ip} all ports)\n4 = Temporary quarantine 24 hours\n\nEnter 1, 2, 3, or 4:`;
            const mode = tPrompt(promptZh, promptEn, "1");
            if (!mode) return;

            let dur = 1440;
            let permanent = true;
            let scope = 'node';
            let targetNodeId = nodeId;
            let targetIP = ip;
            let reason = '用户手动永久屏蔽 (从列表彻底隐藏)';

            if (mode === '2') {
                permanent = true;
                scope = 'cidr';
                targetIP = derivedSubnet || (ip + '/24');
                targetNodeId = targetIP;
                reason = `用户手动永久屏蔽整个网段 (${targetIP})`;
            } else if (mode === '3') {
                permanent = true;
                scope = 'ip';
                targetNodeId = ip;
                reason = `用户永久屏蔽整机 IP (${ip})`;
            } else if (mode === '4') {
                permanent = false;
                reason = '用户临时隔离 24 小时';
            }

            try {
                const res = await fetch('/api/blacklist/add', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({
                        node_id: targetNodeId,
                        ip: targetIP,
                        country: country,
                        duration_minutes: dur,
                        permanent: permanent,
                        scope: scope,
                        reason: reason
                    })
                });
                const ret = await res.json();
                if (!res.ok) {
                    tAlert('屏蔽失败: ' + (ret.error || '未知错误'), 'Block failed: ' + (ret.error || 'Unknown error'));
                    return;
                }
                let msg = '';
                if (permanent) {
                    if (scope === 'cidr') {
                        msg = getLanguage() === 'en' ? `Permanently blocked subnet [${targetIP}] (auto-blocks all future nodes)` : `已永久屏蔽网段 [${targetIP}] (同网段现有及新节点均自动屏蔽)`;
                    } else if (scope === 'ip') {
                        msg = getLanguage() === 'en' ? `Permanently blocked IP [${ip}]` : `已永久屏蔽整机 IP [${ip}]`;
                    } else {
                        msg = getLanguage() === 'en' ? `Permanently blocked [${nodeId}] (hidden)` : `已永久屏蔽节点 [${nodeId}] (已彻底隐藏)`;
                    }
                } else {
                    msg = getLanguage() === 'en' ? `Quarantined [${nodeId}] for 24h` : `已将节点 [${nodeId}] 临时隔离 24 小时`;
                }
                showToast(msg);
                fetchStatus();
                fetchNodes();
            } catch (err) {
                tAlert('请求失败: ' + err, 'Request failed: ' + err);
            }
        }

        async function promptAddCidrBlock() {
            const input = tPrompt(
                '请输入要永久屏蔽的目标网段 (CIDR 格式，例如 1.1.1.0/24 或 198.51.100.0/24)：\n\n该网段现有及未来所有新节点都将自动被物理屏蔽与隐藏！',
                'Enter target subnet to permanently block (CIDR format, e.g. 1.1.1.0/24 or 198.51.100.0/24):\n\nAll current and future nodes in this subnet will be blocked and hidden automatically!',
                '1.1.1.0/24'
            );
            if (!input) return;
            const cidr = input.trim();
            if (!cidr.includes('/')) {
                tAlert('请输入合法的 CIDR 网段格式，例如 1.1.1.0/24', 'Please enter a valid CIDR subnet, e.g. 1.1.1.0/24');
                return;
            }
            try {
                const res = await fetch('/api/blacklist/add', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({
                        node_id: cidr,
                        ip: cidr,
                        permanent: true,
                        scope: 'cidr',
                        reason: `用户手动永久屏蔽网段 (${cidr})`
                    })
                });
                const ret = await res.json();
                if (!res.ok) {
                    tAlert('添加网段屏蔽失败: ' + (ret.error || '未知错误'), 'Failed to block subnet: ' + (ret.error || 'Unknown error'));
                    return;
                }
                showToast(getLanguage() === 'en' ? `Permanently blocked subnet [${cidr}]` : `已永久屏蔽网段 [${cidr}]`);
                await fetchBlacklist();
                fetchStatus();
                fetchNodes();
            } catch (e) {
                tAlert('请求失败: ' + e, 'Request failed: ' + e);
            }
        }

        function renderPortRules() {
            const container = document.getElementById('port-rules-cards');
            if (!container) return;
            const isEn = getLanguage() === 'en';
            if (!currentPortRules || currentPortRules.length === 0) {
                container.innerHTML = `<div class="list-empty">${isEn ? 'No port routing rules. Click "+ Add Port Rule" above to specify port exits.' : '暂未配置自定义端口，点击上方「+ 新增代理端口」添加'}</div>`;
                return;
            }

            const tunnels = (currentState && currentState.tunnels) ? currentState.tunnels : [];

            container.innerHTML = currentPortRules.map(r => {
                let policyBadge = `<span class="badge policy-round"><svg aria-hidden="true" viewBox="0 0 24 24" class="icon-xs icon-stroke"><polyline points="23 4 23 10 17 10"/><path d="M20.49 15a9 9 0 1 1-2.12-9.36L23 10"/></svg> ${isEn ? 'Round-Robin' : '单连接轮询'}</span>`;
                if (r.policy === 'random') {
                    policyBadge = `<span class="badge policy-random"><svg aria-hidden="true" viewBox="0 0 24 24" class="icon-xs icon-stroke"><polygon points="13 2 3 14 12 14 11 22 21 10 12 10 13 2"/></svg> ${isEn ? 'Random' : '随机分发'}</span>`;
                } else if (r.policy === 'interval') {
                    policyBadge = `<span class="badge policy-interval"><svg aria-hidden="true" viewBox="0 0 24 24" class="icon-xs icon-stroke"><circle cx="12" cy="12" r="10"/><polyline points="12 6 12 12 16 14"/></svg> ${isEn ? 'Timed Rotation' : '定时轮换'} (${r.interval_seconds || 300}s)</span>`;
                }

                let authBadge = `<span class="auth-note">${isEn ? 'Random Credentials' : '系统随机账密'}</span>`;
                if (r.auth_mode === 'none') {
                    authBadge = `<span class="auth-note auth-open">${isEn ? 'No Auth (Direct)' : '免密直连'}</span>`;
                } else if (r.auth_mode === 'custom') {
                    authBadge = `<span class="auth-note text-accent">${isEn ? 'Custom: ' : '独立账号: '}${escapeHtml(r.auth_user || (isEn ? 'Not Set' : '未设'))}</span>`;
                }

                let boundBadges = [];
                if (r.bound_group_ids && r.bound_group_ids.length > 0) {
                    r.bound_group_ids.forEach(gid => {
                        const g = currentDynamicGroups.find(item => item.id === gid);
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

        function showAddPortForm() {
            editingPort = null;
            document.getElementById('port-edit-title').innerHTML = `新增代理监听端口与分流绑定`;
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

        function editPortRule(port) {
            const rule = currentPortRules.find(r => r.port === port);
            if (!rule) return;
            editingPort = port;
            document.getElementById('port-edit-title').innerHTML = `编辑端口 [${port}] 绑定与调度规则`;
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

        function hideEditPortForm() { closeEditorDrawer('port-edit-card'); }

        function selectPolicy(pol) {
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

        function selectAuthMode(mode) {
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

        function generateRandomPortAuth() {
            const chars = 'abcdefghijklmnopqrstuvwxyz0123456789';
            let user = 'u_';
            for (let i = 0; i < 6; i++) user += chars[Math.floor(Math.random() * chars.length)];
            let pass = '';
            const passChars = 'abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789';
            for (let i = 0; i < 16; i++) pass += passChars[Math.floor(Math.random() * passChars.length)];
            document.getElementById('rule-auth-user').value = user;
            document.getElementById('rule-auth-pass').value = pass;
        }

        function suggestNextPort() {
            let max = 7927;
            if (currentPortRules && currentPortRules.length > 0) {
                currentPortRules.forEach(r => { if (r.port > max) max = r.port; });
            }
            document.getElementById('rule-port').value = max + 1;
        }

        function setPortVal(val) { document.getElementById('rule-port').value = val; }
        function setIntervalVal(sec) { document.getElementById('rule-interval').value = sec; }

        function renderTunnelCheckboxes(selectedTunnelIds, selectedGroupIds) {
            const container = document.getElementById('rule-tunnels-checkboxes');
            const tunnels = (currentState && currentState.tunnels) ? currentState.tunnels : [];
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
                    <span><strong>全部在线隧道 (默认)</strong> - 自动根据当前所有运行的隧道动态负载均衡</span>
                </label>
            `;

            if (currentDynamicGroups && currentDynamicGroups.length > 0) {
                html += '<div class="check-section-label text-accent">动态出口组 (按规则自动维持并定期轮换):</div>';
                currentDynamicGroups.forEach(g => {
                    const isChecked = !isAllChecked && selGrpMap[g.id];
                    let metricText = g.sort_by === 'speed' ? '最大带宽' : (g.sort_by === 'score' ? '最高评分' : '最低延迟');
                    html += `
                        <label  class="check-row">
                            <input type="checkbox" class="chk-dynamic-group" value="${escapeHtml(g.id)}" ${isChecked ? 'checked' : ''} data-change-action="onSpecificTunnelCheckChanged">
                            <span class="badge badge-system">出口组</span>
                            <strong class="check-name">${escapeHtml(g.name)}</strong>
                            <span  class="text-xs text-muted">(${escapeHtml(g.country || '全部')} · ${escapeHtml(g.ip_type === 'residential' ? '家宽' : (g.ip_type === 'hosting' ? '机房' : '不限'))} · ${escapeHtml(metricText)} Top${escapeHtml(g.target_count)})</span>
                        </label>
                    `;
                });
            }

            if (tunnels.length > 0) {
                html += '<div class="check-section-label text-muted">固定独立在线隧道:</div>';
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

        function onAllTunnelsCheckChanged(event, element) {
            const el = element || (event?.target?.closest ? event.target.closest('input') : null) || event?.target || this;
            if (el && el.checked) {
                document.querySelectorAll('.chk-single-tunnel').forEach(c => c.checked = false);
                document.querySelectorAll('.chk-dynamic-group').forEach(c => c.checked = false);
            }
        }

        function onSpecificTunnelCheckChanged() {
            const anyTunChecked = Array.from(document.querySelectorAll('.chk-single-tunnel')).some(c => c.checked);
            const anyGrpChecked = Array.from(document.querySelectorAll('.chk-dynamic-group')).some(c => c.checked);
            const allChk = document.getElementById('chk-tunnel-all');
            if (anyTunChecked || anyGrpChecked) {
                allChk.checked = false;
            } else {
                allChk.checked = true;
            }
        }

        async function savePortRule() {
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

            let updatedRules = currentPortRules ? [...currentPortRules] : [];
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
                currentPortRules = ret.rules || updatedRules;
                renderPortRules();
                hideEditPortForm();
                tAlert(`端口 [${port}] 规则已保存并实时生效！`, `Port [${port}] rule saved and active!`);
                fetchStatus();
            } catch (err) {
                alert('请求异常: ' + err);
            }
        }

        async function deletePortRule(port) {
            if (!tConfirm(`确认删除并停止代理端口 [${port}] 吗？`, `Are you sure you want to delete and stop proxy port [${port}]?`)) return;
            const updatedRules = currentPortRules.filter(r => r.port !== port);
            try {
                const res = await fetch('/api/proxy/ports', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ rules: updatedRules })
                });
                const ret = await res.json();
                if (!res.ok) { tAlert('删除失败: ' + (ret.error || '未知错误'), 'Delete failed: ' + (ret.error || 'Unknown error')); return; }
                currentPortRules = ret.rules || updatedRules;
                renderPortRules();
                fetchStatus();
            } catch (err) {
                tAlert('请求异常: ' + err, 'Request exception: ' + err);
            }
        }

        // ==========================================
        //  页面视图导航切换引擎 (View Router)
        // ==========================================
        function switchView(viewName) {
            const views = ['dashboard', 'singbox', 'matrix', 'nodes', 'settings'];
            if (!views.includes(viewName)) viewName = 'dashboard';

            // 1. 切换页面卡片视图
            views.forEach(v => {
                const el = document.getElementById('view-' + v);
                if (el) el.classList.remove('active');
            });
            const targetView = document.getElementById('view-' + viewName);
            if (targetView) targetView.classList.add('active');

            // 2. 切换侧边栏高亮项
            document.querySelectorAll('.nav-item').forEach(el => {
                el.classList.remove('active');
                el.removeAttribute('aria-current');
            });
            const navEl = document.getElementById('nav-' + viewName);
            if (navEl) {
                navEl.classList.add('active');
                navEl.setAttribute('aria-current', 'page');
            }

            // 3. 切换移动端底部菜单高亮项
            document.querySelectorAll('.bottom-nav-item').forEach(el => {
                el.classList.remove('active');
                el.removeAttribute('aria-current');
            });
            const bnavEl = document.getElementById('bnav-' + viewName);
            if (bnavEl) {
                bnavEl.classList.add('active');
                bnavEl.setAttribute('aria-current', 'page');
            }

            // 4. 更新顶部大标题
            const isEn = getLanguage() === 'en';
            const titlesZh = {
                dashboard: '运行概览',
                singbox: 'sing-box 节点管理',
                matrix: '端口分流与出口组',
                nodes: 'VPN 节点列表',
                settings: '系统与安全参数设置'
            };
            const titlesEn = {
                dashboard: 'Dashboard Overview',
                singbox: 'Edge Ingress (sing-box)',
                matrix: 'Multi-Port Proxy & Routing Matrix',
                nodes: 'VPN Node Square',
                settings: 'System & Security Settings'
            };
            const titles = isEn ? titlesEn : titlesZh;
            const titleEl = document.getElementById('page-title');
            if (titleEl && titles[viewName]) {
                titleEl.innerText = titles[viewName];
            }
            document.title = `${titles[viewName] || (isEn ? 'Console' : '控制台')} · NXGate`;

            // 5. 同步浏览器 Hash 路由，便于后退/前进
            if (window.location.hash !== '#' + viewName) {
                history.replaceState(null, '', '#' + viewName);
            }

            // 6. 按需触发视图专项数据刷新
            if (viewName === 'singbox') fetchSingBoxOverview();
            if (viewName === 'matrix') {
                if (currentPortRules && currentPortRules.length > 0) renderPortRules();
                if (currentDynamicGroups && currentDynamicGroups.length > 0) renderDynamicGroups();
                fetchPortRules();
                fetchDynamicGroups();
            }
            if (viewName === 'settings') loadSettingsForm();
            if (viewName === 'nodes') { updateCountryFilter(); renderNodes(); }

            if (window.scrollY > 0) {
                window.scrollTo({ top: 0, behavior: 'instant' });
            }
        }

        let activeEditorDrawer = null;

        function openEditorDrawer(id) {
            const drawer = document.getElementById(id);
            const backdrop = document.getElementById('drawer-backdrop');
            if (!drawer) return;
            if (activeEditorDrawer && activeEditorDrawer !== drawer) {
                closeEditorDrawer(activeEditorDrawer.id);
            }
            activeEditorDrawer = drawer;
            drawer.hidden = false;
            if (backdrop) backdrop.hidden = false;
            requestAnimationFrame(() => {
                drawer.classList.add('open');
                backdrop?.classList.add('open');
            });
            document.body.classList.add('drawer-open');
            const firstInput = drawer.querySelector('input:not([disabled]), select');
            firstInput?.focus();
        }

        function closeEditorDrawer(id) {
            const drawer = typeof id === 'string' ? document.getElementById(id) : id;
            const backdrop = document.getElementById('drawer-backdrop');
            if (!drawer) return;
            if (!drawer.classList.contains('open') && drawer.hidden) return;
            drawer.classList.remove('open');
            backdrop?.classList.remove('open');
            activeEditorDrawer = null;
            document.body.classList.remove('drawer-open');
            window.setTimeout(() => {
                if (!drawer.classList.contains('open')) drawer.hidden = true;
                if (backdrop && !document.querySelector('.editor-drawer.open')) backdrop.hidden = true;
            }, 230);
        }

        function enhanceInteractiveElements(root = document) {
            root.querySelectorAll('[data-action]').forEach(element => {
                if (element.matches('button, a, input, select, textarea')) return;
                element.setAttribute('role', 'button');
                element.setAttribute('tabindex', '0');
            });
        }

        function openSingBoxModal() {
            switchView('singbox');
        }

        // ==========================================
        //  侧边栏展开/收起切换引擎 (Sidebar Collapse)
        // ==========================================
        function toggleSidebar() {
            const sb = document.getElementById('app-sidebar');
            if (!sb) return;
            const isCollapsed = sb.classList.toggle('collapsed');
            try {
                localStorage.setItem('nxgate_sidebar_collapsed', isCollapsed ? '1' : '0');
            } catch(e){}
            updateSidebarUI(isCollapsed);
        }

        function updateSidebarUI(isCollapsed) {
            const btn = document.getElementById('sidebar-collapse-btn');
            const hBtn = document.getElementById('btn-toggle-sidebar');
            const isEn = getLanguage() === 'en';
            const tooltip = isCollapsed ? (isEn ? 'Expand Sidebar' : '展开侧边栏') : (isEn ? 'Collapse Sidebar' : '收起侧边栏');
            if (btn) btn.setAttribute('title', tooltip);
            if (hBtn) hBtn.setAttribute('title', tooltip);
        }

        function initSidebarState() {
            try {
                if (localStorage.getItem('nxgate_sidebar_collapsed') === '1' || localStorage.getItem('aimili_sidebar_collapsed') === '1') {
                    const sb = document.getElementById('app-sidebar');
                    if (sb) {
                        sb.classList.add('collapsed');
                        updateSidebarUI(true);
                    }
                }
            } catch(e){}
        }

        // ==========================================
        //  sing-box 边缘抗封锁入站 & 链式代理交互引擎
        // ==========================================
        let singBoxOverview = null;
        let currentSelectedProto = 'reality';

        async function fetchSingBoxOverview(showToastNotice = false) {
            try {
                const res = await fetch('/api/singbox/overview');
                if (!res.ok) {
                    renderSingBox({ ok: false, installed: false });
                    return;
                }
                const data = await res.json();
                singBoxOverview = data;
                renderSingBox(data);
                if (showToastNotice) {
                    showToast('sing-box 入站与链式状态已同步刷新');
                }
            } catch (err) {
                console.warn('拉取 sing-box 概览失败:', err);
                renderSingBox({ ok: false, installed: false });
            }
        }

        function renderSingBox(data) {
            const statusBadge = document.getElementById('sb-status-badge');
            const subBadge = document.getElementById('sb-sub-badge');
            const container = document.getElementById('sb-nodes-container');
            const grid = document.getElementById('sb-nodes-grid');
            const guide = document.getElementById('sb-empty-guide');
            const addBtn = document.getElementById('btn-add-sb-node');

            if (!statusBadge || !grid) return;
            const isEn = getLanguage() === 'en';

            // 1. 服务状态指示徽章
            if (data && data.installed) {
                const isRunning = data.status && data.status.core && data.status.core.running;
                const coreVer = (data.status && data.status.core && data.status.core.version) || (isEn ? 'Installed' : '已安装');
                if (isRunning) {
                    statusBadge.className = 'badge connected';
                    statusBadge.innerHTML = `<span class="status-dot"></span> ${isEn ? 'Running' : '运行中'} (${escapeHtml(coreVer)})`;
                } else {
                    statusBadge.className = 'badge connecting';
                    statusBadge.innerHTML = `<span class="status-dot"></span> ${isEn ? 'Ready' : '服务就绪'} (${escapeHtml(coreVer)})`;
                }
                if (addBtn) addBtn.disabled = false;
            } else {
                statusBadge.className = 'badge disconnected';
                statusBadge.innerHTML = `<span class="status-dot"></span> ${isEn ? 'Not Installed / Offline' : '未安装 / 未运行'}`;
            }

            // 2. 远程订阅状态指示
            if (data && data.subscription && data.subscription.enabled) {
                subBadge.classList.remove('hidden');
                subBadge.innerText = `${isEn ? 'Sub: ' : '订阅: '}(${data.subscription.node_count || 0} ${isEn ? 'nodes' : '节点'})`;
            } else {
                subBadge.classList.add('hidden');
            }

            // 2.5 age 加密状态指示
            const ageBtnText = document.getElementById('age-btn-text');
            const ageBtn = document.getElementById('btn-age-helper');
            const isAgeOn = data && data.subscription && data.subscription.age_encrypt_enabled && data.subscription.age_public_key;
            if (ageBtnText) {
                ageBtnText.innerText = isAgeOn ? (isEn ? 'age Encrypt (On)' : 'age 加密 (开启)') : (isEn ? 'age Encrypt' : 'age 加密');
            }
            if (ageBtn) {
                if (isAgeOn) {
                    ageBtn.classList.add('btn-accent-outline');
                } else {
                    ageBtn.classList.remove('btn-accent-outline');
                }
            }

            const sbNavBadge = document.getElementById('nav-sb-badge');
            if (sbNavBadge) {
                const count = (data && data.nodes) ? data.nodes.length : 0;
                sbNavBadge.innerText = count;
                sbNavBadge.classList.toggle('hidden', count === 0);
            }

            // 3. 未安装引导状态
            if (!data || !data.installed) {
                container.classList.add('hidden');
                guide.classList.remove('hidden');
                guide.innerHTML = `
                    <div  class="empty-title">
                        ${isEn ? 'sing-box service not detected on this system' : '尚未在系统中检测到 sing-box 服务'}
                    </div>
                    <div class="empty-guide-copy">
                        ${isEn ? 'Run the installation command on your VPS terminal to configure VLESS, Hysteria2 and other protocols with gateway egress routing.' : '在 VPS 终端执行安装后，即可在此添加 VLESS、Hysteria2 等协议节点，并支持直连或绑定网关出口分流出海。'}
                    </div>
                    <div class="command-box">
                        <span>bash &lt;(curl -fsSL https://raw.githubusercontent.com/xiumuzidiao0/sing-box/main/install.sh)</span>
                        <button type="button" class="btn btn-outline btn-xs" data-action="copyText" data-args="${jsonAttr(['bash <(curl -fsSL https://raw.githubusercontent.com/xiumuzidiao0/sing-box/main/install.sh)', isEn ? 'sing-box install command' : 'sing-box 一键安装指令'])}">${isEn ? 'Copy' : '复制'}</button>
                    </div>
                `;
                return;
            }

            // 4. 已安装但 0 节点引导
            if (!data.nodes || data.nodes.length === 0) {
                container.classList.add('hidden');
                guide.classList.remove('hidden');
                guide.innerHTML = `
                    <div  class="empty-title">
                        ${isEn ? 'No sing-box nodes configured' : '暂无 sing-box 节点'}
                    </div>
                    <div class="empty-guide-copy compact">
                        ${isEn ? 'Click the button below to add inbound protocol nodes and generate universal or Clash subscriptions.' : '点击下方按钮即可新建 VLESS、Hysteria2 等入站协议节点，并可生成通用与 Clash 订阅链接。'}
                    </div>
                    <button class="btn" data-action="openAddSingBoxModal">
                        + ${isEn ? 'Add Node' : '添加节点'}
                    </button>
                `;
                return;
            }

            // 5. 渲染活跃节点卡片
            guide.classList.add('hidden');
            container.classList.remove('hidden');

            const outbounds = data.available_outbounds || [];

            grid.innerHTML = data.nodes.map(n => {
                let protoPillClass = 'sb-proto-other';
                const pUpper = (n.protocol || '').toUpperCase();
                if (pUpper.includes('REALITY')) protoPillClass = 'sb-proto-reality';
                else if (pUpper.includes('HYSTERIA')) protoPillClass = 'sb-proto-hy2';
                else if (pUpper.includes('TUIC')) protoPillClass = 'sb-proto-tuic';
                else if (pUpper.includes('SHADOWSOCKS') || pUpper === 'SS') protoPillClass = 'sb-proto-ss';

                // 生成出口下拉选单
                const outboundOptions = outbounds.map(ob => {
                    let isSelected = false;
                    if (ob.addr === 'direct') {
                        isSelected = (!n.outbound || n.outbound === 'direct');
                    } else if (ob.port && n.outbound_port) {
                        isSelected = (ob.port === n.outbound_port);
                    } else if (n.outbound) {
                        isSelected = n.outbound.includes(ob.addr);
                    }
                    return `<option value="${escapeHtml(ob.addr)}" ${isSelected ? 'selected' : ''}>${escapeHtml(ob.label)}</option>`;
                }).join('');

                const isChained = n.outbound && n.outbound !== 'direct';
                const chainSelectClass = isChained ? 'sb-chain-select active-chain' : 'sb-chain-select';

                const sniText = n.sni || n.host || (isEn ? 'None' : '无伪装域名');
                const networkText = `${n.network || 'tcp'}${n.flow ? ' (' + n.flow + ')' : ''}`;

                return `
                    <div class="sb-node-card">
                        <div class="sb-node-header">
                            <div class="node-header-left">
                                <span class="sb-proto-pill ${protoPillClass}">${escapeHtml(n.protocol)}</span>
                                <strong class="node-port">:${n.port}</strong>
                            </div>
                            <span class="badge connected badge-mini"><span class="status-dot"></span> ${isEn ? 'Listening' : '在网监听'}</span>
                        </div>

                        <!-- 链式出口动态选择器 -->
                        <div class="sb-chain-box">
                            <div class="sb-chain-label">
                                <svg aria-hidden="true" viewBox="0 0 24 24" class="icon-xs icon-stroke"><path d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71"/><path d="M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71"/></svg>
                                ${isEn ? 'Forwarding Egress Exit' : '出口路由 (Forwarding Exit)'}
                            </div>
                            <select class="${chainSelectClass}" data-change-action="updateNodeOutboundFromSelect" data-node-name="${escapeHtml(n.name)}">
                                ${outboundOptions}
                            </select>
                        </div>

                        <!-- 节点核心参数摘要 -->
                        <div class="sb-node-info">
                            <div class="sb-info-item">
                                <span class="sb-info-lbl">${isEn ? 'SNI / Camouflage' : 'SNI / 伪装域名'}</span>
                                <span class="sb-info-val" title="${escapeHtml(sniText)}">${escapeHtml(sniText)}</span>
                            </div>
                            <div class="sb-info-item">
                                <span class="sb-info-lbl">${isEn ? 'Transport / Flow' : '传输层 / 流控'}</span>
                                <span class="sb-info-val" title="${escapeHtml(networkText)}">${escapeHtml(networkText)}</span>
                            </div>
                        </div>

                        <!-- 卡片底部快捷操作 -->
                        <div class="sb-action-bar">
                            <div  class="row gap-1">
                                <button class="btn btn-outline btn-xs" data-action="copyNodeShareLink" data-args="${jsonAttr([n.url, n.protocol])}">
                                    <svg aria-hidden="true" viewBox="0 0 24 24"><rect x="9" y="9" width="13" height="13" rx="2" ry="2"/><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/></svg>
                                    ${isEn ? 'Copy' : '复制链接'}
                                </button>
                                <button class="btn btn-outline btn-xs" data-action="showNodeQRCode" data-args="${jsonAttr([n.url, `${n.protocol} :${n.port}`])}">
                                    <svg aria-hidden="true" viewBox="0 0 24 24"><rect x="3" y="3" width="7" height="7"/><rect x="14" y="3" width="7" height="7"/><rect x="14" y="14" width="7" height="7"/><rect x="3" y="14" width="7" height="7"/></svg>
                                    ${isEn ? 'QR' : '二维码'}
                                </button>
                            </div>
                            <button class="btn btn-danger btn-xs" title="${isEn ? 'Delete node' : '删除此节点配置'}" data-action="deleteSingBoxNode" data-args="${jsonAttr([n.name])}">
                                <svg aria-hidden="true" viewBox="0 0 24 24"><polyline points="3 6 5 6 21 6"/><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/></svg>
                            </button>
                        </div>
                    </div>
                `;
            }).join('');
        }

        function openAddSingBoxModal() {
            if (!singBoxOverview || !singBoxOverview.installed) {
                tAlert('系统尚未安装 sing-box 服务端，请先在终端执行一键安装！', 'sing-box service not installed. Please install it on the VPS terminal first!');
                return;
            }
            // 填充出口选择器
            const sel = document.getElementById('sb-add-outbound');
            const outbounds = singBoxOverview.available_outbounds || [];
            sel.innerHTML = outbounds.map(ob => {
                return `<option value="${escapeHtml(ob.addr)}" ${ob.is_default ? 'selected' : ''}>${escapeHtml(ob.label)}</option>`;
            }).join('');

            filterProtoGrid('all');
            selectSingBoxProto('reality');
            document.getElementById('sb-add-port').value = 'auto';
            document.getElementById('sb-add-sni').value = 'auto';
            document.getElementById('sb-add-cred').value = 'auto';

            document.getElementById('singbox-add-modal').classList.add('open');
        }

        function closeAddSingBoxModal() {
            document.getElementById('singbox-add-modal').classList.remove('open');
        }

        function selectSingBoxProto(proto) {
            currentSelectedProto = proto;
            document.querySelectorAll('#singbox-add-modal .choice-card').forEach(el => {
                let cardProto = '';
                try {
                    cardProto = JSON.parse(el.dataset.args || '[]')[0] || '';
                } catch(e){}
                el.classList.toggle('selected', cardProto === proto);
            });
            updateProtoHelpText(proto);
        }

        function filterProtoGrid(category) {
            document.querySelectorAll('.proto-filter-bar .pill-btn').forEach(btn => {
                let btnCat = '';
                try {
                    btnCat = JSON.parse(btn.dataset.args || '[]')[0] || '';
                } catch(e){}
                btn.classList.toggle('active', btnCat === category);
            });
            document.querySelectorAll('#singbox-add-modal .choice-card').forEach(card => {
                if (category === 'all' || card.dataset.protoCat === category) {
                    card.classList.remove('hidden');
                } else {
                    card.classList.add('hidden');
                }
            });
        }

        function updateProtoHelpText(proto) {
            const sniLabel = document.getElementById('sb-sni-label');
            const sniInput = document.getElementById('sb-add-sni');
            const credLabel = document.getElementById('sb-cred-label');
            const credInput = document.getElementById('sb-add-cred');
            if (!sniLabel || !credLabel) return;

            const isReality = proto.includes('reality') || proto === 'r' || proto === 'rh2';
            const isTLS = proto.endsWith('tls') || proto === 'trojan' || ['vws', 'wss', 'tws', 'vhu', 'hu', 'thu', 'vh2', 'h2', 'th2'].includes(proto);
            const isPassword = ['hy2', 'tuic', 'ss', 'trojan', 'anytls'].includes(proto) || proto.includes('trojan') || proto.includes('hysteria');
            const isSocks = proto === 'socks';

            if (isReality) {
                sniLabel.innerText = '自定义 SNI 伪装域名';
                if (sniInput) sniInput.placeholder = 'auto (知名权威站，如 www.amazon.com)';
            } else if (isTLS) {
                sniLabel.innerText = '域名 / SNI 证书配置 (需已解析域名)';
                if (sniInput) sniInput.placeholder = 'auto (或输入域名，如 your-domain.com)';
            } else {
                sniLabel.innerText = '自定义 SNI / 伪装域名 (当前协议免配置)';
                if (sniInput) sniInput.placeholder = 'auto (当前协议无需配置)';
            }

            if (isPassword) {
                credLabel.innerText = '自定义连接密码 (Password)';
                if (credInput) credInput.placeholder = 'auto (自动生成高强度随机密码)';
            } else if (isSocks) {
                credLabel.innerText = 'Socks 认证密码 (留空免密)';
                if (credInput) credInput.placeholder = '留空免密直连';
            } else {
                credLabel.innerText = '自定义客户端 UUID 凭据';
                if (credInput) credInput.placeholder = 'auto (自动生成标准 UUID 凭证)';
            }
        }

        function setSingBoxPortAuto() {
            const input = document.getElementById('sb-add-port');
            if (input) input.value = 'auto';
        }

        async function submitAddSingBoxNode(e) {
            if (e) e.preventDefault();
            const btn = document.getElementById('btn-submit-add-sb');
            btn.disabled = true;
            btn.innerText = '正在生成并部署...';

            const portVal = document.getElementById('sb-add-port').value.trim() || 'auto';
            let sniVal = document.getElementById('sb-add-sni').value.trim() || 'auto';
            if (currentSelectedProto === 'anytls' && (sniVal === 'auto' || !sniVal.includes('.'))) {
                sniVal = '';
            }
            const credVal = document.getElementById('sb-add-cred').value.trim() || 'auto';
            const outboundVal = document.getElementById('sb-add-outbound').value;

            const payload = {
                protocol: currentSelectedProto,
                port: portVal,
                sni: sniVal,
                uuid: credVal,
                outbound: outboundVal
            };

            try {
                const res = await fetch('/api/singbox/nodes', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify(payload)
                });
                const ret = await res.json();
                if (!res.ok) {
                    tAlert('创建节点失败: ' + (ret.error || '未知错误'), 'Failed to create node: ' + (ret.error || 'Unknown error'));
                    return;
                }
                closeAddSingBoxModal();
                showToast(`已成功创建 ${ret.node?.protocol || currentSelectedProto} 入站节点 (端口: ${ret.node?.port || 'auto'})`);
                await fetchSingBoxOverview();
            } catch (err) {
                alert('网络请求失败: ' + err);
            } finally {
                btn.disabled = false;
                btn.innerText = '立即创建并部署';
            }
        }

        async function updateNodeOutbound(nodeName, outboundAddr) {
            try {
                const res = await fetch('/api/singbox/nodes/outbound', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ target: nodeName, outbound: outboundAddr })
                });
                const ret = await res.json();
                if (!res.ok) {
                    alert('修改出口失败: ' + (ret.error || '未知错误'));
                    await fetchSingBoxOverview();
                    return;
                }
                showToast(`节点 [${nodeName}] 链式出口已切换为: ${outboundAddr}`);
                await fetchSingBoxOverview();
            } catch (err) {
                alert('网络请求失败: ' + err);
            }
        }

        function updateNodeOutboundFromSelect(event, element) {
            const select = element || (event?.target?.closest ? event.target.closest('select') : null) || event?.target || this;
            if (!select || !select.dataset) return;
            updateNodeOutbound(select.dataset.nodeName, select.value);
        }

        async function batchSetSingBoxOutbound(targetType) {
            if (!singBoxOverview || !singBoxOverview.installed || !singBoxOverview.nodes || singBoxOverview.nodes.length === 0) {
                tAlert('当前没有活跃的 sing-box 节点可供操作', 'No active sing-box nodes available');
                return;
            }

            let targetOutbound = 'direct';
            let label = '直连 (VPS 原生机房网络)';
            if (targetType === 'default') {
                const defOb = (singBoxOverview.available_outbounds || []).find(o => o.is_default);
                targetOutbound = defOb ? defOb.addr : '127.0.0.1:7928';
                label = `NXGate 默认住宅出口 (${targetOutbound})`;
            }

            if (!tConfirm(`确定将所有 sing-box 入站节点批量切换至【${label}】吗？`, `Switch all sing-box inbound nodes to [${label}]?`)) return;

            try {
                const res = await fetch('/api/singbox/nodes/outbound', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ target: 'all', outbound: targetOutbound })
                });
                const ret = await res.json();
                if (!res.ok) {
                    tAlert('批量修改出口失败: ' + (ret.error || '未知错误'), 'Batch egress update failed: ' + (ret.error || 'Unknown error'));
                    return;
                }
                showToast(`已成功将 ${ret.updated_count || '全部'} 个节点切换至: ${targetOutbound}`);
                await fetchSingBoxOverview();
            } catch (err) {
                alert('网络请求失败: ' + err);
            }
        }

        async function deleteSingBoxNode(nodeName) {
            if (!tConfirm(`确认彻底删除 sing-box 入站配置 [${nodeName}] 吗？`, `Are you sure you want to delete sing-box inbound [${nodeName}]?`)) return;
            try {
                const res = await fetch(`/api/singbox/nodes?target=${encodeURIComponent(nodeName)}`, {
                    method: 'DELETE'
                });
                const ret = await res.json();
                if (!res.ok) {
                    alert('删除失败: ' + (ret.error || '未知错误'));
                    return;
                }
                showToast(`节点 [${nodeName}] 已成功删除并重构订阅`);
                await fetchSingBoxOverview();
            } catch (err) {
                alert('网络请求失败: ' + err);
            }
        }

        function sanitizeSubURL(url) {
            if (!url) return '';
            const curHost = window.location.host;
            if (window.location.hostname !== '127.0.0.1' && window.location.hostname !== 'localhost') {
                url = url.replace(/:\/\/(127\.0\.0\.1|localhost)(:\d+)?\//, `://${curHost}/`);
            }
            return url;
        }

        function getGenericSubURL() {
            if (singBoxOverview && singBoxOverview.subscription && singBoxOverview.subscription.sub_url) {
                return sanitizeSubURL(singBoxOverview.subscription.sub_url);
            }
            const origin = window.location.origin;
            const prefix = window.__apiPrefix || '';
            return `${origin}${prefix}/api/singbox/subscription`;
        }

        async function copySingBoxSubURL() {
            if (!singBoxOverview || !singBoxOverview.installed) {
                tAlert('sing-box 未安装，无法获取远程订阅', 'sing-box not installed; cannot fetch remote subscription');
                return;
            }
            const url = getGenericSubURL();
            const isAge = singBoxOverview.subscription && singBoxOverview.subscription.age_encrypt_enabled && singBoxOverview.subscription.age_public_key;
            copyText(url, isAge ? 'age 端到端加密的通用订阅链接 (Base64/Raw)' : 'sing-box 全量通用订阅链接 (Base64/Raw)');
        }

        function getClashSubURL() {
            if (singBoxOverview && singBoxOverview.subscription && singBoxOverview.subscription.clash_sub_url) {
                return sanitizeSubURL(singBoxOverview.subscription.clash_sub_url);
            }
            const origin = window.location.origin;
            const prefix = window.__apiPrefix || '';
            return `${origin}${prefix}/api/singbox/subscription/clash`;
        }

        function copyClashSubURL() {
            if (!singBoxOverview || !singBoxOverview.installed) {
                tAlert('sing-box 未安装，无法获取 Clash 订阅', 'sing-box not installed; cannot fetch Clash subscription');
                return;
            }
            const url = getClashSubURL();
            const isAge = singBoxOverview.subscription && singBoxOverview.subscription.age_encrypt_enabled && singBoxOverview.subscription.age_public_key;
            copyText(url, isAge ? 'age 端到端加密的 Clash Meta 专属订阅链接' : 'Clash Meta / Mihomo 专属订阅链接');
        }

        function downloadClashConfig() {
            const url = getClashSubURL();
            const a = document.createElement('a');
            a.href = url;
            a.download = 'singbox-clash.yaml';
            document.body.appendChild(a);
            a.click();
            document.body.removeChild(a);
            showToast('已触发下载 singbox-clash.yaml');
        }

        function sanitizeShareURL(url) {
            if (!url) return '';
            const host = window.location.hostname || '127.0.0.1';
            return url.replace(/@auto(:|#)/g, `@${host}$1`).replace(/-auto$/g, `-${host}`);
        }

        function copyNodeShareLink(url, proto) {
            url = sanitizeShareURL(url);
            if (!url) {
                tAlert('该节点暂无有效客户端分享链接', 'No valid client share link for this node');
                return;
            }
            copyText(url, `${proto || '代理'} 客户端分享链接`);
        }

        function showNodeQRCode(url, title) {
            url = sanitizeShareURL(url);
            if (!url) {
                tAlert('暂无分享链接', 'No share link available');
                return;
            }
            document.getElementById('sb-qr-title').innerText = title || '客户端配置链接与二维码';
            document.getElementById('sb-qr-url-text').value = url;
            const qrImg = document.getElementById('sb-qr-img');
            qrImg.src = `https://api.qrserver.com/v1/create-qr-code/?size=200x200&data=${encodeURIComponent(url)}`;
            document.getElementById('singbox-qr-modal').classList.add('open');
        }

        function buildFilteredSubParams() {
            const scope = document.getElementById('sub-filter-scope')?.value || '';
            const proto = document.getElementById('sub-filter-proto')?.value || '';
            const ping = document.getElementById('sub-filter-ping')?.value || '';
            const customCountry = (document.getElementById('sub-filter-custom-country')?.value || '').trim();

            const params = new URLSearchParams();
            if (scope === 'favorites') {
                params.set('filter', 'favorites');
            } else if (scope) {
                params.set('countries', scope);
            }
            if (customCountry) {
                params.set('countries', customCountry);
            }
            if (proto) {
                params.set('protocol', proto);
            }
            if (ping) {
                params.set('max_ping', ping);
            }
            return params.toString();
        }

        function getFilteredGenericSubURL() {
            const base = getGenericSubURL();
            const qs = buildFilteredSubParams();
            if (!qs) return base;
            return base.includes('?') ? `${base}&${qs}` : `${base}?${qs}`;
        }

        function getFilteredClashSubURL() {
            const base = getClashSubURL();
            const qs = buildFilteredSubParams();
            if (!qs) return base;
            return base.includes('?') ? `${base}&${qs}` : `${base}?${qs}`;
        }

        function updateSubFilterPreview() {
            const genInput = document.getElementById('sub-filter-preview-generic');
            const clashInput = document.getElementById('sub-filter-preview-clash');
            if (genInput) genInput.value = getFilteredGenericSubURL();
            if (clashInput) clashInput.value = getFilteredClashSubURL();
        }

        function openSubFilterModal() {
            if (!singBoxOverview || !singBoxOverview.installed) {
                tAlert('sing-box 未安装，无法生成定制订阅', 'sing-box not installed; cannot generate customized subscription');
                return;
            }
            updateSubFilterPreview();
            document.getElementById('sub-filter-modal')?.classList.add('open');
        }

        function closeSubFilterModal() {
            document.getElementById('sub-filter-modal')?.classList.remove('open');
        }

        function copyFilteredGenericSub() {
            const url = getFilteredGenericSubURL();
            copyText(url, '定制通用订阅链接 (Base64/Raw)');
        }

        function copyFilteredClashSub() {
            const url = getFilteredClashSubURL();
            copyText(url, '定制 Clash Meta 订阅链接');
        }

        function downloadFilteredClash() {
            const url = getFilteredClashSubURL();
            const a = document.createElement('a');
            a.href = url;
            a.download = 'singbox-clash-filtered.yaml';
            document.body.appendChild(a);
            a.click();
            document.body.removeChild(a);
            showToast('已触发下载定制 singbox-clash-filtered.yaml');
        }

        function showFilteredQR() {
            const url = getFilteredGenericSubURL();
            showNodeQRCode(url, '定制订阅链接与二维码');
        }

        let currentAgeType = 'x25519';

        async function openAgeKeyHelper() {
            try {
                const res = await fetch('/api/settings');
                if (res.ok) {
                    const data = await res.json();
                    const toggleEl = document.getElementById('age-encrypt-enabled');
                    const pubInput = document.getElementById('age-public-key-input');
                    if (toggleEl) toggleEl.checked = !!data.age_encrypt_enabled;
                    if (pubInput) pubInput.value = data.age_public_key || '';
                    if (data.age_public_key && data.age_public_key.startsWith('age1pq')) {
                        switchAgeType('mlkem768-x25519');
                    } else {
                        switchAgeType('x25519');
                    }
                }
            } catch (e) {
                console.warn('获取 age 配置失败:', e);
            }
            document.getElementById('age-key-helper-modal').classList.add('open');
        }

        function closeAgeKeyHelper() {
            document.getElementById('age-key-helper-modal').classList.remove('open');
        }

        function switchAgeType(type) {
            currentAgeType = type;
            const xBtn = document.getElementById('age-type-x25519');
            const pqBtn = document.getElementById('age-type-pq');
            if (type === 'x25519') {
                if (xBtn) xBtn.classList.add('active');
                if (pqBtn) pqBtn.classList.remove('active');
            } else {
                if (pqBtn) pqBtn.classList.add('active');
                if (xBtn) xBtn.classList.remove('active');
            }
        }

        async function generateAgeKey() {
            try {
                const res = await fetch(`/api/singbox/subscription/age/generate?type=${encodeURIComponent(currentAgeType)}`, {
                    method: 'POST'
                });
                const data = await res.json();
                if (!res.ok || !data.ok) {
                    tAlert('生成密钥失败: ' + (data.error || '未知错误'), 'Key generation failed: ' + (data.error || 'Unknown error'));
                    return;
                }
                document.getElementById('age-secret-key-input').value = data.secret_key;
                document.getElementById('age-public-key-input').value = data.public_key;
                showToast(`已成功生成全新 ${data.type} 密钥对！解密私钥请妥善保存。`);
            } catch (err) {
                alert('网络请求失败: ' + err);
            }
        }

        function copyAgeSecretKey() {
            const sec = document.getElementById('age-secret-key-input').value.trim();
            if (!sec) {
                tAlert('请先输入或生成 age 解密私钥', 'Please enter or generate an age secret key first');
                return;
            }
            copyText(sec, 'age 解密私钥');
        }

        async function deriveAgePublicKey() {
            const sec = document.getElementById('age-secret-key-input').value.trim();
            if (!sec) {
                tAlert('请先在私钥框中粘贴或生成 age 解密私钥', 'Please paste or generate an age secret key in the textarea first');
                return;
            }
            try {
                const res = await fetch('/api/singbox/subscription/age/derive', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ secret_key: sec })
                });
                const data = await res.json();
                if (!res.ok || !data.ok) {
                    tAlert('从私钥推导公钥失败: ' + (data.error || '未知错误'), 'Deriving public key failed: ' + (data.error || 'Unknown error'));
                    return;
                }
                document.getElementById('age-public-key-input').value = data.public_key;
                if (data.type === 'MLKEM768-X25519') {
                    switchAgeType('mlkem768-x25519');
                } else {
                    switchAgeType('x25519');
                }
                showToast(`已成功推导出 ${data.type} 加密公钥！`);
            } catch (err) {
                alert('网络请求失败: ' + err);
            }
        }

        function copyAgePublicKey() {
            const pub = document.getElementById('age-public-key-input').value.trim();
            if (!pub) {
                tAlert('请先输入或生成 age 加密公钥', 'Please enter or generate an age public key first');
                return;
            }
            copyText(pub, 'age 加密公钥');
        }

        async function applyAgePublicKey() {
            const pub = document.getElementById('age-public-key-input').value.trim();
            if (!pub) {
                tAlert('请先生成或输入 age 加密公钥', 'Please generate or enter an age public key first');
                return;
            }
            const toggleEl = document.getElementById('age-encrypt-enabled');
            if (toggleEl) toggleEl.checked = true;
            try {
                const res = await fetch('/api/settings', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({
                        age_encrypt_enabled: true,
                        age_public_key: pub
                    })
                });
                if (!res.ok) {
                    const err = await res.json();
                    alert('保存失败: ' + (err.error || '未知错误'));
                    return;
                }
                const cfgPub = document.getElementById('cfg-age-pubkey');
                if (cfgPub) cfgPub.value = pub;
                showToast('已一键填入公钥并开启 age 订阅端到端加密！');
                await fetchSingBoxOverview();
            } catch (err) {
                alert('网络请求失败: ' + err);
            }
        }

        async function onAgeToggleChanged() {
            const toggleEl = document.getElementById('age-encrypt-enabled');
            const isEnabled = toggleEl.checked;
            const pub = document.getElementById('age-public-key-input').value.trim();

            if (isEnabled && !pub) {
                tAlert('请先在下方输入或一键生成 age 加密公钥，再开启加密功能！', 'Please enter or generate an age public key before enabling encryption!');
                toggleEl.checked = false;
                return;
            }

            try {
                const res = await fetch('/api/settings', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({
                        age_encrypt_enabled: isEnabled,
                        age_public_key: pub
                    })
                });
                if (!res.ok) {
                    const err = await res.json();
                    alert('保存开关失败: ' + (err.error || '未知错误'));
                    toggleEl.checked = !isEnabled;
                    return;
                }
                showToast(`age 订阅端到端加密已${isEnabled ? '开启' : '关闭'}`);
                await fetchSingBoxOverview();
            } catch (err) {
                alert('网络请求失败: ' + err);
                toggleEl.checked = !isEnabled;
            }
        }

        function closeSingBoxQRModal() {
            document.getElementById('singbox-qr-modal').classList.remove('open');
        }

        let currentAppProfile = null;

        async function openMobileAppModal() {
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
                        currentAppProfile = data.profile;
                    }
                }
            } catch (e) {
                console.warn('获取服务器导入配置失败:', e);
            }

            renderAppProfileQRCode();
            document.getElementById('mobile-app-modal').classList.add('open');
        }

        function closeMobileAppModal() {
            document.getElementById('mobile-app-modal').classList.remove('open');
        }

        function renderAppProfileQRCode() {
            const name = (document.getElementById('app-server-name')?.value || '').trim() || 'NXGate';
            const proto = document.getElementById('app-server-protocol')?.value || 'http';
            const host = (document.getElementById('app-server-host')?.value || '').trim() || window.location.hostname || '127.0.0.1';
            const port = parseInt(document.getElementById('app-server-port')?.value) || (window.location.port ? parseInt(window.location.port) : 8787);
            const path = (currentAppProfile && currentAppProfile.path) || (currentState && currentState.admin_path) || 'enter';
            const user = (currentAppProfile && currentAppProfile.username) || 'admin';
            const pass = (currentAppProfile && currentAppProfile.password) || '';
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
                proxy_port: (currentAppProfile && currentAppProfile.proxy_port) || 7928,
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

        function onAppProfileFieldChanged() {
            renderAppProfileQRCode();
        }

        async function testMobileApiConnectivity() {
            const testEl = document.getElementById('app-api-test-result');
            if (!testEl) return;
            testEl.classList.remove('hidden');
            testEl.innerHTML = '<span class="text-muted">正在测试连接 /api/app/info ...</span>';
            const start = Date.now();
            try {
                const res = await fetch('/api/app/info');
                const elapsed = Date.now() - start;
                if (res.ok) {
                    const data = await res.json();
                    testEl.innerHTML = `<span class="text-success">API 连通正常！(耗时 ${elapsed}ms, 版本: v${escapeHtml(data.version || '2.5.4')}, 状态: ${escapeHtml(data.status || '就绪')})</span>`;
                } else {
                    testEl.innerHTML = `<span class="text-danger">API 测试返回异常 (HTTP ${res.status})</span>`;
                }
            } catch (err) {
                testEl.innerHTML = `<span class="text-danger">无法连接到 API: ${escapeHtml(err.message)}</span>`;
            }
        }

        function copyFromElement(elId) {
            const el = document.getElementById(elId);
            if (el) {
                copyText(el.value, '链接');
            }
        }

        // ==========================================
        //  系统更新与版本管理引擎 (Self-Update Engine)
        // ==========================================
        let isUpdating = false;

        function openUpdateModal() {
            const modal = document.getElementById('update-modal');
            if (modal) {
                modal.classList.add('open');
                modal.hidden = false;
            }
            checkForUpdates(true);
        }

        function closeUpdateModal() {
            const modal = document.getElementById('update-modal');
            if (modal) {
                modal.classList.remove('open');
                modal.hidden = true;
            }
        }

        async function checkForUpdates(silent = false) {
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
                    t.textContent = '检查中...';
                }
            });
            latestVerElements.forEach(el => {
                if (el) el.textContent = '查询中...';
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
                            t.textContent = '检查失败';
                        }
                    });
                    latestVerElements.forEach(el => {
                        if (el) el.textContent = '无法获取';
                    });
                    if (!silent) showToast(data.error || '检查更新失败', 'error');
                    return;
                }

                const latestVer = 'v' + String(data.latest_version || '').replace(/^v/, '');
                latestVerElements.forEach(el => {
                    if (el) el.textContent = latestVer;
                });

                if (data.release_notes || data.release_name) {
                    const title = data.release_name || `${latestVer} 发行说明`;
                    const dateStr = data.published_at ? new Date(data.published_at).toLocaleDateString('zh-CN') : '';
                    const notesFormatted = escapeHtml(data.release_notes || '已检测到远端主干最新稳定版本。').replace(/\n/g, '<br>');

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
                            t.textContent = '可更新';
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

                    if (!silent) showToast(`发现新版本 ${latestVer}，可点击立即升级！`, 'info');
                } else {
                    statusTags.forEach(t => {
                        if (t) {
                            t.className = 'badge badge-proto';
                            t.textContent = '已是最新';
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

                    if (!silent) showToast('当前已是最新稳定版本 (v' + String(data.current_version || '').replace(/^v/, '') + ')', 'success');
                }
            } catch (err) {
                console.error('检查更新失败:', err);
                statusTags.forEach(t => {
                    if (t) {
                        t.className = 'badge badge-danger';
                        t.textContent = '网络异常';
                    }
                });
                if (!silent) showToast('检查更新失败，请检查网络连接', 'error');
            }
        }

        async function triggerSystemUpdate(force = false) {
            if (isUpdating) return;
            const actionDesc = force ? '强制重新安装当前最新构建' : '升级至最新版本';
            if (!tConfirm(`确认立即执行系统更新（${actionDesc}）并重启服务吗？`, `Proceed with system update (${actionDesc}) and restart service?`)) return;

            isUpdating = true;
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
            progressSteps.forEach(s => s && (s.textContent = '正在下载官方二进制包并比对 SHA-256 校验和...'));
            updateBtns.forEach(b => b && (b.disabled = true));

            try {
                const res = await fetch('/api/update/trigger', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ force: !!force })
                });
                const ret = await res.json();

                if (!res.ok || !ret.ok) {
                    isUpdating = false;
                    updateBtns.forEach(b => b && (b.disabled = false));
                    progressSteps.forEach(s => s && (s.textContent = `${ret.error || '更新失败'}`));
                    tAlert('系统更新失败: ' + (ret.error || '未知错误'), 'System update failed: ' + (ret.error || 'Unknown error'));
                    return;
                }

                // Success! Service is restarting
                let countdown = 5;
                const updateMsg = escapeHtml(ret.message || '更新完成！');
                progressSteps.forEach(s => s && (s.innerHTML = `<strong>${updateMsg}</strong> 页面将在 <strong>${countdown}</strong> 秒后自动刷新...`));

                const timer = setInterval(() => {
                    countdown--;
                    if (countdown <= 0) {
                        clearInterval(timer);
                        window.location.reload();
                    } else {
                        progressSteps.forEach(s => s && (s.innerHTML = `<strong>${updateMsg}</strong> 页面将在 <strong>${countdown}</strong> 秒后自动刷新...`));
                    }
                }, 1000);

            } catch (err) {
                isUpdating = false;
                updateBtns.forEach(b => b && (b.disabled = false));
                progressSteps.forEach(s => s && (s.textContent = `网络异常: ${err}`));
                alert('请求更新失败: ' + err);
            }
        }

        function triggerForceUpdate() {
            triggerSystemUpdate(true);
        }

        const uiActions = {
            toggleSidebar,
            quickConnect,
            disconnectVPN,
            openBlacklistModal,
            openPortMatrixModal,
            clearLogs,
            openAddSingBoxModal,
            batchSetSingBoxOutbound,
            copySingBoxSubURL,
            copyClashSubURL,
            downloadClashConfig,
            fetchSingBoxOverview,
            probeCurrentNodes,
            refreshNodes,
            selectQuickFilter,
            toggleSort,
            closeSettingsModal,
            switchSettingsTab,
            openSystemPrimaryConfig,
            randomPath,
            randomPassword,
            testTelegramAlert,
            switchView,
            closePortMatrixModal,
            switchMatrixTab,
            showAddPortForm,
            suggestNextPort,
            setPortVal,
            selectPolicy,
            setIntervalVal,
            selectAuthMode,
            generateRandomPortAuth,
            hideEditPortForm,
            savePortRule,
            evaluateDynamicGroups,
            showAddDynamicGroupForm,
            hideDynamicGroupForm,
            saveDynamicGroup,
            closeAddSingBoxModal,
            selectSingBoxProto,
            filterProtoGrid,
            setSingBoxPortAuto,
            closeSingBoxQRModal,
            copyFromElement,
            closeBlacklistModal,
            switchBlacklistTab,
            clearTempBlacklist,
            clearPermBlacklist,
            promptAddCidrBlock,
            resurrectBlacklist,
            clearAllBlacklist,
            fetchBlacklist,
            renderNodes,
            saveSettings,
            submitAddSingBoxNode,
            probeTunnelUnlock,
            stopTunnel,
            toggleFavorite,
            connectToNode,
            startNewTunnel,
            addNodeToBlacklist,
            deleteDynamicGroup,
            editDynamicGroup,
            removeNodeFromBlacklist,
            copyText,
            editPortRule,
            deletePortRule,
            onAllTunnelsCheckChanged,
            onSpecificTunnelCheckChanged,
            updateNodeOutbound,
            updateNodeOutboundFromSelect,
            copyNodeShareLink,
            showNodeQRCode,
            deleteSingBoxNode,
            openMobileAppModal,
            closeMobileAppModal,
            onAppProfileFieldChanged,
            testMobileApiConnectivity,
            openUpdateModal,
            closeUpdateModal,
            checkForUpdates,
            triggerSystemUpdate,
            triggerForceUpdate,
            openSubFilterModal,
            closeSubFilterModal,
            updateSubFilterPreview,
            copyFilteredGenericSub,
            copyFilteredClashSub,
            downloadFilteredClash,
            showFilteredQR,
            downloadBackupPackage,
            triggerBackupFileSelect,
            onBackupFileSelected,
            renderMatrixTopology,
            openAgeKeyHelper,
            closeAgeKeyHelper,
            switchAgeType,
            generateAgeKey,
            copyAgeSecretKey,
            deriveAgePublicKey,
            copyAgePublicKey,
            applyAgePublicKey,
            onAgeToggleChanged,
            toggleLanguage,
            onUiLanguageChanged
        };

        function runDataAction(element, dataKey, event) {
            if (!element) return false;
            const actionName = element.dataset?.[dataKey];
            const action = uiActions[actionName];
            if (!action) return false;
            let args = [];
            try {
                args = JSON.parse(element.dataset.args || '[]');
            } catch (error) {
                console.error('Invalid data-args', actionName, error);
            }
            action.call(element, ...args, event, element);
            return true;
        }

        document.addEventListener('click', event => {
            const actionTarget = event.target.closest('[data-action]');
            if (runDataAction(actionTarget, 'action', event)) return;
            const drawerClose = event.target.closest('[data-drawer-close]');
            if (drawerClose) {
                const drawer = drawerClose.closest('.editor-drawer');
                if (drawer) closeEditorDrawer(drawer.id);
                return;
            }
            if (event.target.id === 'drawer-backdrop') {
                closeEditorDrawer(activeEditorDrawer);
                return;
            }
            const target = event.target.closest('[data-view]');
            if (!target) return;
            switchView(target.dataset.view);
        });

        document.addEventListener('change', event => {
            const target = event.target.closest('[data-change-action]');
            runDataAction(target, 'changeAction', event);
        });

        document.addEventListener('input', event => {
            const target = event.target.closest('[data-input-action]');
            runDataAction(target, 'inputAction', event);
        });

        document.addEventListener('submit', event => {
            const target = event.target.closest('[data-submit-action]');
            if (runDataAction(target, 'submitAction', event)) event.preventDefault();
        });

        document.addEventListener('keydown', event => {
            if (event.key === 'Escape' && activeEditorDrawer) {
                closeEditorDrawer(activeEditorDrawer);
                return;
            }
            if ((event.key === 'Enter' || event.key === ' ') && !['BUTTON', 'A', 'INPUT', 'SELECT', 'TEXTAREA'].includes(event.target.tagName)) {
                const target = event.target.closest('[data-action]');
                if (target && runDataAction(target, 'action', event)) {
                    event.preventDefault();
                }
            }
        });

        window.onload = () => {
            applyLanguage(getLanguage());
            enhanceInteractiveElements();
            new MutationObserver(mutations => {
                mutations.forEach(mutation => {
                    mutation.addedNodes.forEach(node => {
                        if (node.nodeType === Node.ELEMENT_NODE) enhanceInteractiveElements(node);
                    });
                });
            }).observe(document.body, { childList: true, subtree: true });
            initSidebarState();
            fetchUnlockCache();
            fetchStatus();
            fetchNodes();
            fetchSingBoxOverview();
            fetchPortRules();
            fetchDynamicGroups();
            setupSSE();
            setTimeout(() => { checkForUpdates(true); }, 2500);

            const initialHash = (window.location.hash || '').replace(/^#/, '');
            if (['dashboard', 'singbox', 'matrix', 'nodes', 'settings'].includes(initialHash)) {
                switchView(initialHash);
            } else {
                switchView('dashboard');
            }

            setInterval(fetchStatus, 2000);
            setInterval(fetchSingBoxOverview, 10000);
            setInterval(fetchNodes, 15000);
            setInterval(fetchUnlockCache, 30000);
        };
    
