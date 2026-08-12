# DIRETRIZ DE ENGENHARIA PROPORCIONAL

**Versão:** 3.1 — agosto de 2026  
**Aplicação:** todos os projetos conduzidos por Márcio Taranto com apoio de ferramentas de IA  
**Substitui:** versão 3.0 — agosto de 2026

---

## 1. Princípio central

Construir a solução profissional mais simples que resolva integralmente o problema validado, com complexidade, ferramentas e controles proporcionais ao risco, sem antecipar necessidades que ainda não existem.

A velocidade da IA não pode produzir mais código do que o processo consegue compreender em nível suficiente, testar, documentar, reverter e manter sob controle.

Simplicidade não significa ausência de rigor, de ferramentas profissionais ou de aprendizagem. Significa adotar somente o que possui função atual, benefício demonstrável e custo de manutenção aceitável.

---

## 2. Objetivo duplo: produto e desenvolvimento profissional

Cada projeto pode atender simultaneamente a dois objetivos:

1. entregar uma solução adequada ao problema real;
2. ampliar a autonomia, o repertório técnico e a empregabilidade de Márcio.

O objetivo pedagógico não autoriza hiperdimensionar o produto, atrasar uma entrega sem necessidade ou transformar todo projeto em laboratório.

Quando uma ferramenta ou prática profissional puder ser aprendida sem comprometer o objetivo principal, ela deve ser considerada.

Quando a introdução dessa ferramenta ameaçar prazo, estabilidade ou clareza, deve ser movida para uma etapa posterior, branch própria ou projeto de estudo relacionado.

---

## 3. Premissa sobre o nível atual de autonomia

Márcio define problemas, requisitos, regras de negócio, limites, critérios de aceite e validações funcionais.

A implementação e a análise detalhada de código ainda dependem intensamente de ferramentas de IA e, em situações sensíveis, de revisão técnica adicional.

Consequências:

- comentários no código não comprovam compreensão;
- concordância entre duas IAs não comprova correção;
- validação funcional não equivale a revisão de código;
- nenhum projeto deve ser apresentado como revisado autonomamente quando isso não ocorreu;
- o rigor da validação deve aumentar conforme o impacto de uma falha;
- a aprendizagem deve reduzir gradualmente a dependência operacional, sem abandonar o uso profissional de IA.

---

## 4. Legibilidade e compreensão humana

Todo código deve ser escrito para ser compreendido e mantido por pessoas, inclusive pelo próprio Márcio no futuro.

A legibilidade humana é requisito de engenharia, não acabamento opcional.

### 4.1 Regras de legibilidade

Preferir:

- nomes claros e específicos;
- funções com responsabilidade identificável;
- fluxo de controle explícito;
- módulos separados por responsabilidade real;
- estruturas previsíveis;
- tratamento de erros visível;
- mensagens de erro úteis;
- consistência entre arquivos;
- testes que funcionem também como exemplos de uso;
- documentação proporcional à complexidade.

Evitar:

- soluções excessivamente engenhosas;
- abreviações obscuras;
- funções que escondem muitos efeitos;
- aninhamento profundo sem necessidade;
- abstrações prematuras;
- duplicação disfarçada por metaprogramação difícil de ler;
- comentários que apenas repetem a sintaxe;
- alterações enormes que dificultem a revisão.

### 4.2 Comentários

Comentários devem explicar:

- intenção;
- restrição;
- contexto;
- decisão não óbvia;
- risco;
- motivo de uma solução aparentemente incomum.

Comentários não devem substituir:

- bons nomes;
- estrutura clara;
- testes;
- documentação;
- compreensão do fluxo.

### 4.3 Critério de aceitação humana

Antes de aceitar uma mudança relevante, Márcio deve conseguir, com apoio da documentação:

- localizar o fluxo principal;
- identificar entrada e saída;
- explicar o objetivo das funções centrais;
- reconhecer os efeitos produzidos;
- executar o teste;
- localizar a forma de reversão;
- registrar o que ainda não compreende.

Não é necessário memorizar toda a implementação. É necessário possuir um mapa mental suficiente para trabalhar com segurança proporcional ao risco.

---

## 5. Competência profissional em duas frentes

A preparação para o mercado deve desenvolver duas competências complementares.

### 5.1 Fundamentos e ferramentas tradicionais

Márcio deve aprender e utilizar, quando adequados:

- Git e GitHub;
- terminal;
- depuradores;
- DevTools do navegador;
- linters;
- formatadores;
- validadores;
- frameworks de teste;
- análise estática;
- testes automatizados;
- integração contínua;
- documentação técnica;
- revisão de diff;
- ferramentas próprias da linguagem e do ecossistema.

### 5.2 Desenvolvimento assistido por IA

Também deve aprender e demonstrar:

- formulação de requisitos para IA;
- criação de prompts técnicos claros;
- decomposição de tarefas;
- revisão crítica de código gerado;
- uso de IA para investigação e depuração;
- comparação entre soluções;
- geração e revisão de testes;
- documentação assistida;
- análise cruzada entre ferramentas;
- reconhecimento de limitações, alucinações e riscos;
- integração responsável da IA ao fluxo de desenvolvimento.

Ferramentas tradicionais não substituem IA. IA não substitui ferramentas tradicionais.

A empregabilidade atual exige capacidade de trabalhar com ambas.

---

## 6. Regra contra automações descartáveis e opacas

Não utilizar pequenos scripts improvisados apenas para copiar, colar e executar quando existir ferramenta profissional consolidada que:

- resolva o mesmo problema;
- seja relevante para o mercado;
- produza resultado reproduzível;
- possa permanecer integrada ao projeto;
- favoreça aprendizagem transferível.

Exemplo de prática a evitar:

- gerar um script Python descartável para verificar HTML, links ou acessibilidade quando o objetivo pedagógico e profissional seria melhor atendido por ferramentas como HTML Validate, Cypress, Playwright, axe, Lighthouse ou linters adequados.

### 6.1 Quando scripts próprios são aceitáveis

Um script próprio pode ser usado quando:

- não existe ferramenta adequada;
- a automação é específica do domínio;
- o script será versionado;
- possui objetivo recorrente;
- é legível;
- está documentado;
- pode ser testado;
- sua manutenção é proporcional;
- Márcio consegue explicar sua função;
- a criação do script desenvolve competência relevante.

Scripts temporários de diagnóstico podem ser usados em investigação técnica quando forem a forma mais simples de obter evidência, mas devem ser explicados e não apresentados como substitutos de uma prática profissional permanente.

---

## 7. Regras essenciais

- Problema antes da tecnologia.
- Toda complexidade precisa atender a um requisito atual, reduzir um risco concreto ou cumprir objetivo pedagógico aprovado.
- Funcionalidades hipotéticas permanecem no backlog.
- Legibilidade vence soluções engenhosas e difíceis de manter.
- Separar código por responsabilidade real, não por aparência de arquitetura.
- Dependências devem ser mínimas e justificadas.
- Trabalhar com alterações pequenas, revisáveis e reversíveis.
- README e portfólio descrevem somente o que está implementado, testado ou observado.
- A IA pode propor e implementar; Márcio controla requisitos, escopo e validação funcional.
- Ferramentas independentes da IA devem produzir evidências sempre que forem adequadas.
- A validação detalhada pode exigir outra IA, ferramentas automáticas ou revisão humana.
- Quando os critérios de aceite forem atendidos, o desenvolvimento para.
- Aprendizagem deve ser planejada, não apenas presumida pela exposição a código gerado.

---

## 8. Portão de complexidade

Antes de acrescentar camada, dependência, serviço, banco de dados, painel, login, integração, framework, ferramenta ou abstração, responder:

1. Qual requisito atual isso atende?
2. Qual risco reduz?
3. Qual objetivo pedagógico ou profissional atende?
4. O que falha, fica inseguro ou deixa de ser aprendido sem isso?
5. Existe alternativa mais simples?
6. Quem utilizará agora e com que frequência?
7. Qual custo de construção e manutenção?
8. Quais riscos novos serão criados?
9. Como essa parte será testada?
10. Como será desfeita ou revertida?
11. A mudança aumenta ou reduz a dependência de IA?
12. O nível de revisão previsto é suficiente?
13. A introdução agora prejudica uma entrega que deveria ser concluída primeiro?

Se a necessidade, o benefício ou a validação não puderem ser demonstrados, o componente não entra naquele momento.

---

## 9. Portão de aprendizagem e empregabilidade

Antes de introduzir uma atividade com finalidade didática, responder:

1. Qual competência será desenvolvida?
2. Essa competência aparece em projetos reais ou no mercado?
3. O aprendizado é transferível para outros projetos?
4. Márcio participará ativamente ou apenas copiará?
5. Haverá prática, falha, depuração e repetição?
6. Será possível demonstrar o resultado em portfólio ou entrevista?
7. A ferramenta permanecerá útil no projeto?
8. Existe momento melhor para aprender sem bloquear a entrega?

Uma atividade não deve ser considerada pedagógica apenas porque contém código.

---

## 10. Convenções de código

- Funções, métodos, tipos, variáveis, constantes, arquivos e módulos em inglês.
- Comentários técnicos em inglês.
- Interface no idioma do público-alvo.
- Commits técnicos preferencialmente em inglês e no padrão Conventional Commits.
- Documentação pública em português e inglês quando o projeto for usado no mercado internacional.
- Nomes claros têm prioridade sobre abreviações.
- APIs internas devem ser pequenas e previsíveis.
- Funções devem expor claramente entrada, saída e efeitos.
- Código repetido pode ser extraído quando a repetição for real e a abstração melhorar a leitura.
- Formatação deve ser automatizada por ferramenta consolidada quando o ecossistema oferecer solução adequada.
- Regras de lint devem ser proporcionais e compreensíveis.

---

## 11. Unidade mínima de mudança

Cada mudança deve ser pequena o suficiente para:

- possuir objetivo único;
- permitir comparação clara do antes e do depois;
- ter procedimento de teste definido;
- ser revertida sem reconstruir o projeto;
- permitir análise por pessoa, ferramenta ou IA;
- evitar reescrita completa sem justificativa;
- produzir diff legível;
- facilitar aprendizagem do fluxo alterado.

Reescritas amplas exigem:

- motivo registrado;
- backup ou branch;
- comparação funcional;
- validação antes e depois;
- revisão proporcional ao risco.

---

## 12. Validação mínima antes de aceitar uma mudança

Registrar:

- problema ou requisito;
- objetivo pedagógico, quando houver;
- comportamento esperado;
- arquivos alterados;
- entradas e saídas principais;
- efeitos colaterais;
- procedimento de teste;
- ferramenta utilizada;
- resultado observado;
- riscos conhecidos;
- limitações;
- forma de reversão;
- responsável pela implementação;
- responsável ou ferramenta de análise adicional.

Comentários e explicações produzidos pela própria IA não substituem testes.

---

## 13. Evidência independente da IA

Sempre que houver ferramenta consolidada e proporcional, utilizar evidência produzida independentemente da resposta da IA.

Exemplos:

### Web

- HTML Validate ou validador equivalente;
- ESLint;
- Stylelint;
- Cypress;
- Playwright;
- axe;
- Lighthouse;
- DevTools;
- verificadores de links;
- testes manuais por teclado.

### JavaScript e TypeScript

- testes unitários e de integração;
- ESLint;
- TypeScript;
- Vitest, Jest ou ferramenta equivalente;
- Cypress ou Playwright para E2E.

### Go

- `go test`;
- `go vet`;
- `go test -race`;
- linters e análise estática adequados;
- benchmarks quando houver requisito de desempenho.

### Git e integração

- revisão de diff;
- pull requests;
- checks;
- GitHub Actions ou CI equivalente;
- proteção de branch quando proporcional.

A lista é orientativa. A ferramenta deve ser adequada ao projeto, não adotada apenas por popularidade.

---

## 14. Níveis de risco e controles

### Nível 1 — baixo risco

Exemplos:

- landing page estática;
- alteração visual;
- texto;
- layout;
- responsividade;
- projeto sem dados sensíveis ou operações destrutivas.

Controles mínimos:

- escopo e critérios de aceite;
- teste manual nos navegadores e tamanhos relevantes;
- validação de links e acessibilidade essencial;
- diff pequeno;
- rollback simples;
- revisão adicional quando a alteração for significativa;
- ferramentas automatizadas quando agregarem qualidade ou aprendizagem sem atrasar indevidamente a entrega.

### Nível 2 — risco moderado

Exemplos:

- banco de dados;
- regras financeiras ou operacionais;
- atualização de registros;
- integrações externas;
- dados pessoais não sensíveis;
- automações com efeitos reais.

Controles mínimos:

- todos os controles do nível 1;
- testes automatizados para regras críticas;
- análise por segunda ferramenta com contexto independente;
- validação de erros e estados excepcionais;
- ambiente de teste;
- backup;
- registro de limitações;
- revisão humana quando a falha puder causar prejuízo relevante.

### Nível 3 — alto risco

Exemplos:

- leitura ou escrita de arquivos locais;
- exclusão ou substituição de dados;
- autenticação;
- autorização;
- pagamentos;
- dados sensíveis;
- execução de comandos;
- operações irreversíveis.

Controles mínimos:

- todos os controles anteriores;
- testes automatizados positivos e negativos;
- análise estática;
- ferramentas de segurança;
- ambiente isolado;
- backup e rollback comprovados;
- confirmação humana explícita;
- análise cruzada;
- revisão humana experiente antes de uso real, quando viável;
- proibição de merge diante de divergência crítica não resolvida.

---

## 15. Protocolo de análise cruzada entre IAs

Quando uma IA implementar e outra analisar:

1. fornecer requisitos, código ou diff e comportamento observado;
2. solicitar problemas concretos, impacto, localização e reprodução;
3. não escolher uma resposta pela confiança do tom;
4. transformar divergências em testes executáveis;
5. usar ferramentas tradicionais para produzir evidência independente;
6. registrar qual hipótese foi confirmada;
7. bloquear mudança quando houver risco crítico não resolvido;
8. buscar revisão humana quando necessário.

Consenso entre IAs é sinal de apoio, não prova.

A revisão técnica independente por outra IA, pelo Codex ou por ferramenta equivalente complementa, mas não substitui, a compreensão progressiva do código por Márcio. Sua função é ampliar a capacidade de detectar erros, riscos, complexidade desnecessária e decisões questionáveis enquanto a autonomia técnica ainda está em desenvolvimento.

---

## 16. Limite mínimo de compreensão

Antes de aceitar uma mudança, Márcio deve conseguir identificar:

- problema resolvido;
- arquivos alterados;
- entrada que inicia o fluxo;
- resultado esperado;
- dados ou recursos modificados;
- ferramentas usadas;
- forma de teste;
- forma de reversão;
- limitações;
- partes que ainda dependem de explicação adicional.

Quanto maior o risco, maior deve ser a compreensão e a revisão independente.

---

## 17. Condições obrigatórias de parada

Interromper a implementação quando:

- o volume da mudança impedir revisão clara;
- não existir teste reproduzível;
- não houver forma segura de reversão;
- houver divergência crítica não resolvida;
- a IA adicionar funcionalidade fora do escopo;
- uma dependência não possuir justificativa;
- a solução manipular dados sem controles proporcionais;
- Márcio não conseguir identificar entradas, saídas e efeitos principais;
- a pressão por entrega exigir aceitar mudança não validada;
- a atividade pedagógica estiver bloqueando uma entrega já pronta sem benefício proporcional;
- a ferramenta estiver sendo usada apenas para ornamentar o portfólio;
- o processo estiver acumulando automações que ninguém compreende.

---

## 18. Engenharia proporcional por tipo de projeto

### Landing pages

Usar:

- HTML semântico;
- CSS organizado;
- JavaScript mínimo;
- SEO;
- acessibilidade;
- validação manual;
- ferramentas de qualidade proporcionais;
- testes E2E quando houver comportamentos relevantes, manutenção futura ou objetivo pedagógico aprovado.

Não incluir banco, login, painel, agenda própria, CRM ou automação complexa sem necessidade.

### Aplicações web

Usar:

- lint;
- testes unitários para regras;
- testes de integração;
- E2E para fluxos críticos;
- CI;
- documentação de execução.

Não buscar cobertura numérica sem valor real.

### Soluções pontuais com dados

Corrigir falhas evidentes, aplicar permissões mínimas, documentar o estado real e encerrar quando o problema estiver resolvido.

### Aplicações com regras de negócio

Testar cálculos, estados, erros e operações críticas.

### Gem Bridge e Gem Extension

Aplicar nível 3.

No Gem Bridge:

- testes Go;
- análise estática;
- `go vet`;
- `go test -race` quando aplicável;
- testes positivos e negativos de arquivos e protocolo;
- ambiente isolado;
- rollback.

No Gem Extension:

- testes adequados à extensão;
- automação de navegador quando útil;
- validação de permissões;
- testes de integração com o host;
- Playwright ou ferramenta equivalente quando houver justificativa.

Nenhuma nova funcionalidade deve entrar antes da validação completa do marco atual.

---

## 19. Desenvolvimento assistido por IA

A IA deve ser utilizada como ferramenta profissional de:

- planejamento;
- implementação;
- revisão;
- explicação;
- depuração;
- geração de hipóteses;
- criação e revisão de testes;
- documentação;
- análise comparativa;
- organização de trabalho.

Regras:

- pedir primeiro a solução mínima;
- exigir justificativa para dependências e abstrações;
- preferir diffs pequenos;
- separar requisito, recomendação, decisão e item adiado;
- usar testes como evidência principal;
- registrar qual ferramenta realizou cada papel;
- não apresentar análise de IA como revisão humana;
- não aceitar código crítico apenas porque funciona no cenário principal;
- estudar o fluxo interno gradualmente;
- usar IA para acelerar aprendizagem, não para ocultar lacunas;
- aprender também a revisar e dirigir código gerado;
- sempre que surgir uma construção de linguagem, padrão ou decisão técnica ainda não familiar a Márcio, a IA deve explicar como ler a sintaxe, o que ela faz, por que foi escolhida e, quando relevante, qual seria a alternativa mais simples;
- essas explicações devem partir do código real em desenvolvimento e ser proporcionais à necessidade, sem transformar cada implementação em uma aula extensa ou bloquear a entrega.

---

## 20. Método de aprendizagem técnica

Para cada nova ferramenta ou conceito:

1. entender o problema que resolve;
2. executar manualmente o comportamento;
3. estudar o comando ou API principal;
4. aplicar em caso pequeno;
5. provocar falha controlada;
6. interpretar o erro;
7. corrigir;
8. repetir em cenário diferente;
9. documentar;
10. explicar com palavras próprias.

A aprendizagem deve produzir competência transferível, não apenas resultado momentâneo.

---

## 21. Definição de pronto

Um projeto está pronto quando:

- resolve o problema combinado;
- atende aos critérios de aceite;
- possui validação proporcional;
- tem procedimento de execução e reversão;
- documenta estado real e limitações;
- não contém componentes sem função;
- não depende de divergência crítica;
- pode ser retomado sem a IA original;
- o código é legível por pessoas;
- as ferramentas permanentes estão documentadas;
- as alegações de aprendizagem e revisão são verdadeiras.

---

## 22. Registro mínimo por projeto

### Antes do início

- problema;
- usuário;
- resultado esperado;
- escopo incluído;
- escopo excluído;
- nível de risco;
- critérios de aceite;
- objetivos pedagógicos aprovados;
- ferramentas previstas.

### Durante

- decisões;
- ferramentas utilizadas;
- testes;
- riscos;
- itens adiados;
- participação da IA;
- evidências independentes;
- aprendizados reais;
- pontos ainda não compreendidos.

### Ao concluir

- resultado funcional;
- limitações;
- procedimento de execução;
- rollback;
- validações realizadas;
- ferramentas incorporadas;
- competências desenvolvidas;
- próximos passos somente quando houver necessidade real.

---

## 23. Síntese

**Profissionalismo obrigatório. Complexidade somente quando necessária. Código legível por pessoas. Validação proporcional ao risco. Ferramentas tradicionais e IA usadas em conjunto. Aprendizagem comprovada por prática, compreensão e evidência — não pela simples cópia de código.**