#!/bin/bash
lockfile="/tmp/sync.lock"

if [[ -f $lockfile ]]; then
  echo "Sync já está rodando."
  exit
else
  touch $lockfile
  echo "[$(date)] Iniciado sync"
  ./sync-freebsd >> sync.log 2>&1
  rm -f $lockfile
fi
