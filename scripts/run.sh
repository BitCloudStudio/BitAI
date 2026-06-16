#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BACKEND_DIR="$ROOT/backend"
FRONTEND_DIR="$ROOT/frontend"
LOG_DIR="$ROOT/logs"
ENV_FILE="$BACKEND_DIR/.env.local"
BACKEND_PORT=8091

stop_port() {
  local port="$1"
  if command -v lsof >/dev/null 2>&1; then
    local pids
    pids="$(lsof -ti "tcp:$port" || true)"
    if [[ -n "$pids" ]]; then
      kill -9 $pids 2>/dev/null || true
    fi
  elif command -v fuser >/dev/null 2>&1; then
    fuser -k "${port}/tcp" >/dev/null 2>&1 || true
  fi
}

http_ok() {
  local url="$1"
  if command -v curl >/dev/null 2>&1; then
    curl -fsS "$url" >/dev/null 2>&1
  elif command -v wget >/dev/null 2>&1; then
    wget -qO- "$url" >/dev/null 2>&1
  elif command -v python3 >/dev/null 2>&1; then
    python3 - "$url" <<'PY'
import sys
from urllib.request import urlopen
try:
    with urlopen(sys.argv[1], timeout=2) as resp:
        raise SystemExit(0 if 200 <= resp.status < 300 else 1)
except Exception:
    raise SystemExit(1)
PY
  else
    return 1
  fi
}

load_env() {
  local path="$1"
  if [[ ! -f "$path" ]]; then
    echo "No backend/.env.local found. Defaults will be used."
    return 0
  fi
  while IFS='=' read -r key value; do
    [[ -z "${key// }" || "${key:0:1}" == "#" ]] && continue
    key="$(echo "$key" | xargs)"
    export "$key=$value"
  done < "$path"
  if [[ "${BITAPI_HTTP_ADDR:-}" == ":8080" ]]; then
    export BITAPI_HTTP_ADDR=":${BACKEND_PORT}"
  fi
}

ensure_frontend() {
  if ! command -v node >/dev/null 2>&1; then
    echo "Node.js is not installed. Please install Node.js 20 or newer." >&2
    exit 1
  fi
  if ! command -v npm >/dev/null 2>&1; then
    echo "npm is not installed." >&2
    exit 1
  fi
  local node_major
  node_major="$(node -p "Number(process.versions.node.split('.')[0])")"
  if [[ "$node_major" -lt 20 ]]; then
    echo "Node.js is too old: $(node -v). Please install Node.js 20 or newer." >&2
    exit 1
  fi
  (
    cd "$FRONTEND_DIR"
    local marker=".node_platform"
    local platform
    platform="$(uname -s)-$(uname -m)-node${node_major}"
    if [[ ! -d node_modules || ! -f "$marker" || "$(cat "$marker" 2>/dev/null)" != "$platform" || package.json -nt "$marker" || package-lock.json -nt "$marker" ]]; then
      echo "Installing frontend dependencies for $platform ..."
      if [[ -f package-lock.json ]]; then
        npm ci --include=dev
      else
        npm install
      fi
      printf '%s\n' "$platform" > "$marker"
    fi
    echo "Building frontend dist..."
    npm run build
  )
}

mkdir -p "$LOG_DIR" "$BACKEND_DIR/data/uploads/avatars"

echo "Stopping old services..."
stop_port 8080
stop_port 5173
stop_port 5181
stop_port "$BACKEND_PORT"

load_env "$ENV_FILE"
ensure_frontend

echo "Starting backend http://localhost:${BACKEND_PORT} ..."
(
  cd "$BACKEND_DIR"
  go run ./cmd/server
) >"$LOG_DIR/backend.log" 2>"$LOG_DIR/backend.err.log" &
BACKEND_PID=$!
echo "$BACKEND_PID" > "$BACKEND_DIR/.server.pid"
deadline=$((SECONDS + 30))
while [[ $SECONDS -lt $deadline ]]; do
  if ! kill -0 "$BACKEND_PID" 2>/dev/null; then
    echo "Backend failed to start. Check $LOG_DIR/backend.err.log" >&2
    tail -n 80 "$LOG_DIR/backend.err.log" >&2 || true
    exit 1
  fi
  if http_ok "http://127.0.0.1:${BACKEND_PORT}/health"; then
    break
  fi
  sleep 1
done
if ! http_ok "http://127.0.0.1:${BACKEND_PORT}/health"; then
  echo "Backend did not become healthy. Check $LOG_DIR/backend.err.log" >&2
  tail -n 80 "$LOG_DIR/backend.err.log" >&2 || true
  exit 1
fi

echo "Started."
echo "Web and API: http://localhost:${BACKEND_PORT}"
echo "Admin login: http://localhost:${BACKEND_PORT}/auth/login"
echo "Backend log: $LOG_DIR/backend.log"
