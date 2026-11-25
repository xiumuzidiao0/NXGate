import { getLanguage, localize, t } from './i18n-runtime.js';

export const countryNames = {
    JP: '日本', US: '美国', KR: '韩国', TW: '台湾', HK: '香港',
    SG: '新加坡', GB: '英国', DE: '德国', FR: '法国', CA: '加拿大',
    AU: '澳大利亚', VN: '越南', TH: '泰国', MY: '马来西亚', IN: '印度',
    RU: '俄罗斯', NL: '荷兰', BR: '巴西', PH: '菲律宾', ID: '印尼',
    IT: '意大利', ES: '西班牙', SE: '瑞典', CH: '瑞士', NZ: '新西兰',
    PL: '波兰', UA: '乌克兰', TR: '土耳其', ZA: '南非', AR: '阿根廷',
    CL: '智利', CO: '哥伦比亚', MX: '墨西哥', NO: '挪威', FI: '芬兰',
    DK: '丹麦', IE: '爱尔兰', AT: '奥地利', BE: '比利时', CZ: '捷克',
    RO: '罗马尼亚', IL: '以色列', AE: '阿联酋', SA: '沙特阿拉伯', CN: '中国',
    PT: '葡萄牙', GR: '希腊', HU: '匈牙利', SK: '斯洛伐克', BG: '保加利亚',
    HR: '克罗地亚', RS: '塞尔维亚', LT: '立陶宛', LV: '拉脱维亚', EE: '爱沙尼亚',
    KZ: '哈萨克斯坦', PK: '巴基斯坦', KH: '柬埔寨', PE: '秘鲁'
};

export const countryNamesEn = {
    JP: 'Japan', US: 'United States', KR: 'South Korea', TW: 'Taiwan', HK: 'Hong Kong',
    SG: 'Singapore', GB: 'United Kingdom', DE: 'Germany', FR: 'France', CA: 'Canada',
    AU: 'Australia', VN: 'Vietnam', TH: 'Thailand', MY: 'Malaysia', IN: 'India',
    RU: 'Russia', NL: 'Netherlands', BR: 'Brazil', PH: 'Philippines', ID: 'Indonesia',
    IT: 'Italy', ES: 'Spain', SE: 'Sweden', CH: 'Switzerland', NZ: 'New Zealand',
    PL: 'Poland', UA: 'Ukraine', TR: 'Turkey', ZA: 'South Africa', AR: 'Argentina',
    CL: 'Chile', CO: 'Colombia', MX: 'Mexico', NO: 'Norway', FI: 'Finland',
    DK: 'Denmark', IE: 'Ireland', AT: 'Austria', BE: 'Belgium', CZ: 'Czech Republic',
    RO: 'Romania', IL: 'Israel', AE: 'UAE', SA: 'Saudi Arabia', CN: 'China',
    PT: 'Portugal', GR: 'Greece', HU: 'Hungary', SK: 'Slovakia', BG: 'Bulgaria',
    HR: 'Croatia', RS: 'Serbia', LT: 'Lithuania', LV: 'Latvia', EE: 'Estonia',
    KZ: 'Kazakhstan', PK: 'Pakistan', KH: 'Cambodia', PE: 'Peru'
};

let displayNamesZh = null;
let displayNamesEn = null;

function getRegionDisplayNames(lang) {
    if (typeof Intl === 'undefined' || typeof Intl.DisplayNames !== 'function') return null;
    try {
        if (lang === 'en') {
            return displayNamesEn || (displayNamesEn = new Intl.DisplayNames(['en'], { type: 'region' }));
        }
        return displayNamesZh || (displayNamesZh = new Intl.DisplayNames(['zh-Hans'], { type: 'region' }));
    } catch {
        return null;
    }
}

export const flagSVGs = {
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

export function getCountryFlagSVG(code) {
    code = (code || '').toUpperCase();
    if (flagSVGs[code]) return flagSVGs[code];
    return '<svg aria-hidden="true" class="flag-img flag-fallback" viewBox="0 0 24 24"><circle cx="12" cy="12" r="10" stroke="#64748b" fill="none"/><line x1="2" y1="12" x2="22" y2="12" stroke="#64748b"/></svg>';
}

export function getCountryName(code) {
    code = (code || '').trim().toUpperCase();
    if (!code) return '';
    const normalized = code === 'UK' ? 'GB' : code;
    const lang = getLanguage() === 'en' ? 'en' : 'zh';
    const map = lang === 'en' ? countryNamesEn : countryNames;
    if (map[normalized]) return map[normalized];
    const dn = getRegionDisplayNames(lang);
    if (dn) {
        try {
            const name = dn.of(normalized);
            if (name && name.toUpperCase() !== normalized) return name;
        } catch {}
    }
    return code;
}

export function escapeHtml(value) {
    return String(value ?? '').replace(/[&<>"']/g, c => ({
        '&': '&amp;',
        '<': '&lt;',
        '>': '&gt;',
        '"': '&quot;',
        "'": '&#39;'
    })[c]);
}

export function jsonAttr(value) {
    return escapeHtml(JSON.stringify(value));
}

export function safeLogLevel(level) {
    const value = String(level || 'info').toLowerCase();
    return ['info', 'warning', 'error'].includes(value) ? value : 'info';
}

export function formatBytes(bytes) {
    if (!bytes || bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
}

export function formatSpeed(bps) { return formatBytes(bps) + '/s'; }

export function appendLog(entry) {
    const term = document.getElementById('terminal');
    const row = document.createElement('div');
    row.className = 'log-line';
    const timeStr = entry.timestamp ? new Date(entry.timestamp).toLocaleTimeString() : new Date().toLocaleTimeString();
    row.innerHTML = `<span class="log-time">[${escapeHtml(timeStr)}]</span> <span class="log-level-${safeLogLevel(entry.level)}">[${escapeHtml(entry.module || 'System')}]</span> ${escapeHtml(entry.message)}`;
    term.appendChild(row);
    term.scrollTop = term.scrollHeight;
}

export function clearLogs() { document.getElementById('terminal').innerHTML = ''; }

export function renderUnlockBadges(u) {
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
    const tpTag = (u.is_probed && u.throughput_bytes_per_sec > 0)
        ? `<span class="badge badge-mini badge-info" title="${isEn ? 'Measured egress download throughput' : '实测隧道吞吐速度'}">⚡ ${formatSpeed(u.throughput_bytes_per_sec)}</span>`
        : '';
    return `<div class="unlock-badges">${tag}${badges.join('')}${tpTag}</div>`;
}

export function showToast(text, type = 'success', duration = 2500) {
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

export function alert(message) {
    const text = String(message ?? (getLanguage() === 'en' ? 'Action incomplete' : '操作未完成'));
    const type = /失败|错误|异常|无法|不能|不存在|请输入|未安装|尚未|暂无|fail|error|cannot|invalid|not|missing/i.test(text) ? 'error' : 'info';
    showToast(text, type, type === 'error' ? 5000 : 3200);
}

export function showAppAlert(message) {
    const el = document.getElementById('app-alert');
    if (!el) return;
    el.textContent = message;
    el.hidden = false;
}

export function clearAppAlert() {
    const el = document.getElementById('app-alert');
    if (!el) return;
    el.hidden = true;
    el.textContent = '';
}

export function copyText(text, label) {
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
    showToast(localize(`已复制: ${label || text}`, `Copied: ${label || text}`));
}

export function sanitizeSubURL(url) {
    if (!url) return '';
    const curOrigin = window.location.origin;
    // 只要当前浏览器访问的不是 127.0.0.1 或 localhost，就统一将其替换为当前浏览器的真实 Origin
    // （支持域名、公网 IP，且自动匹配当前实际使用的 http/https 协议和端口）
    if (window.location.hostname !== '127.0.0.1' && window.location.hostname !== 'localhost') {
        url = url.replace(/^[a-zA-Z]+:\/\/(127\.0\.0\.1|localhost)(:\d+)?/, curOrigin);
    }
    return url;
}

export function sanitizeShareURL(url) {
    if (!url) return '';
    const host = window.location.hostname || '127.0.0.1';
    return url.replace(/@auto(:|#)/g, `@${host}$1`).replace(/-auto$/g, `-${host}`);
}

export function copyFromElement(elId) {
    const el = document.getElementById(elId);
    if (el) {
        copyText(el.value, localize('链接', 'URL'));
    }
}
