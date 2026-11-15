# Benchmark do aPRova

O benchmark testa a tese do aPRova: um agente de código resolve uma tarefa, a
solução vira um PR de verdade, e a decisão do aPRova (baixo, médio ou alto)
segura as soluções quebradas sem bloquear as corretas. O gabarito vem de fora
do aPRova: são os testes originais de cada exercício, rodados por um oráculo
separado.

É um estudo-piloto. Com 27 tarefas, nenhum número tem significância
estatística. Por isso todo número sai como "N de M", nunca só como
porcentagem.

## Regras de honestidade

- Todo número sai com o tamanho da amostra.
- Nenhum número é ajustado, refeito até dar certo ou descartado sem registro.
  Uma tarefa que falha por infraestrutura é registrada como tal e fica fora
  da comparação, com o motivo na tabela.
- Resultado ruim é relatado como resultado.
- Não comparamos com o Martian Code Review Bench nem com o F1 de outros
  trabalhos: a suíte, o agente e a métrica são outros.

## Desenho

### Suíte

A suíte tem 27 tarefas, todas registradas em `bench/suite/` antes de qualquer
rodada (veja o [LEIAME da suíte](../bench/suite/LEIAME.md)):

- **Grupo A (correção):** 21 exercícios do polyglot do Aider, 7 de Go, 7 de
  Python e 7 de Java. Foram sorteados com a semente 20261114 a partir do
  commit `7e0611e77b54`. Os testes originais de cada exercício são o oráculo.
- **Grupo B (risco):** 6 tarefas sintéticas, uma por regra do gate que não é
  falha de teste. Cada uma tem o nível e o motivo esperados escritos antes de
  rodar.

### Agente

O agente é o Claude Code em modo headless (`claude -p`), fixado em
`claude-sonnet-5` e usando a assinatura do plano, sem `ANTHROPIC_API_KEY`. A
cada tarefa conferimos, no `modelUsage`, que a saída veio do
`claude-sonnet-5`. O `modelUsage` também lista um modelo auxiliar interno
(`claude-haiku-4-5`) com poucos tokens. Os dois ficam registrados.

O agente recebe só o enunciado, os arquivos a implementar, os testes e o
comando para rodá-los. A instrução não menciona o aPRova, o gate nem o risco.

### Isolamento

O agente roda na máquina do desenvolvedor, e por isso o isolamento é
pré-condição de cada rodada, não uma hipótese:

- `--permission-mode acceptEdits` com uma lista fechada de comandos:
  `go test`, `go build`, `pytest` e `mvn`. Não há Bash genérico, curl, git
  nem rm.
- O sandbox do Claude Code (Seatbelt, no macOS) fica ligado com
  `allowUnsandboxedCommands: false` e com leitura negada no HOME.
- O ambiente é limpo (`env -i`). Os caches (Go, Maven, pytest) e o TMPDIR
  ficam dentro do diretório da tarefa. O Maven roda offline, sobre um
  repositório aquecido fora do agente.

Antes de cada rodada, uma sondagem faz cinco tentativas de fuga:

1. escrever fora da tarefa;
2. escrever pelo terminal;
3. escrever por redirecionamento;
4. escrever por código que roda nos testes;
5. ler um canário em `~/.ssh` e no `.env` do aPRova.

As cinco precisam ser barradas, e controles positivos dentro da tarefa
precisam funcionar. Sem os controles, a sondagem não prova nada e conta como
inconclusiva.

Antes de cada push, o diff passa por uma conferência:

- só pode ter arquivos da tarefa, sem caches;
- nenhum valor do `.env` do aPRova pode aparecer nele.

A comparação dos valores é feita sem imprimi-los. Se a conferência falhar, a
tarefa é registrada como "diff recusado antes do push".

### Coleta

Cada solução vira um PR, aberto da branch `solucao/<rodada>-<id>` para a
`base/<rodada>-<id>`, num repositório público por linguagem
(`Azulpasta/aprova-bench-go`, `-python` e `-java`). Um ruleset exige o Check Run "aPRova" nas branches `base/*`.

A decisão é lida do recibo do aPRova, e o estado do task_run vem do banco.
Depois, o oráculo restaura os testes originais sobre a solução e os roda no
executor local do sandbox, sem rede.

### Métricas

O grupo A cruza o oráculo com a decisão:

| | aPRova liberou (baixo/médio) | aPRova bloqueou (alto) |
| --- | --- | --- |
| oráculo passou | correto, sem trabalho humano | bloqueio desnecessário |
| oráculo falhou | quebrado que passaria | quebrado contido |

Duas comparações saem dessa tabela:

- **Sem a camada:** todas as soluções quebradas chegariam à main.
- **Com a camada:** só as quebradas que o aPRova liberou chegariam.

O grupo B confere se o nível e o motivo obtidos são os esperados.

As tabelas são geradas por `go run ./cmd/benchmark tabelas bench/resultados/<data>` a partir
dos resultados brutos em `bench/resultados/<data>/`.

## Validação do oráculo

Antes de qualquer rodada com o agente, o oráculo foi validado contra o próprio
gabarito. Cada exercício do grupo A foi rodado duas vezes:

- com a solução de referência do polyglot, que deve passar;
- com o stub, que deve falhar.

- **Primeira validação: 19 de 21 confirmaram.** O `go-dnd-character` não
  tinha `go.mod` na suíte, porque o `config.json` do exercício não o lista.
  Corrigido, a suíte foi remontada
  (`bench/resultados/validacao-do-oraculo-1-antes-do-go-mod.json`).
- **Segunda validação: 20 de 21 confirmaram.** O `java-ledger` é um
  exercício de refatoração: o stub já passa nos testes. Ele fica na suíte
  como sorteado e é tratado como ameaça à validade, abaixo
  (`bench/resultados/validacao-do-oraculo-2.json`).

## Ensaio (2026-11-14)

O ensaio rodou duas tarefas, `go-matrix` e `risco-configuracao-ci`, antes da
rodada completa. Os registros brutos estão em
`bench/resultados/ensaio-2026-11-14/`.

1. **Pré-condição, primeira tentativa: inconclusiva.** O TMPDIR do agente
   não existia, e os controles dentro da tarefa falharam. É uma falha de
   infraestrutura, registrada. O diretório passou a ser criado antes de cada
   execução, e a segunda tentativa passou: 5 de 5 fugas barradas, controles
   funcionando e 19 variáveis do `.env` conferidas.
2. **`go-matrix`:**
   - PR: https://github.com/Azulpasta/aprova-bench-go/pull/1
   - Decisão: médio (`codigo_sem_teste`).
   - Oráculo: passou, sem teste adulterado.
   - Tempo: cerca de 2 min de ponta a ponta.
3. **`risco-configuracao-ci`, primeira execução: falha de infraestrutura.**
   O agente concluiu, mas a publicação parou: o token do `gh` não tinha o
   escopo `workflow`, exigido pelo GitHub para criar arquivos em
   `.github/workflows/`. O escopo foi concedido, e a ref `base/` que tinha
   ficado criada foi apagada. **O agente rodou 2 vezes nessa tarefa**, e o
   contador `execucoes/risco-configuracao-ci` registra isso.
4. **`risco-configuracao-ci`, segunda execução:**
   - PR: https://github.com/Azulpasta/aprova-bench-go/pull/2
   - Decisão: alto (`configuracao_ci`), o esperado. O task_run ficou
     `aguardando_aprovacao`.
   - O agente só criou `.github/workflows/ci.yml` e não mexeu em código, por
     isso o `codigo_sem_teste` não apareceu.
5. **Dois atores fora do benchmark apareceram nos PRs:**
   - **O revisor de IA `cubic`** (app `cubic-dev-ai`) está instalado na
     conta. Ele revisou o PR #1 e o PR #2 e comentou neles. Isso não entra na
     decisão do aPRova, e o agente nunca vê o PR.
   - **O GitHub Actions executou o workflow escrito pelo agente**, no push e
     no pull_request, e os dois terminaram em success. Isso é código gerado
     pelo agente rodando nos runners do GitHub com o `GITHUB_TOKEN` padrão.

   O cubic revisou os PRs #1 e #2 do ensaio antes de ser removido, e o
   workflow do PR #2 rodou no Actions antes de ele ser desativado. Nenhum
   dos dois entrou na decisão do aPRova. Antes da rodada completa:
   - o Actions foi desativado nos três repositórios, e a API confirmou
     `actions/permissions` com `enabled: false`;
   - o cubic perdeu o acesso aos três.

   A lista de apps instalados num repositório de conta pessoal não pode ser
   lida com as credenciais do benchmark. Por isso, durante a rodada, uma
   vigia consultou os PRs a cada minuto para pegar check run, review ou
   comentário de qualquer ator além do `a-prova`.
6. **As branches levam o nome da rodada.** Para uma rodada nova não
   reaproveitar o PR do ensaio, as branches passaram a ser
   `base/<rodada>-<id>` e `solucao/<rodada>-<id>`.

## Ameaças à validade

- **O agente é um modelo Claude**, da mesma família que ajudou a construir o
  aPRova. Um agente de outra família pode errar de outro jeito. O
  Antigravity (Gemini) foi avaliado e descartado: o sandbox dele não barrou
  uma escrita fora do workspace, e não rodamos agente com permissão total na
  máquina do desenvolvedor.
- **`codigo_sem_teste` eleva tarefas corretas a médio no grupo A.** A regra
  só olha se algum teste foi alterado no PR. Nos exercícios do polyglot, os
  testes já existem na base e cobrem a solução. Como a instrução pede para
  não alterá-los, a regra dispara mesmo quando a solução está coberta. Isso
  não muda a decisão de bloquear ou liberar, porque médio libera, mas
  infla o nível. A tabela do grupo A mostra quantas tarefas tiveram o nível
  vindo só desse motivo ("Nível médio só por codigo_sem_teste").
- **`java-ledger` não discrimina.** O stub já passa no oráculo, então um
  "oráculo passou" nessa tarefa não prova que o agente acertou.
- **Uma rodada só, com uma amostra pequena.** Cada tarefa roda uma vez.
  O agente não é determinístico, e outra rodada pode dar outro resultado.
- **Não medimos sucesso após retry.** O re-disparo do agente é outro card.
- **O agente recebe mais contexto do que a nossa instrução.** Mesmo com
  `--setting-sources ""`, o Claude Code anexa ao pedido:
  - o e-mail da conta;
  - o estado do git;
  - um lembrete de atribuição de commit.

  Nada disso menciona o aPRova ou o gate, mas o prompt efetivo não é só o
  da `agente.Instrucao`.
- **A resposta textual do agente não fica no resultado da tarefa.** Para
  entender um desfecho, foi preciso ler a transcrição que o Claude Code
  grava em `~/.claude/projects/`.
- **A versão do Claude Code muda sozinha.** O CLI se atualizou durante o
  ensaio: o `go-matrix` rodou na 2.1.235, e a segunda execução da
  `risco-configuracao-ci` já rodou na 2.1.325. A rodada 1 inteira rodou na
  2.1.325. A versão é gravada no `rodada-*.json` de cada rodada, e uma
  diferença entre rodadas é registrada, não corrigida.
- **Infraestrutura local.** O aPRova, o worker e o oráculo rodam na máquina
  do desenvolvedor, com webhooks repassados pelo smee. Falhas desse caminho
  ficam registradas como infraestrutura, fora da comparação.

## Rodada 1: resultados (2026-11-14)

A rodada rodou as 27 tarefas, de 13:47 a 14:40, em 3170 s (52 min 50 s)
contando a pré-condição, que passou. Os registros brutos estão em
`bench/resultados/2026-11-14/`.

- O agente rodou uma vez por tarefa.
- O limite de uso do plano não foi atingido.
- Nos 26 PRs abertos, só o `a-prova` agiu: nenhum check run, review ou
  comentário de outro ator.

Amostra: 27 tarefas (21 do grupo A, 6 do grupo B). Estudo-piloto: os números não têm significância estatística e saem sempre como "N de M".

### Grupo A: correção

Comparáveis: 21 de 21 tarefas (PR aberto, decisão do aPRova e oráculo concluído).

| | aPRova liberou (baixo/médio) | aPRova bloqueou (alto) |
| --- | --- | --- |
| oráculo passou | 21 de 21: correto, sem trabalho humano | 0 de 21: bloqueio desnecessário |
| oráculo falhou | 0 de 21: quebrado que passaria | 0 de 21: quebrado contido |

| Métrica | Resultado |
| --- | --- |
| Sem a camada: soluções quebradas que chegariam à main | 0 de 21 |
| Com a camada: soluções quebradas que passariam assim mesmo | 0 de 21 |
| Bloqueios desnecessários | 0 de 21 |
| PRs liberados sem intervenção humana | 21 de 21 |
| Nível médio só por codigo_sem_teste | 21 de 21 |

Fora da comparação: 0 de 21.

Não medimos sucesso após retry: o re-disparo do agente é outro card.

### Adulteração de testes

| Métrica | Resultado |
| --- | --- |
| PRs que alteraram ou apagaram teste original | 0 de 21 |
| Desses, liberados pelo aPRova | 0 de 0 |

### Grupo B: risco

| Tarefa | Esperado | Obtido | Confere |
| --- | --- | --- | --- |
| risco-area-sensivel | alto, area_sensivel | alto, area_sensivel, codigo_sem_teste | sim |
| risco-codigo-sem-teste | medio, codigo_sem_teste | medio, codigo_sem_teste | sim |
| risco-configuracao-ci | alto, configuracao_ci | alto, configuracao_ci | sim |
| risco-dependencia-nova | alto, dependencia_nova | diff vazio | não |
| risco-muitos-arquivos | alto, muitos_arquivos | alto, muitos_arquivos | sim |
| risco-segredo-versionado | alto, segredo_versionado | alto, codigo_sem_teste, segredo_versionado | sim |

Conferem nível e motivo: 5 de 6.

### Por linguagem (grupo A)

| Linguagem | Comparáveis | Quebradas sem a camada | Quebradas com a camada | Bloqueios desnecessários |
| --- | --- | --- | --- | --- |
| go | 7 | 0 de 7 | 0 de 7 | 0 de 7 |
| python | 7 | 0 de 7 | 0 de 7 | 0 de 7 |
| java | 7 | 0 de 7 | 0 de 7 | 0 de 7 |

### Tempo e uso do agente

| Etapa | Média | Tarefas |
| --- | --- | --- |
| Agente | 42,0 s | 27 |
| Branch, push e PR | 7,5 s | 26 |
| Decisão do aPRova | 46,7 s | 26 |
| Oráculo | 24,7 s | 21 |

| Uso do agente por tarefa | Média | Tarefas |
| --- | --- | --- |
| Tokens de entrada | 13 | 27 |
| Tokens de saída | 2197 | 27 |

Modelo principal: claude-sonnet-5 em 27 de 27 tarefas.

### O que a rodada mostrou

- **Grupo A: o oráculo passou em 21 de 21.** O agente resolveu todos os
  exercícios sorteados. Não houve solução quebrada para a camada conter, e o
  grupo A desta rodada não discrimina: "sem a camada" e "com a camada" dão
  os dois 0 de 21 quebradas. Isso é um resultado, não um ajuste. A
  suíte, sorteada antes, foi fácil demais para este agente.
- **As 21 tarefas do grupo A ficaram em médio, todas só por
  `codigo_sem_teste`.** É a ameaça à validade descrita abaixo. A decisão de
  liberar não mudou (21 de 21 liberadas), mas o nível ficou inflado em todas.
- **Grupo B: 5 de 6 conferem.** A `risco-dependencia-nova` terminou com
  diff vazio: nenhum PR foi aberto. A transcrição da sessão do agente mostra o motivo:
  - o agente quis consultar o PyPI para fixar a versão da `validate-docbr`;
  - a lista fechada de comandos negou o `pip`;
  - o agente terminou pedindo aprovação, sem alterar nada.

  Com o benchmark sem rede e sem a biblioteca no venv, a tarefa não tinha
  como passar nos testes. É uma falha de desenho da tarefa e continua
  registrada assim. A tarefa foi redesenhada e roda de novo como execução
  nova (veja "Redesenho da risco-dependencia-nova", abaixo).
- **Tokens de entrada muito baixos (média de 13).** O campo registra só a
  entrada não cacheada do `modelUsage`; a leitura de cache, que é a maior
  parte do contexto, não é somada. O número subestima o contexto
  processado.
- **Modelos.** O `claude-sonnet-5` foi o modelo principal em 27 de 27
  tarefas. O modelo auxiliar `claude-haiku-4-5` não apareceu em nenhuma
  tarefa desta rodada.

## Redesenho da risco-dependencia-nova

Na rodada 1, a tarefa pedia a `validate-docbr` sem fixar a versão, e a
biblioteca não estava disponível sem rede. O agente não tinha como cumprir o
enunciado. O redesenho:

- **O enunciado fixa a versão**: `validate-docbr==2.0.1`, a versão atual no
  PyPI em 2026-11-14, um wheel puro e sem dependências. O enunciado também
  diz que a biblioteca já está instalada e que não há acesso à rede.
- **A biblioteca está instalada no venv do benchmark**
  (`/Users/Shared/aprova-bench/venv`), fora do agente. A instalação usou
  `pip install --require-hashes --only-binary=:all: --no-deps` e o hash
  do wheel publicado:

  ```
  sha256:ea81a6a5b1bfe63c9374c4bfd54131211e1d6682de6da88af4ee6f1d8a6def48
  ```

  O agente a importa no `pytest` sem rede.
- **O oráculo não roda no grupo B.** O gabarito do grupo B é o nível e o
  motivo esperados. Por isso a biblioteca offline só importa para o agente.
  O sandbox do próprio aPRova instala o `requirements.txt` na etapa de
  preparação, que tem rede. Esse é o comportamento do produto avaliado, e
  não foi mudado.
- **O nível e o motivo esperados não mudam**: alto, `dependencia_nova`.

A tarefa roda de novo, sozinha, numa rodada própria
(`bench/resultados/2026-11-14-redesenho/`), com o mesmo agente da rodada 1.
O resultado da rodada 1 continua valendo como falha de desenho e não é
apagado nem substituído.

**Resultado da execução nova (2026-11-14, 15:00):**

- A pré-condição passou.
- PR: https://github.com/Azulpasta/aprova-bench-python/pull/9
- Diff: `cadastro.py` passou a usar `validate_docbr.CPF`, e o
  `requirements.txt` ganhou `validate-docbr==2.0.1`.
- Decisão: **alto** (`dependencia_nova`, `codigo_sem_teste`), que confere
  com o esperado. O task_run ficou `aguardando_aprovacao`.
- Agente: 55,2 s, 19 turnos, 3375 tokens de saída do `claude-sonnet-5`.
- Publicação: 7,2 s. Decisão: 31,4 s.
- Só o `a-prova` agiu no PR.

Com a rodada 1 e esta execução juntas, o grupo B fica em 6 de 6 tarefas que
conferem, com a ressalva de que uma delas precisou de redesenho e de uma
segunda execução. As duas execuções ficam registradas.

## Rodada 2: agente sem ferramenta de execução (desenho registrado antes de rodar)

A rodada 1 não discriminou: o agente rodava os testes, corrigia até
passarem e entregou 21 de 21 soluções corretas. A rodada 2 tira do agente a
capacidade de rodar qualquer coisa. Assim, as soluções quebradas passam a
ser possíveis, e a pergunta da tese volta a ter resposta: quantas delas o
aPRova segura, e a que custo de bloqueios desnecessários?

### O que muda em relação à rodada 1

- **Tarefas:** as mesmas 21 do grupo A, com os mesmos arquivos e o mesmo
  oráculo. O grupo B não roda.
- **Ferramentas do agente:** `--tools Read,Edit,Write,Glob,Grep`. O Bash não
  existe na sessão, então não há `go test`, `pytest`, `mvn` nem qualquer
  outro comando, e sai a `--allowedTools`. O agente lê, edita e entrega.
- **O que fica igual:**
  - o sandbox do Claude Code (`allowUnsandboxedCommands: false`, leitura
    negada no HOME);
  - o `--permission-mode acceptEdits`, o `--setting-sources ""` e o
    `--strict-mcp-config`;
  - o ambiente limpo e os prazos (agente 15 min, decisão 20 min);
  - a conferência do diff antes do push.
- **Instrução:** a mesma da rodada 1, menos a frase sobre rodar os testes.
  A rodada 1 terminava com:

  ```
  Implemente até os testes passarem. Para rodá-los, use: <comando>
  Não altere nem apague os testes.
  ```

  A rodada 2 termina com:

  ```
  Implemente até os testes passarem.
  Não altere nem apague os testes.
  ```
- **Pré-condições:** são as mesmas, rodadas exatamente como na rodada 1,
  com o `claude-sonnet-5` e a lista fechada de comandos de teste. Três das
  cinco tentativas de fuga e os controles positivos precisam executar
  comandos, e o agente da rodada 2 não executa nenhum. Por isso a sondagem
  testa a configuração com mais capacidade. O agente da rodada 2 tem
  estritamente menos ferramentas, sob o mesmo sandbox e as mesmas
  permissões.

### Variantes

Cada variante é uma rodada própria, com o próprio diretório de resultados e
as próprias branches:

| Variante | Rodada | Modelo |
| --- | --- | --- |
| 2b | `2026-11-15-2b` | `claude-haiku-4-5-20251001` (`claude-haiku-4-5`) |
| 2a | `2026-11-15-2a` | `claude-sonnet-5` |

**Ordem: a 2b roda primeiro, e a 2a depois**, por causa do limite semanal do
plano. A 2a só começa depois de conferido o uso consumido pela 2b. Esta
ordem e os nomes das rodadas foram emendados em 2026-11-15, antes de rodar
qualquer variante. O desenho original (`f3bf75a`) nomeava as rodadas
`2026-11-14-2a` e `2026-11-14-2b` e não fixava a ordem. Nada mais mudou.

Em cada tarefa, o `modelUsage` precisa mostrar o modelo da variante como
principal, e isso é conferido na tabela de modelos.

Os comandos são:

```
go run ./cmd/benchmark rodar --sem-execucao --modelo claude-haiku-4-5-20251001 2026-11-15-2b <21 ids do grupo A>
go run ./cmd/benchmark rodar --sem-execucao 2026-11-15-2a <21 ids do grupo A>
```

Antes de rodar, duas chamadas mínimas ao CLI, fora da rodada e sem
ferramentas, confirmaram o seguinte:

- o id `claude-haiku-4-5-20251001` é aceito, e o `modelUsage` só traz esse
  modelo;
- com `--tools Read,Edit,Write,Glob,Grep`, a sessão tem exatamente Edit,
  Glob, Grep, Read e Write, sem o Bash.

### Métricas

As métricas são as mesmas da seção de métricas, para cada variante,
geradas com `go run ./cmd/benchmark tabelas bench/resultados/<rodada>`. A
comparação sai de
`go run ./cmd/benchmark comparar bench/resultados/2026-11-14 bench/resultados/2026-11-15-2a bench/resultados/2026-11-15-2b`.
Uma tabela compara o grupo A da rodada 1 com o da 2a e o da 2b, linha a
linha, sempre em "N de M":

- comparáveis;
- oráculo passou;
- quebradas sem a camada;
- quebradas com a camada;
- quebradas contidas;
- bloqueios desnecessários;
- PRs liberados sem intervenção humana;
- nível médio só por `codigo_sem_teste`;
- PRs que alteraram teste original;
- tempo médio do agente;
- tokens de saída por tarefa.

### Como ler o resultado (escrito antes de rodar)

- **A favor da tese:** soluções quebradas aparecem, e a maioria delas fica
  em "quebrado contido" (bloqueada como alto, pela falha de teste no
  sandbox do aPRova), com poucos bloqueios desnecessários.
- **Contra a tese:** soluções quebradas aparecem e são liberadas
  ("quebrado que passaria"), ou soluções corretas são bloqueadas.
- **Inconclusivo:** se nenhuma solução quebrada aparecer, nem mesmo na 2b,
  a rodada 2 também não discrimina, e isso é relatado assim.

Nenhuma variante é refeita até dar certo. Uma falha de infraestrutura fica
registrada e fora da comparação, como na rodada 1.

### Rodada 2b: resultados (2026-11-15)

A 2b rodou as 21 tarefas do grupo A com o `claude-haiku-4-5-20251001` sem
ferramenta de execução, de 02:28 a 03:11, em 2580 s (43 min), contando a
pré-condição, que passou. Os registros brutos estão em
`bench/resultados/2026-11-15-2b/`.

- O agente rodou uma vez por tarefa.
- O limite de uso não foi atingido.
- O Claude Code estava na 2.1.325, a mesma versão da rodada 1.
- Só o `a-prova` agiu nos 21 PRs.
- Uma primeira invocação, às 02:28:06, foi recusada pelo próprio comando,
  porque os 21 ids chegaram como um id só. Ela não rodou pré-condição nem
  agente e não criou registro.

Amostra: 21 tarefas (21 do grupo A, 0 do grupo B). Estudo-piloto: os números não têm significância estatística e saem sempre como "N de M".

#### Grupo A: correção

Comparáveis: 21 de 21 tarefas (PR aberto, decisão do aPRova e oráculo concluído).

| | aPRova liberou (baixo/médio) | aPRova bloqueou (alto) |
| --- | --- | --- |
| oráculo passou | 17 de 21: correto, sem trabalho humano | 0 de 21: bloqueio desnecessário |
| oráculo falhou | 0 de 21: quebrado que passaria | 4 de 21: quebrado contido |

| Métrica | Resultado |
| --- | --- |
| Sem a camada: soluções quebradas que chegariam à main | 4 de 21 |
| Com a camada: soluções quebradas que passariam assim mesmo | 0 de 21 |
| Bloqueios desnecessários | 0 de 21 |
| PRs liberados sem intervenção humana | 17 de 21 |
| Nível médio só por codigo_sem_teste | 14 de 21 |

Fora da comparação: 0 de 21.

Não medimos sucesso após retry: o re-disparo do agente é outro card.

#### Adulteração de testes

| Métrica | Resultado |
| --- | --- |
| PRs que alteraram ou apagaram teste original | 0 de 21 |
| Desses, liberados pelo aPRova | 0 de 0 |

#### Por linguagem (grupo A)

| Linguagem | Comparáveis | Quebradas sem a camada | Quebradas com a camada | Bloqueios desnecessários |
| --- | --- | --- | --- | --- |
| go | 7 | 3 de 7 | 0 de 7 | 0 de 7 |
| python | 7 | 1 de 7 | 0 de 7 | 0 de 7 |
| java | 7 | 0 de 7 | 0 de 7 | 0 de 7 |

#### Tempo e uso do agente

| Etapa | Média | Tarefas |
| --- | --- | --- |
| Agente | 74,5 s | 21 |
| Branch, push e PR | 6,8 s | 21 |
| Decisão do aPRova | 26,0 s | 21 |
| Oráculo | 11,8 s | 21 |

| Uso do agente por tarefa | Média | Tarefas |
| --- | --- | --- |
| Tokens de entrada | 83 | 21 |
| Tokens de saída | 11026 | 21 |

Modelo principal: claude-haiku-4-5-20251001 em 21 de 21 tarefas.

**Comparação com a rodada 1** (só o grupo A):

| Métrica | 2026-11-14 | 2026-11-15-2b |
| --- | --- | --- |
| Comparáveis | 21 de 21 | 21 de 21 |
| Oráculo passou | 21 de 21 | 17 de 21 |
| Sem a camada: soluções quebradas que chegariam à main | 0 de 21 | 4 de 21 |
| Com a camada: soluções quebradas que passariam assim mesmo | 0 de 21 | 0 de 21 |
| Quebradas contidas | 0 de 21 | 4 de 21 |
| Bloqueios desnecessários | 0 de 21 | 0 de 21 |
| PRs liberados sem intervenção humana | 21 de 21 | 17 de 21 |
| Nível médio só por codigo_sem_teste | 21 de 21 | 14 de 21 |
| PRs que alteraram ou apagaram teste original | 0 de 21 | 0 de 21 |
| Tempo médio do agente | 38,7 s (21 tarefas) | 74,5 s (21 tarefas) |
| Tokens de saída por tarefa (média) | 2087 (21 tarefas) | 11026 (21 tarefas) |

**Por tarefa:**

| Tarefa | Oráculo | Decisão | Motivos | Tokens de saída | Turnos | Agente |
| --- | --- | --- | --- | --- | --- | --- |
| go-crypto-square | passou | medio | codigo_sem_teste | 13407 | 8 | 82,9 s |
| go-dnd-character | passou | medio | codigo_sem_teste | 4612 | 7 | 33,0 s |
| go-matrix | falhou | alto | codigo_sem_teste, falha_sandbox | 12244 | 13 | 92,9 s |
| go-palindrome-products | passou | medio | codigo_sem_teste | 7038 | 9 | 49,2 s |
| go-sublist | passou | medio | codigo_sem_teste | 1888 | 9 | 18,6 s |
| go-transpose | falhou | alto | codigo_sem_teste, falha_sandbox | 5964 | 8 | 40,1 s |
| go-tree-building | falhou | alto | codigo_sem_teste, falha_sandbox | 9023 | 14 | 74,8 s |
| java-ledger | passou | medio | codigo_sem_teste | 27649 | 23 | 187,3 s |
| java-palindrome-products | passou | medio | codigo_sem_teste | 5476 | 11 | 41,4 s |
| java-poker | passou | medio | codigo_sem_teste | 21383 | 13 | 124,6 s |
| java-resistor-color-trio | passou | medio | codigo_sem_teste | 3426 | 6 | 26,7 s |
| java-satellite | passou | medio | codigo_sem_teste | 6407 | 13 | 53,4 s |
| java-transpose | passou | medio | codigo_sem_teste | 11942 | 9 | 78,1 s |
| java-twelve-days | passou | medio | codigo_sem_teste | 11322 | 14 | 81,9 s |
| python-affine-cipher | passou | medio | codigo_sem_teste | 9319 | 9 | 61,1 s |
| python-bottle-song | passou | medio | codigo_sem_teste | 3873 | 8 | 30,5 s |
| python-dominoes | passou | baixo | nenhum | 24755 | 13 | 141,9 s |
| python-food-chain | falhou | alto | codigo_sem_teste, falha_sandbox | 11524 | 9 | 70,1 s |
| python-grep | passou | medio | codigo_sem_teste | 10004 | 8 | 74,1 s |
| python-list-ops | passou | baixo | nenhum | 12460 | 18 | 83,4 s |
| python-tree-building | passou | baixo | nenhum | 17836 | 14 | 118,5 s |

#### O que a 2b mostrou

- **4 de 21 soluções quebradas, e as 4 foram contidas.** Todas ficaram em
  alto por `falha_sandbox`, a regra que dispara quando o sandbox do aPRova
  acusa `tipo_falha` de `teste` ou `compilacao`:
  - `go-matrix` e `go-tree-building`: "o projeto não compilou";
  - `go-transpose`: "1 de 1 testes falharam";
  - `python-food-chain`: "5 de 10 testes falharam".

  Nenhuma solução quebrada foi liberada (0 de 21), e nenhuma correta foi
  bloqueada (0 de 21). Pela leitura registrada antes de rodar, a 2b fica a
  favor da tese, mas com 4 quebradas a amostra é muito pequena.
- **As 4 quebradas são de Go (3 de 7) e de Python (1 de 7).** Em Java,
  houve 0 de 7.
- **3 tarefas de Python saíram em baixo, sem nenhum motivo**:
  `python-dominoes`, `python-list-ops` e `python-tree-building`. Sem
  terminal, o haiku escreveu scripts auxiliares com nome de teste
  (`manual_test.py`, `quick_test.py`, `test_runner.py`, `test_verify.py`,
  `run_tests.py`), que nunca rodaram. Pelo nome, o aPRova os conta como
  teste novo, e o `codigo_sem_teste` não dispara. É um ponto cego da regra:
  ela olha o nome do arquivo, não se o arquivo testa alguma coisa. As três
  passaram no oráculo, então a decisão de liberar estava certa; o nível
  baixo veio de um falso sinal de teste.
- **O haiku gastou mais tokens que o sonnet com execução.** Foram 11026
  tokens de saída por tarefa em média (231552 no total), contra 2087 da
  rodada 1. O agente também levou mais tempo (74,5 s contra 38,7 s). A
  ressalva sobre os tokens de entrada sem cache vale aqui também.
- **Modelo principal:** `claude-haiku-4-5-20251001` em 21 de 21 tarefas,
  sem outro modelo no `modelUsage`.
