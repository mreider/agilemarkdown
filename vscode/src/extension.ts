import * as vscode from 'vscode';
import * as path from 'path';
import * as fs from 'fs';
import * as cp from 'child_process';
import { AmClient } from './am-client';
import { locateAm } from './locate-am';
import { SidebarProvider } from './sidebar';

let client: AmClient | null = null;
let panel: vscode.WebviewPanel | null = null;
let logChannel: vscode.OutputChannel | null = null;
let sidebar: SidebarProvider | null = null;
let currentBacklog: string | null = null;

type WelcomeState = 'noFolder' | 'uninitialized' | 'ready';

function pickAmRoot(): { folder: vscode.WorkspaceFolder | null; hasAm: boolean } {
  const folders = vscode.workspace.workspaceFolders;
  if (!folders || folders.length === 0) return { folder: null, hasAm: false };
  for (const f of folders) {
    if (fs.existsSync(path.join(f.uri.fsPath, '.am', 'config.yaml'))) {
      return { folder: f, hasAm: true };
    }
  }
  return { folder: folders[0], hasAm: false };
}

function computeState(): WelcomeState {
  const folders = vscode.workspace.workspaceFolders;
  if (!folders || folders.length === 0) return 'noFolder';
  const { hasAm } = pickAmRoot();
  return hasAm ? 'ready' : 'uninitialized';
}

async function refreshState(): Promise<void> {
  const state = computeState();
  await vscode.commands.executeCommand('setContext', 'agilemarkdown.state', state);
  log(`welcome state = ${state}`);
}

export async function activate(context: vscode.ExtensionContext) {
  logChannel = vscode.window.createOutputChannel('Agile Markdown');
  context.subscriptions.push(logChannel);

  await refreshState();

  context.subscriptions.push(
    vscode.workspace.onDidChangeWorkspaceFolders(async () => {
      await refreshState();
      sidebar?.refresh();
      await maybeAutoOpen(context);
    }),
  );

  const watcher = vscode.workspace.createFileSystemWatcher('**/.am/config.yaml');
  watcher.onDidCreate(async () => {
    await refreshState();
    sidebar?.refresh();
    await maybeAutoOpen(context);
  });
  watcher.onDidDelete(async () => {
    await refreshState();
    sidebar?.refresh();
  });
  context.subscriptions.push(watcher);

  sidebar = new SidebarProvider({
    amClient: () => client,
    currentBacklog: () => currentBacklog,
    isReady: () => computeState() === 'ready',
    selectBacklog: async (name: string) => {
      currentBacklog = name;
      sidebar?.refresh();
      if (panel) await sendBoardSnapshot();
    },
  });
  const treeView = vscode.window.createTreeView('agilemarkdown.welcome', {
    treeDataProvider: sidebar,
  });
  treeView.onDidChangeVisibility(async e => {
    if (e.visible) {
      await maybeAutoOpen(context);
      sidebar?.refresh();
    }
  });
  context.subscriptions.push(treeView);

  context.subscriptions.push(
    vscode.commands.registerCommand('agilemarkdown.openBoard', () => openBoard(context)),
    vscode.commands.registerCommand('agilemarkdown.refreshBoard', () => {
      sidebar?.refresh();
      panel?.webview.postMessage({ type: 'refresh' });
    }),
    vscode.commands.registerCommand('agilemarkdown.installCli', async () => {
      // Setting cliPath to empty triggers locate flow on next openBoard.
      await vscode.workspace.getConfiguration('agilemarkdown').update('cliPath', '', vscode.ConfigurationTarget.Global);
      await openBoard(context);
    }),
    vscode.commands.registerCommand('agilemarkdown.createBacklog', () => createBacklog(context)),
    vscode.commands.registerCommand('agilemarkdown.openWorkspaceFolder', async () => {
      await vscode.commands.executeCommand('workbench.action.files.openFolder');
    }),
    vscode.commands.registerCommand('agilemarkdown.initFolder', () => initFolder(context)),
    vscode.commands.registerCommand('agilemarkdown.switchBacklog', async (name: string) => {
      currentBacklog = name;
      sidebar?.refresh();
      if (!panel) await openBoard(context);
      else { panel.reveal(undefined, true); await sendBoardSnapshot(); }
    }),
    vscode.commands.registerCommand('agilemarkdown.newStoryInCurrentBacklog', async () => {
      if (!currentBacklog) {
        vscode.window.showInformationMessage('Pick a backlog first.');
        return;
      }
      const title = await vscode.window.showInputBox({ prompt: `New story title (in ${currentBacklog})`, placeHolder: 'Login flow' });
      if (!title) return;
      const c = await ensureClient(vscode.workspace.workspaceFolders![0].uri.fsPath);
      if (!c) return;
      try {
        await c.createItem(currentBacklog, title);
        sidebar?.refresh();
        if (panel) await sendBoardSnapshot();
      } catch (e: any) {
        vscode.window.showErrorMessage(`Create story failed: ${e?.message || e}`);
      }
    }),
    vscode.commands.registerCommand('agilemarkdown.sync', async () => {
      const c = await ensureClient(vscode.workspace.workspaceFolders![0].uri.fsPath);
      if (!c) return;
      try {
        await c.sync();
        sidebar?.refresh();
        if (panel) await sendBoardSnapshot();
        vscode.window.showInformationMessage('Sync complete.');
      } catch (e: any) {
        vscode.window.showErrorMessage(`Sync failed: ${e?.message || e}`);
      }
    }),
  );

  // First activation: if the workspace is already a backlog, jump
  // straight to the board.
  await maybeAutoOpen(context);
}

async function maybeAutoOpen(context: vscode.ExtensionContext): Promise<void> {
  if (computeState() !== 'ready') return;
  if (panel) {
    panel.reveal(undefined, true);
    return;
  }
  await openBoard(context);
}

async function initFolder(context: vscode.ExtensionContext) {
  const folders = vscode.workspace.workspaceFolders;
  if (!folders || folders.length === 0) {
    const choice = await vscode.window.showInformationMessage(
      'Open a folder first. Agile Markdown stores stories alongside your code.',
      'Open Folder',
    );
    if (choice === 'Open Folder') {
      await vscode.commands.executeCommand('workbench.action.files.openFolder');
    }
    return;
  }

  const target = folders[0];
  const cwd = target.uri.fsPath;

  const am = await locateAm();
  if (!am) {
    vscode.window.showErrorMessage('Agile Markdown: no `am` binary available. Run "Install am CLI".');
    return;
  }

  // `am init` walks up looking for `.git` and errors out if missing.
  // Seed an empty repo silently so the init flow is one click for new folders.
  if (!fs.existsSync(path.join(cwd, '.git'))) {
    try {
      cp.execFileSync('git', ['init', '-q'], { cwd, stdio: 'ignore' });
    } catch (e: any) {
      vscode.window.showErrorMessage(`git init failed: ${e?.message || e}`);
      return;
    }
  }

  try {
    cp.execFileSync(am, ['init'], { cwd, stdio: 'pipe' });
  } catch (e: any) {
    const stderr = e?.stderr?.toString?.() || '';
    vscode.window.showErrorMessage(`am init failed: ${stderr || e?.message || e}`);
    return;
  }

  await refreshState();
  vscode.window.showInformationMessage(`Initialized Agile Markdown in ${target.name}.`);
}

async function createBacklog(context: vscode.ExtensionContext) {
  const ws = vscode.workspace.workspaceFolders?.[0];
  if (!ws) {
    const choice = await vscode.window.showInformationMessage(
      'Open a folder first. Agile Markdown stores stories alongside your code.',
      'Open Folder',
    );
    if (choice === 'Open Folder') {
      await vscode.commands.executeCommand('workbench.action.files.openFolder');
    }
    return;
  }

  const name = await vscode.window.showInputBox({
    prompt: 'Name for the new backlog folder',
    placeHolder: 'product',
    validateInput: (v) => {
      if (!v) return 'Required';
      if (!/^[a-z][a-z0-9-]*$/.test(v)) return 'Use lowercase letters, digits, and hyphens (start with a letter).';
      return undefined;
    },
  });
  if (!name) return;

  if (!client) {
    client = await ensureClient(ws.uri.fsPath);
    if (!client) return;
  }

  try {
    await client.createBacklog(name);
    currentBacklog = name;
    sidebar?.refresh();
    vscode.window.showInformationMessage(`Created backlog "${name}".`);
    if (panel) await sendBoardSnapshot();
    else await openBoard(context);
  } catch (e: any) {
    vscode.window.showErrorMessage(`Create backlog failed: ${e?.message || e}`);
  }
}

export function deactivate() {
  client = null;
}

async function ensureClient(root: string): Promise<AmClient | null> {
  const am = await locateAm();
  if (!am) {
    vscode.window.showErrorMessage('Agile Markdown: no `am` binary available. Run "Install am CLI".');
    return null;
  }
  return new AmClient(am, root);
}

async function openBoard(context: vscode.ExtensionContext) {
  const ws = vscode.workspace.workspaceFolders?.[0];
  if (!ws) {
    vscode.window.showErrorMessage('Open an agilemarkdown repo as a workspace first.');
    return;
  }

  if (!client) {
    client = await ensureClient(ws.uri.fsPath);
    if (!client) return;
  }

  if (panel) {
    panel.reveal();
    return;
  }

  panel = vscode.window.createWebviewPanel(
    'agilemarkdown.board',
    'Agile Markdown',
    vscode.ViewColumn.Active,
    {
      enableScripts: true,
      retainContextWhenHidden: true,
      localResourceRoots: [vscode.Uri.file(path.join(context.extensionPath, 'dist')), vscode.Uri.file(path.join(context.extensionPath, 'media'))],
    },
  );

  panel.iconPath = vscode.Uri.file(path.join(context.extensionPath, 'media', 'icon.png'));

  panel.onDidDispose(() => {
    panel = null;
  });

  panel.webview.html = renderHtml(panel.webview, context);

  panel.webview.onDidReceiveMessage(msg => onMessage(msg).catch(err => {
    log(`message handler error: ${err?.message || err}`);
    panel?.webview.postMessage({ type: 'error', payload: { message: String(err?.message || err) } });
  }));
}

async function onMessage(msg: { type: string; payload?: any }): Promise<void> {
  if (!client) return;
  switch (msg.type) {
    case 'load': {
      await sendBoardSnapshot();
      return;
    }
    case 'whoami': {
      try {
        const me = await client.whoami();
        panel?.webview.postMessage({ type: 'whoami', payload: me });
      } catch {
        // Best-effort; some workspaces won't have a git user set.
      }
      return;
    }
    case 'search': {
      const query = msg.payload?.query ?? '';
      if (!query.trim()) {
        panel?.webview.postMessage({ type: 'search-results', payload: { query, hits: [], count: 0 } });
        return;
      }
      try {
        const res = await client.search(query, 20);
        panel?.webview.postMessage({ type: 'search-results', payload: res });
      } catch (e: any) {
        panel?.webview.postMessage({ type: 'search-results', payload: { query, hits: [], count: 0, error: String(e?.message || e) } });
      }
      return;
    }
    case 'open-docs': {
      await vscode.env.openExternal(vscode.Uri.parse('https://agilemarkdown.com'));
      return;
    }
    case 'create-backlog': {
      // Inline empty-state CTA — re-uses the same flow as the sidebar.
      await vscode.commands.executeCommand('agilemarkdown.createBacklog');
      return;
    }
    case 'new-story-prompt': {
      // The webview's "Add story" button. Prompt for a title, then
      // create the story in the targeted backlog and refresh.
      const target = msg.payload?.backlog || currentBacklog;
      if (!target) return;
      const title = await vscode.window.showInputBox({
        prompt: `New story title (in ${target})`,
        placeHolder: 'Login flow',
      });
      if (!title) return;
      await client.createItem(target, title);
      sidebar?.refresh();
      await sendBoardSnapshot();
      return;
    }
    case 'load-sprint-plan': {
      const snap = await client.sprintPlan(msg.payload?.backlog || (await client.listBacklogs()).backlogs?.[0]);
      panel?.webview.postMessage({ type: 'sprint-plan', payload: snap });
      return;
    }
    case 'open-detail': {
      const detail = await client.getItem(msg.payload.path);
      // Fill in iteration for unstarted-in-priority items the server
      // couldn't bucket (it has no priority context). One band per
      // velocity points, starting at the current iteration.
      if (!detail.iteration && !detail.iteration_label && currentBacklog) {
        try {
          const pri = await client.priorityList(currentBacklog);
          const idx = pri.items.findIndex(r => r.path.endsWith(detail.path) || detail.path.endsWith(r.path));
          if (idx >= 0 && pri.velocity > 0) {
            // Compute current iteration number on the client side; the
            // server's IterationNumberFor lives in Go. Approximate with
            // weeks-since-2000-01-03 / length (assumed 1-week here).
            const epoch = new Date(Date.UTC(2000, 0, 3));
            const today = new Date();
            const weeks = Math.floor((today.getTime() - epoch.getTime()) / (7 * 24 * 3600 * 1000));
            const band = Math.floor(idx / pri.velocity);
            detail.iteration = weeks + 1 + band;
          } else if (idx >= 0) {
            detail.iteration_label = 'backlog';
          } else {
            detail.iteration_label = 'icebox';
          }
        } catch {
          // Best-effort; leave iteration blank.
        }
      }
      const [comments, tasks, history] = await Promise.all([
        client.getComments(msg.payload.path).catch(() => ({ comments: [], count: 0 })),
        client.listTasks(msg.payload.path).catch(() => ({ tasks: [], count: 0 })),
        client.history(msg.payload.path, 20).catch(() => ({ entries: [], count: 0 })),
      ]);
      panel?.webview.postMessage({ type: 'detail', payload: { item: detail, comments, tasks, history } });
      return;
    }
    case 'set-status': {
      await client.setStatus(msg.payload.path, msg.payload.status);
      await sendBoardSnapshot();
      return;
    }
    case 'set-estimate': {
      await client.setEstimate(msg.payload.path, String(msg.payload.estimate));
      await sendBoardSnapshot();
      return;
    }
    case 'set-assigned': {
      await client.setAssigned(msg.payload.path, msg.payload.assignees);
      await sendBoardSnapshot();
      return;
    }
    case 'set-tags': {
      await client.setTags(msg.payload.path, msg.payload.tags);
      await sendBoardSnapshot();
      return;
    }
    case 'set-epic': {
      await client.setEpic(msg.payload.path, msg.payload.epic || '');
      await sendBoardSnapshot();
      return;
    }
    case 'set-description': {
      await client.setDescription(msg.payload.path, msg.payload.body);
      await sendBoardSnapshot();
      return;
    }
    case 'rank-item': {
      await client.rankItem(msg.payload);
      await sendBoardSnapshot();
      return;
    }
    case 'move-to-icebox': {
      await client.moveToIcebox(msg.payload);
      await sendBoardSnapshot();
      return;
    }
    case 'move-to-priority': {
      await client.moveToPriority(msg.payload);
      await sendBoardSnapshot();
      return;
    }
    case 'create-item': {
      await client.createItem(msg.payload.backlog, msg.payload.title, msg.payload.user);
      await sendBoardSnapshot();
      return;
    }
    case 'block': {
      await client.blockItem(msg.payload.path, msg.payload.reason);
      await sendBoardSnapshot();
      return;
    }
    case 'unblock': {
      await client.unblockItem(msg.payload.path);
      await sendBoardSnapshot();
      return;
    }
    case 'reject': {
      // Mirrors the CLI `am reject ITEM --reason "..."`. Captures the
      // reason in the body's "## Rejection notes" section. Distinct
      // from set-status:rejected (which would not record a reason).
      await client.rejectItem(msg.payload.path, msg.payload.reason || '');
      await sendBoardSnapshot();
      // Refresh detail too so the reason becomes visible.
      panel?.webview.postMessage({ type: 'detail', payload: { item: await client.getItem(msg.payload.path), comments: await client.getComments(msg.payload.path), tasks: await client.listTasks(msg.payload.path) } });
      return;
    }
    case 'add-comment': {
      await client.addComment(msg.payload.path, msg.payload.text, msg.payload.author);
      panel?.webview.postMessage({ type: 'detail', payload: { item: await client.getItem(msg.payload.path), comments: await client.getComments(msg.payload.path), tasks: await client.listTasks(msg.payload.path) } });
      return;
    }
    case 'add-task': {
      await client.addTask(msg.payload.path, msg.payload.text);
      panel?.webview.postMessage({ type: 'detail', payload: { item: await client.getItem(msg.payload.path), comments: await client.getComments(msg.payload.path), tasks: await client.listTasks(msg.payload.path) } });
      return;
    }
    case 'tick-task': {
      await client.setTaskDone(msg.payload.path, msg.payload.index, msg.payload.done);
      panel?.webview.postMessage({ type: 'detail', payload: { item: await client.getItem(msg.payload.path), comments: await client.getComments(msg.payload.path), tasks: await client.listTasks(msg.payload.path) } });
      return;
    }
    default:
      log(`unknown message: ${msg.type}`);
  }
}

async function sendBoardSnapshot(): Promise<void> {
  if (!client || !panel) return;
  const backlogs = (await client.listBacklogs()).backlogs || [];
  // currentBacklog may become invalid when the list of backlogs
  // changes (rename, delete, init). Fall back to the first available
  // backlog and update the global so the sidebar reflects the same
  // choice.
  if (!currentBacklog || !backlogs.includes(currentBacklog)) {
    currentBacklog = backlogs[0] || null;
    sidebar?.refresh();
  }
  const backlog = currentBacklog;
  if (!backlog) {
    panel.webview.postMessage({ type: 'snapshot', payload: { backlog: '', priority: { items: [], count: 0, velocity: 0 }, icebox: { items: [], count: 0 }, dashboard: null, velocity: { rows: [] }, epics: [], backlogs } });
    return;
  }
  const [priority, icebox, dashboard, velocity, mix, burnup, cfd] = await Promise.all([
    client.priorityList(backlog),
    client.iceboxList(backlog),
    client.dashboard().catch(() => null),
    client.velocityHistory(backlog).catch(() => ({ rows: [] })),
    client.typeMix().catch(() => ({ rows: [], total: 0 })),
    client.burnupChart(backlog, 0).catch(() => null),
    client.cumulativeFlow(30).catch(() => null),
  ]);
  const epicSlugs = uniqueEpicSlugs([...priority.items, ...icebox.items]);
  const epics = await Promise.all(epicSlugs.map(slug => client!.epicProgress(slug).catch(() => null)));
  panel.webview.postMessage({
    type: 'snapshot',
    payload: {
      backlog,
      backlogs,
      priority,
      icebox,
      dashboard,
      velocity,
      typeMix: mix,
      burnup,
      cfd,
      epics: epics.filter(Boolean),
    },
  });
}

function uniqueEpicSlugs(rows: Array<{ epic?: string }>): string[] {
  const set = new Set<string>();
  for (const r of rows) {
    if (r.epic && r.epic.trim()) set.add(r.epic.trim());
  }
  return Array.from(set);
}

function renderHtml(webview: vscode.Webview, context: vscode.ExtensionContext): string {
  const scriptUri = webview.asWebviewUri(vscode.Uri.file(path.join(context.extensionPath, 'dist', 'webview.js')));
  const nonce = randNonce();
  const csp = `default-src 'none'; img-src ${webview.cspSource} https: data:; style-src ${webview.cspSource} 'unsafe-inline'; script-src 'nonce-${nonce}';`;
  // Inline shell renders before webview.js loads: a centered "Loading"
  // line plus an error sink that catches anything thrown during bundle
  // execution. React mounts into #root and replaces the shell. If
  // webview.js fails to load (CSP, bad path, corrupt bundle), the user
  // sees an actionable message instead of a blank panel.
  return `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8" />
  <meta http-equiv="Content-Security-Policy" content="${csp}" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>Agile Markdown</title>
  <style nonce="${nonce}">
    html, body { margin: 0; padding: 0; height: 100%; background: var(--vscode-editor-background); color: var(--vscode-foreground); font-family: var(--vscode-font-family); }
    #root { height: 100%; }
    #shell { display: flex; flex-direction: column; align-items: center; justify-content: center; height: 100%; gap: 8px; font-size: 13px; opacity: 0.7; }
    #shell pre { max-width: 80ch; white-space: pre-wrap; color: var(--vscode-errorForeground); font-size: 12px; }
  </style>
</head>
<body>
  <div id="root">
    <div id="shell">
      <div>Loading Agile Markdown…</div>
      <pre id="shell-err"></pre>
    </div>
  </div>
  <script nonce="${nonce}">
    window.addEventListener('error', function(e) {
      var el = document.getElementById('shell-err');
      if (el) el.textContent = 'Webview failed: ' + (e && e.message ? e.message : 'unknown error') + (e && e.filename ? ' @ ' + e.filename : '');
    });
  </script>
  <script nonce="${nonce}" src="${scriptUri}"></script>
</body>
</html>`;
}

function randNonce(): string {
  const chars = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789';
  let s = '';
  for (let i = 0; i < 32; i++) s += chars.charAt(Math.floor(Math.random() * chars.length));
  return s;
}

function log(msg: string) {
  logChannel?.appendLine(`[${new Date().toISOString()}] ${msg}`);
}
