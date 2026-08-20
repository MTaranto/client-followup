#!/bin/sh
set -eu

command -v systemctl >/dev/null 2>&1 || {
    echo "systemctl não está disponível." >&2
    exit 1
}

command -v curl >/dev/null 2>&1 || {
    echo "curl não está disponível." >&2
    exit 1
}

command -v xdg-open >/dev/null 2>&1 || {
    echo "xdg-open não está disponível." >&2
    exit 1
}

systemctl --user start client-followup.service

attempt=0
while [ "$attempt" -lt 60 ]; do
    if curl --fail --silent --max-time 2 \
        http://127.0.0.1:8080/health >/dev/null; then
        exec xdg-open http://127.0.0.1:8080
    fi

    attempt=$((attempt + 1))
    sleep 1
done

echo "Client Follow-up não respondeu em http://127.0.0.1:8080/health" >&2
exit 1
