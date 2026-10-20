// EventSource owns reconnection. A single fallback timer exists only while disconnected.
export function createStatusTransport({url, fetchStatus, renderStatus, onOpen = () => {},
    EventSourceClass = globalThis.EventSource, setTimer = setInterval, clearTimer = clearInterval}) {
    let source = null;
    let poll = null;
    const stopPolling = () => { if (poll !== null) { clearTimer(poll); poll = null; } };
    const startPolling = () => { if (poll === null) poll = setTimer(fetchStatus, 2000); };
    return {
        start() {
            if (source) return source;
            // Covers browsers without EventSource and slow/failed initial connections.
            startPolling();
            if (!EventSourceClass) return null;
            try { source = new EventSourceClass(url); } catch { return null; }
            source.onopen = () => { stopPolling(); onOpen(); };
            source.onerror = startPolling;
            source.addEventListener('status', event => {
                try { renderStatus(JSON.parse(event.data)); }
                catch (error) { console.error('Invalid status event:', error); startPolling(); return; }
                stopPolling();
            });
            return source;
        },
        stop() { if (source) source.close(); source = null; stopPolling(); }
    };
}
