// Run with: node --test bridge/client/nodra.test.mjs
import assert from "node:assert/strict";
import { test } from "node:test";
import { dispatch, getState, remoteState } from "./nodra.js";

// fakeRuntime stands in for window.mygo: it records calls, and lets a test push frames into the open watch and end it.
function fakeRuntime() {
  const calls = [];
  const watches = [];
  return {
    calls,
    watches,
    channel(onmessage) {
      return { onmessage, closed: false, close() { this.closed = true; } };
    },
    call(method, ...args) {
      calls.push([method, ...args]);
      if (method.endsWith(".Watch")) {
        let resolve, reject;
        const done = new Promise((res, rej) => { resolve = res; reject = rej; });
        const channel = args[2];
        watches.push({ name: args[0], after: args[1], channel, resolve, reject, send: (frame) => channel.onmessage(frame) });
        return done;
      }
      return Promise.resolve({ name: args[0], version: 1, value: "x" });
    },
  };
}

const flush = () => new Promise((resolve) => setTimeout(resolve, 0));

test("the first subscriber starts the watch and the last one stops it", async () => {
  const runtime = fakeRuntime();
  const state = remoteState("counter", { runtime });
  assert.equal(runtime.calls.length, 0, "nothing is asked before a subscriber");
  let seen = 0;
  const a = state.subscribe(() => seen++);
  const b = state.subscribe(() => seen++);
  assert.equal(runtime.watches.length, 1);
  assert.deepEqual(runtime.calls[0], ["Nodra.Watch", "counter", 0, runtime.watches[0].channel]);

  runtime.watches[0].send({ name: "counter", version: 3, value: { count: 1 } });
  assert.deepEqual(state.getSnapshot(), { count: 1 });
  assert.equal(state.getVersion(), 3);
  assert.equal(seen, 2);

  a();
  assert.equal(runtime.watches[0].channel.closed, false);
  b();
  assert.equal(runtime.watches[0].channel.closed, true);
});

test("the snapshot keeps its identity until a frame arrives", () => {
  const runtime = fakeRuntime();
  const state = remoteState("s", { runtime });
  const stop = state.subscribe(() => {});
  runtime.watches[0].send({ name: "s", version: 1, value: { a: 1 } });
  const first = state.getSnapshot();
  assert.equal(state.getSnapshot(), first);
  runtime.watches[0].send({ name: "s", version: 2, value: { a: 2 } });
  assert.notEqual(state.getSnapshot(), first);
  stop();
});

test("a new watch resumes from the version the page holds, and frames of the old one are dropped", () => {
  const runtime = fakeRuntime();
  const state = remoteState("s", { runtime });
  let stop = state.subscribe(() => {});
  runtime.watches[0].send({ name: "s", version: 5, value: "five" });
  stop();
  stop = state.subscribe(() => {});
  assert.equal(runtime.watches[1].after, 5);
  runtime.watches[0].send({ name: "s", version: 6, value: "stale" });
  assert.equal(state.getSnapshot(), "five");
  runtime.watches[1].send({ name: "s", version: 7, value: "seven" });
  assert.equal(state.getSnapshot(), "seven");
  stop();
});

test("a failed watch tells its listeners, and a later subscriber tries again", async () => {
  const runtime = fakeRuntime();
  const state = remoteState("gone", { runtime });
  let calls = 0;
  const stop = state.subscribe(() => calls++);
  const reason = new Error("nodra bridge: closed");
  runtime.watches[0].reject(reason);
  await flush();
  assert.equal(state.getError(), reason);
  assert.equal(calls, 1);
  const again = state.subscribe(() => {});
  assert.equal(runtime.watches.length, 2);
  assert.equal(state.getError(), undefined);
  stop();
  again();
});

test("ready resolves with the first value and releases its subscription", async () => {
  const runtime = fakeRuntime();
  const state = remoteState("s", { runtime });
  const ready = state.ready();
  runtime.watches[0].send({ name: "s", version: 1, value: 42 });
  assert.equal(await ready, 42);
  assert.equal(runtime.watches[0].channel.closed, true, "ready alone keeps no watch open");
});

test("ready rejects when the watch fails before a value", async () => {
  const runtime = fakeRuntime();
  const state = remoteState("missing", { runtime });
  const ready = state.ready();
  const reason = new Error("nodra bridge: state or action not found");
  runtime.watches[0].reject(reason);
  await assert.rejects(ready, reason);
});

test("ready copes with a runtime that delivers a frame while subscribing", async () => {
  const runtime = fakeRuntime();
  const call = runtime.call;
  runtime.call = (method, ...args) => {
    const done = call(method, ...args);
    if (method.endsWith(".Watch")) args[2].onmessage({ name: args[0], version: 1, value: "now" });
    return done;
  };
  assert.equal(await remoteState("s", { runtime }).ready(), "now");
});

test("dispatch and getState call the service by name", async () => {
  const runtime = fakeRuntime();
  await dispatch("counter/add", { by: 2 }, { runtime });
  await dispatch("reset", undefined, { runtime, service: "Other" });
  assert.deepEqual((await getState("counter", { runtime })).value, "x");
  assert.deepEqual(runtime.calls, [
    ["Nodra.Dispatch", "counter/add", { by: 2 }],
    ["Other.Dispatch", "reset", null],
    ["Nodra.Get", "counter"],
  ]);
});

test("outside a MyGo window the client says so", () => {
  assert.throws(() => dispatch("a"), /MyGo runtime is missing/);
});

test("each subscription is its own, even with the same listener", () => {
  const runtime = fakeRuntime();
  const state = remoteState("s", { runtime });
  let calls = 0;
  const render = () => calls++;
  const a = state.subscribe(render);
  const b = state.subscribe(render);
  runtime.watches[0].send({ name: "s", version: 1, value: 1 });
  assert.equal(calls, 2, "both subscriptions hear the frame");
  a();
  a();
  assert.equal(runtime.watches[0].channel.closed, false, "the watch closed although a subscription is live");
  runtime.watches[0].send({ name: "s", version: 2, value: 2 });
  assert.equal(calls, 3);
  b();
  assert.equal(runtime.watches[0].channel.closed, true);
});

// reportErrors stands in for the page's error reporting while fn runs.
async function reportErrors(fn) {
  const reported = [];
  const had = Object.hasOwn(globalThis, "reportError");
  const previous = globalThis.reportError;
  globalThis.reportError = (error) => reported.push(error);
  try {
    await fn();
  } finally {
    if (had) globalThis.reportError = previous;
    else delete globalThis.reportError;
  }
  return reported;
}

test("a listener that throws does not keep the others from a frame", async () => {
  const runtime = fakeRuntime();
  const state = remoteState("s", { runtime });
  const bug = new Error("widget bug");
  let second = 0;
  const reported = await reportErrors(async () => {
    state.subscribe(() => { throw bug; });
    state.subscribe(() => second++);
    const ready = state.ready();
    runtime.watches[0].send({ name: "s", version: 1, value: 1 });
    assert.equal(second, 1, "the second listener missed the frame");
    assert.equal(await ready, 1);
  });
  assert.deepEqual(reported, [bug]);
});

test("a listener that throws does not keep the others from the watch's failure", async () => {
  const runtime = fakeRuntime();
  const state = remoteState("s", { runtime });
  const bug = new Error("widget bug");
  let second = 0;
  const reported = await reportErrors(async () => {
    state.subscribe(() => { throw bug; });
    state.subscribe(() => second++);
    runtime.watches[0].reject(new Error("nodra bridge: closed"));
    await flush();
  });
  assert.equal(second, 1, "the second listener missed the failure");
  assert.deepEqual(reported, [bug]);
});

test("ready without a running watch asks Go for the current value, not the one the page held", async () => {
  const runtime = fakeRuntime();
  const state = remoteState("s", { runtime });
  const first = state.ready();
  runtime.watches[0].send({ name: "s", version: 1, value: "old" });
  assert.equal(await first, "old");
  assert.equal(runtime.watches[0].channel.closed, true);
  // Go moves on to "new" while the page follows nothing.
  const second = state.ready();
  assert.equal(runtime.watches.length, 2);
  assert.equal(runtime.watches[1].after, 0, "the watch must send the current frame whatever the page holds");
  assert.equal(runtime.watches[1].channel.closed, false, "the watch closed before Go answered");
  runtime.watches[1].send({ name: "s", version: 2, value: "new" });
  assert.equal(await second, "new");
  assert.equal(runtime.watches[1].channel.closed, true, "ready alone keeps no watch open");
});

test("ready while a watch runs resolves with the value it keeps current", async () => {
  const runtime = fakeRuntime();
  const state = remoteState("s", { runtime });
  const stop = state.subscribe(() => {});
  runtime.watches[0].send({ name: "s", version: 1, value: "one" });
  assert.equal(await state.ready(), "one");
  assert.equal(runtime.watches.length, 1);
  assert.equal(runtime.watches[0].channel.closed, false);
  stop();
});

test("two ready calls in one tick both wait for Go's current frame", async () => {
  const runtime = fakeRuntime();
  const state = remoteState("s", { runtime });
  const first = state.ready();
  runtime.watches[0].send({ name: "s", version: 1, value: "old" });
  assert.equal(await first, "old");
  // Go moves on to "new" while the page follows nothing; two handlers then ask at once.
  let settled = 0;
  const a = state.ready().then((v) => { settled++; return v; });
  const b = state.ready().then((v) => { settled++; return v; });
  await flush();
  assert.equal(settled, 0, "a ready() settled before Go answered, from the value held by an ended watch");
  assert.equal(runtime.watches.length, 2, "the second ready() shares the watch the first one started");
  assert.equal(runtime.watches[1].after, 0);
  runtime.watches[1].send({ name: "s", version: 2, value: "new" });
  assert.deepEqual(await Promise.all([a, b]), ["new", "new"]);
  assert.equal(runtime.watches[1].channel.closed, true, "ready alone keeps no watch open");
});

test("ready after a subscription resumed waits for a frame Go is sure to send", async () => {
  const runtime = fakeRuntime();
  const state = remoteState("s", { runtime });
  let stop = state.subscribe(() => {});
  runtime.watches[0].send({ name: "s", version: 1, value: "old" });
  stop();
  // The component mounts again, which resumes from version 1, and a handler asks at once.
  let seen = 0;
  stop = state.subscribe(() => seen++);
  assert.equal(runtime.watches[1].after, 1);
  let settled = false;
  const ready = state.ready().then((v) => { settled = true; return v; });
  await flush();
  assert.equal(settled, false, "ready() settled from the held value before Go answered");
  assert.equal(runtime.watches.length, 3, "the resumed watch, which Go may never send a frame, was not restarted");
  assert.equal(runtime.watches[1].channel.closed, true);
  assert.equal(runtime.watches[2].after, 0);
  runtime.watches[2].send({ name: "s", version: 2, value: "new" });
  assert.equal(await ready, "new");
  assert.equal(seen, 1, "the subscriber follows the restarted watch");
  assert.equal(runtime.watches[2].channel.closed, false, "the subscriber's watch stays open");
  runtime.watches[2].send({ name: "s", version: 3, value: "newer" });
  assert.equal(seen, 2);
  stop();
  assert.equal(runtime.watches[2].channel.closed, true);
});

test("ready joins a running watch that resumed once it sent a frame", async () => {
  const runtime = fakeRuntime();
  const state = remoteState("s", { runtime });
  let stop = state.subscribe(() => {});
  runtime.watches[0].send({ name: "s", version: 1, value: "old" });
  stop();
  stop = state.subscribe(() => {});
  runtime.watches[1].send({ name: "s", version: 2, value: "new" });
  assert.equal(await state.ready(), "new");
  assert.equal(runtime.watches.length, 2);
  assert.equal(runtime.watches[1].channel.closed, false);
  stop();
});
