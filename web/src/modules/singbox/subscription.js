import { appState } from '../../state.js';
import { localize, tAlert } from '../../i18n-runtime.js';
import { copyText, sanitizeShareURL, sanitizeSubURL, showToast } from '../../utils.js';

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

export function closeSingBoxQRModal() {
    document.getElementById('singbox-qr-modal').classList.remove('open');
}
