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
- backup local automático do banco SQLite.

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

## Instalação

A distribuição final para Linux será feita como **binário compilado**, de forma que o usuário final não precise instalar Go nem o utilitário `sqlite3` para utilizar a aplicação.

A camada de instalação está em preparação e será concluída antes da distribuição final. Ela prevê:

- launcher Linux por arquivo `.desktop`, com nome e ícone próprios no menu de aplicativos e/ou Área de Trabalho;
- execução normal da aplicação pelo launcher;
- opção de configuração para inicialização automática na sessão do usuário.

As instruções definitivas de instalação serão adicionadas aqui após essa etapa ser concluída e validada.

## Qualidade

O projeto possui testes automatizados para regras de negócio e persistência, além de verificações com `go vet`, detector de corrida, validação de JavaScript e GitHub Actions.

Os principais fluxos também passaram por validação manual no navegador, incluindo cadastro, busca, resolução de homônimos, telefone, ciclo de vida das pendências, sincronização do dashboard, relatórios, responsividade e impressão.

## Método S.C.A.L.E.

O projeto segue o **Método S.C.A.L.E.**, uma metodologia desenvolvida por mim, baseada em engenharia proporcional: adotar a solução profissional mais simples que resolva integralmente o problema validado, ampliando controles, testes e documentação conforme o risco real.

A implementação privilegia mudanças pequenas, estados explícitos, dependências locais, validação reproduzível, caminhos claros de reversão e aceite humano antes da integração.
