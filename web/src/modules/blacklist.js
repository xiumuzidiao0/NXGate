import { appState } from '../state.js';
import { getLanguage, tAlert, tConfirm, tPrompt } from '../i18n-runtime.js';
import { fetchNodes } from './nodes.js';
import { fetchStatus } from '../transport.js';
import { fetchDynamicGroups } from './matrix/groups.js';
import { escapeHtml, getCountryFlagSVG, jsonAttr, showToast } from '../utils.js';

export async function openBlacklistModal() {
    document.getElementById('blacklist-modal').classList.add('open');
    await fetchBlacklist();
}

export function closeBlacklistModal() {
    document.getElementById('blacklist-modal').classList.remove('open');
}

export async function fetchBlacklist(showNotice = false) {
    try {
        const res = await fetch('/api/blacklist');
        if (!res.ok) return;
        appState.currentBlacklist = await res.json() || [];
        renderBlacklist(appState.currentBlacklist);
        if (showNotice) {
            showToast(getLanguage() === 'en' ? 'Blacklist updated' : '屏蔽库列表已刷新');
        }
    } catch (err) {
        console.error("fetch blacklist error:", err);
    }
}


export function switchBlacklistTab(tab) {
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

export function renderBlacklist(items) {
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

export async function removeNodeFromBlacklist(nodeId, ip) {
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

export async function clearTempBlacklist() {
    if (!(await tConfirm('确定清空临时故障隔离池吗？隔离节点将立即放回可用候选池重测。', 'Clear temporary quarantine? Nodes will return to candidate pool for probing.'))) return;
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

export async function clearPermBlacklist() {
    if (!(await tConfirm('确定清空所有用户永久黑名单吗？被拉黑的节点将重新在节点列表中显示。', 'Are you sure you want to clear all permanent blacklisted nodes? They will reappear in the node list.'))) return;
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

export async function resurrectBlacklist() {
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

export async function addNodeToBlacklist(nodeId, ip, country, groupId = '') {
    let derivedSubnet = '';
    if (ip) {
        const parts = ip.split('.');
        if (parts.length === 4) {
            derivedSubnet = `${parts[0]}.${parts[1]}.${parts[2]}.0/24`;
        }
    }

    const promptZh = `请选择对节点 [${nodeId}] 的屏蔽方式：\n\n1 = 永久屏蔽此节点 (单机，永不连接，从列表彻底隐藏) [推荐]\n2 = 永久屏蔽整个 /24 网段 (${derivedSubnet || ip + '/24'} 动态规则，未来该网段新节点均自动屏蔽)\n3 = 永久屏蔽整机 IP (${ip} 所有端口)\n4 = 临时隔离 24 小时 (仅在故障隔离池中观察)\n\n请输入 1, 2, 3 或 4:`;
    const promptEn = `Choose blocking method for [${nodeId}]:\n\n1 = Permanent block (Single node, never connect, hidden) [Recommended]\n2 = Permanent /24 subnet block (${derivedSubnet || ip + '/24'} dynamic rule, auto-blocks future nodes)\n3 = Permanent IP block (${ip} all ports)\n4 = Temporary quarantine 24 hours\n\nEnter 1, 2, 3, or 4:`;
    const mode = await tPrompt(promptZh, promptEn, "1");
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
        const payload = {
            node_id: targetNodeId,
            ip: targetIP,
            country: country,
            duration_minutes: dur,
            permanent: permanent,
            scope: scope,
            reason: reason
        };
        if (groupId) {
            payload.group_id = groupId;
        }
        const res = await fetch('/api/blacklist/add', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(payload)
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
        if (Array.isArray(ret.evicted_tunnels) && ret.evicted_tunnels.length > 0) {
            msg += getLanguage() === 'en'
                ? ` · Evicted ${ret.evicted_tunnels.length} exit(s), rotating group`
                : ` · 已断开 ${ret.evicted_tunnels.length} 个在用出口并触发重评换线`;
        }
        showToast(msg);
        fetchStatus();
        fetchDynamicGroups();
        fetchNodes();
    } catch (err) {
        tAlert('请求失败: ' + err, 'Request failed: ' + err);
    }
}

export async function promptAddCidrBlock() {
    const input = await tPrompt(
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
