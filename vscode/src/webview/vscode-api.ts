// VS Code injects `acquireVsCodeApi` into the webview iframe and only
// allows one call per page. Multiple modules calling it themselves
// throws "An instance of the VS Code API has already been acquired"
// and stops the bundle dead. Acquire once here, import everywhere.

export const vscode = (typeof acquireVsCodeApi === 'function')
  ? acquireVsCodeApi()
  : null;
