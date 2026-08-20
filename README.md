# Client Follow-up

**Português** | [English](README.en.md)

Aplicação local para acompanhamento de pendências de clientes, prazos, histórico e ações de retorno. O projeto foi desenvolvido para um fluxo operacional real, com foco em simplicidade, clareza e confiabilidade.

Funciona localmente no Linux, com dados persistidos em SQLite e interface web servida pela própria aplicação.

## Principais funcionalidades

- cadastro e edição de clientes;
- criação, edição e exclusão de pendências abertas;
- busca incremental de clientes sem distinção de maiúsculas/minúsculas e acentos;
- tratamento explícito de clientes homônimos;
- confirmação controlada para alteração e duplicidade de telefone;
- datas de início e limite, prioridade, descrição, encaminhamento e observações;
- dashboard com indicadores operacionais e alertas de prazo;
- ciclo de estados `PENDING → COMPLETED → PENDING` e `COMPLETED → ARCHIVED`;
- ficha individual do cliente com histórico operacional;
- consulta e filtragem de registros por período, cliente, encaminhamento, prioridade e status;
- impressão de relatórios pelo navegador para exportação em PDF;
- interface responsiva para desktop, tablet e celular;
- backup local automático com pontos de recuperação rotativos do banco SQLite.

Pendências arquivadas deixam o fluxo operacional principal, mas permanecem disponíveis para consulta histórica.

## Tecnologias

- **Go** — servidor HTTP, regras de negócio e acesso a dados;
- **SQLite** — persistência local;
- **HTMX** — interações dinâmicas sem framework frontend;
- **HTML templates** — renderização no servidor;
- **JavaScript** — apenas onde necessário para comportamento de interface;
- **CSS** — layout responsivo e apresentação;
- **GitHub Actions** — validação contínua do projeto.

O HTMX é armazenado localmente no projeto, portanto a aplicação não depende de CDN durante a execução normal.

## Arquitetura

```text
Navegador
   │
   ▼
Aplicação Go
   │
   ├── HTML templates
   ├── HTMX
   └── JavaScript mínimo
   │
   ▼
SQLite
```

A aplicação escuta localmente em `127.0.0.1`, mantendo o fluxo e os dados sob controle do próprio usuário.

## Distribuição Linux

A distribuição homologada tem como alvo **Linux x86-64 / amd64** e é gerada como o bundle compilado `client-followup-linux-amd64.tar.gz`. O usuário final não precisa instalar Go nem o utilitário `sqlite3` para executar a aplicação.

Dependências operacionais esperadas no sistema:

- `systemctl` com suporte a serviços do usuário;
- `curl`;
- `xdg-open`.

### Instalação

Para uma versão publicada em GitHub Releases, baixe e extraia o bundle e execute o instalador a partir da pasta extraída:

```bash
tar -xzf client-followup-linux-amd64.tar.gz
cd client-followup-linux-amd64
./install.sh
```

A instalação ocorre no espaço do próprio usuário e **não utiliza `sudo`**. O aplicativo passa a aparecer como **Client Follow-up** no menu de aplicativos.

O serviço `systemd --user` é iniciado sob demanda quando o aplicativo é aberto. Não há autostart na entrada da sessão.

Arquivos da aplicação são instalados em:

```text
~/.local/share/client-followup/app/
```

O banco de dados fica separado da aplicação:

```text
~/.local/share/client-followup/data/client-followup.db
```

Os backups e pontos de recuperação ficam em:

```text
~/.local/share/client-followup/backups/
```

## Backup e recuperação

O mecanismo utiliza snapshots SQLite completos, não backup incremental.

A janela normal de recuperação mantém espaço limitado:

- **1 baseline diário** — `client-followup-AAAA-MM-DD.db`, representando o estado da primeira abertura do dia;
- **até 3 recovery snapshots rotativos** — `recent-1.db`, `recent-2.db` e `recent-3.db`, preservando estados anteriores às últimas alterações persistidas;
- **1 proteção pré-restore** — `pre-restore.db`, criada ao restaurar um ponto de recuperação para preservar o banco que estava ativo antes da restauração.

`recent-1` representa o estado imediatamente anterior à última alteração persistida, seguido por `recent-2` e `recent-3`.

### Restaurar um ponto de recuperação

Para ver a sintaxe e os pontos disponíveis:

```bash
~/.local/share/client-followup/restore-backup.sh
```

Exemplos de restauração:

```bash
~/.local/share/client-followup/restore-backup.sh recent-1
~/.local/share/client-followup/restore-backup.sh recent-2
~/.local/share/client-followup/restore-backup.sh recent-3
~/.local/share/client-followup/restore-backup.sh daily
~/.local/share/client-followup/restore-backup.sh 2026-08-20
```

O script valida o ponto escolhido, para o serviço, preserva o banco ativo em `pre-restore.db`, restaura o snapshot, limpa os arquivos SQLite WAL/SHM, reinicia o serviço e aguarda o endpoint `/health` responder.

Após uma restauração bem-sucedida, atualize a página com `F5`. O estado anterior à restauração também passa a ficar disponível como `recent-1`, permitindo desfazer imediatamente a própria restauração.

## Desinstalação

Execute:

```bash
~/.local/share/client-followup/uninstall.sh
```

A desinstalação remove o serviço, o launcher, o ícone e os arquivos executáveis da aplicação, mas **preserva o banco de dados e os backups** em:

```text
~/.local/share/client-followup/
```

## Qualidade

O projeto possui testes automatizados para regras de negócio e persistência, além de verificações com `go vet`, detector de corrida, validação de JavaScript e GitHub Actions.

Os principais fluxos também passaram por validação manual no navegador, incluindo cadastro, busca, resolução de homônimos, telefone, ciclo de vida das pendências, sincronização do dashboard, relatórios, responsividade e impressão. A distribuição Linux também foi homologada com instalação user-level, execução pelo launcher, persistência de dados, Backup & Recovery e desinstalação com preservação dos dados.

## Método S.C.A.L.E.

O projeto segue o **Método S.C.A.L.E.**, uma metodologia desenvolvida por mim, baseada em engenharia proporcional: adotar a solução profissional mais simples que resolva integralmente o problema validado, ampliando controles, testes e documentação conforme o risco real.

A implementação privilegia mudanças pequenas, estados explícitos, dependências locais, validação reproduzível, caminhos claros de reversão e aceite humano antes da integração.
