#!/bin/sh
set -eu

project_directory=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
dist_directory="$project_directory/dist"
bundle_name="client-followup-linux-amd64"
bundle_directory="$dist_directory/$bundle_name"
archive_file="$dist_directory/$bundle_name.tar.gz"

for command_name in go tar file ldd; do
    command -v "$command_name" >/dev/null 2>&1 || {
        echo "Ferramenta de build não encontrada: $command_name" >&2
        exit 1
    }
done

[ "$(go env GOOS)" = "linux" ] || {
    echo "Este release deve ser compilado para Linux." >&2
    exit 1
}

[ "$(go env GOARCH)" = "amd64" ] || {
    echo "Este release deve ser compilado para amd64." >&2
    exit 1
}

[ "$(go env CGO_ENABLED)" = "1" ] || {
    echo "CGO precisa estar habilitado para este build." >&2
    exit 1
}

mkdir -p "$dist_directory"

rm -rf -- "$bundle_directory"
rm -f -- "$archive_file"

mkdir -p "$bundle_directory"

(
    cd "$project_directory"
    go build -o "$bundle_directory/client-followup" .
)

cp -a \
    "$project_directory/templates" \
    "$bundle_directory/templates"

cp -a \
    "$project_directory/static" \
    "$bundle_directory/static"

install -m 0755 \
    "$project_directory/scripts/install-user-service.sh" \
    "$bundle_directory/install.sh"

install -m 0755 \
    "$project_directory/scripts/uninstall-user-service.sh" \
    "$bundle_directory/uninstall.sh"

install -m 0755 \
    "$project_directory/scripts/open-when-ready.sh" \
    "$bundle_directory/open-when-ready.sh"

install -m 0644 \
    "$project_directory/systemd/client-followup.service" \
    "$bundle_directory/client-followup.service"

install -m 0644 \
    "$project_directory/desktop/client-followup.desktop" \
    "$bundle_directory/client-followup.desktop"

install -m 0644 \
    "$project_directory/assets/icons/client-follow-up-desktop-icon.png" \
    "$bundle_directory/client-followup.png"

sh -n "$bundle_directory/install.sh"
sh -n "$bundle_directory/uninstall.sh"
sh -n "$bundle_directory/open-when-ready.sh"

if command -v desktop-file-validate >/dev/null 2>&1; then
    desktop-file-validate "$bundle_directory/client-followup.desktop"
fi

echo
echo "=== EXECUTÁVEL ==="
file "$bundle_directory/client-followup"

echo
echo "=== DEPENDÊNCIAS DINÂMICAS ==="
ldd "$bundle_directory/client-followup"

tar \
    -C "$dist_directory" \
    -czf "$archive_file" \
    "$bundle_name"

echo
echo "Release criado:"
echo "$archive_file"
