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
| `falhou` | `failure`, com o diagnóstico | não muda |
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

### A aprovação vale para o commit revisado

Cada revisão chega do GitHub com o `review.commit_id`, o commit que o revisor
viu, e a aprovação guarda esse commit em `approvals.commit_sha`. Ela só decide
o task_run cujo `head_sha` é esse commit. A API registra a revisão no task_run
mais recente do PR; se o commit revisado não for o analisado, porque houve um
push depois da revisão, a aprovação fica registrada, não decide nada, e o
worker escreve no log os dois SHAs (`commit_revisado` e `commit_analisado`).
Assim, uma aprovação de um commit antigo, mesmo que chegue atrasada ou seja
reenviada, nunca libera um commit que o revisor não viu.

### Análise que falhou: `failure`

Quando o task_run termina em `falhou` (GitHub fora do ar até esgotar as
tentativas, sandbox que não roda, gate sem veredito, falta de permissão), o
Check Run é concluído como `failure`, com o diagnóstico no resumo.

O princípio do projeto é **na dúvida, bloqueia**. Uma análise que não terminou
é dúvida: o aPRova não sabe se o pull request é seguro. Por isso o check não
fica `neutral`. O GitHub aceita `success`, `neutral` e `skipped` como
conclusões que satisfazem um check obrigatório, então `neutral` liberaria o
merge sem análise.

O texto do check diz o diagnóstico e o caminho: uma nova análise, que um novo
push no pull request dispara. O botão Reexecutar vem com o dashboard.

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

Quando o task_run falha, o motor tenta fechar o Check Run como `failure`. Se
nem isso der certo, por exemplo pelo mesmo 403, o task_run termina em `falhou`
assim mesmo e o diário registra que o check pode ter ficado em andamento. Um
check em andamento também não satisfaz a proteção de branch, então o merge
continua bloqueado.

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

## Roteiro da demonstração

O caso da demonstração é o PR #1 de `Azulpasta/aPRova-TESTES`, branch
`teste-peso-zero`, com um teste que falha de propósito. A proteção da `main`
desse repositório exige só o check `aPRova`, amarrado ao app pelo `app_id`, e
tem `enforce_admins` ligado: nem um admin vê o atalho de "bypass".

Para deixar o PR bloqueado antes da apresentação:

1. Se o PR tiver uma aprovação, dispense-a no GitHub (na review, "Dismiss
   review"), com o motivo "reanálise para a demonstração".
2. Faça um commit vazio na branch e envie:

   ```bash
   git switch teste-peso-zero
   git commit --allow-empty -m "chore: reanálise do aPRova"
   git push
   ```

3. Com a API, o worker e o repasse do webhook rodando, o push chega como
   `synchronize`, e o aPRova analisa o commit novo. O PR fica com:
   - o Check Run `aPRova` em `action_required` no commit novo, por
     `falha_sandbox`;
   - o merge bloqueado pela proteção de branch;
   - o comentário do aPRova editado com o commit novo, os motivos e o hash do
     recibo.

A aprovação dispensada não libera o commit novo por dois motivos: ela é de
outro commit, e é anterior ao bloqueio (o aPRova só aceita aprovação feita
depois de o PR entrar em espera).

Durante a apresentação, quem tem permissão de escrita aprova o PR no GitHub.
O Check Run passa a `success` com "aprovado por @login", e o merge fica
disponível. **Não faça o merge**: o PR #1 é o caso da demonstração e precisa
continuar aberto. Depois, repita os passos 1 e 2 para voltar ao bloqueio.

Para rodar localmente, enquanto o núcleo não tem endereço público, o webhook do
app entrega num canal do smee.io, repassado para a API local:

```bash
make up && make migrate-up
make run-api
make run-worker
npx smee-client --url <canal do smee configurado no app> --target http://localhost:<PORT>/webhooks/github
```
