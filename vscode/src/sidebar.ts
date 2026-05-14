// Tree-data provider for the Agile Markdown activity-bar sidebar.
// Two sections, both data-driven:
//   • Backlogs — one row per backlog, click to switch the board
//   • Actions  — New story / New backlog / Sync
// Empty state surfaces "Create your first backlog" instead of dead text.

import * as vscode from 'vscode';
import { AmClient } from './am-client';

type Section = 'backlogs' | 'actions';

export interface SidebarHost {
  amClient(): AmClient | null;
  currentBacklog(): string | null;
  selectBacklog(name: string): Promise<void>;
  isReady(): boolean;
}

export class SidebarProvider implements vscode.TreeDataProvider<Node> {
  private _onDidChange = new vscode.EventEmitter<Node | undefined | void>();
  readonly onDidChangeTreeData = this._onDidChange.event;

  private backlogs: string[] = [];
  private loaded = false;

  constructor(private readonly host: SidebarHost) {}

  refresh(): void {
    this.loaded = false;
    this._onDidChange.fire();
  }

  getTreeItem(node: Node): vscode.TreeItem {
    return node.toTreeItem();
  }

  async getChildren(node?: Node): Promise<Node[]> {
    if (!node) {
      // When the project isn't initialized (no .am or no folder open),
      // return [] so viewsWelcome can render the bootstrap CTA.
      if (!this.host.isReady()) return [];
      // Top-level: section headers. Load backlog list lazily so the
      // tree renders instantly even if the CLI is slow on first call.
      await this.ensureLoaded();
      const out: Node[] = [];
      if (this.backlogs.length === 0) {
        out.push(new MessageNode('This project has no backlogs yet.'));
      } else {
        out.push(new SectionNode('backlogs', `Backlogs (${this.backlogs.length})`));
      }
      out.push(new SectionNode('actions', 'Actions'));
      return out;
    }
    if (node instanceof SectionNode) {
      if (node.section === 'backlogs') {
        const current = this.host.currentBacklog();
        return this.backlogs.map(b => new BacklogNode(b, b === current));
      }
      if (node.section === 'actions') {
        const actions: Node[] = [];
        const current = this.host.currentBacklog();
        if (this.backlogs.length === 0) {
          actions.push(new ActionNode('Create your first backlog', 'agilemarkdown.createBacklog', 'plus'));
        } else {
          if (current) {
            actions.push(new ActionNode(`New story in ${current}`, 'agilemarkdown.newStoryInCurrentBacklog', 'plus'));
          }
          actions.push(new ActionNode('New backlog…', 'agilemarkdown.createBacklog', 'plus'));
          actions.push(new ActionNode('Sync', 'agilemarkdown.sync', 'sync'));
        }
        return actions;
      }
    }
    return [];
  }

  private async ensureLoaded(): Promise<void> {
    if (this.loaded) return;
    const client = this.host.amClient();
    if (!client) {
      this.backlogs = [];
      this.loaded = true;
      return;
    }
    try {
      const res = await client.listBacklogs();
      this.backlogs = res.backlogs || [];
    } catch {
      this.backlogs = [];
    }
    this.loaded = true;
  }
}

abstract class Node {
  abstract toTreeItem(): vscode.TreeItem;
}

class SectionNode extends Node {
  constructor(public readonly section: Section, private readonly label: string) {
    super();
  }
  toTreeItem(): vscode.TreeItem {
    const item = new vscode.TreeItem(this.label, vscode.TreeItemCollapsibleState.Expanded);
    item.contextValue = 'section';
    return item;
  }
}

class BacklogNode extends Node {
  constructor(private readonly name: string, private readonly active: boolean) {
    super();
  }
  toTreeItem(): vscode.TreeItem {
    const item = new vscode.TreeItem(this.name, vscode.TreeItemCollapsibleState.None);
    item.iconPath = new vscode.ThemeIcon(this.active ? 'circle-filled' : 'circle-outline');
    item.description = this.active ? 'current' : undefined;
    item.command = {
      command: 'agilemarkdown.switchBacklog',
      title: 'Switch to backlog',
      arguments: [this.name],
    };
    item.contextValue = 'backlog';
    return item;
  }
}

class ActionNode extends Node {
  constructor(private readonly label: string, private readonly cmd: string, private readonly icon: string) {
    super();
  }
  toTreeItem(): vscode.TreeItem {
    const item = new vscode.TreeItem(this.label, vscode.TreeItemCollapsibleState.None);
    item.iconPath = new vscode.ThemeIcon(this.icon);
    item.command = { command: this.cmd, title: this.label };
    item.contextValue = 'action';
    return item;
  }
}

class MessageNode extends Node {
  constructor(private readonly text: string) {
    super();
  }
  toTreeItem(): vscode.TreeItem {
    const item = new vscode.TreeItem(this.text, vscode.TreeItemCollapsibleState.None);
    item.contextValue = 'message';
    return item;
  }
}
