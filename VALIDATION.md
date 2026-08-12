# Registro de validação — 12/08/2026

## Escopo e risco

- Requisito: MVP local descrito em `ARCHITECTURE.md`.
- Usuária: Roberta Meletti.
- Nível tratado: risco moderado, por persistência e dados pessoais não sensíveis.
- Implementação: Codex, com evidência independente produzida pelas ferramentas Go e requisições HTTP locais.
- Revisão humana: ainda não realizada.

## Evidências executadas

Comandos aprovados nesta passagem:

```bash
gofmt -w *.go
go test -count=1 ./...
go vet ./...
go test -count=1 -race ./...
go build .
sh -n scripts/*.sh
node --check static/js/app.js
git diff --check
```

Resultados observados:

- testes Go: aprovados;
- `go vet`: aprovado;
- detector de corrida: aprovado;
- build: aprovado;
- sintaxe dos scripts e do JavaScript: aprovada;
- whitespace/diff: aprovado;
- banco temporário: schema, CRUD necessário, busca, estados, persistência e backup aprovados;
- fluxo HTTP local: cadastro, dashboard, busca/autofill, alerta D-1, finalizar, reabrir, arquivar e consulta de arquivados aprovados.

## Limitações da validação

- `agent-browser` não está instalado no ambiente, portanto não houve automação visual real, inspeção de console, teste de teclado ou impressão do navegador.
- `systemd-analyze --user verify` leu a unidade, mas não conseguiu acessar o barramento de usuário isolado nem encontrar o binário no caminho final antes da instalação. O script e a unidade precisam ser validados no Zorin com a instalação real.
- O início no login, a ausência de duplicação de janelas, a ergonomia e “Salvar como PDF” exigem teste manual na máquina-alvo.

## Reversão

- O desinstalador remove serviço e executáveis, preservando `data/` e `backups/`.
- A restauração de backup está documentada no `README.md`.
- Não houve commit, instalação no perfil do usuário ou uso de dados reais nesta passagem.
