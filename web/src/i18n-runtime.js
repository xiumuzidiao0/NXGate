import { I18N } from './i18n.js';
import { appState } from './state.js';
import { renderStatus } from './modules/dashboard.js';
import { renderDynamicGroups, renderPortRules, renderTunnelCheckboxes } from './modules/matrix.js';
import { renderNodes, updateCountryFilter } from './modules/nodes.js';
import { switchView } from './modules/shell.js';
import { renderSingBox, updateProtoHelpText } from './modules/singbox.js';
import { alert, showToast } from './utils.js';

export function getLanguage() {
    try {
        const saved = localStorage.getItem('nxgate_lang');
        if (saved === 'en' || saved === 'zh') return saved;
    } catch(e) {}
    return 'zh';
}

export function t(key, fallback) {
    const lang = getLanguage();
    if (I18N[lang] && I18N[lang][key] !== undefined) {
        return I18N[lang][key];
    }
    if (I18N.zh && I18N.zh[key] !== undefined) {
        return I18N.zh[key];
    }
    return fallback !== undefined ? fallback : key;
}

export function tAlert(zhMsg, enMsg) {
    alert(getLanguage() === 'en' ? enMsg : zhMsg);
}

// Localize app-owned dynamic copy, not user names, backend errors, or logs.
export function localize(zh, en) {
    return getLanguage() === 'en' ? en : zh;
}

export const NODE_SOURCE_MAP_EN = {
    'GitHub Pages 实时镜像': 'GitHub Pages Live Mirror',
    'VPNGate 官方 HTTPS API': 'VPNGate Official HTTPS API',
    'VPNGate 官方 HTTP API': 'VPNGate Official HTTP API',
    'NXGate 实时镜像 (Data 分支)': 'NXGate Live Mirror (Data)',
    'NXGate 高速压缩镜像 (Gzip)': 'NXGate Gzip Mirror',
    'GitHub Raw 直链镜像': 'GitHub Raw Mirror',
    'Fastly 全球加速 CDN': 'Fastly Global CDN',
    'GitHub 镜像加速源': 'GitHub Accelerated Mirror',
    '用户 GitHub 镜像源 (Main 种子)': 'GitHub Main Seed Mirror',
    'jsDelivr 全球加速 CDN': 'jsDelivr Global CDN',
    '本地快照': 'Local Snapshot',
    '未知': 'Unknown',
};

function pluralUnit(count, singular, plural) {
    return `${count} ${count === 1 ? singular : plural}`;
}

function translateNodeSourcePart(part) {
    const trimmed = String(part || '').trim();
    return NODE_SOURCE_MAP_EN[trimmed] || trimmed;
}

export function localizeNodeSource(src) {
    if (!src) return localize('未知', 'Unknown');
    if (getLanguage() !== 'en') return src;
    const raw = String(src).trim();
    let m;
    if ((m = raw.match(/^多源聚合\s*\((.+?)\s*等\s*(\d+)\s*个在线数据源\)$/))) {
        const primary = translateNodeSourcePart(m[1]);
        const count = parseInt(m[2], 10);
        return `Multi-Source (${primary} + ${pluralUnit(count, 'online source', 'online sources')})`;
    }
    if ((m = raw.match(/^本地快照\s*\((.+)\)$/))) {
        return `Local Snapshot (${translateNodeSourcePart(m[1])})`;
    }
    if (raw === '本地快照') {
        return 'Local Snapshot';
    }
    return raw.split(/\s*\+\s*/).map(translateNodeSourcePart).join(' + ');
}

export function localizeLastMessage(msg) {
    if (!msg) return localize('服务正常运行中', 'Service running normally');
    if (getLanguage() !== 'en') return msg;
    const s = String(msg);
    const exact = {
        'VPN 服务已就绪，未连接': 'VPN service ready (disconnected)',
        '服务就绪': 'Service ready',
        '隧道池未初始化': 'Tunnel pool not initialized',
        '连接超时': 'Connection timed out',
        '连接异常中断 (进程已退出)': 'Connection interrupted (process exited)',
        'VPN 隧道已成功连通并就绪': 'VPN tunnel connected and ready',
        '检测到隧道中断，已转交主出口动态组按既定国家/纯净度约束重新选路...': 'Tunnel interrupted; handed off to primary egress group for re-routing...',
        '检测到隧道中断，正在挑选最优新节点自动切换...': 'Tunnel interrupted; selecting optimal replacement node...',
    };
    if (exact[s]) return exact[s];
    let m;
    if ((m = s.match(/^用户态免 TUN 直连已就绪 \(虚拟 IP: ([^,]+), 端口: (\d+)\)$/))) {
        return `Zero-TUN userland connected (Virtual IP: ${m[1]}, Port: ${m[2]})`;
    }
    if ((m = s.match(/^正在发起对节点 (\S+) \(([^)]+)\) 的内核 OpenVPN 连接\.\.\.$/))) {
        return `Connecting to ${m[1]} (${m[2]}) via kernel OpenVPN...`;
    }
    if ((m = s.match(/^正在通过用户态 SSTP \(免 TUN\) 连接 (\S+) \(([^)]+)\)\.\.\.$/))) {
        return `Connecting to ${m[1]} (${m[2]}) via userland Zero-TUN (SSTP)...`;
    }
    if ((m = s.match(/^用户态隧道 (\S+) 握手成功但无法转发 IP 报文$/))) {
        return `Userland tunnel ${m[1]} completed handshake but failed to forward packets`;
    }
    if ((m = s.match(/^启动主隧道失败:\s*(.+)$/))) {
        return `Failed to start primary tunnel: ${m[1]}`;
    }
    if ((m = s.match(/^用户态 SSTP 直连失败:\s*(.+)$/))) {
        return `Zero-TUN SSTP connection failed: ${m[1]}`;
    }
    if ((m = s.match(/^已断开连接:\s*(.+)$/))) {
        return `Disconnected: ${m[1]}`;
    }
    return s;
}

export function localizeGroupStatus(text) {
    if (!text) return localize('正常', 'Normal');
    if (getLanguage() !== 'en') return text;
    const exact = {
        '正常': 'Normal',
        '运行中': 'Running',
        '主网关正常运行': 'Primary Gateway Running',
        '未开启': 'Disabled',
        '降级运行中': 'Fallback Active',
        '正在选路...': 'Selecting...',
    };
    return exact[text] || text;
}

export function formatOutboundLabel(ob) {
    if (getLanguage() !== 'en') return ob.label;
    if (ob.type === 'direct' || ob.addr === 'direct') {
        return 'Direct Exit (VPS native network, no chain)';
    }
    const portMatch = ob.port || (ob.addr && ob.addr.match(/:(\d+)/)?.[1]) || '';
    const hasAuth = typeof ob.auth_enabled === 'boolean'
        ? ob.auth_enabled
        : ((ob.addr && ob.addr.includes('@')) || (ob.label && ob.label.includes('密码保护')));
    const authNote = hasAuth ? 'Auth' : 'No Auth';
    if (ob.is_default) {
        return `NXGate Default Exit (PORT ${portMatch || '7928'} - SOCKS5 TCP+UDP ${authNote})`;
    }
    const tunnelCount = typeof ob.tunnel_count === 'number'
        ? ob.tunnel_count
        : parseInt(ob.label?.match(/(\d+)\s*隧道/)?.[1] || '0', 10);
    const groupCount = typeof ob.group_count === 'number'
        ? ob.group_count
        : parseInt(ob.label?.match(/(\d+)\s*动态组/)?.[1] || '0', 10);
    const parts = [];
    if (tunnelCount > 0) parts.push(pluralUnit(tunnelCount, 'tunnel', 'tunnels'));
    if (groupCount > 0) parts.push(pluralUnit(groupCount, 'group', 'groups'));
    const boundDesc = parts.length ? ` - ${parts.join(', ')}` : '';
    return `Multi-Port Exit (PORT ${portMatch} - SOCKS5 TCP+UDP ${authNote}${boundDesc})`;
}

export function tConfirm(zhMsg, enMsg) {
    return confirm(getLanguage() === 'en' ? enMsg : zhMsg);
}

export function tPrompt(zhMsg, enMsg, defaultVal) {
    return prompt(getLanguage() === 'en' ? enMsg : zhMsg, defaultVal);
}

export function applyLanguage(lang = getLanguage(), container = document) {
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

export function toggleLanguage() {
    const nextLang = getLanguage() === 'zh' ? 'en' : 'zh';
    setLanguage(nextLang);
}

export function onUiLanguageChanged(event) {
    const newLang = event?.target?.value || 'zh';
    setLanguage(newLang);
}

export function setLanguage(lang) {
    try {
        localStorage.setItem('nxgate_lang', lang);
    } catch(e) {}
    applyLanguage(lang);
    const hash = (window.location.hash || '#dashboard').replace(/^#/, '');
    switchView(hash);
    if (appState.currentState) {
        renderStatus(appState.currentState);
    } else {
        const badge = document.getElementById('conn-badge');
        if (badge) badge.innerHTML = `<span class="status-dot"></span> ${t('header.checking')}`;
    }
    if (appState.allNodes.length > 0) renderNodes(appState.allNodes);
    if (appState.singBoxOverview) renderSingBox(appState.singBoxOverview);
    if (appState.currentDynamicGroups) renderDynamicGroups();
    if (appState.currentPortRules) renderPortRules();
    updateCountryFilter();
    if (document.getElementById('port-edit-card')?.classList.contains('open')) {
        const selectedTunnels = [...document.querySelectorAll('.chk-single-tunnel:checked')].map(el => el.value);
        const selectedGroups = [...document.querySelectorAll('.chk-dynamic-group:checked')].map(el => el.value);
        renderTunnelCheckboxes(selectedTunnels, selectedGroups);
    }
    updateProtoHelpText(appState.currentSelectedProto);
    showToast(lang === 'en' ? 'Language switched to English' : '已切换至简体中文');
}
