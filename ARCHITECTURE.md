# ARCHITECTURE — Client Follow-up

**Projeto:** `client-followup`  
**Responsável:** Márcio Taranto  
**Usuária:** Roberta Meletti  
**Ambiente-alvo:** Lenovo IdeaPad S145, 8 GB RAM, Zorin OS  
**Stack definida:** Go + HTMX + SQLite + Go Templates + CSS + JavaScript mínimo  
**Execução:** aplicação local, iniciada no login da sessão  
**Data:** 12 de agosto de 2026

---

## 1. Instrução principal ao Codex

Implementar o MVP funcional do `client-followup` usando este documento como especificação de produto e o repositório abaixo como referência técnica para reaproveitamento:

```text
https://github.com/MTaranto/estacionamento-go-htmx
```

Também será fornecida ao Codex a:

```text
Diretriz_Engenharia_Proporcional_v3.md
```

Ela deve ser seguida durante toda a implementação.

### Relação entre os documentos

- `ARCHITECTURE.md` define **o que o produto precisa fazer**.
- `Diretriz_Engenharia_Proporcional_v3.md` define **como a solução deve ser implementada, validada e mantida**.

Aplicar especialmente os princípios de:

- solução profissional mais simples que resolva o problema;
- reaproveitamento do que já funciona;
- complexidade proporcional ao risco;
- legibilidade;
- dependências mínimas e justificadas;
- separação por responsabilidade real;
- testes proporcionais;
- backup e reversibilidade;
- uso de `go test`, `go vet` e demais ferramentas adequadas;
- não introduzir abstrações, frameworks, serviços ou funcionalidades sem requisito atual;
- parar quando os critérios de aceite estiverem atendidos.

O objetivo da primeira passagem não é criar apenas scaffolding. O Codex deve deixar o MVP funcional e pronto para validação e ajuste fino.

---

## 2. Objetivo do sistema

Criar uma aplicação local para Roberta controlar pendências relacionadas às clientes atendidas no trabalho do Mais Magra.

O sistema deve permitir:

- cadastrar clientes;
- cadastrar pendências;
- definir data de início e data limite;
- definir prioridade;
- registrar para quem a pendência deve ser encaminhada;
- registrar observação;
- marcar pendência como realizada;
- reabrir uma pendência finalizada;
- arquivar uma pendência finalizada;
- receber alertas de prazo;
- pesquisar clientes e pendências;
- consultar registros por filtros;
- gerar relatórios na tela;
- exportar relatórios para PDF pela impressão do navegador;
- manter backup local.

---

## 3. Arquitetura

```text
Login no Zorin OS
        │
        ├── inicia client-followup
        │
        └── abre o painel no navegador
                         │
                         ▼
                  Go HTTP Server
                         │
                  HTMX + Templates
                         │
                         ▼
                      SQLite
```

Servidor:

```text
127.0.0.1:8080
```

O sistema deve funcionar sem internet.

O HTMX deve ser armazenado localmente no projeto.

---

## 4. Reaproveitamento do Estacionamento

O novo projeto deve ser criado em repositório próprio:

```text
client-followup
```

Não criar um fork.

Antes de implementar, inspecionar o estado atual de:

```text
MTaranto/estacionamento-go-htmx
```

e reaproveitar código ou padrões genéricos sempre que forem compatíveis.

### Reaproveitar/adaptar

Do backend Go:

- inicialização do servidor HTTP;
- abertura e uso do SQLite com `database/sql`;
- `github.com/mattn/go-sqlite3`;
- criação automática de schema;
- mecanismo simples de evolução de schema preservando dados, quando necessário;
- configuração de arquivos estáticos;
- handlers HTTP;
- queries parametrizadas;
- `INSERT` e `UPDATE`;
- padrão de fragmentos HTMX;
- `HX-Trigger`;
- busca incremental com `LIKE`;
- autofill a partir de registro existente;
- tratamento de datas em `America/Sao_Paulo`;
- tratamento de valores nulos;
- padrão de modal carregado dinamicamente.

Do front-end:

- contêiner de modal;
- fechamento por `Escape`;
- fechamento pelo overlay;
- bloqueio/restauração do scroll de fundo;
- foco automático;
- eventos customizados do HTMX;
- padrões de tabela;
- padrões de botões;
- estrutura responsiva útil.

### Adaptar, não copiar literalmente

Não transportar regras específicas do domínio do estacionamento.

Não reutilizar HTML montado por concatenação quando houver conteúdo livre digitado pela usuária.

Para nome, descrição, observação e `Encaminhar para`, preferir `html/template` para escaping automático.

---

## 5. Organização do projeto

Manter estrutura simples e legível.

Todos os arquivos Go podem permanecer no mesmo `package main`.

Estrutura inicial sugerida:

```text
client-followup/
├── ARCHITECTURE.md
├── README.md
├── go.mod
├── go.sum
├── main.go
├── db.go
├── models.go
├── handlers.go
├── reminders.go
├── backup.go
├── templates/
│   ├── base.html
│   ├── dashboard.html
│   ├── followup-form.html
│   ├── client-detail.html
│   ├── reports.html
│   └── partials/
├── static/
│   ├── css/
│   │   └── style.css
│   ├── js/
│   │   └── app.js
│   └── vendor/
│       └── htmx.min.js
├── scripts/
│   ├── install-user-service.sh
│   ├── uninstall-user-service.sh
│   └── open-when-ready.sh
├── systemd/
│   └── client-followup.service
├── autostart/
│   └── client-followup.desktop
├── data/
│   └── .gitkeep
├── backups/
│   └── .gitkeep
└── .gitignore
```

O Codex pode simplificar essa estrutura se a Diretriz de Engenharia Proporcional indicar que algum arquivo ou separação não agrega valor.

Não criar camadas como repository/service/controller/interfaces apenas por convenção arquitetural.

---

## 6. Modelo de dados

Banco:

```text
data/client-followup.db
```

### 6.1 `clients`

Campos:

```text
id
name
contact
created_at
updated_at
```

Regras:

- `id`: chave primária;
- `name`: obrigatório;
- `contact`: opcional;
- timestamps controlados pela aplicação.

### 6.2 `followups`

Campos:

```text
id
client_id
start_date
due_date
description
forward_to
priority
status
notes
completed_at
archived_at
created_at
updated_at
```

Regras:

- `client_id`: obrigatório;
- `start_date`: padrão = hoje, editável;
- `due_date`: obrigatória;
- `description`: obrigatória;
- `forward_to`: texto livre;
- `priority`: `HIGH`, `MEDIUM`, `LOW`;
- prioridade padrão: `MEDIUM`;
- `status`: `PENDING`, `COMPLETED`, `ARCHIVED`;
- status inicial: `PENDING`;
- `notes`: opcional;
- `completed_at`: preenchido ao finalizar;
- `archived_at`: preenchido ao arquivar.

Não realizar exclusão física de pendências no fluxo normal.

---

## 7. Estados e ações

Internamente:

```text
PENDING
COMPLETED
ARCHIVED
```

Na interface:

```text
Pendente
Finalizado
Arquivado
```

Fluxo:

```text
PENDING
   │
   └── checkbox "Realizado"
            │
            ▼
       COMPLETED
       ├── Reabrir  → PENDING
       └── Arquivar → ARCHIVED
```

### Ao marcar como realizado

- mudar para `COMPLETED`;
- preencher `completed_at`;
- remover dos alertas;
- exibir ações `Reabrir` e `Arquivar`.

### Ao reabrir

- voltar para `PENDING`;
- limpar `completed_at`;
- voltar a participar dos alertas.

### Ao arquivar

- mudar para `ARCHIVED`;
- preencher `archived_at`;
- remover do dashboard;
- remover da ficha operacional da cliente;
- manter disponível em consultas/relatórios.

---

## 8. Prioridades e ordenação

Prioridades:

```text
HIGH   → Alta
MEDIUM → Média
LOW    → Baixa
```

Padrão:

```text
Média
```

A prioridade não substitui o prazo.

Ordenação operacional:

1. atrasadas;
2. prazo mais próximo;
3. prioridade Alta;
4. prioridade Média;
5. prioridade Baixa.

Uma pendência atrasada deve aparecer antes de uma pendência futura, independentemente da prioridade.

---

## 9. Lembretes

Fuso:

```text
America/Sao_Paulo
```

As regras consideram somente a data.

Para pendências `PENDING`:

- prazo amanhã → `Vence amanhã`;
- prazo hoje → `Vence hoje`;
- prazo anterior a hoje → `Atrasada`;
- demais → sem alerta.

Pendências `COMPLETED` e `ARCHIVED` não aparecem nos alertas.

### Popup

Ao abrir o dashboard, se houver alertas, exibir um único modal com todos os itens relevantes.

Cada item deve mostrar:

- cliente;
- descrição;
- data limite;
- prioridade;
- situação do prazo.

Para atrasadas, informar:

```text
Atrasada há N dias
```

Ordenar:

1. atrasadas;
2. vencem hoje;
3. vencem amanhã;
4. prazo;
5. prioridade.

### Interação

O modal deve permitir:

- fechar;
- clicar em uma pendência.

Ao clicar:

1. fechar o modal;
2. abrir a ficha da cliente;
3. mostrar suas pendências atuais;
4. destacar a pendência selecionada.

---

## 10. Telas

### 10.1 Dashboard

Deve conter:

- `Nova pendência`;
- busca;
- filtros;
- indicadores;
- tabela/lista operacional.

Indicadores:

- Atrasadas;
- Vencem hoje;
- Vencem amanhã;
- Pendentes;
- Finalizadas.

Tabela:

- Cliente;
- Descrição;
- Encaminhar para;
- Data limite;
- Prioridade;
- Status;
- Ações.

Arquivadas não aparecem no dashboard.

### 10.2 Nova pendência

Tela explícita de cadastro.

Campos:

```text
Cliente
Contato
Data de início
Data limite
Prioridade
Descrição
Encaminhar para
Observação
```

Comportamento:

- Data de início = hoje por padrão;
- prioridade = Média por padrão;
- Data limite obrigatória;
- Descrição obrigatória.

### 10.3 Cliente

Ao digitar o nome:

- pesquisar clientes existentes;
- mostrar sugestões;
- permitir seleção;
- preencher contato automaticamente.

Se a cliente ainda não existir, permitir cadastrá-la no mesmo fluxo da nova pendência.

### 10.4 Ficha da cliente

Mostrar:

- nome;
- contato;
- edição simples desses dados;
- pendências `PENDING`;
- pendências `COMPLETED`.

Não mostrar `ARCHIVED`.

### 10.5 Consulta / Relatórios

Filtros:

- período;
- cliente;
- Encaminhar para;
- prioridade;
- status;
- atrasado.

Status:

- Pendente;
- Finalizado;
- Arquivado.

Arquivados aparecem quando solicitados pela consulta.

---

## 11. Busca

### Busca de cliente

Pesquisa incremental durante a digitação.

Pesquisar por:

- nome;
- contato, quando houver termo compatível.

Usar query parametrizada e pequeno delay no `hx-trigger`.

### Busca operacional

Pesquisar por:

- cliente;
- descrição;
- Encaminhar para.

SQLite `LIKE` é suficiente.

---

## 12. HTMX e JavaScript

HTMX deve ser o mecanismo principal para:

- salvar pendência;
- atualizar tabela;
- concluir;
- reabrir;
- arquivar;
- buscar cliente;
- abrir ficha da cliente;
- aplicar filtros;
- carregar alertas;
- exibir mensagens.

JavaScript deve permanecer pequeno e ser usado apenas quando necessário para:

- modal;
- `Escape`;
- overlay;
- scroll;
- foco;
- destaque visual;
- impressão.

Não transformar a aplicação em SPA.

---

## 13. Relatórios e PDF

Não adicionar biblioteca de geração de PDF.

Criar versão adequada para impressão usando:

```css
@media print
```

Botão:

```text
Exportar PDF
```

deve abrir a impressão do navegador para uso de:

```text
Salvar como PDF
```

O relatório deve refletir os filtros aplicados e mostrar:

- data de emissão;
- cliente;
- descrição;
- Encaminhar para;
- prazo;
- prioridade;
- status.

Elementos de navegação e botões não devem sair na impressão.

---

## 14. Backup

O banco local deve ter backup simples.

Ao iniciar a aplicação:

- verificar se já existe backup do dia;
- se não existir, criar um.

Diretório:

```text
backups/
```

Nome:

```text
client-followup-YYYY-MM-DD.db
```

Retenção:

```text
14 backups diários
```

Ignorar no Git:

```text
data/*.db
backups/*.db
```

---

## 15. Inicialização no Zorin OS

O sistema deve iniciar quando Roberta entra na sessão.

### Serviço

Criar serviço `systemd --user`.

Responsabilidades:

- iniciar o binário;
- reiniciar em caso de falha simples;
- usar `127.0.0.1:8080`.

### Painel

Criar entrada de autostart para abrir o navegador quando o servidor estiver pronto.

Usar:

```text
scripts/open-when-ready.sh
```

O script deve aguardar `GET /health` responder antes de abrir:

```text
http://127.0.0.1:8080
```

Evitar abrir janelas duplicadas.

Criar scripts de instalação e remoção do serviço.

---

## 16. Privacidade dos dados

Armazenar somente dados necessários ao controle operacional:

- nome;
- contato;
- descrição operacional;
- Encaminhar para;
- observação administrativa.

O sistema não deve ser usado como prontuário.

Evitar registrar informações clínicas ou médicas nas observações.

Nenhum banco real ou backup deve entrar no Git.

---

## 17. Validação

Seguir a `Diretriz_Engenharia_Proporcional_v3.md`.

Como o sistema possui banco de dados, dados pessoais e operações reais, tratar a implementação como risco moderado e validar proporcionalmente.

### Testes automatizados mínimos

Testar regras novas e críticas.

#### Lembretes

- D-1 → Vence amanhã;
- D → Vence hoje;
- data anterior → Atrasada;
- data futura além de D-1 → sem alerta;
- Finalizado → sem alerta;
- Arquivado → sem alerta.

#### Estados

- Pendente → Finalizado;
- Finalizado → Pendente;
- Finalizado → Arquivado;
- impedir arquivamento inválido;
- timestamps coerentes.

#### Ordenação

- atraso tem precedência;
- prazo tem precedência sobre prioridade;
- prioridade desempata situações equivalentes.

#### Banco

- criação inicial;
- cadastro de cliente;
- cadastro de pendência;
- busca parcial;
- atualização de status;
- persistência após reinício.

Usar banco temporário nos testes.

### Ferramentas mínimas

Executar:

```bash
gofmt
go test ./...
go vet ./...
```

Executar também:

```bash
go test -race ./...
```

quando aplicável à implementação.

O Codex deve corrigir falhas encontradas antes de considerar a primeira passagem concluída.

---

## 18. Critérios de aceite

O MVP estará funcional quando Roberta conseguir:

1. fazer login no Zorin e ter o sistema iniciado;
2. ver o painel abrir automaticamente;
3. cadastrar uma nova cliente durante o cadastro de pendência;
4. localizar uma cliente existente pela digitação;
5. ter o contato preenchido automaticamente;
6. criar pendência com data inicial e data limite;
7. definir prioridade Alta, Média ou Baixa;
8. preencher Descrição;
9. preencher Encaminhar para;
10. preencher Observação opcional;
11. visualizar a pendência no dashboard;
12. pesquisar e filtrar pendências;
13. receber alerta de prazo para amanhã;
14. receber alerta no dia do prazo;
15. receber alerta de atraso;
16. clicar no alerta e abrir a cliente correspondente;
17. ver a pendência selecionada destacada;
18. marcar como Realizado;
19. ver o status Finalizado;
20. reabrir;
21. arquivar;
22. ver arquivados somente em consulta;
23. gerar relatório filtrado;
24. salvar relatório como PDF pela impressão;
25. fechar e reabrir o sistema sem perder dados;
26. encontrar o backup diário;
27. utilizar o sistema sem internet.

---

## 19. Entrega esperada do Codex

Na primeira passagem, deixar prontos:

- estrutura de diretórios;
- arquivos Go;
- schema SQLite;
- inicialização e persistência;
- templates;
- CSS funcional;
- JavaScript mínimo;
- HTMX local;
- dashboard;
- cadastro de pendência;
- cadastro/busca/autofill de cliente;
- ficha da cliente;
- prioridades;
- estados;
- conclusão;
- reabertura;
- arquivamento;
- popup de lembretes;
- busca;
- filtros;
- relatórios;
- impressão/PDF;
- backup;
- `/health`;
- systemd user service;
- autostart;
- scripts de instalação/remoção;
- `.gitignore`;
- testes;
- README com execução e instalação.

A fase posterior deve se concentrar em:

- validação funcional;
- correções;
- ergonomia;
- CSS;
- nomenclatura;
- ajuste fino com Roberta.

---

## 20. Regra de conclusão

Não ampliar o produto além deste documento durante a primeira implementação.

Quando os critérios de aceite estiverem satisfeitos e as validações previstas na Diretriz de Engenharia Proporcional estiverem aprovadas, considerar o MVP pronto para uso.
