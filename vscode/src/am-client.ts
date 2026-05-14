// Per-verb shell-out to the `am` CLI. One `execFile` per call, JSON on
// stdout for reads, exit code for writes. No long-lived child process,
// no MCP framing.
//
// The shape mirrors the previous MCP-backed client so callers in
// extension.ts and the webview message router don't change.

import * as cp from 'child_process';
import * as path from 'path';
import {
  PriorityListResult,
  IceboxListResult,
  ListItemsResult,
  DashboardResult,
  VelocityHistoryResult,
  BurnupResult,
  CFDResult,
  TypeMixResult,
  GetCommentsResult,
  ListTasksResult,
  EpicProgressResult,
  GetItemResult,
  HistoryResult,
  SearchResult,
} from './shared/types';

export class AmClient {
  constructor(private readonly amPath: string, private readonly root: string) {}

  // --- read verbs (JSON on stdout) ---

  listBacklogs(): Promise<{ backlogs: string[] }> {
    return this.readJSON(['list-backlogs']);
  }

  listItems(backlog: string, status?: string, tag?: string): Promise<ListItemsResult> {
    const args = ['list-items', backlog];
    if (status) args.push('--status', status);
    if (tag) args.push('--tag', tag);
    return this.readJSON(args);
  }

  priorityList(backlog: string): Promise<PriorityListResult> {
    return this.readJSON(['show', 'priority', '--json'], { cwd: this.backlogDir(backlog) });
  }

  iceboxList(backlog: string): Promise<IceboxListResult> {
    return this.readJSON(['show', 'icebox', '--json'], { cwd: this.backlogDir(backlog) });
  }

  dashboard(): Promise<DashboardResult> {
    return this.readJSON(['dashboard', '--json']);
  }

  velocityHistory(_backlog: string): Promise<VelocityHistoryResult> {
    // velocity --json works from any subdir of the project.
    return this.readJSON(['velocity', '--json']);
  }

  burnupChart(backlog: string, offset = 0): Promise<BurnupResult> {
    const args = ['show', 'burnup', '--json'];
    if (offset !== 0) args.push(String(offset));
    return this.readJSON(args, { cwd: this.backlogDir(backlog) });
  }

  typeMix(): Promise<TypeMixResult> {
    return this.readJSON(['type-mix']);
  }

  cumulativeFlow(days = 30): Promise<CFDResult> {
    return this.readJSON(['show', 'cfd', '--json', '--days', String(days)]);
  }

  whoami(): Promise<{ name: string; email: string }> {
    return this.readJSON(['whoami']);
  }

  history(itemPath: string, limit = 50): Promise<HistoryResult> {
    return this.readJSON(['history', '--limit', String(limit), itemPath]);
  }

  search(query: string, limit = 20): Promise<SearchResult> {
    return this.readJSON(['search', '--limit', String(limit), query]);
  }

  getItem(itemPath: string): Promise<GetItemResult> {
    return this.readJSON(['get-item', itemPath]);
  }

  getComments(itemPath: string): Promise<GetCommentsResult> {
    return this.readJSON(['get-comments', itemPath]);
  }

  listTasks(itemPath: string): Promise<ListTasksResult> {
    return this.readJSON(['task', 'list', '--json', itemPath]);
  }

  epicProgress(slug: string): Promise<EpicProgressResult> {
    return this.readJSON(['show', 'epic', '--json', slug]);
  }

  sprintPlan(backlog: string): Promise<any> {
    return this.readJSON(['sprint', 'plan', '--json'], { cwd: this.backlogDir(backlog) });
  }

  // --- write verbs (exit code only) ---

  async createBacklog(name: string): Promise<{ ok: boolean }> {
    await this.run(['create-backlog', name]);
    return { ok: true };
  }

  async createItem(backlog: string, title: string, _user?: string): Promise<{ path: string }> {
    // `am create-item` writes to cwd's backlog. cd into the backlog folder.
    const { stdout } = await this.run(['create-item', title], { cwd: this.backlogDir(backlog) });
    // The CLI prints the created path; parse it loosely.
    const m = stdout.match(/([^\s]+\.md)/);
    return { path: m ? path.resolve(this.backlogDir(backlog), m[1]) : '' };
  }

  async setStatus(itemPath: string, status: string): Promise<{ ok: boolean }> {
    const verb = statusVerb(status);
    if (!verb) throw new Error(`unsupported status: ${status}`);
    await this.run([verb, itemPath]);
    return { ok: true };
  }

  async setEstimate(itemPath: string, estimate: string): Promise<{ ok: boolean }> {
    await this.run(['estimate', itemPath, estimate]);
    return { ok: true };
  }

  async setAssigned(itemPath: string, assignees: string[]): Promise<{ ok: boolean }> {
    await this.run(['assign', itemPath, ...assignees]);
    return { ok: true };
  }

  async setTags(itemPath: string, tags: string[]): Promise<{ ok: boolean }> {
    await this.run(['tag', itemPath, ...tags]);
    return { ok: true };
  }

  async setEpic(itemPath: string, slug: string): Promise<{ ok: boolean }> {
    if (!slug) {
      await this.run(['epic', '--unset', itemPath]);
    } else {
      await this.run(['epic', itemPath, slug]);
    }
    return { ok: true };
  }

  async setDescription(itemPath: string, body: string): Promise<{ ok: boolean }> {
    await this.run(['set-description', itemPath], { stdin: body });
    return { ok: true };
  }

  async rankItem(args: { backlog: string; item_path: string; position?: 'top' | 'bottom'; after?: string; before?: string }): Promise<{ ok: boolean }> {
    const cliArgs = ['rank', args.item_path];
    if (args.position === 'top') cliArgs.push('--top');
    else if (args.position === 'bottom') cliArgs.push('--bottom');
    else if (args.after) cliArgs.push('--after', args.after);
    else if (args.before) cliArgs.push('--before', args.before);
    await this.run(cliArgs, { cwd: this.backlogDir(args.backlog) });
    return { ok: true };
  }

  async moveToIcebox(args: { backlog: string; item_path: string; position?: 'top' | 'bottom' }): Promise<{ ok: boolean }> {
    const cliArgs = ['ice', args.item_path];
    if (args.position === 'top') cliArgs.push('--top');
    await this.run(cliArgs, { cwd: this.backlogDir(args.backlog) });
    return { ok: true };
  }

  async moveToPriority(args: { backlog: string; item_paths: string[]; position?: 'top' | 'bottom'; after?: string }): Promise<{ ok: boolean }> {
    if (args.item_paths.length === 0) throw new Error('item_paths is required');
    // `am unice` takes a single ITEM_PATH (or --all). For bulk moves we call once per item.
    for (const p of args.item_paths) {
      const cliArgs = ['unice', p];
      if (args.position === 'top') cliArgs.push('--top');
      else if (args.after) cliArgs.push('--after', args.after);
      await this.run(cliArgs, { cwd: this.backlogDir(args.backlog) });
    }
    return { ok: true };
  }

  async blockItem(itemPath: string, reason?: string): Promise<{ ok: boolean }> {
    const cliArgs = ['block', itemPath];
    if (reason) cliArgs.push('--reason', reason);
    await this.run(cliArgs);
    return { ok: true };
  }

  async unblockItem(itemPath: string): Promise<{ ok: boolean }> {
    await this.run(['unblock', itemPath]);
    return { ok: true };
  }

  async rejectItem(itemPath: string, reason: string): Promise<{ ok: boolean }> {
    const cliArgs = ['reject', itemPath];
    if (reason) cliArgs.push('--reason', reason);
    await this.run(cliArgs);
    return { ok: true };
  }

  async addComment(itemPath: string, text: string, author?: string): Promise<{ ok: boolean }> {
    const cliArgs = ['comment'];
    if (author) cliArgs.push('--author', author);
    cliArgs.push(itemPath, text);
    await this.run(cliArgs);
    return { ok: true };
  }

  async addTask(itemPath: string, text: string): Promise<{ ok: boolean }> {
    await this.run(['task', 'add', itemPath, text]);
    return { ok: true };
  }

  async setTaskDone(itemPath: string, index: number, done: boolean): Promise<{ ok: boolean }> {
    const cliArgs = ['task', 'tick'];
    if (!done) cliArgs.push('--undo');
    cliArgs.push(itemPath, String(index));
    await this.run(cliArgs);
    return { ok: true };
  }

  async sync(): Promise<{ ok: boolean }> {
    await this.run(['sync']);
    return { ok: true };
  }

  // --- internals ---

  private backlogDir(backlog: string): string {
    return path.join(this.root, backlog);
  }

  private async readJSON<T>(args: string[], opts: { cwd?: string } = {}): Promise<T> {
    const { stdout } = await this.run(args, opts);
    const trimmed = stdout.trim();
    if (!trimmed) throw new Error(`am ${args.join(' ')} produced no JSON`);
    try {
      return JSON.parse(trimmed) as T;
    } catch (err: any) {
      throw new Error(`am ${args.join(' ')} returned non-JSON output: ${trimmed.slice(0, 200)}`);
    }
  }

  private run(args: string[], opts: { cwd?: string; stdin?: string } = {}): Promise<{ stdout: string; stderr: string }> {
    return new Promise((resolve, reject) => {
      const child = cp.execFile(
        this.amPath,
        args,
        { cwd: opts.cwd || this.root, maxBuffer: 16 * 1024 * 1024 },
        (err, stdout, stderr) => {
          if (err) {
            const msg = (stderr || '').toString().trim() || (err as any).message || String(err);
            const verb = args[0] || '';
            reject(new Error(`am ${verb} failed: ${msg}`));
            return;
          }
          resolve({ stdout: stdout.toString(), stderr: stderr.toString() });
        },
      );
      if (opts.stdin !== undefined) {
        child.stdin?.write(opts.stdin);
        child.stdin?.end();
      }
    });
  }
}

function statusVerb(status: string): string | null {
  switch (status) {
    case 'unstarted': return null; // no direct CLI verb; transitions are forward-only
    case 'started': return 'start';
    case 'finished': return 'finish';
    case 'delivered': return 'deliver';
    case 'accepted': return 'accept';
    case 'rejected': return 'reject';
    default: return null;
  }
}
