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
