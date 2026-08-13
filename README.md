# Client Follow-up

Aplicação local para controle das pendências de clientes da Roberta. O MVP usa Go, HTMX, templates HTML e SQLite, funciona sem internet e mantém os textos livres protegidos pelo escaping automático de `html/template`.

## O que está implementado

- cadastro de cliente no mesmo fluxo da nova pendência;
- busca incremental por nome e preenchimento automático do telefone;
- datas de início e limite, prioridade, descrição, encaminhamento e observação;
- painel com indicadores, busca e filtros operacionais;
- ficha da cliente com edição de nome/telefone e pendências não arquivadas;
- fluxo `PENDING → COMPLETED → PENDING` e `COMPLETED → ARCHIVED`;
- alertas únicos para atrasadas, vencimento hoje e amanhã, com acesso e destaque da pendência;
- consulta histórica por período, cliente, encaminhamento, prioridade, status e atraso;
- impressão do relatório para “Salvar como PDF” no navegador;
- backup SQLite diário com retenção dos 14 backups mais recentes;
- endpoint `GET /health`, serviço `systemd --user` e abertura do navegador no login;
- HTMX 2.0.8 armazenado em `static/vendor`, sem dependência de internet em execução.

Arquivadas não aparecem no painel ou na ficha operacional. Elas permanecem no banco e podem ser consultadas nos relatórios. Não existe exclusão física no fluxo normal.

## Mapa do código

- `main.go`: inicialização, fuso, backup e ciclo do servidor HTTP;
- `db.go`: schema versionado, persistência, consultas e transições de estado;
- `handlers.go`: rotas HTTP, validação de formulários e renderização;
- `models.go`: tipos e valores permitidos;
- `reminders.go`: regras e ordenação dos alertas;
- `backup.go`: backup diário e retenção;
- `templates/`: páginas e fragmentos HTMX com escaping automático;
- `static/`: CSS, JavaScript mínimo e HTMX local;
- `scripts/`, `systemd/` e `autostart/`: instalação e início no login.

## Reaproveitamento do Estacionamento

O estado atual de `MTaranto/estacionamento-go-htmx` foi inspecionado antes da implementação. Foram adaptados os padrões compatíveis de servidor `net/http`, SQLite com `database/sql` e `go-sqlite3`, criação/evolução automática do schema, consultas parametrizadas, fragmentos e eventos HTMX, busca incremental, autofill, fuso `America/Sao_Paulo`, arquivos estáticos e interação de modal (Escape, overlay, scroll e foco).

Não foram transportadas regras do estacionamento. A geração de HTML por concatenação também não foi copiada: nome, descrição, observação e encaminhamento passam por `html/template`.

## Executar durante o desenvolvimento

Requisitos: Go 1.26 ou compatível com o `go.mod`, GCC e cabeçalhos necessários ao `go-sqlite3`.

```bash
go run .
```

Abra `http://127.0.0.1:8080`. O banco será criado em `data/client-followup.db` e o backup diário em `backups/`.

Opções disponíveis:

```bash
go run . -addr 127.0.0.1:8081 -db /caminho/dados.db -backups /caminho/backups
```

## Validação técnica

```bash
gofmt -w *.go
go test ./...
go vet ./...
go test -race ./...
```

Os testes usam bancos temporários e cobrem lembretes, ordenação, criação do schema, cadastro, busca parcial, transições/timestamps, persistência após reabertura, backup/restauração, retenção e escaping de conteúdo digitado.

O workflow `.github/workflows/ci.yml` repete formatação, testes, `go vet` e detector de corrida no GitHub Actions quando o repositório for publicado.

## Instalar no Zorin OS

Execute no diretório do projeto:

```bash
./scripts/install-user-service.sh
```

O instalador compila e copia a aplicação para `~/.local/share/client-followup`, habilita o serviço de usuário e instala a entrada de autostart. Consulte o estado com:

```bash
systemctl --user status client-followup.service
journalctl --user -u client-followup.service
```

Para remover o serviço e os arquivos executáveis:

```bash
./scripts/uninstall-user-service.sh
```

O desinstalador preserva deliberadamente `data/` e `backups/`.

## Backup, restauração e reversão

Na inicialização, o SQLite cria no máximo um backup por dia pelo comando consistente `VACUUM INTO`. Os arquivos seguem o nome `client-followup-AAAA-MM-DD.db`.

Para restaurar, pare o serviço, guarde uma cópia do banco atual e copie o backup escolhido para o caminho do banco:

```bash
systemctl --user stop client-followup.service
cp ~/.local/share/client-followup/data/client-followup.db ~/client-followup-before-restore.db
cp ~/.local/share/client-followup/backups/client-followup-AAAA-MM-DD.db ~/.local/share/client-followup/data/client-followup.db
systemctl --user start client-followup.service
```

Se existirem arquivos `client-followup.db-wal` ou `client-followup.db-shm` após uma interrupção anormal, preserve-os junto com o banco atual antes de restaurar e solicite revisão técnica.

## Privacidade e limites

O sistema é local, sem autenticação, porque escuta exclusivamente `127.0.0.1`. Qualquer pessoa com acesso à sessão ou aos arquivos do usuário poderá acessar os dados. Use apenas nome, telefone e informações administrativas necessárias; o sistema não é prontuário e não deve receber dados clínicos ou médicos.

Esta primeira passagem não inclui sincronização, múltiplos usuários, login, envio de notificações, geração própria de PDF ou integração externa. A validação funcional final no Lenovo/Zorin e a revisão humana do código continuam necessárias antes do uso real.

## Registro de engenharia

- **Problema e usuária:** controle local de pendências operacionais para Roberta.
- **Escopo:** somente os itens e critérios de aceite do `ARCHITECTURE.md`.
- **Risco:** moderado por persistência e dados pessoais não sensíveis.
- **Dependência externa:** apenas `github.com/mattn/go-sqlite3`; HTMX é um arquivo estático local.
- **Efeitos:** gravação do banco, backups diários e atualização de estado sem exclusão física.
- **Reversão:** desinstalador preserva dados; restauração documentada acima; alterações de código podem ser revertidas pelo Git após um commit aprovado.
- **Implementação:** assistida por Codex; testes automatizados e ferramentas Go fornecem evidência independente, mas não equivalem a revisão humana.
