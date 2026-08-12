#!/bin/sh
set -eu

application_directory=${XDG_DATA_HOME:-"$HOME/.local/share"}/client-followup
service_directory=${XDG_CONFIG_HOME:-"$HOME/.config"}/systemd/user
autostart_directory=${XDG_CONFIG_HOME:-"$HOME/.config"}/autostart

systemctl --user disable --now client-followup.service 2>/dev/null || true
rm -f "$service_directory/client-followup.service"
rm -f "$autostart_directory/client-followup.desktop"
systemctl --user daemon-reload

rm -f "$application_directory/client-followup"
rm -rf "$application_directory/templates" "$application_directory/static" "$application_directory/scripts"

echo "Serviço e aplicação removidos. Os diretórios data/ e backups/ foram preservados em:"
echo "$application_directory"
