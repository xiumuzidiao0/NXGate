import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
const code = readFileSync(new URL('../src/status-transport.js', import.meta.url), 'utf8');
const { createStatusTransport } = await import('data:text/javascript;base64,' + Buffer.from(code).toString('base64'));
let timers = new Map(), next = 0, fetches = 0, renders = 0;
class ES {
 constructor() { ES.count++; this.handlers = {}; }
 addEventListener(name, fn) { this.handlers[name] = fn; }
 close() { this.closed = true; }
}
ES.count = 0;
const transport = createStatusTransport({url:'/api/events', fetchStatus:()=>fetches++, renderStatus:()=>renders++,
 EventSourceClass: ES, setTimer:fn=>{timers.set(++next,fn);return next;}, clearTimer:id=>timers.delete(id)});
const source = transport.start();
assert.equal(transport.start(), source); assert.equal(ES.count,1); assert.equal(timers.size,1);
source.onopen(); assert.equal(timers.size,0);
for(let i=0;i<10;i++) source.handlers.status({data:'{"vpn":{}}'});
assert.equal(renders,10); assert.equal(fetches,0);
source.onerror(); source.onerror(); assert.equal(timers.size,1);
for(const fn of timers.values()) fn(); assert.equal(fetches,1);
source.onopen(); assert.equal(timers.size,0);
const oldConsole = console.error; console.error = () => {};
source.handlers.status({data:'broken'}); assert.equal(timers.size,1);
console.error = oldConsole; source.handlers.status({data:'{}'}); assert.equal(timers.size,0);
transport.stop(); assert.equal(source.closed,true); assert.equal(timers.size,0);
const unsupported = createStatusTransport({url:'/',fetchStatus:()=>fetches++,renderStatus:()=>{},EventSourceClass:null,
 setTimer:fn=>{timers.set(++next,fn);return next;},clearTimer:id=>timers.delete(id)});
unsupported.start(); unsupported.start(); assert.equal(timers.size,1); unsupported.stop(); assert.equal(timers.size,0);
console.log('status transport: stable SSE, single fallback, recovery, unsupported browser passed');
