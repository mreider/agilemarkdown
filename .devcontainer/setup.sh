#!/usr/bin/env bash
# Codespaces postCreate: install agilemarkdown into the container so
# `am` and the VS Code extension both work out of the box. Runs once
# when the codespace is first created; subsequent starts skip this.

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${REPO_ROOT}"

echo "▶ building am CLI"
mkdir -p "${HOME}/go/bin"
go build -o "${HOME}/go/bin/am" .
# Many CLI verbs accept either name for back-compat.
cp "${HOME}/go/bin/am" "${HOME}/go/bin/agilemarkdown"

echo "▶ installing VS Code extension dependencies"
cd "${REPO_ROOT}/vscode"
npm ci

echo "▶ packaging extension"
npx webpack --mode production
npx vsce package --allow-missing-repository

echo "▶ installing extension into this codespace"
# `code` is provided by Codespaces. In a local devcontainer this is
# also wired by the VS Code Server installation.
VSIX="$(ls -t "${REPO_ROOT}/vscode"/*.vsix | head -1)"
if command -v code >/dev/null 2>&1; then
  code --install-extension "${VSIX}"
fi

echo "▶ seeding a demo project at /workspaces/am-demo"
cd "${REPO_ROOT}"
bash scripts/build-demo.sh /workspaces/am-demo

echo
echo "Done. Open /workspaces/am-demo in VS Code (File ▸ Open Folder...) to play with the extension."
echo "Or run \`am show priority\` inside that folder to read the CLI surface."
