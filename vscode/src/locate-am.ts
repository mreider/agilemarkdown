// Locate the `am` binary across user setups. Order:
//   1. agilemarkdown.cliPath setting
//   2. PATH (`which am`)
//   3. ~/go/bin/am
//   4. ~/.am/bin/am
//   5. prompt: Download | Browse | Skip

import * as vscode from 'vscode';
import * as fs from 'fs';
import * as os from 'os';
import * as path from 'path';
import * as cp from 'child_process';
import * as https from 'https';

const RELEASE_API = 'https://api.github.com/repos/mreider/agilemarkdown/releases/latest';

export async function locateAm(): Promise<string | null> {
  const cfg = vscode.workspace.getConfiguration('agilemarkdown').get<string>('cliPath');
  if (cfg && fs.existsSync(cfg)) {
    return cfg;
  }
  const fromPath = which('am');
  if (fromPath) return fromPath;
  for (const candidate of [
    path.join(os.homedir(), 'go', 'bin', 'am'),
    path.join(os.homedir(), '.am', 'bin', 'am'),
    path.join(os.homedir(), 'go', 'bin', 'agilemarkdown'),
  ]) {
    if (fs.existsSync(candidate)) return candidate;
  }
  return promptInstall();
}

function which(bin: string): string | null {
  try {
    const out = cp.execSync(process.platform === 'win32' ? `where ${bin}` : `which ${bin}`, {
      stdio: ['ignore', 'pipe', 'ignore'],
    }).toString().trim();
    if (out) {
      // `where` may return multiple lines on Windows; take the first.
      return out.split(/\r?\n/)[0].trim();
    }
  } catch {
    // not on PATH
  }
  return null;
}

async function promptInstall(): Promise<string | null> {
  const choice = await vscode.window.showWarningMessage(
    'Agile Markdown: the `am` CLI was not found.',
    { modal: true, detail: 'The extension shells out to the `am` CLI. Download the matching release binary, browse to one you already have, or cancel.' },
    'Download',
    'Browse',
    'Skip',
  );
  if (choice === 'Download') {
    return downloadLatest();
  }
  if (choice === 'Browse') {
    const picked = await vscode.window.showOpenDialog({ canSelectMany: false, openLabel: 'Use this am binary' });
    if (picked && picked[0]) {
      const p = picked[0].fsPath;
      await vscode.workspace.getConfiguration('agilemarkdown').update('cliPath', p, vscode.ConfigurationTarget.Global);
      return p;
    }
  }
  return null;
}

interface ReleaseAsset { name: string; browser_download_url: string; }
interface ReleaseInfo { tag_name: string; assets: ReleaseAsset[]; }

async function downloadLatest(): Promise<string | null> {
  try {
    const info = await fetchJson<ReleaseInfo>(RELEASE_API);
    const triple = platformTriple();
    const asset = info.assets.find(a => a.name.toLowerCase().includes(triple));
    if (!asset) {
      vscode.window.showErrorMessage(`No release asset matched ${triple}. Found: ${info.assets.map(a => a.name).join(', ')}`);
      return null;
    }
    const dest = path.join(os.homedir(), '.am', 'bin');
    await fs.promises.mkdir(dest, { recursive: true });
    const target = path.join(dest, 'am');
    await downloadFile(asset.browser_download_url, target);
    fs.chmodSync(target, 0o755);
    await vscode.workspace.getConfiguration('agilemarkdown').update('cliPath', target, vscode.ConfigurationTarget.Global);
    vscode.window.showInformationMessage(`Installed am ${info.tag_name} at ${target}`);
    return target;
  } catch (err: any) {
    vscode.window.showErrorMessage(`am download failed: ${err?.message || err}`);
    return null;
  }
}

function platformTriple(): string {
  switch (process.platform) {
    case 'darwin':
      return process.arch === 'arm64' ? 'darwin-arm64' : 'darwin-amd64';
    case 'linux':
      return process.arch === 'arm64' ? 'linux-arm64' : 'linux-amd64';
    case 'win32':
      return 'windows-amd64';
    default:
      return process.platform;
  }
}

function fetchJson<T>(url: string): Promise<T> {
  return new Promise((resolve, reject) => {
    https.get(url, { headers: { 'User-Agent': 'agilemarkdown-vscode' } }, res => {
      if (res.statusCode === 302 || res.statusCode === 301) {
        return fetchJson<T>(res.headers.location!).then(resolve, reject);
      }
      let body = '';
      res.on('data', d => body += d);
      res.on('end', () => {
        try { resolve(JSON.parse(body)); } catch (e) { reject(e); }
      });
    }).on('error', reject);
  });
}

function downloadFile(url: string, target: string): Promise<void> {
  return new Promise((resolve, reject) => {
    https.get(url, { headers: { 'User-Agent': 'agilemarkdown-vscode' } }, res => {
      if (res.statusCode === 302 || res.statusCode === 301) {
        return downloadFile(res.headers.location!, target).then(resolve, reject);
      }
      const out = fs.createWriteStream(target);
      res.pipe(out);
      out.on('finish', () => out.close(() => resolve()));
      out.on('error', reject);
    }).on('error', reject);
  });
}
