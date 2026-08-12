#!/bin/sh
set -eu

runtime_directory=${XDG_RUNTIME_DIR:-/tmp}
lock_file="$runtime_directory/client-followup-browser.lock"

exec 9>"$lock_file"
flock -n 9 || exit 0

attempt=0
while [ "$attempt" -lt 60 ]; do
    if curl --fail --silent --max-time 2 http://127.0.0.1:8080/health >/dev/null; then
        xdg-open http://127.0.0.1:8080 >/dev/null 2>&1
        exit 0
    fi
    attempt=$((attempt + 1))
    sleep 1
done

echo "Client Follow-up não respondeu em http://127.0.0.1:8080/health" >&2
exit 1
