#!/bin/sh
set -eu

application_directory="$HOME/.local/share/client-followup"
data_directory="$application_directory/data"
backup_directory="$application_directory/backups"
database_file="$data_directory/client-followup.db"
pre_restore_backup="$backup_directory/pre-restore.db"

show_available_backups() {
    echo "Pontos de recuperação disponíveis:" >&2
    found=0
    for baseline in "$backup_directory"/client-followup-[0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9].db; do
        [ -f "$baseline" ] || continue
        name=${baseline##*/}
        date=${name#client-followup-}
        date=${date%.db}
        echo "  daily      início do dia — $date" >&2
        found=1
        break
    done
    if [ -f "$backup_directory/recent-1.db" ]; then
        echo "  recent-1   estado anterior à última alteração" >&2
        found=1
    fi
    if [ -f "$backup_directory/recent-2.db" ]; then
        echo "  recent-2   dois estados atrás" >&2
        found=1
    fi
    if [ -f "$backup_directory/recent-3.db" ]; then
        echo "  recent-3   três estados atrás" >&2
        found=1
    fi
    [ "$found" -eq 1 ] || echo "  nenhum" >&2
}

usage() {
    echo "Uso: $0 [daily | recent-1 | recent-2 | recent-3 | AAAA-MM-DD]" >&2
    show_available_backups
}

[ "$#" -eq 1 ] || {
    usage
    exit 2
}

choice=$1
selected_backup=""

case "$choice" in
    daily)
        for baseline in "$backup_directory"/client-followup-[0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9].db; do
            if [ -f "$baseline" ]; then
                selected_backup="$baseline"
                break
            fi
        done
        if [ -z "$selected_backup" ]; then
            echo "Nenhum baseline diário encontrado." >&2
            show_available_backups
            exit 1
        fi
        ;;
    recent-1|recent-2|recent-3)
        selected_backup="$backup_directory/$choice.db"
        if [ ! -f "$selected_backup" ]; then
            echo "Ponto de recuperação \"$choice\" não encontrado." >&2
            show_available_backups
            exit 1
        fi
        ;;
    [0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9])
        selected_backup="$backup_directory/client-followup-$choice.db"
        if [ ! -f "$selected_backup" ]; then
            echo "Backup da data \"$choice\" não encontrado." >&2
            show_available_backups
            exit 1
        fi
        ;;
    *)
        echo "Opção de recuperação inválida: $choice" >&2
        usage
        exit 2
        ;;
esac

for command_name in systemctl curl; do
    command -v "$command_name" >/dev/null 2>&1 || {
        echo "Dependência necessária não encontrada: $command_name" >&2
        exit 1
    }
done

if ! systemctl --user stop client-followup.service; then
    echo "Erro: não foi possível parar client-followup.service." >&2
    exit 1
fi

mkdir -p "$data_directory" "$backup_directory"
chmod 700 "$data_directory" "$backup_directory"

if [ -f "$database_file" ]; then
    cp -- "$database_file" "$pre_restore_backup"
    chmod 600 "$pre_restore_backup"
fi

rm -f "$database_file-wal" "$database_file-shm"
cp -- "$selected_backup" "$database_file"
chmod 600 "$database_file"
rm -f "$database_file-wal" "$database_file-shm"

rm -f "$backup_directory"/recent-[123].db
if [ -f "$pre_restore_backup" ]; then
    cp -- "$pre_restore_backup" "$backup_directory/recent-1.db"
    chmod 600 "$backup_directory/recent-1.db"
fi

if ! systemctl --user start client-followup.service; then
    echo "Aviso: o banco foi restaurado, mas falhou ao iniciar client-followup.service." >&2
    if [ -f "$pre_restore_backup" ]; then
        echo "O banco anterior foi preservado em: $pre_restore_backup" >&2
    fi
    exit 1
fi

attempt=0
while [ "$attempt" -lt 30 ]; do
    if curl --fail --silent --max-time 2 http://127.0.0.1:8080/health >/dev/null; then
        echo "Ponto de recuperação \"$choice\" restaurado com sucesso."
        if [ -f "$pre_restore_backup" ]; then
            echo "O banco anterior foi preservado em:"
            echo "  $pre_restore_backup"
            echo "Ele também está disponível como recent-1 para desfazer a restauração."
        fi
        echo "Atualize a página (F5) no navegador."
        exit 0
    fi
    attempt=$((attempt + 1))
    sleep 1
done

echo "Aviso: o banco foi restaurado, mas o serviço não respondeu em http://127.0.0.1:8080/health." >&2
if [ -f "$pre_restore_backup" ]; then
    echo "O banco anterior foi preservado em: $pre_restore_backup" >&2
fi
exit 1
