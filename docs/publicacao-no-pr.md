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

O link para `docs/verificacao-recibo.md` aponta para a branch `main`, onde o
passo a passo já está publicado.

## Roteiro da demonstração

O caso da demonstração é o PR #6 de `PI-GRUPO/aPRova-TESTES`, da branch
`teste-peso-zero` para a `main`, com um teste Go que falha de propósito
(`TestPesoZeroRetornaErro`, descrito no `BUG.md`). A proteção da `main` desse
repositório exige só o check `aPRova`, amarrado ao app pelo `app_id`, e tem
`enforce_admins` ligado: nem um admin vê o atalho de "bypass". O check `Go` do
CI do próprio repositório também falha, pelo mesmo teste, mas não é
obrigatório.

O PR #6 foi aberto pela conta `Azulpasta`, e o GitHub não deixa o autor aprovar
o próprio pull request. A aprovação precisa vir de outra conta com permissão de
escrita no repositório. Na demonstração, ela é feita pelo Slack, com o GitHub
como plano B.

### Só uma instância do aPRova no canal do smee

O smee.io entrega cada evento a **todos** os clientes conectados no canal.
Se duas pessoas rodam o aPRova com o mesmo canal, as duas analisam cada push:
o commit ganha dois Check Runs `aPRova`, cada um com um recibo assinado por uma
chave diferente, e a aprovação é processada duas vezes. Os dois aparecem no
GitHub como o mesmo app `a-prova`; o que distingue um do outro é o `key_id` do
recibo no resumo do check.

Em 2026-11-20, mesmo depois de o grupo combinar quem rodaria o aPRova, uma
segunda instância continuou analisando os pushes do PR #6. O webhook do app foi
então trocado para um canal novo do smee. A URL do canal fica só no `.env` de
quem apresenta, na variável `SMEE_URL`, e nunca nesta documentação nem em outro
arquivo versionado: quem conhece a URL recebe todos os eventos do app e passa a
ser mais uma instância no canal.

O teste de que só há uma instância é o próprio commit vazio do passo 2 abaixo:
depois dele, o commit novo deve ter um único Check Run `aPRova`, com o `key_id`
da sua instância.

### Subir a demonstração

O webhook do app entrega no canal do smee.io de `SMEE_URL`, repassado para a
API local. O código Go do repositório de teste fica na pasta `go/`, então o worker precisa
de `SANDBOX_DIRETORIOS` e de `SANDBOX_LINGUAGENS` para o repositório. Sem
`SANDBOX_DIRETORIOS`, a análise falha com "repositório sem go.mod na raiz". As
variáveis vão na linha do comando: o `godotenv` não sobrescreve o que já está no
ambiente, então o `.env` não muda. Com `APROVA_URL_PUBLICA` apontando para a API
local, o link "Verificar no aPRova" do check abre na máquina da apresentação;
sem ela, o link usa o endereço provisório `https://aprova.example.com`.

A API e o worker sobem com `go run`, e não com `make run-api` e
`make run-worker`: o `Makefile` exporta um `DATABASE_URL` padrão, na porta
5432, e o `godotenv` não o sobrescreve com o do `.env`. Com o Postgres do
projeto em outra porta, os alvos do `make` conectam no banco errado.

```bash
make up && make migrate-up
go run ./cmd/api
SANDBOX_DIRETORIOS="PI-GRUPO/aPRova-TESTES=go" \
SANDBOX_LINGUAGENS="PI-GRUPO/aPRova-TESTES=go" \
APROVA_URL_PUBLICA="http://localhost:8080" \
N8N_WEBHOOK_URL="" \
go run ./cmd/worker
npx smee-client --url "$(grep '^SMEE_URL=' .env | cut -d= -f2-)" --target http://localhost:8080/webhooks/github
```

Se o `.env` já define `SANDBOX_DIRETORIOS` ou `SANDBOX_LINGUAGENS` para outros
repositórios, repita essas entradas na linha do comando, separadas por
vírgula: o valor da linha substitui o do `.env` inteiro.

### Deixar o PR bloqueado

1. Se o PR tiver uma aprovação, dispense-a no GitHub (na review, "Dismiss
   review"), com o motivo "reanálise para a demonstração".
2. Faça um commit vazio a partir da branch da organização e envie. O clone
   local de `aPRova-TESTES` pode apontar para outro remoto, então o comando
   busca e envia pela URL do repositório da organização:

   ```bash
   git fetch https://github.com/PI-GRUPO/aPRova-TESTES.git teste-peso-zero:org-teste-peso-zero
   git switch org-teste-peso-zero
   git commit --allow-empty -m "chore: reanálise do aPRova"
   git push https://github.com/PI-GRUPO/aPRova-TESTES.git org-teste-peso-zero:teste-peso-zero
   ```

3. Com a API, o worker e o smee rodando, o push chega como `synchronize`, e o
   aPRova analisa o commit novo. O PR fica com:
   - um único Check Run `aPRova`, em `action_required` no commit novo, por
     `falha_sandbox`;
   - o merge bloqueado pela proteção de branch;
   - o comentário do aPRova editado com o commit novo, os motivos e o hash do
     recibo.

A aprovação dispensada não libera o commit novo por dois motivos: ela é de
outro commit, e é anterior ao bloqueio (o aPRova só aceita aprovação feita
depois de o PR entrar em espera).

### Durante a apresentação

Quem tem permissão de escrita aprova o PR pelo Slack, no alerta de risco alto,
com uma justificativa. O fluxo do Slack publica a review pelo app, com o texto
"Aprovado por @login via Slack:" seguido da justificativa. O webhook dessa
review chega à instância local pelo smee, e o aPRova registra a decisão em nome
de quem clicou. O Check Run passa a `success` com "aprovado por @login", a
aprovação é gravada em `approvals` com o `commit_sha` do commit analisado, a
origem `slack` e a justificativa, e o merge fica disponível.

Se o Slack não responder, o plano B é aprovar pelo GitHub: na aba "Files
changed", "Review changes" e "Approve". O resultado é o mesmo, com a origem
`github` e o corpo da review como justificativa.

**Não faça o merge**: o PR #6 é o caso da demonstração e precisa continuar
aberto. Depois, repita os passos 1 e 2 para voltar ao bloqueio.

No ensaio de 2026-11-20, a aprovação pelo Slack funcionou com a instância
local: o check do commit `2bba201` passou a `success` com "aprovado por
@julia-ed2", e `approvals` gravou a origem `slack`, o `commit_sha` do commit
analisado e a justificativa digitada no Slack.
