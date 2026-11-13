# Calibração por histórico

A calibração é a camada 2 do gate de risco. Ela compara a análise atual com
análises passadas parecidas do mesmo tenant. Quando humanos aprovaram e
mergearam todas elas sem reversão, ela desce o nível do gate determinístico **um
degrau**. Ela nunca sobe o nível, nunca desce dois degraus e nunca decide
sozinha.

- O modelo de embedding só mede semelhança entre textos. Ele não opina sobre o
  PR.
- Quem decide é uma regra fixa, em `internal/calibracao/regra.go`, sobre o que
  humanos fizeram com os casos parecidos.
- Na dúvida, o nível é o do gate determinístico.
- Desligada (`CALIBRACAO_LIGADA=false`, o padrão), o aPRova se comporta
  exatamente como antes: o recibo, o Check Run, o comentário e os eventos são
  os mesmos.

## Onde ela roda

Tudo acontece em `avaliando_risco`, depois do gate determinístico e antes do
recibo:

```
gate determinístico ─► calibração (se ligada) ─► transação: risk_gate + code_memory + recibo
```

1. O gate devolve o nível, os motivos e as dependências novas.
2. O calibrador monta a assinatura do PR, pede o embedding ao provedor e, se a
   análise for calibrável, busca os vizinhos e aplica a regra. Essas chamadas
   externas acontecem **antes** da transação. O estado `avaliando_risco` já
   está gravado, então um crash no meio só faz o worker repetir a avaliação.
3. A decisão, o rastro da calibração, a assinatura com o vetor e o recibo
   assinado são gravados numa só transação: ou tudo fica, ou nada fica.
4. Na retomada depois de um crash, com o `risk_gate` já gravado, o motor usa o
   que está gravado. O embedding e a calibração não são chamados de novo.

Os eventos do n8n continuam saindo do `risk_gate` gravado.

## O que pode e o que nunca pode ser calibrado

A lista é fechada, no código (`calibracao.Calibravel`).

| Motivo | Calibrável | Por quê |
| --- | --- | --- |
| `area_sensivel` | sim | o mesmo tipo de mudança em `auth/` aprovada muitas vezes é evidência de padrão seguro |
| `muitos_arquivos` | sim | PRs grandes e repetitivos, como renomeações, têm histórico próprio |
| `codigo_sem_teste` | sim | médio por natureza; o histórico pode mostrar que aquele padrão não quebra |
| `falha_sandbox` | **nunca** | teste falhou; nenhum histórico torna um teste vermelho aceitável |
| `segredo_versionado` | **nunca** | um segredo versionado é incidente, não padrão |
| `dependencia_nova` | **nunca** | slopsquatting e CVE: cada pacote novo precisa de olho humano |
| `configuracao_ci` | **nunca** | cadeia de suprimentos: mexer no pipeline é mexer em quem constrói o código |

Qualquer motivo fora da lista, inclusive um que ainda venha a existir, conta como
não calibrável. Se **qualquer** motivo não calibrável estiver presente, a
calibração nem roda: o nível só cai se todos os motivos que o produziram forem
calibráveis. Nível baixo não tem o que reduzir.

## A assinatura do PR

O texto que vira embedding é montado de forma determinística a partir do que o
gate já tem (`calibracao.Assinatura`):

```
motivos: area_sensivel, codigo_sem_teste
caminhos: *, *.md, internal/auth/*.go, migrations/*.sql
adicionados: 2
alterados: 4
removidos: 1
```

- **motivos:** os ids dos motivos do gate, em ordem alfabética, sem repetição
  e sem o `calibracao_historico`.
- **caminhos:** cada arquivo vira `diretório/*.extensão`, em ordem e sem
  repetição. Arquivo na raiz vira `*.extensão`; arquivo sem extensão vira
  `diretório/*` ou `*`.
- **contagens:** `added` conta como adicionado e `removed` como removido; todo
  o resto (`modified`, `renamed` e os demais) conta como alterado.

A mesma entrada gera o mesmo texto em qualquer ordem de arquivos ou motivos, e
há teste para isso. **O código do PR nunca vai ao provedor de embedding:** nem
o diff, nem o conteúdo, nem o nome dos arquivos. Só diretório, extensão,
contagens e ids de regras.

## O embedding

O pacote `internal/embedding` tem a interface `Embedder` e três implementações:

| Implementação | Uso |
| --- | --- |
| `Gemini` | produção: `gemini-embedding-001`, `taskType: SEMANTIC_SIMILARITY`, `outputDimensionality: 768` |
| `Falso` | testes: vetor determinístico derivado do SHA-256 do texto |
| `Indisponivel` | calibração ligada sem `GEMINI_API_KEY`: toda chamada falha com o motivo |

Como o Gemini é chamado:

- **Chave:** vai no cabeçalho `x-goog-api-key`, nunca na URL, e nenhuma
  mensagem de erro a repete.
- **Normalização:** o Gemini só entrega o vetor normalizado em 3072 dimensões.
  Em 768, o aPRova normaliza o vetor antes de gravar.
- **Busca:** a similaridade é o cosseno no pgvector, pelo índice HNSW
  `code_memory_embedding`, com varredura iterativa para o filtro por tenant não
  esvaziar o resultado.
- **Histórico:** o vetor de **toda** análise avaliada com a calibração ligada é
  gravado em `code_memory`, inclusive quando a calibração não rodou ou não
  mudou nada. É assim que o histórico cresce.

## Os sinais de desfecho

O desfecho de um vizinho **não é gravado na linha dele**. Ele é derivado na hora
da consulta a partir de três fatos, porque muda com o tempo: um PR aprovado
hoje pode ser revertido amanhã.

| Fato | Tabela | De onde vem |
| --- | --- | --- |
| aprovação humana do commit exato | `approvals` (`decision='aprovada'`, `commit_sha`) | webhook `pull_request_review` |
| PR fechado, com ou sem merge | `pull_request_outcomes` | webhook `pull_request` com `action: closed` (migration 18) |
| PR revertido | `pull_request_reverts` | webhook `closed` com merge de um PR que reverte outro (migration 18) |

**Fechamento.**
- O registro guarda `merged`, o head no fechamento e o commit do merge.
- O `merge_commit_sha` só é gravado quando houve merge. O GitHub manda um
  commit de merge de teste mesmo em PR fechado sem merge, e ele não interessa.
- Reentrega não duplica nada. Num PR reaberto e fechado de novo, vale o
  fechamento mais recente, mesmo que o antigo chegue depois.

**Reversão.** Só vale o formato do botão **Revert** do GitHub:

- a **primeira linha** do corpo do PR que reverte é exatamente
  `Reverts <dono>/<repo>#<n>`;
- o repositório é o mesmo do evento;
- o PR que reverte foi mergeado.

Não há heurística de título ou de texto livre. Uma linha `Reverts` fora da
primeira posição, com texto depois do número ou de outro repositório não marca
nada.

### Como cada vizinho é classificado

| Desfecho | Condição | Conta como |
| --- | --- | --- |
| `revertido` | o PR do vizinho consta em `pull_request_reverts` | **negativo** |
| `fechado_sem_merge` | o PR foi fechado com `merged=false` | **negativo** |
| `aprovado_e_mergeado` | o PR foi mergeado no commit do vizinho e há aprovação humana com `commit_sha` igual a esse commit | **positivo** |
| `mergeado_sem_aprovacao` | o PR foi mergeado no commit do vizinho, sem aprovação desse commit | ignorado |
| `substituido` | o PR foi mergeado em outro commit | ignorado |
| `em_aberto` | o PR não foi fechado | ignorado |

O negativo vale para o PR inteiro, e o positivo só para o commit exato que foi
aprovado e mergeado. É a mesma assimetria do resto do gate.

**Por que `mergeado_sem_aprovacao` é ignorado.** Um PR calibrado de alto para
médio é mergeado sem aprovação humana. Se isso bloqueasse, cada calibração
impediria a próxima, e a calibração oscilaria. Se contasse como positivo, a
calibração se alimentaria das próprias decisões e derivaria. Ignorando, só a
aprovação humana do commit exato vale como positivo, e só reversão e
fechamento sem merge valem como negativo.

**PR ajustado.** Num PR que recebeu alterações solicitadas e depois foi ajustado
e aprovado, o commit rejeitado nunca chegou à main e é ignorado como
`substituido`. O commit ajustado e aprovado conta como positivo: é o sinal
"ajustado e depois aprovado".

## A regra

Com os vizinhos do mesmo tenant:

1. A busca pega os `k` mais parecidos (`CALIBRACAO_VIZINHOS`, padrão 10). Os
   outros task_runs do **próprio PR** ficam de fora, para o PR não contar a
   favor de si mesmo.
2. Ficam só os que estão acima da similaridade mínima
   (`CALIBRACAO_SIMILARIDADE_MINIMA`, padrão 0,97).
3. Se **um** deles for negativo, não calibra, com quaisquer motivos.
4. Um positivo só conta se o gate determinístico dele acionou **exatamente o
   mesmo conjunto de motivos** da análise atual, sem o `calibracao_historico`.
   Um motivo a mais ou a menos e ele não conta, mesmo com similaridade 0,99.
5. Se houver menos positivos assim que o mínimo
   (`CALIBRACAO_MINIMO_DE_DESFECHOS`, padrão 5), não calibra.
6. Cumprido tudo isso, o nível cai **um** degrau: alto vira médio e médio vira
   baixo.

**Por que o mesmo conjunto de motivos.** A similaridade sozinha separa mal:
a margem medida entre o pior par legítimo e o melhor par que deve ficar de fora
é de 0,006 (veja os números abaixo). O conjunto de motivos é um filtro exato e
determinístico, que não depende do modelo. Os motivos de cada vizinho vêm de
`risk_gates.rules_hit` dele.

**Por que o filtro não vale para os negativos.** O filtro existe para
restringir a calibração, nunca para ampliá-la. Aplicado aos negativos, ele
faria um vizinho revertido com um motivo a mais deixar de bloquear, e a
calibração passaria a acontecer onde antes não acontecia. Por isso um negativo
acima do limiar bloqueia com quaisquer motivos.

Histórico só da mesma instalação: decisão de outro tenant nunca é usada. A
consulta filtra por tenant, e a regra descarta de novo qualquer vizinho de
outro tenant. Dentro do tenant, o histórico é unificado: todos os repositórios
e linguagens juntos.

O limiar padrão de 0,97 saiu do teste manual com o Gemini real, descrito
abaixo.

## Configuração

| Variável | Padrão | O que faz |
| --- | --- | --- |
| `CALIBRACAO_LIGADA` | `false` | liga a calibração no worker |
| `CALIBRACAO_VIZINHOS` | `10` | quantos vizinhos mais parecidos olhar (1 a 1000) |
| `CALIBRACAO_SIMILARIDADE_MINIMA` | `0.97` | similaridade de cosseno mínima para um vizinho contar (0 a 1) |
| `CALIBRACAO_MINIMO_DE_DESFECHOS` | `5` | quantos positivos são exigidos |
| `CALIBRACAO_TIMEOUT` | `5s` | prazo único para o embedding e a busca juntos |
| `GEMINI_API_KEY` | vazia | chave do Gemini |

## Quando algo falha

Qualquer falha deixa a análise sem calibração, com o nível determinístico, e o
motivo fica registrado. A calibração nunca derruba uma análise e nunca a atrasa
além de `CALIBRACAO_TIMEOUT`, mesmo que o provedor ou o banco não respeitem o
cancelamento.

| Situação | `calibration_status` | Exemplo de `calibration_reason` |
| --- | --- | --- |
| calibração desligada | `desligada` | calibração desligada |
| nível baixo | `sem_motivo_a_calibrar` | nível baixo: não há o que reduzir |
| motivo fora da lista | `motivo_nao_calibravel` | motivo não calibrável: falha_sandbox |
| sem chave, Gemini fora do ar ou timeout | `embedding_indisponivel` | embedding indisponível: timeout de 5s |
| busca de vizinhos falhou | `historico_indisponivel` | histórico indisponível: … |
| poucos casos | `historico_insuficiente` | histórico insuficiente: 2 de 5 |
| algum vizinho negativo | `vizinho_negativo` | caso parecido com desfecho negativo: 1 revertido(s), 0 fechado(s) sem merge |
| calibrou | `calibrada` | 7 casos parecidos aprovados e mergeados, nenhum revertido |

Com um motivo não calibrável e o embedding fora do ar ao mesmo tempo, o estado
é o do motivo, que é determinístico, e a razão registra as duas coisas.

## O rastro e o recibo

O formato do recibo **não mudou**. A calibração cabe no payload que já existia:

- `nivel_deterministico` e `nivel` passam a poder ser diferentes, e os dois já
  eram assinados;
- quando o nível cai, `motivos` ganha `calibracao_historico`, com uma evidência
  por vizinho usado:

```
<task_run_id> <dono>/<repo>#<n> similaridade 0.9712 aprovado_e_mergeado
```

A similaridade tem quatro casas, a mesma que fica gravada em `risk_gates`.

O que fica **fora** do recibo, em `risk_gates` (migration 20):

| Coluna | Conteúdo |
| --- | --- |
| `calibration_status` | o estado da tabela acima |
| `calibration_reason` | a razão em texto |
| `calibration_neighbors` | os vizinhos considerados, em JSON: `task_run_id`, `repositorio`, `numero_pr`, `similaridade`, `desfecho`, `motivos` |

E em `code_memory` (migration 19), uma linha por análise: o `task_run_id`, o
texto da assinatura em `summary` e o vetor em `embedding`.

## Como auditar uma calibração pelo recibo

1. **Verifique o recibo.** Use `GET /receipts/{id}/verify` ou o passo a passo de
   [verificacao-recibo.md](verificacao-recibo.md), só com a chave pública. Se a
   assinatura confere, nada do que segue foi alterado depois da decisão.
2. **Compare os níveis.** `nivel_deterministico` diferente de `nivel` quer
   dizer que a calibração agiu. A diferença é sempre de um degrau e sempre
   para baixo; o código garante isso, e há teste com milhares de entradas.
3. **Confira os motivos.** Todos os outros motivos do recibo têm de estar na
   lista calibrável. Um `falha_sandbox`, `segredo_versionado`,
   `dependencia_nova` ou `configuracao_ci` ao lado de `calibracao_historico`
   seria um defeito.
4. **Confira cada vizinho.** Cada evidência de `calibracao_historico` traz o
   task_run, o PR, a similaridade e o desfecho no momento da decisão. Todos
   têm de ser `aprovado_e_mergeado`, com similaridade pelo menos igual ao
   limiar, e pelo menos o mínimo configurado. O `rules_hit` do gate de cada
   vizinho, sem o `calibracao_historico`, tem de ter exatamente os mesmos ids
   que os outros motivos deste recibo.
5. **Confira os fatos de cada vizinho.** Para cada vizinho, consulte:
   - o fechamento do PR em `pull_request_outcomes`;
   - a aprovação do commit em `approvals` (`commit_sha` igual ao head do
     task_run vizinho);
   - a ausência de reversão em `pull_request_reverts`.

   O recibo registra o desfecho no momento da decisão. Uma reversão posterior
   aparece nessas tabelas, mas não muda o recibo, que é imutável.

O mesmo texto aparece no PR:

- **Check Run e comentário:** ganham a seção "Calibração por histórico", por
  exemplo "Risco alto pelas regras, reduzido a médio pelo histórico: 7 casos
  parecidos aprovados e mergeados, nenhum revertido", com o link de cada PR
  vizinho.
- **Calibração que rodou e não mudou o nível:** a seção diz por quê.
- **Calibração desligada ou nível baixo:** a seção não aparece.

Tudo é montado a partir dos dados, sem modelo.

## Teste manual com o Gemini

O teste `TestGeminiRealSimilaridadeDasAssinaturas`, em
`internal/embedding/gemini_manual_test.go`, só roda com
`APROVA_TESTE_GEMINI_REAL=1` e `GEMINI_API_KEY` no ambiente, e nunca no CI:

```sh
GEMINI_API_KEY=... APROVA_TESTE_GEMINI_REAL=1 go test ./internal/embedding/ -run GeminiReal -v -count=1
```

Ele gera o embedding de oito assinaturas de referência e mostra a similaridade
de cosseno entre todas elas. Resultado de 2026-11-13, com
`gemini-embedding-001` em 768 dimensões:

| Par | Similaridade |
| --- | --- |
| `area_sensivel` em `internal/auth/`, com teste (três variações entre si) | 0,9723 a 0,9941 |
| o mesmo, com `migrations/*.sql` junto | 0,9508 a 0,9633 |
| `area_sensivel` em `internal/auth/` contra `internal/payments/` | 0,9421 a 0,9665 |
| o mesmo em `auth/`, com `codigo_sem_teste` a mais | 0,9485 a 0,9582 |
| assinaturas sem nada em comum (frontend sem teste, renomeação em massa) | 0,8656 a 0,9031 |

**O piso é alto.** Textos curtos e estruturados como a assinatura ficam todos
perto uns dos outros: até assinaturas sem relação passam de 0,90, como
`codigo_sem_teste` no frontend contra `area_sensivel` em `auth/` (0,9007) e
`area_sensivel` em `payments/` contra uma renomeação de 25 arquivos (0,9031).
Com o limiar antigo de 0,90, esses pares contariam como casos parecidos.

**Por que 0,97.** Nesta amostra, é o valor que aceita só "mesmos motivos e
mesma área" e recusa todo o resto. A margem é estreita: o pior par legítimo dá
0,9723 e o melhor par que deve ficar de fora dá 0,9665.

**Por que isso não basta.** Com uma margem de 0,006, o limiar sozinho é frágil,
e por isso a regra também exige o mesmo conjunto de motivos. Esse filtro não
depende do modelo. Nos pares da tabela, ele já recusa sozinho:
- `auth/` com `codigo_sem_teste` a mais;
- o frontend sem teste;
- a renomeação em massa.

Sobram para o limiar só os pares com os mesmos motivos em áreas diferentes,
como `auth/` contra `payments/` ou `auth/` com `migrations/`. A regra
continua segura dos dois lados do limiar, porque um vizinho negativo bloqueia e
cinco positivos são exigidos. Vale repetir o teste quando houver histórico real
e ajustar o limiar com `CALIBRACAO_SIMILARIDADE_MINIMA`.
