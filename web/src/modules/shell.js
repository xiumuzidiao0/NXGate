import { appState } from '../state.js';
import { getLanguage } from '../i18n-runtime.js';
import { fetchDynamicGroups, fetchPortRules, renderDynamicGroups, renderPortRules } from './matrix.js';
import { initNodesMap, renderNodes, updateCountryFilter } from './nodes.js';
import { loadSettingsForm } from './settings.js';
import { fetchSingBoxOverview } from './singbox.js';

export function switchView(viewName) {
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
        nodes: 'VPN Nodes',
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
        if (appState.currentPortRules && appState.currentPortRules.length > 0) renderPortRules();
        if (appState.currentDynamicGroups && appState.currentDynamicGroups.length > 0) renderDynamicGroups();
        fetchPortRules();
        fetchDynamicGroups();
    }
    if (viewName === 'settings') loadSettingsForm();
    if (viewName === 'nodes') {
        updateCountryFilter();
        renderNodes();
        initNodesMap();
        setTimeout(() => { if (appState.nodesMap) appState.nodesMap.invalidateSize(); }, 150);
    }

    if (window.scrollY > 0) {
        window.scrollTo({ top: 0, behavior: 'instant' });
    }
}


export function openEditorDrawer(id) {
    const drawer = document.getElementById(id);
    const backdrop = document.getElementById('drawer-backdrop');
    if (!drawer) return;
    if (appState.activeEditorDrawer && appState.activeEditorDrawer !== drawer) {
        closeEditorDrawer(appState.activeEditorDrawer.id);
    }
    appState.activeEditorDrawer = drawer;
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

export function closeEditorDrawer(id) {
    const drawer = typeof id === 'string' ? document.getElementById(id) : id;
    const backdrop = document.getElementById('drawer-backdrop');
    if (!drawer) return;
    if (!drawer.classList.contains('open') && drawer.hidden) return;
    drawer.classList.remove('open');
    backdrop?.classList.remove('open');
    appState.activeEditorDrawer = null;
    document.body.classList.remove('drawer-open');
    window.setTimeout(() => {
        if (!drawer.classList.contains('open')) drawer.hidden = true;
        if (backdrop && !document.querySelector('.editor-drawer.open')) backdrop.hidden = true;
    }, 230);
}

export function enhanceInteractiveElements(root = document) {
    root.querySelectorAll('[data-action]').forEach(element => {
        if (element.matches('button, a, input, select, textarea')) return;
        element.setAttribute('role', 'button');
        element.setAttribute('tabindex', '0');
    });
}

// ==========================================
//  侧边栏展开/收起切换引擎 (Sidebar Collapse)
// ==========================================
export function toggleSidebar() {
    const sb = document.getElementById('app-sidebar');
    if (!sb) return;
    const isCollapsed = sb.classList.toggle('collapsed');
    try {
        localStorage.setItem('nxgate_sidebar_collapsed', isCollapsed ? '1' : '0');
        localStorage.removeItem('aimili_sidebar_collapsed');
    } catch(e){}
    updateSidebarUI(isCollapsed);
}

export function updateSidebarUI(isCollapsed) {
    const btn = document.getElementById('sidebar-collapse-btn');
    const hBtn = document.getElementById('btn-toggle-sidebar');
    const isEn = getLanguage() === 'en';
    const tooltip = isCollapsed ? (isEn ? 'Expand Sidebar' : '展开侧边栏') : (isEn ? 'Collapse Sidebar' : '收起侧边栏');
    if (btn) btn.setAttribute('title', tooltip);
    if (hBtn) hBtn.setAttribute('title', tooltip);
}

export function initSidebarState() {
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
