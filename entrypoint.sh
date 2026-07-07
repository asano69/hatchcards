#!/bin/bash
set -e

if [ -d "/certs" ] && [ "$(ls -A /certs/*.crt 2>/dev/null)" ]; then
  cp /certs/*.crt /usr/local/share/ca-certificates/
  update-ca-certificates
fi

exec su-exec hashcards:hashcards "$@"
