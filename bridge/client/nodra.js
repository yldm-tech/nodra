// The client of Nodra's bridge for the web pages of a MyGo app: it follows a state that Go exposes with bridge.Expose or bridge.ExposeProjection, and dispatches the actions Go registers with bridge.Handle. It calls the bridge's bound service through the runtime MyGo injects into every page as window.mygo, so it needs no package and runs with or without a bundler. Its types are in nodra.d.ts.

const DEFAULT_SERVICE = "Nodra";

function runtimeOf(options) {
  const runtime = options?.runtime ?? globalThis.mygo;
  if (!runtime) {
    throw new Error("nodra: the MyGo runtime is missing; is this page running inside a MyGo window?");
  }
  return runtime;
}

function serviceOf(options) {
  return options?.service ?? DEFAULT_SERVICE;
}

/**
 * Follows the state Go exposes under name. Nothing is asked of Go until the first listener subscribes; the watch stops when the last one unsubscribes, and the next subscriber starts it again from the version the page holds. subscribe and getSnapshot fit React's useSyncExternalStore: the snapshot is the same object until a new frame arrives.
 */
export function remoteState(name, options = {}) {
  const service = serviceOf(options);
  const listeners = new Set();
  let value;
  let version = 0;
  let received = false;
  let error;
  let channel = null;
  // Each watch has a generation: frames and results of a stopped watch are ignored.
  let generation = 0;

  const notify = () => {
    for (const listener of [...listeners]) listener();
  };

  function start() {
    const runtime = runtimeOf(options);
    const current = ++generation;
    error = undefined;
    channel = runtime.channel((frame) => {
      if (current !== generation) return;
      value = frame.value;
      version = frame.version;
      received = true;
      notify();
    });
    runtime.call(`${service}.Watch`, name, received ? version : 0, channel).then(
      () => {
        if (current === generation) channel = null;
      },
      (reason) => {
        if (current !== generation) return;
        channel = null;
        error = reason;
        notify();
      },
    );
  }

  function stop() {
    generation++;
    const open = channel;
    channel = null;
    open?.close();
  }

  const state = {
    name,
    getSnapshot: () => value,
    getVersion: () => version,
    getError: () => error,
    hasValue: () => received,
    subscribe(listener) {
      listeners.add(listener);
      if (!channel) start();
      return () => {
        if (listeners.delete(listener) && listeners.size === 0) stop();
      };
    },
    ready() {
      if (received && channel) return Promise.resolve(value);
      return new Promise((resolve, reject) => {
        // A runtime may deliver the first frame while subscribe runs, before unsubscribe exists: settle again after it.
        let unsubscribe = null;
        let settled = false;
        const settle = () => {
          if (settled || !unsubscribe) return;
          if (error !== undefined) {
            settled = true;
            unsubscribe();
            reject(error);
          } else if (received) {
            settled = true;
            unsubscribe();
            resolve(value);
          }
        };
        unsubscribe = state.subscribe(settle);
        settle();
      });
    },
  };
  return state;
}

/** Returns the current frame of the state Go exposes under name, once, without following it. */
export function getState(name, options) {
  return runtimeOf(options).call(`${serviceOf(options)}.Get`, name);
}

/** Runs the action Go registered under name with arg, which is sent as JSON. It resolves once the action returns and rejects with its error. */
export function dispatch(name, arg, options) {
  return runtimeOf(options).call(`${serviceOf(options)}.Dispatch`, name, arg === undefined ? null : arg);
}
