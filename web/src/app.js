import { appState } from './state.js';
import { applyLanguage, getLanguage, onUiLanguageChanged, toggleLanguage } from './i18n-runtime.js';
import { addNodeToBlacklist, clearPermBlacklist, clearTempBlacklist, closeBlacklistModal, fetchBlacklist, openBlacklistModal, promptAddCidrBlock, removeNodeFromBlacklist, resurrectBlacklist, switchBlacklistTab } from './modules/blacklist.js';
import { closePortMatrixModal, deleteDynamicGroup, deletePortRule, editDynamicGroup, editPortRule, evaluateDynamicGroups, fetchDynamicGroups, fetchPortRules, generateRandomPortAuth, hideDynamicGroupForm, hideEditPortForm, onAllTunnelsCheckChanged, onSpecificTunnelCheckChanged, openPortMatrixModal, renderMatrixTopology, saveDynamicGroup, savePortRule, selectAuthMode, selectPolicy, setIntervalVal, setPortVal, showAddDynamicGroupForm, showAddPortForm, startNewTunnel, stopTunnel, suggestNextPort, switchMatrixTab } from './modules/matrix.js';
import { connectToNode, disconnectVPN, fetchNodes, probeCurrentNodes, probeSingleNode, quickConnect, refreshNodes, renderNodes, resetMapView, selectNodeScope, selectQuickFilter, toggleFavorite, toggleMapCollapse, toggleSort } from './modules/nodes.js';
import { checkForUpdates, closeMobileAppModal, closeSettingsModal, closeUpdateModal, downloadBackupPackage, onAppProfileFieldChanged, onBackupFileSelected, onDisableAuthChanged, openMobileAppModal, openSystemPrimaryConfig, openUpdateModal, randomPassword, randomPath, saveSettings, switchSettingsTab, testMobileApiConnectivity, testTelegramAlert, triggerBackupFileSelect, triggerForceUpdate, triggerSystemUpdate } from './modules/settings.js';
import { closeEditorDrawer, enhanceInteractiveElements, initSidebarState, switchView, toggleSidebar } from './modules/shell.js';
import { applyAgePublicKey, batchSetSingBoxOutbound, closeAddSingBoxModal, closeAgeKeyHelper, closeSingBoxQRModal, closeSubFilterModal, copyAgePublicKey, copyAgeSecretKey, copyClashSubURL, copyFilteredClashSub, copyFilteredGenericSub, copyNodeShareLink, copyQuickRelayedSub, copySingBoxSubURL, deleteSingBoxNode, deriveAgePublicKey, downloadClashConfig, fetchSingBoxOverview, filterProtoGrid, generateAgeKey, onAgeToggleChanged, openAddSingBoxModal, openAgeKeyHelper, openSubFilterModal, selectSingBoxProto, setSingBoxPortAuto, showFilteredQR, showNodeQRCode, submitAddSingBoxNode, switchAgeType, updateNodeOutbound, updateNodeOutboundFromSelect, updateSubFilterPreview } from './modules/singbox.js';
import { fetchStatus, fetchUnlockCache, probeTunnelUnlock, setupSSE } from './transport.js';
import { clearLogs, copyFromElement, copyText } from './utils.js';
import './transport.js';

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
    probeSingleNode,
    refreshNodes,
    selectNodeScope,
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
    fetchBlacklist,
    renderNodes,
    saveSettings,
    onDisableAuthChanged,
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
    copyQuickRelayedSub,
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
    onUiLanguageChanged,
    resetMapView,
    toggleMapCollapse
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
        closeEditorDrawer(appState.activeEditorDrawer);
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
    if (event.key === 'Escape' && appState.activeEditorDrawer) {
        closeEditorDrawer(appState.activeEditorDrawer);
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


    setInterval(fetchSingBoxOverview, 10000);
    setInterval(fetchNodes, 15000);
    setInterval(fetchUnlockCache, 30000);
};
    
