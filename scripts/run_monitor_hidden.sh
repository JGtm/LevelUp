#!/usr/bin/env bash
# Équivalent Linux de run_monitor_hidden.vbs
# Lance monitor_uptime.py en arrière-plan (détaché du terminal)

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(dirname "$SCRIPT_DIR")"

nohup "$REPO_ROOT/.venv/bin/python" "$SCRIPT_DIR/monitor_uptime.py" \
    >> "$REPO_ROOT/data/logs/monitor.log" 2>&1 &
