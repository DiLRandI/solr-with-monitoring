#!/usr/bin/env bash
set -euo pipefail

url="${1:?usage: open-url.sh <url>}"

if command -v xdg-open >/dev/null 2>&1; then
  nohup xdg-open "$url" >/dev/null 2>&1 &
elif command -v open >/dev/null 2>&1; then
  nohup open "$url" >/dev/null 2>&1 &
elif command -v python3 >/dev/null 2>&1; then
  nohup python3 -m webbrowser "$url" >/dev/null 2>&1 &
else
  printf '%s\n' "$url"
fi

