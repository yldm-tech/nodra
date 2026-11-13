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

// Reports a listener's error as an uncaught one without unwinding the caller, as MyGo's runtime does for its own callbacks, so one failing listener cannot keep the others from a frame or a failure.
function report(error) {
  if (typeof globalThis.reportError === "function") {
    globalThis.reportError(error);
  } else {
    setTimeout(() => {
      throw error;
    });
  }
}

/**
 * Follows the state Go exposes under name. Nothing is asked of Go until the first listener subscribes; the watch stops when the last one unsubscribes, and the next subscriber starts it again from the version the page holds. subscribe and getSnapshot fit React's useSyncExternalStore: the snapshot is the same object until a new frame arrives.
 */
export function remoteState(name, options = {}) {
  const service = serviceOf(options);
  // One entry per subscription, so a listener subscribed twice is two subscriptions, each ended by its own unsubscribe.
  const listeners = new Set();
  let value;
  let version = 0;
  let received = false;
  let error;
  let channel = null;
  // Whether the running watch delivered a frame, which makes the value current.
  let framed = false;
  // Whether the running watch resumed from the version the page held: Go skips its first frame while that version is current, so it may never send one.
  let resumed = false;
  // Each watch has a generation: frames and results of a stopped watch are ignored.
  let generation = 0;

  const notify = () => {
    for (const entry of [...listeners]) {
      try {
        entry.listener();
      } catch (error) {
        report(error);
      }
    }
  };

  // Starts a watch; Go skips its first frame when that frame's version is after.
  function start(after) {
    const runtime = runtimeOf(options);
    const current = ++generation;
    error = undefined;
    framed = false;
    resumed = after !== 0;
    channel = runtime.channel((frame) => {
      if (current !== generation) return;
      value = frame.value;
      version = frame.version;
      received = true;
      framed = true;
      notify();
    });
    runtime.call(`${service}.Watch`, name, after, channel).then(
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

  // Adds a subscription and starts a watch if none runs, from the version the page holds. A fresh subscription needs a watch that will send Go's current frame: it starts one from no version, and restarts a running watch that resumed from the version the page holds and has sent nothing, since Go may never send that one a frame.
  function listen(listener, fresh) {
    const entry = { listener };
    listeners.add(entry);
    if (!channel) {
      start(received && !fresh ? version : 0);
    } else if (fresh && resumed && !framed) {
      stop();
      start(0);
    }
    return () => {
      if (listeners.delete(entry) && listeners.size === 0) stop();
    };
  }

  const state = {
    name,
    getSnapshot: () => value,
    getVersion: () => version,
    getError: () => error,
    hasValue: () => received,
    subscribe: (listener) => listen(listener, false),
    ready() {
      // Only a frame of the running watch makes the value current; a value held from an ended watch may be stale.
      if (channel && framed) return Promise.resolve(value);
      return new Promise((resolve, reject) => {
        // ready waits for a watch that sends Go's current frame, and only a frame or the failure of that watch settles it.
        // A runtime may deliver the first frame while subscribe runs, before unsubscribe exists: settle again after it.
        let unsubscribe = null;
        let settled = false;
        const settle = () => {
          if (settled || !unsubscribe) return;
          if (error !== undefined) {
            settled = true;
            unsubscribe();
            reject(error);
          } else if (framed) {
            settled = true;
            unsubscribe();
            resolve(value);
          }
        };
        unsubscribe = listen(settle, true);
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
