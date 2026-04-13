#!/bin/sh
set -e

# DATA_DIR: path to the data directory mounted from the host.
# entrypoint.sh uses this to match the host directory owner UID/GID.
# Default: /app/data
DATA_DIR="${DATA_DIR:-/app/data}"
if [ -d "$DATA_DIR" ]; then
	RUN_UID=$(stat -c '%u' "$DATA_DIR")
	RUN_GID=$(stat -c '%g' "$DATA_DIR")
else
	RUN_UID=1000
	RUN_GID=1000
fi

if [ "$(id -u)" = "0" ]; then
	if ! getent group "$RUN_GID" >/dev/null 2>&1; then
		addgroup -g "$RUN_GID" appgroup
	fi
	if ! getent passwd "$RUN_UID" >/dev/null 2>&1; then
		adduser -u "$RUN_UID" -G "$(getent group "$RUN_GID" | cut -d: -f1)" \
			-H -s /sbin/nologin -D appuser
	fi
	# CONFIG_FILE: path to the hashwrap config file. Default: /app/config.json
	exec su-exec "$RUN_UID" hashwrap "${CONFIG_FILE:-/app/config.json}"
else
	exec hashwrap "${CONFIG_FILE:-/app/config.json}"
fi
