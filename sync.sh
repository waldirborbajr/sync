#!/bin/bash
set -euo pipefail
lockfile="/tmp/sync.lock"

cleanup() {
  rm -f "$lockfile"
}
trap cleanup EXIT

if [[ -f $lockfile ]]; then
  echo "Sync já está rodando."
  exit 0
fi

touch "$lockfile"
echo "[$(date)] Iniciado sync"
./sync-freebsd >> sync.log 2>&1 || {
  rc=$?
  echo "Erro: sync terminou com exit code $rc. Verifique sync.log para detalhes."
  exit $rc
}
echo "[$(date)] sync finalizado com sucesso"