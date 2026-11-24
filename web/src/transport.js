import { createStatusTransport } from './status-transport.js';
import { appState } from './state.js';
import { getLanguage, localize, tAlert } from './i18n-runtime.js';
import { renderStatus } from './modules/dashboard.js';
import { renderNodes, updateNodePoolSummary } from './modules/nodes.js';
import { appendLog, clearAppAlert, showAppAlert } from './utils.js';

export const apiPrefix = (() => {
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
            showAppAlert(localize(`服务返回 ${response.status}，部分数据可能未更新。`, `Service returned ${response.status}. Some data may be out of date.`));
        } else if (response.ok) {
            clearAppAlert();
        }
        return response;
    } catch (error) {
        showAppAlert(localize('无法连接管理服务，请检查网络或服务状态。', 'Cannot reach the management service. Check the connection and service status.'));
        throw error;
    }
};

export async function fetchUnlockCache() {
    try {
        const res = await fetch('/api/unlock');
        if (res.ok) {
            appState.cachedUnlockMap = await res.json() || {};
            renderNodes();
        }
    } catch(e) {}
}

export async function probeTunnelUnlock(tunnelId) {
    const isEn = getLanguage() === 'en';
    appendLog({ level: 'INFO', module: 'Unlock', message: localize(`正在对出口 [${tunnelId}] 启动流媒体与 AI 解锁探测...`, `Probing AI & streaming unlocks for exit [${tunnelId}]...`) });
    const btn = document.querySelector(`[data-action="probeTunnelUnlock"][data-args*="${tunnelId}"]`);
    if (btn) {
        btn.disabled = true;
        btn.innerHTML = `<span class="status-dot status-dot-warning animate-pulse"></span> ${isEn ? 'Probing...' : '探测中...'}`;
    }
    try {
        const res = await fetch('/api/unlock/probe', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ tunnel_id: tunnelId })
        });
        if (!res.ok) {
            const errData = await res.json().catch(() => ({}));
            tAlert('探测启动失败: ' + (errData.error || res.statusText), 'Probe failed: ' + (errData.error || res.statusText));
            if (btn) {
                btn.disabled = false;
                btn.innerText = isEn ? 'Unlock' : '测解锁';
            }
            return;
        }
        tAlert('已启动解锁探测，正在实时检测三大 AI 与流媒体，请稍候...', 'Unlock probing started, checking AI & streaming services...');
        let attempts = 0;
        const pollInterval = setInterval(async () => {
            attempts++;
            await fetchUnlockCache();
            fetchStatus();
            if (attempts >= 4) {
                clearInterval(pollInterval);
                if (btn) {
                    btn.disabled = false;
                    btn.innerText = isEn ? 'Unlock' : '测解锁';
                }
            }
        }, 2500);
    } catch(err) {
        tAlert('探测请求失败: ' + err, 'Probe request failed: ' + err);
        if (btn) {
            btn.disabled = false;
            btn.innerText = isEn ? 'Unlock' : '测解锁';
        }
    }
}

export async function fetchStatus() {
    if (appState.statusFetchInFlight) return;
    appState.statusFetchInFlight = true;
    const eventVersion = appState.statusEventVersion;
    const controller = new AbortController();
    const deadline = setTimeout(() => controller.abort(), 8000);
    try {
        const res = await fetch('/api/status', { signal: controller.signal });
        if (res.ok) {
            const data = await res.json();
            // Do not overwrite a newer pushed snapshot with an older HTTP response.
            if (eventVersion === appState.statusEventVersion) {
                renderStatus(data);
                updateNodePoolSummary();
            }
        }
    } catch (err) {
        console.error("fetch status error:", err);
    } finally { clearTimeout(deadline); appState.statusFetchInFlight = false; }
}

export function setupSSE() {
    if (appState.statusTransport) return;
    appState.statusTransport = createStatusTransport({
        url: (apiPrefix || '') + '/api/events',
        fetchStatus,
        renderStatus: data => { appState.statusEventVersion++; renderStatus(data); updateNodePoolSummary(); },
        onOpen: () => appendLog({ level: 'INFO', module: 'SSE', message: localize('已成功连通服务器实时推流总线', 'Live server updates connected') })
    });
    const es = appState.statusTransport.start();
    if (!es) return;
    es.addEventListener('log', (e) => {
        try {
            const entry = JSON.parse(e.data);
            appendLog(entry);
        } catch(err){}
    });

}
