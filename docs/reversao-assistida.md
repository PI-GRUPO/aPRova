# Reversão assistida

Quando uma análise do aPRova deixou passar algo que não devia, a pergunta
seguinte é: **o que exatamente entrou na base do PR por causa dela, e como
desfazer?**
A reversão assistida responde a partir do recibo. Ela confere o recibo,
encontra o commit que entrou na base do PR (em geral a `main`) e devolve o que uma pessoa precisa para
reverter: o comando `git` pronto e o título e o corpo do PR de reversão.

## O que ela nunca faz

O aPRova **não executa nada**: não cria branch, não faz commit, não abre PR e
não faz merge. Quem reverte é uma pessoa, na própria máquina, com as próprias
credenciais. O PR de reversão passa pelo aPRova como qualquer outro.

A reversão assistida só lê:

- do banco, o recibo e o desfecho do PR (`pull_request_outcomes`);
- do GitHub, um único `GET /repos/{dono}/{repo}/commits/{sha}`, para contar os
  pais do commit que entrou.

Essa leitura é coberta por `contents: read`. **O "só leitura" vale para
`contents` e para esta funcionalidade, não para o app inteiro.** O app tem
`checks: write` e `pull_requests: write` para publicar o check e o comentário
da análise, como descrito em [publicacao-no-pr.md](publicacao-no-pr.md#permissões-do-app).
Nada da reversão assistida usa essas permissões, e um teste garante isso: o
cliente do GitHub usado nos testes da rota falha em qualquer `POST`, `PUT`,
`PATCH` ou `DELETE`.

## Por que a versão completa ficou para depois

A versão completa abriria o PR de reversão sozinha. Para isso o app precisaria
criar uma branch e um commit no repositório, o que exige `contents: write`.
Com essa permissão, um defeito ou um vazamento de credencial no aPRova passaria
a poder **escrever código** nos repositórios instalados, não só comentar. É uma
mudança de categoria de risco, para uma economia pequena: a pessoa que reverte
já está decidindo reverter e só precisa colar um comando. Por isso a versão
atual entrega a instrução e deixa a execução com quem tem o direito de
escrever.

## Como o commit que entrou é encontrado

O recibo assinado guarda o repositório, o número do PR e o `head_sha`
analisado. Quando o PR é fechado, o webhook `closed` grava em
`pull_request_outcomes` se ele foi mergeado, a base (`pull_request.base.ref`,
coluna `base_ref`, migração 24), o head no fechamento, o `merge_commit_sha` e
quantos commits o PR tinha (coluna `commits`, migração 23). A ligação entre recibo e desfecho é `(tenant, repositório, número do PR)`.

O GitHub informa o `merge_commit_sha` nas três formas de merge, mas não diz
qual delas foi usada. O comando de reversão muda conforme a forma:

| Pais do commit | Commits no PR | Forma | Comando |
| --- | --- | --- | --- |
| 2 | qualquer | merge commit | `git revert -m 1 <sha>` |
| 1 | 1 | squash, ou rebase de um commit | `git revert <sha>` |
| 1 | mais de 1 | squash **ou** rebase | nenhum: ambíguo |

No merge commit, `-m 1` diz ao git que o lado a preservar é o primeiro pai, a
base de antes do merge. No squash, o PR inteiro virou um commit só. No rebase
de N commits, o `merge_commit_sha` é a ponta dos N commits reaplicados em
sequência na base, e a reversão é a faixa `<sha>~N..<sha>`.

Com um pai e mais de um commit no PR, squash e rebase têm a mesma aparência
vista de fora. Daria para desempatar com outra leitura no GitHub
(`/commits/{sha~1}/pulls`), mas essa associação entre commit e PR é a parte
menos confiável da API. **Na dúvida, o aPRova não sugere**: devolve o aviso e
pede o tipo, que no executável vem por `-tipo`.

PRs fechados antes da migração 23 têm `commits = 0`. Para eles, só o merge
commit (2 pais) gera comando; o resto cai no caso ambíguo.

O comando sempre parte de uma branch nova a partir da base remota do PR, que
nem sempre é a `main`. No ensaio de ponta a ponta, o PR mirava
`base/reversao-e2e`, e um comando fixo em `origin/main` teria revertido na
branch errada:

```sh
git fetch origin && git switch -c revert/pr-21 origin/base/reversao-e2e && git revert --no-edit -m 1 <sha>
```

PRs fechados antes da migração 24 não têm a base gravada. Para eles não há
comando: a recusa é `base_desconhecida`, porque supor a `main` é exatamente o
erro que a coluna nova corrige.

## Quando não há instrução

| `recusa` | Quando |
| --- | --- |
| `recibo_invalido` | a assinatura, o hash, a chave, o conteúdo ou o elo com o anterior não conferem |
| `pr_aberto` | o PR ainda não foi fechado |
| `pr_nao_mergeado` | o PR foi fechado sem merge; nada dele entrou na base |
| `commit_diferente` | houve push depois da análise: o recibo não cobre o que entrou |
| `tipo_ambiguo` | um pai e mais de um commit, sem o tipo informado |
| `base_desconhecida` | o PR foi fechado antes da migração 24, ou o executável não recebeu `-base` |
| `dados_do_merge_invalidos` | SHA ou base fora do formato, tipo desconhecido ou rebase sem a contagem de commits |

Recibo inválido não consulta banco nem GitHub: a recusa sai antes de qualquer
leitura. No caso `commit_diferente`, a explicação mostra os dois SHAs, o
analisado e o que entrou.

Os SHAs e a base entram num comando de shell. Os SHAs só são aceitos em
hexadecimal minúsculo de 40 ou 64 caracteres. A base é mais delicada: o git
aceita nomes de branch com `;`, `$` e crase, que o shell interpretaria. Por
isso só passam segmentos de letras, dígitos, `.`, `_` e `-`, separados por `/`,
sem começar com `-` e sem `..`. Qualquer outra coisa é recusada em vez de
virar comando.

## O texto do PR de reversão

O corpo começa exatamente com a linha que o botão "Revert" do GitHub usaria:

```text
Reverts Azulpasta/aPRova-TESTES#1
```

Essa linha é o sinal para a calibração. Quando o PR de reversão é mergeado, o
webhook `closed` reconhece a linha e marca o PR original como revertido em
`pull_request_reverts`; a calibração passa a tratar aquele PR como vizinho de
desfecho negativo, que impede reduzir o nível de casos parecidos (ver
[calibracao.md](calibracao.md)). Um
teste da rota passa o corpo gerado pelo mesmo detector que o webhook usa.

Depois dessa linha, o corpo diz em português qual commit está sendo revertido
e em que base ele entrou, o nível e os motivos do recibo original com as
evidências (ou "sem motivos de risco", quando o recibo não tem nenhum), o link de
verificação do recibo e o aviso de que o PR de reversão também passa pelo
aPRova.

## Pela API

```sh
curl -s "$APROVA/receipts/$ID/revert"
```

O link do recibo no corpo usa `APROVA_URL_PUBLICA` da API; sem ela, sai o
endereço provisório `https://aprova.example.com`.

A resposta traz `verificacao`, o mesmo objeto de `/receipts/{id}/verify`, mais
`repositorio`, `numero_pr`, `head_sha` e `commit_na_main`; `disponivel`
verdadeiro com `tipo`, `comando`, `titulo` e `corpo`, ou falso com `recusa` e
`explicacao`.

| Status | Quando |
| --- | --- |
| 200 | com instrução ou com a recusa explicada |
| 404 | recibo inexistente ou id malformado |
| 502 | a leitura do commit no GitHub falhou |
| 503 | sem as credenciais do GitHub App ou sem verificador de recibos |

A rota é só de leitura: `POST`, `PUT`, `PATCH` e `DELETE` respondem 405.

## Pelo executável, sem banco e sem rede

O subcomando `reverter` do `verificar-recibo` faz a mesma coisa a partir do
JSON do recibo (`GET /receipts/{id}`) e dos dados do merge, que quem reverte lê
no PR fechado do GitHub. Sem rede não dá para contar os pais do commit, então
o tipo vem por `-tipo`:

```sh
go run ./cmd/verificar-recibo reverter \
  -chave "$CHAVE" -url-publica "$APROVA" \
  -anterior recibo-anterior.json \
  -base main -merge-sha <merge_commit_sha> -head-final <head no fechamento> \
  -tipo rebase -commits 3 \
  recibo.json
```

- Sem `-merge-sha`, o PR é tratado como não mergeado.
- `-base` é a branch em que o PR entrou; sem ela não há comando.
- `-head-final` é o head do PR no fechamento; se não for o `head_sha` do
  recibo, a saída avisa que o recibo não cobre o que entrou.
- `-anterior` é opcional. Sem ele, quando o recibo cita um anterior, a saída
  começa com `ATENÇÃO: ELO NÃO CONFERIDO`, porque a assinatura sozinha não
  prova que o recibo não foi tirado da cadeia. O primeiro recibo da cadeia não
  precisa dele.

Sai com 0 quando há instrução, 1 quando não há (recibo inválido ou recusa) e 2
quando a chave, o recibo ou as opções não puderam ser lidos.

## Onde está o código

- `internal/reversao`: a regra, pura, sem banco nem rede, usada pela API e
  pelo executável.
- `internal/api/reversao_de_recibo.go`: a rota.
- `internal/platform/commits.go`: a leitura dos pais do commit.
- `cmd/verificar-recibo/reverter.go`: o subcomando.
