#!/bin/sh
set -eu

project_directory=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
application_directory=${XDG_DATA_HOME:-"$HOME/.local/share"}/client-followup
service_directory=${XDG_CONFIG_HOME:-"$HOME/.config"}/systemd/user
autostart_directory=${XDG_CONFIG_HOME:-"$HOME/.config"}/autostart

command -v go >/dev/null 2>&1 || { echo "Go não está instalado." >&2; exit 1; }
command -v curl >/dev/null 2>&1 || { echo "curl não está instalado." >&2; exit 1; }
command -v flock >/dev/null 2>&1 || { echo "flock não está instalado." >&2; exit 1; }
command -v xdg-open >/dev/null 2>&1 || { echo "xdg-open não está instalado." >&2; exit 1; }

mkdir -p "$application_directory"
install -d -m 0700 "$application_directory/data" "$application_directory/backups"
mkdir -p "$application_directory/scripts" "$service_directory" "$autostart_directory"

(cd "$project_directory" && go build -o "$application_directory/client-followup" .)
rm -rf "$application_directory/templates" "$application_directory/static"
cp -R "$project_directory/templates" "$application_directory/templates"
cp -R "$project_directory/static" "$application_directory/static"
install -m 0755 "$project_directory/scripts/open-when-ready.sh" "$application_directory/scripts/open-when-ready.sh"
install -m 0644 "$project_directory/systemd/client-followup.service" "$service_directory/client-followup.service"
install -m 0644 "$project_directory/autostart/client-followup.desktop" "$autostart_directory/client-followup.desktop"

systemctl --user daemon-reload
systemctl --user enable --now client-followup.service

echo "Client Follow-up instalado. O serviço já está ativo e o painel abrirá nos próximos logins."
echo "Dados: $application_directory/data"
echo "Backups: $application_directory/backups"
