import { describe, it, expect, beforeAll, afterAll } from 'vitest';
import * as cp from 'child_process';
import * as fs from 'fs';
import * as os from 'os';
import * as path from 'path';
import { AmClient } from './am-client';

// Resolve the local agilemarkdown binary. Skip the suite gracefully when
// the developer has not built it yet.
const REPO_ROOT = path.resolve(__dirname, '..', '..');
const LOCAL_AM = path.join(REPO_ROOT, 'agilemarkdown');

function ensureBinary(): boolean {
  if (fs.existsSync(LOCAL_AM)) return true;
  // try to build once
  try {
    cp.execSync('go build -o agilemarkdown .', { cwd: REPO_ROOT, stdio: 'ignore' });
    return fs.existsSync(LOCAL_AM);
  } catch {
    return false;
  }
}

function makeFixture(): string {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'am-vsc-'));
  cp.execSync('git init -q && git config user.email tester@example.com && git config user.name "Tester"', { cwd: dir, shell: '/bin/sh' });
  cp.execSync(`${LOCAL_AM} create-backlog product`, { cwd: dir });
  cp.execSync(`${LOCAL_AM} create-item "Search relevance"`, { cwd: path.join(dir, 'product') });
  // Sync to materialize _priority.md / _icebox.md AND to leave a real
  // git commit on disk so per-item history tests have something to
  // read back. `am sync` commits.
  cp.execSync(`${LOCAL_AM} sync`, { cwd: dir });
  return dir;
}

describe('AmClient end-to-end', () => {
  if (!ensureBinary()) {
    it.skip('skipped: ./agilemarkdown not built', () => {});
    return;
  }
  let dir: string;
  let client: AmClient;

  beforeAll(async () => {
    dir = makeFixture();
    client = new AmClient(LOCAL_AM, dir);
  }, 60_000);

  afterAll(() => {
    if (dir) fs.rmSync(dir, { recursive: true, force: true });
  });

  it('lists backlogs and items', async () => {
    const backlogs = await client.listBacklogs();
    expect(backlogs.backlogs).toContain('product');
    const items = await client.listItems('product');
    expect(items.count).toBeGreaterThanOrEqual(1);
    expect(items.items.some(i => /search relevance/i.test(i.title))).toBe(true);
  });

  it('reads priority list with count', async () => {
    const pri = await client.priorityList('product');
    expect(typeof pri.count).toBe('number');
  });

  it('blocks and unblocks an item', async () => {
    const items = await client.listItems('product');
    const target = items.items.find(i => /search relevance/i.test(i.title)) || items.items[0];
    expect(target).toBeDefined();
    await client.blockItem(target.path, 'waiting on infra');
    const after = await client.getItem(target.path);
    expect(after.blocked).toBe(true);
    expect(after.blocked_reason).toBe('waiting on infra');
    await client.unblockItem(target.path);
    const cleared = await client.getItem(target.path);
    expect(cleared.blocked).toBeFalsy();
  });

  it('adds and ticks a task', async () => {
    const items = await client.listItems('product');
    const target = items.items.find(i => /search relevance/i.test(i.title)) || items.items[0];
    await client.addTask(target.path, 'first task');
    let listed = await client.listTasks(target.path);
    expect(listed.tasks.find(t => t.text === 'first task')).toBeDefined();
    await client.setTaskDone(target.path, 1, true);
    listed = await client.listTasks(target.path);
    expect(listed.tasks[0].done).toBe(true);
  });

  it('adds a comment and reads it back', async () => {
    const items = await client.listItems('product');
    const target = items.items.find(i => /search relevance/i.test(i.title)) || items.items[0];
    await client.addComment(target.path, 'hello world', 'tester');
    const got = await client.getComments(target.path);
    expect(got.count).toBeGreaterThanOrEqual(1);
    expect(got.comments[got.comments.length - 1].text).toContain('hello world');
  });

  it('sets multi-owner via assignees', async () => {
    const items = await client.listItems('product');
    const target = items.items.find(i => /search relevance/i.test(i.title)) || items.items[0];
    await client.setAssigned(target.path, ['alice', 'bob']);
    const after = await client.getItem(target.path);
    expect(after.assignees?.sort()).toEqual(['alice', 'bob']);
  });

  it('stamps started: when transitioning a story to started', async () => {
    const items = await client.listItems('product');
    const target = items.items.find(i => /search relevance/i.test(i.title)) || items.items[0];
    // Reset to unstarted first, then to started.
    await client.setStatus(target.path, 'unstarted').catch(() => undefined);
    await client.setStatus(target.path, 'started');
    const after = await client.getItem(target.path);
    expect(after.started).toBeTruthy();
  });

  it('whoami returns a non-empty git user', async () => {
    const me = await client.whoami();
    expect(me.email).toContain('@');
    expect(me.name.length).toBeGreaterThan(0);
  });

  it('search returns ranked hits for a known title fragment', async () => {
    const res = await client.search('relevance', 10);
    expect(res.count).toBeGreaterThan(0);
    expect(res.hits[0].title.toLowerCase()).toContain('relevance');
  });

  it('history returns at least the initial commit for a story', async () => {
    const items = await client.listItems('product');
    const target = items.items.find(i => /search relevance/i.test(i.title)) || items.items[0];
    const res = await client.history(target.path, 20);
    expect(res.count).toBeGreaterThanOrEqual(1);
    expect(res.entries[0].hash.length).toBeGreaterThanOrEqual(7);
  });

  it('cumulativeFlow returns rows with three band fields', async () => {
    const cfd = await client.cumulativeFlow(10);
    expect(cfd.rows.length).toBeGreaterThan(0);
    const sample = cfd.rows[cfd.rows.length - 1];
    expect(sample).toHaveProperty('accepted');
    expect(sample).toHaveProperty('in_flight');
    expect(sample).toHaveProperty('backlog');
  });
});
