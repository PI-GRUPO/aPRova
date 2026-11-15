# SARIF e code scanning

O aPRova envia o resultado do gate ao code scanning do GitHub, em SARIF 2.1.0.
Com isso os motivos aparecem na aba Security do repositório e como anotações no
próprio pull request.

Isso é paridade de mercado, não novidade do aPRova: Codacy, CodeRabbit e
SonarQube já publicam os achados deles em SARIF. O que o aPRova tem de próprio
continua sendo o gate determinístico, o recibo assinado e o Check Run que
bloqueia o merge.

## O que o SARIF nunca faz

O SARIF é uma saída a mais. Ele não decide nada.

- Não chama o gate. O documento é montado a partir da decisão já gravada em
  `risk_gates`, a mesma que o recibo assina.
- Não muda o nível, o estado do task_run nem o Check Run `aPRova`.
- Não reprova o pull request. Nenhum resultado sai com nível `error` e nenhuma
  regra tem `security-severity`, então o check que o code scanning cria no PR
  não falha com o limiar padrão.
- Não usa modelo de IA.
- Não falha a análise. Um envio que dá errado fica no banco e no log, e o
  task_run segue o caminho dele.

A gravidade real de cada motivo está no Check Run `aPRova` e no recibo. No
SARIF ela aparece rebaixada de propósito (ver "Níveis").

## Ligar e desligar

`SARIF_LIGADO`, com padrão `false`. Desligada, o worker não monta o enviador e
o fluxo é exatamente o de antes: as mesmas chamadas ao GitHub e os mesmos
estados.

O app precisa da permissão `security_events: write` (em "Code scanning alerts",
Read and write). Repositório público tem code scanning sem custo; repositório
privado exige GitHub Advanced Security.

## Quando o envio acontece

Logo depois de cada Check Run publicado:

- risco alto: ao entrar em `aguardando_aprovacao`, para o alerta aparecer
  enquanto o PR espera a revisão, e de novo ao passar por `aplicando`;
- risco baixo ou médio: ao passar por `aplicando`.

O motor só pede o envio e segue. O trabalho roda em segundo plano, então a
transição de estado não espera o GitHub.

O segundo pedido de um mesmo task_run não vira um segundo envio: com o id do
envio gravado, o enviador não manda nada de novo.

## De onde vem cada campo

| Campo do SARIF | Origem |
| --- | --- |
| `tool.driver.name` | fixo, `aPRova SARIF` (ver "O nome da ferramenta") |
| `tool.driver.version` | revisão do git embutida no binário (`vcs.revision`); `desenvolvimento` quando o build não a traz |
| `tool.driver.informationUri` | fixo, o repositório do aPRova |
| `tool.driver.rules` | catálogo fixo em `internal/sarif/regras.go`, uma regra por motivo do gate |
| `results[].ruleId` | `rules_hit[].id` |
| `results[].level` | `rules_hit[].nivel`, convertido (ver "Níveis") |
| `results[].message.text` | `rules_hit[].descricao` |
| `artifactLocation.uri` | `rules_hit[].evidencias`, o caminho do arquivo |
| `region.startLine` | patch do arquivo, lido do GitHub na hora do envio |
| `partialFingerprints.primaryLocationLineHash` | SHA-256 de motivo e arquivo |
| `results[].properties.recibo` | id e SHA-256 do recibo assinado da análise |
| `runs[].properties.aprova` | `risk_gates`: nível determinístico, nível final, estado da calibração e os motivos que não viraram resultado |

A linha é o único dado que não vem de `risk_gates`: o gate grava o nome do
arquivo, não o patch. A leitura dos arquivos do PR no envio serve só para
escolher a linha. Se ela falhar, os resultados saem na linha 1.

## Níveis

| Nível do motivo no gate | Nível no SARIF |
| --- | --- |
| alto | `warning` |
| médio | `note` |

O GitHub reprova o check do code scanning quando há resultado `error` (ou de
severidade de segurança alta). Se um motivo alto virasse `error`, todo
PR de risco alto ganharia um segundo check vermelho, que continuaria vermelho
mesmo depois da aprovação humana. Como o SARIF não decide, ele também não
reprova: quem bloqueia o merge é só o Check Run `aPRova`.

## Localização por motivo

| Motivo | Resultados | Onde |
| --- | --- | --- |
| `dependencia_nova` | um por manifesto | primeira linha adicionada que cita uma dependência nova; sem nenhuma, a primeira linha adicionada |
| `segredo_versionado`, `configuracao_ci`, `area_sensivel` | um por arquivo | primeira linha adicionada |
| `muitos_arquivos`, `codigo_sem_teste` | um só | primeiro arquivo da evidência, em ordem alfabética, na primeira linha adicionada |
| `falha_sandbox` | nenhum | `runs[].properties.aprova.motivosSemResultado`, com a razão `sem_arquivo` |
| `calibracao_historico` | nenhum | `runs[].properties.aprova.calibracao` |

`muitos_arquivos` e `codigo_sem_teste` falam do PR inteiro. O arquivo é só a
âncora do alerta, e a mensagem diz isso.

`falha_sandbox` fica fora porque o resultado do sandbox não sabe qual arquivo
falhou. Ancorar o alerta em um arquivo qualquer criaria uma anotação em uma
linha sem relação com a falha.

Casos de borda:

- **Patch ausente** (arquivo grande ou binário): resultado no arquivo, linha 1.
  O GitHub só anota no PR linhas que estão no diff, então esse caso vira alerta
  sem anotação.
- **Arquivo apagado**: não vira resultado, porque o arquivo não existe no commit
  analisado. O motivo vai para `motivosSemResultado` com a razão
  `arquivo_removido`.

## Impressão do alerta

O GitHub usa `partialFingerprints.primaryLocationLineHash` para saber se dois
resultados são o mesmo alerta. O aPRova calcula essa impressão a partir do
motivo e do arquivo, sem a linha. Um push novo que desloca o código não cria um
alerta duplicado.

Nos motivos do PR inteiro a impressão usa só o motivo, porque o arquivo de
âncora pode mudar de um push para o outro.

## Determinismo

A mesma decisão gravada e os mesmos arquivos produzem os mesmos bytes, em
qualquer ordem de leitura dos arquivos. O teste compara a saída com um arquivo
de referência versionado (`internal/sarif/testdata/todos_os_motivos.sarif`).

## Envio

`POST /repos/{owner}/{repo}/code-scanning/sarifs`, com:

- `sarif`: o documento em gzip e depois base64;
- `commit_sha`: o head analisado;
- `ref`: `refs/pull/<número>/head`;
- `tool_name`: `aPRova SARIF`, o mesmo nome declarado no documento.

O `202` só diz que o GitHub recebeu o arquivo. O enviador então consulta
`GET .../sarifs/{id}` a cada 5 segundos, por até 2 minutos, até `complete` ou
`failed`.

| Estado em `sarif_uploads.status` | Significado |
| --- | --- |
| `aceito` | o GitHub recebeu o arquivo e o processamento ainda não tem desfecho conhecido |
| `processado` | o GitHub processou o arquivo |
| `recusado` | o GitHub recusou o arquivo no processamento; os erros ficam em `detail` |
| `falhou` | o GitHub não aceitou o envio; é o único estado sem id |

| Resposta do GitHub | O que o enviador faz |
| --- | --- |
| 5xx, 429, ou 403 com limite de requisições | nova tentativa, com espera que dobra, até `WORKER_MAX_TENTATIVAS` |
| 403 sem limite de requisições | grava `falhou` e desliga o envio para aquele repositório |
| Outros 4xx | grava `falhou`, sem nova tentativa |

### Idempotência

O id do envio fica em `sarif_uploads` (migration `000022`). Na retomada:

- com id e estado `processado` ou `recusado`, nada é chamado;
- com id e estado `aceito`, só o processamento é consultado de novo;
- sem id (`falhou`), o envio é refeito.

O banco nunca troca o id de um envio aceito.

Dois pedidos simultâneos do mesmo task_run no mesmo processo viram um envio só.
Entre réplicas do worker pode acontecer um envio repetido; ele não cria alerta
duplicado, por causa da impressão.

### Envio desligado para um repositório

Um 403 sem limite de requisições quer dizer que o app não tem
`security_events: write` ou que o code scanning não está disponível. Repetir
não adianta, então o repositório entra em `sarif_disabled_repositories` com o
motivo, e os próximos task_runs dele não chamam o GitHub.

Depois de corrigir a permissão, religue apagando a linha:

```sql
DELETE FROM sarif_disabled_repositories WHERE repository = 'dono/repositorio';
```

### Encerramento do worker

Ao desligar, o worker interrompe os envios em curso. Um envio que já tinha id
fica como `aceito`; o alerta chega ao GitHub do mesmo jeito, só o desfecho do
processamento deixa de ser registrado.

## O que nunca é enviado

- O diff e o conteúdo dos arquivos. O SARIF leva caminho e linha, nunca código.
- A saída do sandbox. `falha_sandbox` leva só a contagem que já está na
  evidência do gate.
- Os vizinhos da calibração. Eles citam outros pull requests do tenant; o SARIF
  leva só o estado da calibração e se ela reduziu o nível.
- A assinatura e o conteúdo do recibo. Vão o id e o hash, que bastam para ligar
  o alerta à prova em `/receipts/{id}/verify`.
- Segredos. `segredo_versionado` aponta o arquivo; o valor não é lido.

O motivo é o mesmo em todos os casos: o code scanning é visível para quem tem
acesso de leitura aos alertas do repositório, e o SARIF só precisa dizer onde
olhar.

## Validação

O schema oficial do SARIF 2.1.0 (errata 01) está versionado em
`internal/sarif/testdata/sarif-schema-2.1.0.json`. Os testes validam contra ele,
sem rede, o SARIF do PR #1 do aPRova-TESTES (válido e sem resultados, porque o
único motivo dele é `falha_sandbox`) e o de cada motivo do gate.

## Ponta a ponta

Rodado em 2026-11-15 contra o PR #20 de `Azulpasta/aprova-bench-go`, criado só
para este teste (branches `base/sarif-e2e` e `solucao/sarif-e2e`), com a mesma
mudança da tarefa `risco-area-sensivel` da rodada 1 do benchmark em um commit
novo. Worker local, com `SARIF_LIGADO=true` só naquela execução.

| O que | Resultado |
| --- | --- |
| Gate | alto, por `area_sensivel` (alto) e `codigo_sem_teste` (médio) |
| Check Run `aPRova` (app a-prova) | `action_required`, merge bloqueado |
| Envio | aceito 3 s depois da transição para `aguardando_aprovacao` e processado 6 s depois de aceito |
| `sarif_uploads` | `processado`, com o id do envio |
| Alertas | dois, em `auth/senha.go`, linha 3: `area_sensivel` como `warning` e `codigo_sem_teste` como `note` |
| Anotações no PR | duas, na linha 3 do diff, em uma review de `github-advanced-security[bot]` |
| Check do code scanning | `success`, com o título "2 new alerts" |

- PR: <https://github.com/Azulpasta/aprova-bench-go/pull/20>
- Alertas do PR na aba Security:
  <https://github.com/Azulpasta/aprova-bench-go/security/code-scanning?query=pr%3A20>
- Check do code scanning:
  <https://github.com/Azulpasta/aprova-bench-go/runs/132842631042>

O ponta a ponta rodou com o nome antigo da ferramenta, `aPRova`. O nome atual,
`aPRova SARIF`, foi adotado por causa do que esse teste mostrou (seção
seguinte) e não passou por um ponta a ponta próprio. O PR #20 foi fechado e as
branches dele apagadas depois do teste.

## O nome da ferramenta

A ferramenta se chama `aPRova SARIF`, e não `aPRova`, para que o check que o
code scanning cria no pull request nunca tenha o nome do Check Run obrigatório.

A documentação do GitHub chama esse check de "Code scanning results". No ponta
a ponta, para um SARIF enviado pela API, o check veio com o nome da ferramenta,
criado pelo app `github-advanced-security`. Com a ferramenta chamada `aPRova`,
o PR ficou com dois checks `aPRova`, um de cada app.

Isso era um risco para quem exige o check só pelo nome. Uma regra de proteção
que pede "um check chamado `aPRova`" poderia ser satisfeita pelo check do code
scanning, que fica `success` porque nenhum resultado sai como `error`. O merge
seria liberado sem a decisão do aPRova.

Duas proteções fecham esse caminho, e uma não depende da outra:

- o nome da ferramenta é diferente do nome do Check Run, e um teste garante
  isso comparando com `publicacao.NomeDoCheckRun`;
- o check obrigatório é amarrado ao app a-prova: na `main` do aPRova-TESTES pelo
  `app_id` e nos rulesets dos `aprova-bench-*` pelo `integration_id` 4810767.
  Assim, nenhum check de mesmo nome vindo de outro app conta.

Ao instalar o aPRova em um repositório novo, amarre o check obrigatório ao app
antes de ligar `SARIF_LIGADO`.

Trocar o nome muda a ferramenta para o code scanning. Os alertas enviados com
o nome antigo continuam sob a ferramenta `aPRova` e não se juntam aos novos.
