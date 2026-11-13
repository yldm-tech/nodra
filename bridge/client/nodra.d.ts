/** A state as Go sends it: its name, the version of the source it was taken at, and the value Go's projection returned. */
export interface Frame<T> {
  name: string;
  version: number;
  value: T;
}

/** What the client needs of the MyGo runtime, which MyGo injects as window.mygo. Tests pass their own. */
export interface Runtime {
  call<R = unknown>(method: string, ...args: unknown[]): Promise<R>;
  channel<V = unknown>(onmessage?: (value: V) => void): { close(): void };
}

export interface Options {
  /** The name the bridge was bound under in Go, "Nodra" unless BindAs gave another. */
  service?: string;
  /** The MyGo runtime; window.mygo by default. */
  runtime?: Runtime;
}

/** A state Go exposes, followed while it has listeners. */
export interface RemoteState<T> {
  readonly name: string;
  /** The latest value, or undefined before the first frame. The same object until a new frame arrives. */
  getSnapshot(): T | undefined;
  /** The version of the latest value, 0 before the first frame. */
  getVersion(): number;
  /** Why the watch ended, such as a state that is not exposed or no longer is; undefined while it runs. */
  getError(): unknown;
  /** Whether a frame arrived. */
  hasValue(): boolean;
  /** Calls listener after every frame and when the watch fails. Each call is its own subscription, even for a listener already subscribed, and its function ends only that one. An error a listener throws is reported as uncaught, with reportError where the page has it, and does not keep the other listeners from being called. The first subscriber starts the watch; the last one to unsubscribe stops it. */
  subscribe(listener: () => void): () => void;
  /** Resolves with the current value: once the running watch has sent a frame, the one it keeps; otherwise the next frame of a watch Go sends its current frame to whatever version the page holds. ready starts that watch when none runs, waits for one that started from no version, and restarts one that resumed from the version the page holds and has sent nothing, which Go may never send a frame; its subscribers follow the restarted watch. So a value held from an ended watch is not taken for the current one. Rejects when that watch fails first. */
  ready(): Promise<T>;
}

export function remoteState<T>(name: string, options?: Options): RemoteState<T>;
export function getState<T>(name: string, options?: Options): Promise<Frame<T>>;
export function dispatch<A = unknown>(name: string, arg?: A, options?: Options): Promise<void>;
