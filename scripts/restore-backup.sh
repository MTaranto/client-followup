#!/bin/sh
set -eu

application_directory="$HOME/.local/share/client-followup"
data_directory="$application_directory/data"
backup_directory="$application_directory/backups"
database_file="$data_directory/client-followup.db"
pre_restore_backup="$backup_directory/pre-restore.db"

show_available_backups() {
    echo "Backups disponíveis:" >&2
    found=0
    for backup_file in "$backup_directory"/client-followup-[0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9].db; do
        [ -f "$backup_file" ] || continue
        backup_name=${backup_file##*/}
        backup_date=${backup_name#client-followup-}
        backup_date=${backup_date%.db}
        echo "  $backup_date" >&2
        found=1
    done
    [ "$found" -eq 1 ] || echo "  nenhum" >&2
}

usage() {
    echo "Uso: $0 AAAA-MM-DD" >&2
    show_available_backups
}

[ "$#" -eq 1 ] || {
    usage
    exit 2
}

restore_date=$1
case "$restore_date" in
    [0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9]) ;;
    *)
        echo "Data de backup inválida: $restore_date" >&2
        usage
        exit 2
        ;;
esac

backup_file="$backup_directory/client-followup-$restore_date.db"
[ -f "$backup_file" ] || {
    echo "Backup não encontrado: $backup_file" >&2
    show_available_backups
    exit 1
}

command -v systemctl >/dev/null 2>&1 || {
    echo "systemctl não está disponível." >&2
    exit 1
}

systemctl --user stop client-followup.service

mkdir -p "$data_directory" "$backup_directory"
chmod 700 "$data_directory" "$backup_directory"

if [ -f "$database_file" ]; then
    cp -- "$database_file" "$pre_restore_backup"
    chmod 600 "$pre_restore_backup"
fi

rm -f "$database_file-wal" "$database_file-shm"
cp -- "$backup_file" "$database_file"
chmod 600 "$database_file"
rm -f "$database_file-wal" "$database_file-shm"

echo "Backup de $restore_date restaurado."
if [ -f "$pre_restore_backup" ]; then
    echo "O banco anterior foi preservado em:"
    echo "$pre_restore_backup"
fi
echo "Abra \"Client Follow-up\" pelo menu de aplicativos."
