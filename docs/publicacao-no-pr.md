# Publicação do resultado no pull request

O aPRova leva o resultado de cada análise ao próprio pull request de duas
formas: um **Check Run** chamado `aPRova`, que a proteção de branch usa para
liberar ou bloquear o merge, e **um comentário** na conversa do PR, que
explica a decisão em português e mostra como verificar o recibo.

O aPRova nunca faz merge. Quem bloqueia é o GitHub: quando o check `aPRova`
está marcado como obrigatório na proteção da branch, o botão de merge só
libera com o check numa conclusão aceita.

O código está em `internal/publicacao` (o `Publicador`), em
`internal/platform` (as chamadas à API) e em `internal/worker` (quando o motor
publica).

## Quando o motor publica

| Momento | Check Run | Comentário |
| --- | --- | --- |
| O task_run entra em `mapeando_contexto` | criado `in_progress` | — |
| Risco baixo, ao entrar em `aplicando` e antes de `concluido` | `success` | criado ou editado |
| Risco médio, ao entrar em `aplicando` e antes de `concluido` | `success`, com os pontos de atenção | criado ou editado |
| Risco alto, antes de ir para `aguardando_aprovacao` | `action_required`, com `details_url` na verificação do recibo | criado ou editado |
| Aprovação humana (`aguardando_aprovacao` → `aplicando`) | `success`, "aprovado por @login" | editado |
| `rejeitado` | não muda: já está em `action_required` | não muda |
| `falhou` | `neutral`, com o diagnóstico | não muda |
| Substituído por commit mais novo | nada | nada |

O bloqueio de risco alto é publicado **antes** da transição para
`aguardando_aprovacao`. Assim, se o GitHub falhar, a retomada volta a
`avaliando_risco`, reaproveita a decisão e o recibo já gravados e publica de
novo, sem emitir outro recibo. Se a publicação viesse depois da transição, um
erro deixaria o task_run parado esperando aprovação com o Check Run ainda em
andamento no PR.

Na aprovação, o login publicado é o de quem decidiu. O cursor de aprovação
(`last_approval_seen`) para logo antes da aprovação que liberou o PR: as
anteriores, de quem só tem leitura, foram puladas, e ele não anda depois da
decisão. Por isso a próxima aprovação depois do cursor é sempre a que decidiu.

### O caso do `neutral`

O card pede `neutral` para um task_run que falhou, para o PR não ficar com um
check em andamento para sempre. O efeito na proteção de branch precisa ficar
claro: o GitHub aceita `success`, `neutral` e `skipped` como conclusões que
satisfazem um check obrigatório. **Uma análise que falha não bloqueia o
merge.** O texto do Check Run diz isso. Se a decisão for bloquear também nesse
caso, a troca é de `neutral` para `failure` ou `action_required`, num card
próprio.

## Onde ficam os ids

- **O id do Check Run** fica em `task_runs.check_run_id`. O Check Run é por
  commit, e cada task_run analisa um commit, então a relação é de um para um.
- **O id do comentário** fica na tabela `pull_request_comments`, com chave
  primária `(tenant_id, repository, pr_number)`. O comentário é por pull
  request e sobrevive a vários task_runs, um a cada push. Se ele morasse em
  `task_runs`, cada análise nova teria de copiar o id da anterior, e "um
  comentário por PR" seria só uma convenção do código. Com a chave primária, é
  o banco que garante.

## Idempotência

O worker pode morrer entre criar algo no GitHub e gravar o id no banco. Na
retomada, o `Publicador`:

- **Check Run:** com id gravado, atualiza. Sem id gravado, procura nos check
  runs de nome `aPRova` do `head_sha` o que tem `external_id` igual ao id do
  task_run. Achou, grava o id e atualiza; não achou, cria.
- **Comentário:** com id gravado, edita. Se o comentário foi apagado à mão (o
  GitHub responde 404), ou se não há id gravado, procura na conversa do PR um
  comentário escrito por `<slug do app>[bot]` que contenha a marca invisível
  `<!-- aprova:resultado-da-analise -->`. Achou, grava o id e edita; não achou,
  cria. Exigir o autor e a marca juntos impede que o aPRova edite o comentário
  de uma pessoa que copiou o texto dele.

Por isso o worker exige `GITHUB_APP_SLUG`: é com ele que reconhece o próprio
comentário.

## Erros do GitHub

| Resposta | Classificação | O que o motor faz |
| --- | --- | --- |
| 5xx, ou 429 / 403 com limite de requisições | transitório | nova tentativa, com espera que dobra |
| 403 sem limite de requisições | `platform.ErrSemPermissao` | `falhou`, com o diagnóstico "confira checks: write e pull_requests: write" |
| Outros 4xx | definitivo | `falhou`, com a resposta no detalhe |

Quando o task_run falha, o motor tenta fechar o Check Run como `neutral`. Se
nem isso der certo, por exemplo pelo mesmo 403, o task_run termina em `falhou`
assim mesmo e o diário registra que o check pode ter ficado em andamento.

## Permissões do app

O app precisa de `checks: write` e `pull_requests: write`. No GitHub, todo
pull request é também uma issue, e os comentários da conversa usam a API de
issues; para um número que é de PR, `pull_requests: write` basta e
`issues: write` não é necessária.

## O comentário

É montado só a partir dos motivos do gate, sem modelo de IA. Mostra o nível e
o que ele significa para o merge, cada motivo com as evidências (até 20 por
motivo), o resultado do sandbox, o recibo (hash, `key_id`, link para
`/receipts/{id}/verify` e link para `docs/verificacao-recibo.md`) e o commit
analisado em forma curta. Evidências com crase são escritas com uma cerca
maior, para não quebrar o Markdown.

O link para `docs/verificacao-recibo.md` aponta para a branch `main`. Ele só
abre depois que essa documentação chegar à `main`. Até lá, o link de
verificação do recibo e o próprio passo a passo estão na `dev`.
