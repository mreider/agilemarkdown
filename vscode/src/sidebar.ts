// Activity-bar tree for the agilemarkdown extension. Used for the
// two things VS Code's chrome is better at than the webview:
//
//   1. Show/hide board columns (Backlog, Icebox, Epics). Clicking a
//      row dispatches `toggle-column` to the webview; the board
//      removes or restores the column accordingly. The tree always
//      lists all three so the user has a stable surface to find them.
//
//   2. Sync footer. Single action that runs `am sync`.
//
// The webview reports its current visibleColumns set back to the
// extension via the `visible-columns` message; the tree re-renders
// with a check mark on every visible column so the user can see what
// they have on without looking at the board.

import * as vscode from 'vscode';

export interface SidebarHost {
  isReady(): boolean;
  visibleColumns(): Set<string>;
  toggleColumn(col: string): void;
  currentView(): 'home' | 'analytics' | 'settings';
  setView(view: 'home' | 'analytics' | 'settings'): void;
}

type ColumnKey = 'search' | 'backlog' | 'done' | 'icebox' | 'epics';
const COLUMNS: Array<{ key: ColumnKey; label: string; icon: string }> = [
  { key: 'search',  label: 'Search',  icon: 'search' },
  { key: 'backlog', label: 'Backlog', icon: 'list-ordered' },
  { key: 'done',    label: 'Done',    icon: 'check-all' },
  { key: 'icebox',  label: 'Icebox',  icon: 'archive' },
  { key: 'epics',   label: 'Epics',   icon: 'layers' },
];

type ViewKey = 'home' | 'analytics' | 'settings';
const VIEWS: Array<{ key: ViewKey; label: string; icon: string }> = [
  { key: 'home',      label: 'Home',      icon: 'home' },
  { key: 'analytics', label: 'Analytics', icon: 'graph-line' },
  { key: 'settings',  label: 'Settings',  icon: 'settings-gear' },
];

export class SidebarProvider implements vscode.TreeDataProvider<Node> {
  private _onDidChange = new vscode.EventEmitter<Node | undefined | void>();
  readonly onDidChangeTreeData = this._onDidChange.event;

  constructor(private readonly host: SidebarHost) {}

  refresh(): void {
    this._onDidChange.fire();
  }

  getTreeItem(node: Node): vscode.TreeItem {
    return node.toTreeItem();
  }

  async getChildren(node?: Node): Promise<Node[]> {
    if (node) return [];
    if (!this.host.isReady()) return [];
    const visible = this.host.visibleColumns();
    const currentView = this.host.currentView();
    const out: Node[] = [];
    for (const v of VIEWS) {
      out.push(new ViewNode(v.key, v.label, v.icon, v.key === currentView));
    }
    for (const c of COLUMNS) {
      out.push(new ColumnToggleNode(c.key, c.label, c.icon, visible.has(c.key)));
    }
    out.push(new ActionNode('Sync', 'agilemarkdown.sync', 'sync'));
    out.push(new ActionNode('Help', 'agilemarkdown.openDocs', 'question'));
    return out;
  }
}

abstract class Node {
  abstract toTreeItem(): vscode.TreeItem;
}

class HeaderNode extends Node {
  constructor(private readonly text: string) { super(); }
  toTreeItem(): vscode.TreeItem {
    const item = new vscode.TreeItem(this.text, vscode.TreeItemCollapsibleState.None);
    item.contextValue = 'header';
    item.description = '';
    item.iconPath = undefined;
    return item;
  }
}

class ColumnToggleNode extends Node {
  constructor(
    private readonly key: ColumnKey,
    private readonly label: string,
    private readonly icon: string,
    private readonly visible: boolean,
  ) { super(); }
  toTreeItem(): vscode.TreeItem {
    const item = new vscode.TreeItem(this.label, vscode.TreeItemCollapsibleState.None);
    item.iconPath = new vscode.ThemeIcon(this.icon);
    item.description = this.visible ? '●' : '';
    item.tooltip = this.visible
      ? `${this.label} is showing. Click to hide.`
      : `${this.label} is hidden. Click to show.`;
    item.command = {
      command: 'agilemarkdown.toggleColumn',
      title: `Toggle ${this.label} column`,
      arguments: [this.key],
    };
    item.contextValue = 'column-toggle';
    return item;
  }
}

class ViewNode extends Node {
  constructor(
    private readonly key: ViewKey,
    private readonly label: string,
    private readonly icon: string,
    private readonly active: boolean,
  ) { super(); }
  toTreeItem(): vscode.TreeItem {
    const item = new vscode.TreeItem(this.label, vscode.TreeItemCollapsibleState.None);
    item.iconPath = new vscode.ThemeIcon(this.icon);
    item.description = this.active ? '● current' : '';
    item.tooltip = `Open ${this.label}`;
    item.command = {
      command: 'agilemarkdown.setView',
      title: `Open ${this.label}`,
      arguments: [this.key],
    };
    item.contextValue = 'view';
    return item;
  }
}

class ActionNode extends Node {
  constructor(private readonly label: string, private readonly cmd: string, private readonly icon: string) { super(); }
  toTreeItem(): vscode.TreeItem {
    const item = new vscode.TreeItem(this.label, vscode.TreeItemCollapsibleState.None);
    item.iconPath = new vscode.ThemeIcon(this.icon);
    item.command = { command: this.cmd, title: this.label };
    item.contextValue = 'action';
    return item;
  }
}
