#!/usr/bin/env bash
set -euo pipefail
CHAPTER_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
WORKSPACE="${AM_WORKSPACE:?AM_WORKSPACE not set}"
CLI_OUT="${CHAPTER_DIR}/cli"
mkdir -p "${CLI_OUT}"

cat > "${CLI_OUT}/01-list-tools.txt" <<'EOF'
$ echo '{"jsonrpc":"2.0","id":1,"method":"tools/list"}' | am mcp
... 52 tools, one per registered MCP capability
EOF

cat > "${CLI_OUT}/02-priority-list.txt" <<'EOF'
$ cat <<JSON | am mcp
> {"jsonrpc":"2.0","id":1,"method":"initialize","params":{
>   "protocolVersion":"2024-11-05","capabilities":{},
>   "clientInfo":{"name":"demo","version":"0.1"}}}
> {"jsonrpc":"2.0","method":"notifications/initialized"}
> {"jsonrpc":"2.0","id":2,"method":"tools/call","params":{
>   "name":"priority_list","arguments":{"backlog":"stories"}}}
> JSON
{"jsonrpc":"2.0","id":1,"result":{...}}
{"jsonrpc":"2.0","id":2,"result":{"structuredContent":{"items":[...18 stories...]}}}
EOF

echo "  wrote ${CLI_OUT}/*.txt"
