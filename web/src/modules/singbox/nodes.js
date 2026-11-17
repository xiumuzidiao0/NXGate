import { appState } from '../../state.js';
import { formatOutboundLabel, getLanguage, localize, tAlert, tConfirm } from '../../i18n-runtime.js';
import { alert, escapeHtml, jsonAttr, showToast } from '../../utils.js';

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

    if (!(await tConfirm(`确定将所有 sing-box 入站节点批量切换至【${label}】吗？`, `Switch all sing-box inbound nodes to [${label}]?`))) return;

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
    if (!(await tConfirm(`确认彻底删除 sing-box 入站配置 [${nodeName}] 吗？`, `Are you sure you want to delete sing-box inbound [${nodeName}]?`))) return;
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
