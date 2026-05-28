// Minimal shape of the bridge VS Code injects into webview iframes.
declare function acquireVsCodeApi(): {
  postMessage(msg: unknown): void;
  setState(state: unknown): void;
  getState<T = unknown>(): T | undefined;
};
