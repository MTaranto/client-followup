#!/bin/sh
set -eu

bundle_directory=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)

application_directory="$HOME/.local/share/client-followup"
app_directory="$application_directory/app"
service_directory="$HOME/.config/systemd/user"
desktop_directory="$HOME/.local/share/applications"
icon_directory="$HOME/.local/share/icons/hicolor/512x512/apps"

[ "$#" -eq 0 ] || {
    echo "Uso: ./install.sh" >&2
    exit 2
}

for command_name in systemctl curl xdg-open; do
    command -v "$command_name" >/dev/null 2>&1 || {
        echo "Dependência necessária não encontrada: $command_name" >&2
        exit 1
    }
done

for required_file in \
    client-followup \
    open-when-ready.sh \
    restore-backup.sh \
    uninstall.sh \
    client-followup.service \
    client-followup.desktop \
    client-followup.png
do
    [ -f "$bundle_directory/$required_file" ] || {
        echo "Bundle incompleto: $required_file não encontrado." >&2
        exit 1
    }
done

for required_directory in templates static; do
    [ -d "$bundle_directory/$required_directory" ] || {
        echo "Bundle incompleto: $required_directory/ não encontrado." >&2
        exit 1
    }
done

systemctl --user stop client-followup.service 2>/dev/null || true

mkdir -p \
    "$app_directory" \
    "$application_directory/data" \
    "$application_directory/backups" \
    "$service_directory" \
    "$desktop_directory" \
    "$icon_directory"

chmod 700 \
    "$application_directory/data" \
    "$application_directory/backups"

rm -f "$app_directory/client-followup"
rm -rf "$app_directory/templates" "$app_directory/static"

install -m 0755 \
    "$bundle_directory/client-followup" \
    "$app_directory/client-followup"

cp -R "$bundle_directory/templates" "$app_directory/templates"
cp -R "$bundle_directory/static" "$app_directory/static"

install -m 0755 \
    "$bundle_directory/open-when-ready.sh" \
    "$application_directory/open-when-ready.sh"

install -m 0755 \
    "$bundle_directory/restore-backup.sh" \
    "$application_directory/restore-backup.sh"

install -m 0755 \
    "$bundle_directory/uninstall.sh" \
    "$application_directory/uninstall.sh"

install -m 0644 \
    "$bundle_directory/client-followup.service" \
    "$service_directory/client-followup.service"

install -m 0644 \
    "$bundle_directory/client-followup.desktop" \
    "$desktop_directory/client-followup.desktop"

install -m 0644 \
    "$bundle_directory/client-followup.png" \
    "$icon_directory/client-followup.png"

systemctl --user daemon-reload

if command -v update-desktop-database >/dev/null 2>&1; then
    update-desktop-database "$desktop_directory" 2>/dev/null || true
fi

echo "Client Follow-up instalado."
echo "Abra \"Client Follow-up\" pelo menu de aplicativos."
echo "Dados: $application_directory/data"
echo "Backups: $application_directory/backups"
