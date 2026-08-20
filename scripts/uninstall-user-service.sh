#!/bin/sh
set -eu

application_directory="$HOME/.local/share/client-followup"
service_file="$HOME/.config/systemd/user/client-followup.service"
desktop_file="$HOME/.local/share/applications/client-followup.desktop"
icon_file="$HOME/.local/share/icons/hicolor/512x512/apps/client-followup.png"

systemctl --user stop client-followup.service 2>/dev/null || true

rm -f "$service_file"
systemctl --user daemon-reload

rm -f "$desktop_file"
rm -f "$icon_file"

rm -rf "$application_directory/app"
rm -f "$application_directory/open-when-ready.sh"
rm -f "$application_directory/restore-backup.sh"
rm -f "$application_directory/uninstall.sh"

if command -v update-desktop-database >/dev/null 2>&1; then
    update-desktop-database "$HOME/.local/share/applications" 2>/dev/null || true
fi

echo "Client Follow-up removido."
echo "Seus dados e backups foram preservados em:"
echo "$application_directory"
