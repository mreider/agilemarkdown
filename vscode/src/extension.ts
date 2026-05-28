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
// Module-level context handle so onMessage() (also module-level) can
// reach globalState without threading the parameter through every
// caller.
let extensionContext: vscode.ExtensionContext | null = null;
// Activity-bar sidebar drives the visible-columns set (Backlog,
// Icebox, Epics). Persisted to context.globalState so the user's
// chosen layout survives VS Code restarts and matches across the
// sidebar tree, the board, and any future surfaces.
const VISIBLE_COLUMNS_KEY = 'am.visibleColumns';
const ALL_COLUMNS = ['search', 'backlog', 'done', 'icebox', 'epics'];
let visibleColumns = new Set<string>(['backlog']);
// Active webview view; switched by the activity-bar Views list.
type AmView = 'home' | 'analytics' | 'settings';
const VIEW_KEY = 'am.currentView';
const ALL_VIEWS: AmView[] = ['home', 'analytics', 'settings'];
let currentView: AmView = 'home';
// Path of the story the DetailPanel is currently bound to (set on
// open-detail, cleared on detail-closed). Used by afterWrite() so a
// write that targets that path re-emits a fresh detail payload, keeping
// the panel and the card from diverging on status / estimate / tags.
let currentDetailPath: string | null = null;

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

// closeBoardIfStale tears down the open board panel when the
// workspace stops being a valid agilemarkdown project (no folder, or
// .am/config.yaml went away). Without this the panel kept rendering
// the last snapshot from the previous workspace while every click
// hit a dead client; the UX looked functional but nothing worked.
//
// Disposing also resets the cached AmClient so the next openBoard
// rebuilds against whatever workspace the user opens next, rather
// than firing CLI calls in the now-irrelevant old root.
function closeBoardIfStale(): void {
  if (computeState() === 'ready') return;
  if (panel) {
    log('workspace no longer agilemarkdown-ready; disposing board panel');
    panel.dispose();
    panel = null;
  }
  client = null;
  currentBacklog = null;
  currentDetailPath = null;
}

export async function activate(context: vscode.ExtensionContext) {
  extensionContext = context;
  logChannel = vscode.window.createOutputChannel('Agile Markdown');
  context.subscriptions.push(logChannel);

  // Hydrate the visible-columns set from globalState before anything
  // reads it. Default to a single Backlog column on a fresh install.
  const persisted = context.globalState.get<string[]>(VISIBLE_COLUMNS_KEY);
  if (Array.isArray(persisted)) {
    visibleColumns = new Set(persisted.filter(c => ALL_COLUMNS.includes(c)));
  }
  const persistedView = context.globalState.get<string>(VIEW_KEY);
  if (persistedView && (ALL_VIEWS as string[]).includes(persistedView)) {
    currentView = persistedView as AmView;
  }

  await refreshState();

  context.subscriptions.push(
    vscode.workspace.onDidChangeWorkspaceFolders(async () => {
      await refreshState();
      sidebar?.refresh();
      // Always tear down on workspace change, even when the new
      // workspace is also agilemarkdown-ready: the existing client +
      // panel are bound to the OLD root and would surface stale data
      // (or worse, write to it) under the new workspace's UI. Force a
      // fresh mount via maybeAutoOpen below.
      if (panel) {
        panel.dispose();
        panel = null;
      }
      client = null;
      currentBacklog = null;
      currentDetailPath = null;
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
    // .am went away. Any panel still open is now non-functional;
    // tear it down so the user falls back to the viewsWelcome CTA
    // instead of clicking dead controls.
    closeBoardIfStale();
  });
  context.subscriptions.push(watcher);

  sidebar = new SidebarProvider({
    isReady: () => computeState() === 'ready',
    visibleColumns: () => visibleColumns,
    toggleColumn: (col: string) => {
      if (visibleColumns.has(col)) visibleColumns.delete(col);
      else visibleColumns.add(col);
      context.globalState.update(VISIBLE_COLUMNS_KEY, Array.from(visibleColumns));
      sidebar?.refresh();
      panel?.webview.postMessage({ type: 'visible-columns', payload: Array.from(visibleColumns) });
    },
    currentView: () => currentView,
    setView: (view) => {
      currentView = view;
      context.globalState.update(VIEW_KEY, view);
      sidebar?.refresh();
      panel?.webview.postMessage({ type: 'set-view', payload: view });
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
    vscode.commands.registerCommand('agilemarkdown.setView', (view: string) => {
      if (!(ALL_VIEWS as string[]).includes(view)) return;
      currentView = view as AmView;
      context.globalState.update(VIEW_KEY, view);
      sidebar?.refresh();
      panel?.webview.postMessage({ type: 'set-view', payload: view });
    }),
    vscode.commands.registerCommand('agilemarkdown.toggleColumn', (col: string) => {
      if (!ALL_COLUMNS.includes(col)) return;
      if (visibleColumns.has(col)) visibleColumns.delete(col);
      else visibleColumns.add(col);
      context.globalState.update(VISIBLE_COLUMNS_KEY, Array.from(visibleColumns));
      sidebar?.refresh();
      panel?.webview.postMessage({ type: 'visible-columns', payload: Array.from(visibleColumns) });
    }),
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
    vscode.commands.registerCommand('agilemarkdown.openStoryDetail', async (storyPath: string) => {
      // Open (or focus) the board, then ask the webview to render the
      // detail panel for the given path. Mirror what a board-card
      // click would do, so sidebar rows and card clicks behave the
      // same downstream.
      if (!panel) await openBoard(context);
      panel?.reveal(undefined, true);
      panel?.webview.postMessage({ type: 'open-detail', payload: { path: storyPath } });
    }),
    vscode.commands.registerCommand('agilemarkdown.pullNext', async (storyPath: string) => {
      // The Next-pull sidebar row's primary action: open the detail
      // panel so the human can confirm the story before pulling. The
      // panel already has a Start button; starting from there gives
      // the coach (and the /am-align skill) a chance to fire.
      if (!panel) await openBoard(context);
      panel?.reveal(undefined, true);
      panel?.webview.postMessage({ type: 'open-detail', payload: { path: storyPath } });
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
    vscode.commands.registerCommand('agilemarkdown.openDocs', async () => {
      await vscode.env.openExternal(vscode.Uri.parse('https://agilemarkdown.com'));
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

  // Push the project name on creation so the topbar renders the
  // label on the very first paint. Without this, the brand text
  // flashed "Agile Markdown" (the fallback) until the slower snapshot
  // round trip landed and replaced it. The `project` message is
  // separately handled by App.tsx so we don't have to wait for
  // listBacklogs / sprint_plan / etc. to fire.
  try {
    const folder = pickAmRoot().folder;
    const folderName = folder ? path.basename(folder.uri.fsPath) : 'agilemarkdown';
    let project = folderName;
    if (client) {
      try {
        const info = await client.project();
        if (info?.name && info.name.trim()) project = info.name.trim();
      } catch { /* fall back to folder basename */ }
    }
    // `name` is the resolved label for the topbar (config override
    // or folder). `folderName` is the raw workspace folder name so
    // Settings can render it as the Display-name placeholder.
    panel.webview.postMessage({ type: 'project', payload: { name: project, folderName } });
    // Seed visible-columns so the board renders the user's chosen
    // layout on first paint instead of falling back to localStorage.
    panel.webview.postMessage({ type: 'visible-columns', payload: Array.from(visibleColumns) });
    panel.webview.postMessage({ type: 'set-view', payload: currentView });
  } catch (e: any) {
    log(`project push failed: ${e?.message || e}`);
  }
}

async function onMessage(msg: { type: string; payload?: any }): Promise<void> {
  if (!client) return;
  // Defense against late arrivals: a webview message that races a
  // workspace change can land after closeBoardIfStale ran. If state
  // is no longer ready, swallow the message instead of firing the
  // CLI against a root that may not even be agilemarkdown anymore.
  if (computeState() !== 'ready') {
    log(`ignoring ${msg.type}: workspace no longer agilemarkdown-ready`);
    return;
  }
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
      // The webview now collects the name inline. When a name is in
      // the payload, create directly. Falling back to the sidebar
      // command (showInputBox) is the legacy path; left in for the
      // activity-bar entry that doesn't have a webview to host an
      // inline form.
      const name = (msg.payload?.name || '').trim();
      if (name) {
        try {
          await client.createBacklog(name);
          currentBacklog = name;
          sidebar?.refresh();
          await afterWrite();
        } catch (e: any) {
          vscode.window.showErrorMessage(`Create backlog failed: ${e?.message || e}`);
        }
        return;
      }
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
    case 'toggle-column-from-webview': {
      const col = String(msg.payload?.col || '');
      const show = !!msg.payload?.show;
      if (!ALL_COLUMNS.includes(col)) return;
      if (show) visibleColumns.add(col); else visibleColumns.delete(col);
      extensionContext?.globalState.update(VISIBLE_COLUMNS_KEY, Array.from(visibleColumns));
      sidebar?.refresh();
      return;
    }
    case 'detail-closed': {
      // The webview tells us when the user dismisses the panel so
      // afterWrite() stops refreshing detail for a closed view. Cheap
      // and explicit, no guesswork.
      currentDetailPath = null;
      return;
    }
    case 'open-detail': {
      currentDetailPath = msg.payload.path;
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
      // autoAction lets a caller (card-level Reject) request that the
      // panel render directly into a specific form on mount, instead
      // of forcing the human through Accept → ceremony → Reject. Pass
      // through verbatim; DetailPanel reads it once per item.path.
      panel?.webview.postMessage({
        type: 'detail',
        payload: { item: detail, comments, tasks, history, autoAction: msg.payload?.autoAction || null },
      });
      return;
    }
    case 'set-status': {
      // Auto-promote on Start: if the user is moving an icebox story
      // to 'started' from anywhere (state-machine button, card,
      // sidebar), promote it to priority first so it doesn't end up
      // stranded in the icebox with status=started. The card-level
      // pull-from-icebox path used to be the only caller that did
      // this; centralizing it here means the detail-panel state
      // machine, the sidebar, and any future caller all get the same
      // semantics for free.
      if (msg.payload.status === 'started') {
        await maybePromoteFromIcebox(msg.payload.path);
      }
      await client.setStatus(msg.payload.path, msg.payload.status);
      await afterWrite(msg.payload.path);
      return;
    }
    case 'set-estimate': {
      await client.setEstimate(msg.payload.path, String(msg.payload.estimate));
      await afterWrite(msg.payload.path);
      return;
    }
    case 'set-release-date': {
      // Release markers stake a target ship date. Drives the marker's
      // position in the iteration view; once today is past, the bar
      // surfaces a Ship affordance.
      await client.setReleaseDate(msg.payload.path, String(msg.payload.release_date || ''));
      await afterWrite(msg.payload.path);
      return;
    }
    case 'set-type': {
      // Coach rule: bugs and chores are not pointed. Flipping a
      // story's type to bug/chore strips any existing estimate in the
      // same call so the next snapshot doesn't render a points pill on
      // a card that shouldn't have one. Releases also have no
      // estimate, so the same rule applies. Features keep whatever
      // estimate they already carried.
      const t = String(msg.payload.type || '').toLowerCase();
      await client.setType(msg.payload.path, t);
      if (t === 'bug' || t === 'chore' || t === 'release') {
        try { await client.setEstimate(msg.payload.path, ''); } catch { /* best-effort */ }
      }
      await afterWrite(msg.payload.path);
      return;
    }
    case 'set-assigned': {
      await client.setAssigned(msg.payload.path, msg.payload.assignees);
      await afterWrite(msg.payload.path);
      return;
    }
    case 'set-tags': {
      await client.setTags(msg.payload.path, msg.payload.tags);
      await afterWrite(msg.payload.path);
      return;
    }
    case 'set-epic': {
      await client.setEpic(msg.payload.path, msg.payload.epic || '');
      await afterWrite(msg.payload.path);
      return;
    }
    case 'set-description': {
      await client.setDescription(msg.payload.path, msg.payload.body);
      await afterWrite(msg.payload.path);
      return;
    }
    case 'rank-item': {
      await client.rankItem(msg.payload);
      await afterWrite();
      return;
    }
    case 'move-to-icebox': {
      await client.moveToIcebox(msg.payload);
      await afterWrite();
      return;
    }
    case 'move-to-priority': {
      await client.moveToPriority(msg.payload);
      await afterWrite();
      return;
    }
    case 'pull-from-icebox': {
      // Promote then start: the user clicked Start on an icebox card,
      // which by intent means "this story is current now." Do both
      // moves so the card jumps from Icebox to Current as one motion.
      // moveToPriority cd's into the backlog so basename is fine, but
      // setStatus runs from repo root and needs the backlog prefix so
      // `am start` resolves the file correctly.
      const { backlog, item_path } = msg.payload;
      const qualified = `${backlog}/${item_path}`;
      try {
        await client.moveToPriority({ backlog, item_paths: [item_path], position: 'top' });
        await client.setStatus(qualified, 'started');
      } catch (e: any) {
        vscode.window.showErrorMessage(`Pull failed: ${e?.message || e}`);
      }
      await afterWrite(qualified);
      return;
    }
    case 'create-item': {
      // Target column comes from the webview (which + button was
      // clicked); position is read from the user's config and
      // defaults to bottom. The CLI's `am create-item --target …
      // --position …` flags do the actual ranking, so the handler
      // is now a thin dispatch.
      const { backlog, title, user, target } = msg.payload;
      let position: 'top' | 'bottom' = 'bottom';
      try {
        const cfg = await client.configShow().catch(() => null);
        const tgt = target === 'priority' ? 'priority' : 'icebox';
        const field = tgt === 'priority' ? 'PriorityPosition' : 'IceboxPosition';
        if (cfg?.NewStory?.[field] === 'top') position = 'top';
      } catch { /* fall back to bottom */ }
      const tgt: 'priority' | 'icebox' = target === 'priority' ? 'priority' : 'icebox';
      await client.createItem(backlog, title, user, { target: tgt, position });
      await afterWrite();
      return;
    }
    case 'config-show': {
      try {
        const cfg = await client.configShow();
        panel?.webview.postMessage({ type: 'config', payload: cfg });
      } catch (e: any) {
        panel?.webview.postMessage({ type: 'error', payload: { message: String(e?.message || e) } });
      }
      return;
    }
    case 'config-save': {
      await client.configSave(msg.payload);
      try {
        const cfg = await client.configShow();
        panel?.webview.postMessage({ type: 'config', payload: cfg });
      } catch { /* best-effort */ }
      await afterWrite();
      return;
    }
    case 'block': {
      await client.blockItem(msg.payload.path, msg.payload.reason);
      await afterWrite(msg.payload.path);
      return;
    }
    case 'unblock': {
      await client.unblockItem(msg.payload.path);
      await afterWrite(msg.payload.path);
      return;
    }
    case 'reject': {
      // Mirrors the CLI `am reject ITEM --reason "..."`. Captures the
      // reason in the body's "## Rejection notes" section. Distinct
      // from set-status:rejected (which would not record a reason).
      await client.rejectItem(msg.payload.path, msg.payload.reason || '');
      await afterWrite(msg.payload.path);
      return;
    }
    case 'add-comment': {
      await client.addComment(msg.payload.path, msg.payload.text, msg.payload.author);
      await afterWrite(msg.payload.path);
      return;
    }
    case 'add-task': {
      await client.addTask(msg.payload.path, msg.payload.text);
      await afterWrite(msg.payload.path);
      return;
    }
    case 'tick-task': {
      await client.setTaskDone(msg.payload.path, msg.payload.index, msg.payload.done);
      await afterWrite(msg.payload.path);
      return;
    }
    case 'delete-item': {
      // Hard delete: the story file and its priority/icebox entries
      // are gone after this call. Tell the webview to close the
      // panel first; the path no longer exists so any follow-up
      // open-detail would 404 against the CLI. Then snapshot so the
      // card vanishes from the columns in lock-step.
      await client.deleteItem(msg.payload.path);
      currentDetailPath = null;
      // Tell the webview to drop the open detail panel; the story it
      // was bound to no longer exists on disk.
      panel?.webview.postMessage({ type: 'close-detail' });
      await sendBoardSnapshot();
      return;
    }
    default:
      log(`unknown message: ${msg.type}`);
  }
}

// maybePromoteFromIcebox runs the icebox→priority promotion when the
// caller is about to start an item that currently lives in the icebox.
// Centralizing this here means every "set status to started" caller
// (card button, detail-panel state machine, sidebar) gets the same
// "I'm starting this so it must be current" semantics, instead of the
// card path being the only one that promoted. Paths come in as
// `<backlog>/<basename>`; we look up the icebox of that backlog and
// promote on a hit. No-op when the item is already in priority.
async function maybePromoteFromIcebox(qualifiedPath: string): Promise<void> {
  if (!client) return;
  const norm = qualifiedPath.replace(/\\/g, '/').replace(/^\.\//, '');
  const slash = norm.indexOf('/');
  if (slash < 0) return; // bare basename: can't infer backlog, skip
  const backlog = norm.slice(0, slash);
  const basename = norm.slice(slash + 1);
  try {
    const ice = await client.iceboxList(backlog);
    const inIcebox = (ice.items || []).some(r => r.path === basename || r.path.endsWith('/' + basename));
    if (!inIcebox) return;
    await client.moveToPriority({ backlog, item_paths: [basename], position: 'top' });
  } catch (e: any) {
    // Best-effort. If the lookup fails we fall through to the normal
    // setStatus call, preserving today's behavior rather than blocking.
    log(`maybePromoteFromIcebox failed for ${qualifiedPath}: ${e?.message || e}`);
  }
}

// samePath returns true when two webview-visible paths refer to the
// same story file. The webview passes paths in `<backlog>/<basename>`
// form, but background callers may forget the prefix. Compare on
// basename + folder to stay robust against either shape.
function samePath(a: string | null | undefined, b: string | null | undefined): boolean {
  if (!a || !b) return false;
  const norm = (p: string) => p.replace(/\\/g, '/').replace(/^\.\//, '');
  return norm(a) === norm(b) || norm(a).endsWith('/' + norm(b)) || norm(b).endsWith('/' + norm(a));
}

// sendDetail re-fetches the open story's detail + comments + tasks +
// history and posts a fresh 'detail' message to the webview. Used by
// afterWrite() so any write that mutates the open story keeps the
// DetailPanel coherent with the board's snapshot.
async function sendDetail(path: string): Promise<void> {
  if (!client || !panel) return;
  try {
    const [item, comments, tasks, history] = await Promise.all([
      client.getItem(path),
      client.getComments(path).catch(() => ({ comments: [], count: 0 })),
      client.listTasks(path).catch(() => ({ tasks: [], count: 0 })),
      client.history(path, 20).catch(() => ({ entries: [], count: 0 })),
    ]);
    panel.webview.postMessage({ type: 'detail', payload: { item, comments, tasks, history } });
  } catch (e: any) {
    log(`sendDetail failed for ${path}: ${e?.message || e}`);
  }
}

// afterWrite is the post-write fan-out every onMessage write branch
// goes through. It refreshes the board snapshot AND, if the write
// targeted the path the DetailPanel is currently bound to, refreshes
// the detail too so the panel and the card cannot disagree on status,
// estimate, tags, blocked, or epic. Without this seam, set-status from
// the detail-panel state-machine left the panel with stale status
// while the card updated from the snapshot, and the user saw a
// contradictory UI.
async function afterWrite(writtenPath?: string): Promise<void> {
  await sendBoardSnapshot();
  if (currentDetailPath && writtenPath && samePath(currentDetailPath, writtenPath)) {
    await sendDetail(currentDetailPath);
  }
}

async function sendBoardSnapshot(): Promise<void> {
  if (!client || !panel) return;
  // Every snapshot reflects a possible state change, so the pulse
  // sidebar's cached aggregation invalidates here. Cheap: the tree
  // re-fetches on next getChildren, batching multiple writes into
  // one reload.
  sidebar?.refresh();
  // Project label for the topbar. Priority:
  //   1. `project.name` from `.am/config.yaml` (if the user set one)
  //   2. workspace folder basename (the natural default)
  //   3. literal "agilemarkdown" if nothing else is available
  // The config-set name lets the human pick a display label that
  // differs from the on-disk directory (e.g. "Services Guide" with
  // a space, when the folder is `services-guide`).
  const wsFolder = pickAmRoot().folder;
  const fallback = wsFolder ? path.basename(wsFolder.uri.fsPath) : 'agilemarkdown';
  let project = fallback;
  try {
    const info = await client.project();
    if (info?.name && info.name.trim()) project = info.name.trim();
  } catch {
    // best-effort; fall back to the workspace folder name.
  }
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
    panel.webview.postMessage({ type: 'snapshot', payload: { project, backlog: '', priority: { items: [], count: 0, velocity: 0 }, icebox: { items: [], count: 0 }, done: { items: [], count: 0 }, dashboard: null, velocity: { rows: [] }, epics: [], backlogs } });
    return;
  }
  const [priority, icebox, done, dashboard, velocity, mix, burnup, cfd] = await Promise.all([
    client.priorityList(backlog),
    client.iceboxList(backlog),
    client.doneList(backlog).catch(() => ({ items: [], count: 0 } as any)),
    client.dashboard().catch(() => null),
    client.velocityHistory(backlog).catch(() => ({ rows: [] })),
    client.typeMix().catch(() => ({ rows: [], total: 0 })),
    client.burnupChart(backlog, 0).catch(() => null),
    client.cumulativeFlow(30).catch(() => null),
  ]);
  const epicSlugs = uniqueEpicSlugs([...priority.items, ...icebox.items, ...done.items]);
  const epics = await Promise.all(epicSlugs.map(slug => client!.epicProgress(slug).catch(() => null)));
  panel.webview.postMessage({
    type: 'snapshot',
    payload: {
      project,
      backlog,
      backlogs,
      priority,
      icebox,
      done,
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
