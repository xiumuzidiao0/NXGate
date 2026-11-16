import { appState } from '../state.js';
import { formatOutboundLabel, getLanguage, localize, tAlert, tConfirm } from '../i18n-runtime.js';
import { alert, copyText, escapeHtml, jsonAttr, sanitizeShareURL, sanitizeSubURL, showToast } from '../utils.js';

export async function fetchSingBoxOverview(showToastNotice = false) {
    try {
        const res = await fetch('/api/singbox/overview');
        if (!res.ok) {
            renderSingBox({ ok: false, installed: false });
            return;
        }
        const data = await res.json();
        appState.singBoxOverview = data;
        renderSingBox(data);
        if (showToastNotice) {
            showToast(localize('sing-box 入站与链式状态已同步刷新', 'sing-box inbounds and egress chains refreshed'));
        }
    } catch (err) {
        console.warn('拉取 sing-box 概览失败:', err);
        renderSingBox({ ok: false, installed: false });
    }
}

export function renderSingBox(data) {
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
            return `<option value="${escapeHtml(ob.addr)}" ${isSelected ? 'selected' : ''}>${escapeHtml(formatOutboundLabel(ob))}</option>`;
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
                    <select class="${chainSelectClass}" data-change-action="updateNodeOutboundFromSelect" data-node-name="${escapeHtml(n.name)}" aria-label="${isEn ? 'Egress Exit' : '出口路由'}">
                        ${outboundOptions}
                    </select>
                </div>

                <!-- 节点核心参数摘要 -->
                <div class="sb-node-info">
                    <div class="sb-info-item">
                        <span class="sb-info-lbl">SNI</span>
                        <span class="sb-info-val" title="${escapeHtml(sniText)}">${escapeHtml(sniText)}</span>
                    </div>
                    <div class="sb-info-item">
                        <span class="sb-info-lbl">${isEn ? 'Transport' : '传输流控'}</span>
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

export function openAddSingBoxModal() {
    if (!appState.singBoxOverview || !appState.singBoxOverview.installed) {
        tAlert('系统尚未安装 sing-box 服务端，请先在终端执行一键安装！', 'sing-box service not installed. Please install it on the VPS terminal first!');
        return;
    }
    // 填充出口选择器
    const sel = document.getElementById('sb-add-outbound');
    const outbounds = appState.singBoxOverview.available_outbounds || [];
    sel.innerHTML = outbounds.map(ob => {
        return `<option value="${escapeHtml(ob.addr)}" ${ob.is_default ? 'selected' : ''}>${escapeHtml(formatOutboundLabel(ob))}</option>`;
    }).join('');

    filterProtoGrid('all');
    selectSingBoxProto('reality');
    document.getElementById('sb-add-port').value = 'auto';
    document.getElementById('sb-add-sni').value = 'auto';
    document.getElementById('sb-add-cred').value = 'auto';

    document.getElementById('singbox-add-modal').classList.add('open');
}

export function closeAddSingBoxModal() {
    document.getElementById('singbox-add-modal').classList.remove('open');
}

export function selectSingBoxProto(proto) {
    appState.currentSelectedProto = proto;
    document.querySelectorAll('#singbox-add-modal .choice-card').forEach(el => {
        let cardProto = '';
        try {
            cardProto = JSON.parse(el.dataset.args || '[]')[0] || '';
        } catch(e){}
        el.classList.toggle('selected', cardProto === proto);
    });
    updateProtoHelpText(proto);
}

export function filterProtoGrid(category) {
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

export function updateProtoHelpText(proto) {
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
        sniLabel.innerText = localize('自定义 SNI 伪装域名', 'Custom SNI hostname');
        if (sniInput) sniInput.placeholder = localize('auto (知名权威站，如 www.amazon.com)', 'auto (e.g. www.amazon.com)');
    } else if (isTLS) {
        sniLabel.innerText = localize('域名 / SNI 证书配置 (需已解析域名)', 'Domain / SNI certificate (DNS must be configured)');
        if (sniInput) sniInput.placeholder = localize('auto (或输入域名，如 your-domain.com)', 'auto (or a domain, e.g. your-domain.com)');
    } else {
        sniLabel.innerText = localize('自定义 SNI / 伪装域名 (当前协议免配置)', 'SNI hostname (not required for this protocol)');
        if (sniInput) sniInput.placeholder = localize('auto (当前协议无需配置)', 'auto (not required for this protocol)');
    }

    if (isPassword) {
        credLabel.innerText = localize('自定义连接密码 (Password)', 'Connection password');
        if (credInput) credInput.placeholder = localize('auto (自动生成高强度随机密码)', 'auto (generate a strong random password)');
    } else if (isSocks) {
        credLabel.innerText = localize('Socks 认证密码 (留空免密)', 'SOCKS password (blank disables authentication)');
        if (credInput) credInput.placeholder = localize('留空免密直连', 'Leave blank for no authentication');
    } else {
        credLabel.innerText = localize('自定义客户端 UUID 凭据', 'Client UUID');
        if (credInput) credInput.placeholder = localize('auto (自动生成标准 UUID 凭证)', 'auto (generate a UUID)');
    }
}

export function setSingBoxPortAuto() {
    const input = document.getElementById('sb-add-port');
    if (input) input.value = 'auto';
}

export async function submitAddSingBoxNode(e) {
    if (e) e.preventDefault();
    const btn = document.getElementById('btn-submit-add-sb');
    btn.disabled = true;
    btn.innerText = localize('正在生成并部署...', 'Creating and deploying...');

    const portVal = document.getElementById('sb-add-port').value.trim() || 'auto';
    let sniVal = document.getElementById('sb-add-sni').value.trim() || 'auto';
    if (appState.currentSelectedProto === 'anytls' && (sniVal === 'auto' || !sniVal.includes('.'))) {
        sniVal = '';
    }
    const credVal = document.getElementById('sb-add-cred').value.trim() || 'auto';
    const outboundVal = document.getElementById('sb-add-outbound').value;

    const payload = {
        protocol: appState.currentSelectedProto,
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
        showToast(localize(`已成功创建 ${ret.node?.protocol || appState.currentSelectedProto} 入站节点 (端口: ${ret.node?.port || 'auto'})`, `Created ${ret.node?.protocol || appState.currentSelectedProto} inbound (port ${ret.node?.port || 'auto'})`));
        await fetchSingBoxOverview();
    } catch (err) {
        alert(localize('网络请求失败: ', 'Network request failed: ') + err);
    } finally {
        btn.disabled = false;
        btn.innerText = localize('立即创建并部署', 'Create and deploy');
    }
}

export async function updateNodeOutbound(nodeName, outboundAddr) {
    try {
        const res = await fetch('/api/singbox/nodes/outbound', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ target: nodeName, outbound: outboundAddr })
        });
        const ret = await res.json();
        if (!res.ok) {
            alert(localize('修改出口失败: ', 'Failed to change egress: ') + (ret.error || localize('未知错误', 'Unknown error')));
            await fetchSingBoxOverview();
            return;
        }
        showToast(localize(`节点 [${nodeName}] 链式出口已切换为: ${outboundAddr}`, `[${nodeName}] egress changed to: ${outboundAddr}`));
        await fetchSingBoxOverview();
    } catch (err) {
        alert(localize('网络请求失败: ', 'Network request failed: ') + err);
    }
}

export function updateNodeOutboundFromSelect(event, element) {
    const select = element || (event?.target?.closest ? event.target.closest('select') : null) || event?.target || this;
    if (!select || !select.dataset) return;
    updateNodeOutbound(select.dataset.nodeName, select.value);
}

export async function batchSetSingBoxOutbound(targetType) {
    if (!appState.singBoxOverview || !appState.singBoxOverview.installed || !appState.singBoxOverview.nodes || appState.singBoxOverview.nodes.length === 0) {
        tAlert('当前没有活跃的 sing-box 节点可供操作', 'No active sing-box nodes available');
        return;
    }

    let targetOutbound = 'direct';
    let label = localize('直连 (VPS 原生机房网络)', 'Direct (VPS network)');
    if (targetType === 'default') {
        const defOb = (appState.singBoxOverview.available_outbounds || []).find(o => o.is_default);
        targetOutbound = defOb ? defOb.addr : '127.0.0.1:7928';
        label = localize(`NXGate 默认住宅出口 (${targetOutbound})`, `NXGate default egress (${targetOutbound})`);
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
        showToast(localize(`已成功将 ${ret.updated_count || '全部'} 个节点切换至: ${targetOutbound}`, `Updated ${ret.updated_count || 'all'} nodes to egress: ${targetOutbound}`));
        await fetchSingBoxOverview();
    } catch (err) {
        alert(localize('网络请求失败: ', 'Network request failed: ') + err);
    }
}

export async function deleteSingBoxNode(nodeName) {
    if (!tConfirm(`确认彻底删除 sing-box 入站配置 [${nodeName}] 吗？`, `Are you sure you want to delete sing-box inbound [${nodeName}]?`)) return;
    try {
        const res = await fetch(`/api/singbox/nodes?target=${encodeURIComponent(nodeName)}`, {
            method: 'DELETE'
        });
        const ret = await res.json();
        if (!res.ok) {
            alert(localize('删除失败: ', 'Delete failed: ') + (ret.error || localize('未知错误', 'Unknown error')));
            return;
        }
        showToast(localize(`节点 [${nodeName}] 已成功删除并重构订阅`, `Deleted [${nodeName}] and rebuilt subscriptions`));
        await fetchSingBoxOverview();
    } catch (err) {
        alert(localize('网络请求失败: ', 'Network request failed: ') + err);
    }
}

export function getGenericSubURL() {
    if (appState.singBoxOverview && appState.singBoxOverview.subscription && appState.singBoxOverview.subscription.sub_url) {
        return sanitizeSubURL(appState.singBoxOverview.subscription.sub_url);
    }
    const origin = window.location.origin;
    const prefix = window.__apiPrefix || '';
    return `${origin}${prefix}/api/singbox/subscription`;
}

export async function copySingBoxSubURL() {
    if (!appState.singBoxOverview || !appState.singBoxOverview.installed) {
        tAlert('sing-box 未安装，无法获取远程订阅', 'sing-box not installed; cannot fetch remote subscription');
        return;
    }
    const url = getGenericSubURL();
    const isAge = appState.singBoxOverview.subscription && appState.singBoxOverview.subscription.age_encrypt_enabled && appState.singBoxOverview.subscription.age_public_key;
    copyText(url, isAge ? localize('age 端到端加密的通用订阅链接 (Base64/Raw)', 'age-encrypted subscription URL (Base64/Raw)') : localize('sing-box 全量通用订阅链接 (Base64/Raw)', 'Full sing-box subscription URL (Base64/Raw)'));
}

export function getClashSubURL() {
    if (appState.singBoxOverview && appState.singBoxOverview.subscription && appState.singBoxOverview.subscription.clash_sub_url) {
        return sanitizeSubURL(appState.singBoxOverview.subscription.clash_sub_url);
    }
    const origin = window.location.origin;
    const prefix = window.__apiPrefix || '';
    return `${origin}${prefix}/api/singbox/subscription/clash`;
}

export function copyClashSubURL() {
    if (!appState.singBoxOverview || !appState.singBoxOverview.installed) {
        tAlert('sing-box 未安装，无法获取 Clash 订阅', 'sing-box not installed; cannot fetch Clash subscription');
        return;
    }
    const url = getClashSubURL();
    const isAge = appState.singBoxOverview.subscription && appState.singBoxOverview.subscription.age_encrypt_enabled && appState.singBoxOverview.subscription.age_public_key;
    copyText(url, isAge ? localize('age 端到端加密的 Clash Meta 专属订阅链接', 'age-encrypted Clash Meta subscription URL') : localize('Clash Meta / Mihomo 专属订阅链接', 'Clash Meta / Mihomo subscription URL'));
}

export function downloadClashConfig() {
    const url = getClashSubURL();
    const a = document.createElement('a');
    a.href = url;
    a.download = 'singbox-clash.yaml';
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    showToast(localize('已触发下载 singbox-clash.yaml', 'Downloading singbox-clash.yaml'));
}

export function copyNodeShareLink(url, proto) {
    url = sanitizeShareURL(url);
    if (!url) {
        tAlert('该节点暂无有效客户端分享链接', 'No valid client share link for this node');
        return;
    }
    copyText(url, localize(`${proto || '代理'} 客户端分享链接`, `${proto || 'Proxy'} client share link`));
}

export function showNodeQRCode(url, title) {
    url = sanitizeShareURL(url);
    if (!url) {
        tAlert('暂无分享链接', 'No share link available');
        return;
    }
    document.getElementById('sb-qr-title').innerText = title || localize('客户端配置链接与二维码', 'Client configuration URL and QR code');
    document.getElementById('sb-qr-url-text').value = url;
    const qrImg = document.getElementById('sb-qr-img');
    qrImg.src = `https://api.qrserver.com/v1/create-qr-code/?size=200x200&data=${encodeURIComponent(url)}`;
    document.getElementById('singbox-qr-modal').classList.add('open');
}

export function buildFilteredSubParams() {
    const scope = document.getElementById('sub-filter-scope')?.value || '';
    const proto = document.getElementById('sub-filter-proto')?.value || '';
    const ping = document.getElementById('sub-filter-ping')?.value || '';
    const customCountry = (document.getElementById('sub-filter-custom-country')?.value || '').trim();

    const params = new URLSearchParams();
    if (scope === 'favorites') {
        params.set('filter', 'favorites');
    } else if (scope === 'proxy' || scope === 'relayed') {
        params.set('filter', 'proxy');
    } else if (scope === 'direct') {
        params.set('filter', 'direct');
    } else if (scope) {
        params.set('countries', scope);
    }
    if (customCountry) {
        // 如果用户输入的是有效国家代码（例如 JP,US,KR），则设置，防止输入整条 URL 导致污染
        const sanitizedCountry = customCountry.split(',')
            .map(c => c.trim().toUpperCase())
            .filter(c => c && /^[A-Z]{2,10}$/.test(c))
            .join(',');
        if (sanitizedCountry) {
            params.set('countries', sanitizedCountry);
        }
    }
    if (proto) {
        params.set('protocol', proto);
    }
    if (ping) {
        params.set('max_ping', ping);
    }
    return params;
}

export function getFilteredGenericSubURL() {
    const base = getGenericSubURL();
    const u = new URL(base, window.location.origin);
    const params = buildFilteredSubParams();
    // 先清除原有可能会重复的参数
    ['filter', 'countries', 'country', 'protocol', 'protocols', 'max_ping', 'max_latency'].forEach(k => u.searchParams.delete(k));
    params.forEach((v, k) => u.searchParams.set(k, v));
    return u.toString();
}

export function getFilteredClashSubURL() {
    const base = getClashSubURL();
    const u = new URL(base, window.location.origin);
    const params = buildFilteredSubParams();
    // 先清除原有可能会重复的参数
    ['filter', 'countries', 'country', 'protocol', 'protocols', 'max_ping', 'max_latency'].forEach(k => u.searchParams.delete(k));
    params.forEach((v, k) => u.searchParams.set(k, v));
    return u.toString();
}

export function updateSubFilterPreview() {
    const genInput = document.getElementById('sub-filter-preview-generic');
    const clashInput = document.getElementById('sub-filter-preview-clash');
    if (genInput) genInput.value = getFilteredGenericSubURL();
    if (clashInput) clashInput.value = getFilteredClashSubURL();
}

export function openSubFilterModal() {
    if (!appState.singBoxOverview || !appState.singBoxOverview.installed) {
        tAlert('sing-box 未安装，无法生成定制订阅', 'sing-box not installed; cannot generate customized subscription');
        return;
    }
    updateSubFilterPreview();
    document.getElementById('sub-filter-modal')?.classList.add('open');
}

export function closeSubFilterModal() {
    document.getElementById('sub-filter-modal')?.classList.remove('open');
}

export function copyFilteredGenericSub() {
    const url = getFilteredGenericSubURL();
    copyText(url, localize('定制通用订阅链接 (Base64/Raw)', 'Filtered subscription URL (Base64/Raw)'));
}

export function copyFilteredClashSub() {
    const url = getFilteredClashSubURL();
    copyText(url, localize('定制 Clash Meta 订阅链接', 'Filtered Clash Meta subscription URL'));
}

export function copyQuickRelayedSub() {
    if (!appState.singBoxOverview || !appState.singBoxOverview.installed) {
        tAlert('sing-box 未安装，无法生成定制订阅', 'sing-box not installed; cannot generate customized subscription');
        return;
    }
    const base = getGenericSubURL();
    const u = new URL(base, window.location.origin);
    u.searchParams.set('filter', 'proxy');
    copyText(u.toString(), localize('所有转接出口定制订阅链接', 'Relayed egress subscription URL'));
}

export function showFilteredQR() {
    const url = getFilteredGenericSubURL();
    showNodeQRCode(url, localize('定制订阅链接与二维码', 'Filtered subscription URL and QR code'));
}


export async function openAgeKeyHelper() {
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

export function closeAgeKeyHelper() {
    document.getElementById('age-key-helper-modal').classList.remove('open');
}

export function switchAgeType(type) {
    appState.currentAgeType = type;
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

export async function generateAgeKey() {
    try {
        const res = await fetch(`/api/singbox/subscription/age/generate?type=${encodeURIComponent(appState.currentAgeType)}`, {
            method: 'POST'
        });
        const data = await res.json();
        if (!res.ok || !data.ok) {
            tAlert('生成密钥失败: ' + (data.error || '未知错误'), 'Key generation failed: ' + (data.error || 'Unknown error'));
            return;
        }
        document.getElementById('age-secret-key-input').value = data.secret_key;
        document.getElementById('age-public-key-input').value = data.public_key;
        showToast(localize(`已成功生成全新 ${data.type} 密钥对！解密私钥请妥善保存。`, `Generated a new ${data.type} key pair. Keep the private key safe.`));
    } catch (err) {
        alert(localize('网络请求失败: ', 'Network request failed: ') + err);
    }
}

export function copyAgeSecretKey() {
    const sec = document.getElementById('age-secret-key-input').value.trim();
    if (!sec) {
        tAlert('请先输入或生成 age 解密私钥', 'Please enter or generate an age secret key first');
        return;
    }
    copyText(sec, localize('age 解密私钥', 'age private key'));
}

export async function deriveAgePublicKey() {
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
        showToast(localize(`已成功推导出 ${data.type} 加密公钥！`, `Derived the ${data.type} public key.`));
    } catch (err) {
        alert(localize('网络请求失败: ', 'Network request failed: ') + err);
    }
}

export function copyAgePublicKey() {
    const pub = document.getElementById('age-public-key-input').value.trim();
    if (!pub) {
        tAlert('请先输入或生成 age 加密公钥', 'Please enter or generate an age public key first');
        return;
    }
    copyText(pub, localize('age 加密公钥', 'age public key'));
}

export async function applyAgePublicKey() {
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
            alert(localize('保存失败: ', 'Save failed: ') + (err.error || localize('未知错误', 'Unknown error')));
            return;
        }
        const cfgPub = document.getElementById('cfg-age-pubkey');
        if (cfgPub) cfgPub.value = pub;
        showToast(localize('已一键填入公钥并开启 age 订阅端到端加密！', 'Public key applied and age subscription encryption enabled.'));
        await fetchSingBoxOverview();
    } catch (err) {
        alert(localize('网络请求失败: ', 'Network request failed: ') + err);
    }
}

export async function onAgeToggleChanged() {
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
            alert(localize('保存开关失败: ', 'Failed to save setting: ') + (err.error || localize('未知错误', 'Unknown error')));
            toggleEl.checked = !isEnabled;
            return;
        }
        showToast(localize(`age 订阅端到端加密已${isEnabled ? '开启' : '关闭'}`, `age subscription encryption ${isEnabled ? 'enabled' : 'disabled'}`));
        await fetchSingBoxOverview();
    } catch (err) {
        alert(localize('网络请求失败: ', 'Network request failed: ') + err);
        toggleEl.checked = !isEnabled;
    }
}

export function closeSingBoxQRModal() {
    document.getElementById('singbox-qr-modal').classList.remove('open');
}
