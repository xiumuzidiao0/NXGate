/**
 * Shared mutable WebUI application state.
 */
export const appState = {
    allNodes: [],
    nodeScope: 'active',
    currentState: null,
    cachedUnlockMap: {},
    activeQuickFilter: 'all',
    statusFetchInFlight: false,
    statusEventVersion: 0,
    currentSort: 'latency_asc',
    nodesMap: null,
    nodesMarkerGroup: null,
    isMapCollapsed: false,
    statusTransport: null,
    currentPortRules: [],
    currentDynamicGroups: [],
    editingPort: null,
    editingGroupId: null,
    currentBlacklist: [],
    activeBlacklistTab: 'temp',
    activeEditorDrawer: null,
    singBoxOverview: null,
    currentSelectedProto: 'reality',
    currentAgeType: 'x25519',
    currentAppProfile: null,
    isUpdating: false,
};
