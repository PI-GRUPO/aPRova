# Análise estática no Codacy

O que o Codacy analisa neste repositório, o que fica de fora e por quê.

## Onde cada decisão mora

| Arquivo | O que controla | Quando vale |
| --- | --- | --- |
| `.codacy.yml` | Caminhos excluídos da análise inteira e de uma ferramenta inteira (`engines.<ferramenta>.exclude_paths`) | Só a versão da branch padrão |
| `.hadolint.yaml` | Regras do Hadolint | Versão da branch analisada, com "usar arquivo de configuração" ligado no Hadolint |
| `.markdownlint.json` | Regras do markdownlint | Versão da branch analisada, com "usar arquivo de configuração" ligado no markdownlint |
| Interface do Codacy | Issue individual ignorada | Imediatamente |

O `.codacy.yml` não desliga um padrão isolado nem ignora uma dependência específica. Para isso servem o arquivo de configuração da própria ferramenta ou a interface.

### O arquivo de configuração substitui o coding standard inteiro

Com "usar arquivo de configuração" ligado, o Codacy deixa de aplicar à ferramenta as regras do coding standard e passa a rodá-la só com o arquivo do repositório. O que o arquivo não menciona segue o padrão da própria ferramenta, e não o do Codacy.

Foi o que aconteceu ao ligar o `.markdownlint.json`: o arquivo só configurava o MD024, o markdownlint voltou com todas as regras ligadas e a análise ganhou 422 issues novas, 404 delas do MD013 (linha de até 80 colunas), que o coding standard deixava desligado. No Hadolint, o DL3018 apareceu pelo mesmo motivo.

Por isso, ao ligar o arquivo de uma ferramenta, compare a análise antes e depois e desligue no arquivo as regras que o coding standard já deixava de fora.

## Exclusões

| Caminho | Ferramenta | Motivo |
| --- | --- | --- |
| `tests/integration/testdata/**` | todas | Repositórios de exemplo, defasados ou quebrados de propósito, usados para testar o aPRova |
| `bench/suite/**`, `bench/resultados/**` | todas | Exercícios de terceiros do benchmark e saídas das rodadas |
| `design/**` | todas | Protótipo do dashboard, fora do produto |
| `**/*.sql` | TSQLLint | O SQL do projeto é PostgreSQL; o TSQLLint aplica regras de T-SQL |
| `tests/**`, `**/*_test.go`, `cmd/benchmark/**`, `internal/benchmark/**`, `workflows/n8n/tests/**` | Opengrep | Leitura de arquivo, subprocesso, `new Function` e caminho não literal nesses lugares vêm do próprio repositório |

Código de produção continua inteiro no Opengrep.

## Regras desligadas

| Regra | Arquivo | Motivo |
| --- | --- | --- |
| Hadolint DL3008 | `.hadolint.yaml` | As imagens base já são fixadas pelo digest sha256; fixar a versão de cada pacote do apt quebraria a build quando o Debian retirar a versão |
| Hadolint DL3018 | `.hadolint.yaml` | Mesmo motivo do DL3008, para o `apk add` da imagem do sandbox Go, que fixa a base Alpine pelo digest |
| markdownlint MD013 | `.markdownlint.json` | Markdown não quebra linha em 80 colunas; tabelas e parágrafos longos são normais nesta documentação |
| markdownlint MD024 só entre irmãos | `.markdownlint.json` | `docs/benchmark.md` repete títulos como "Métricas" em seções de rodadas diferentes |
| Opengrep `codacy.generic.sql.rac-table-access` | Interface (Code patterns) | Exige tabelas com prefixo `RAC_*`, convenção que não existe neste projeto; acusava 75 consultas e migrations |

## Ferramentas do lado do cliente

GolangCI-Lint, Gosec, Staticcheck, Aligncheck e Deadcode não rodam nos servidores do Codacy: o job `envio` do workflow `codacy-cliente` roda cada uma no CI e envia o resultado para o commit analisado.

| Ferramenta | Como roda | Conversor |
| --- | --- | --- |
| GolangCI-Lint | Mesma versão e `.golangci.yml` do job `verificacao` | `codacy/codacy-golangci-lint`, versão e SHA-256 fixados no workflow |
| Gosec | `go install` com versão fixada | `codacy/codacy-gosec`, versão e SHA-256 fixados |
| Staticcheck | `go install` com versão fixada | `codacy/codacy-staticcheck`, versão e SHA-256 fixados |
| Aligncheck e Deadcode | Codacy Analysis CLI com a imagem fixada por versão | Próprio CLI, com `--upload` |

`scripts/codacy-cliente.sh` baixa e confere cada conversor, converte, envia e, no fim, chama `resultsFinal`. Os conversores leem a entrada em ASCII, então o script reescreve os caracteres acentuados como escapes `\uXXXX` antes de converter; o JSON continua o mesmo.

### Ativação, nesta ordem

1. No Codacy, em Settings → API tokens, criar um token em "Repository API tokens".
2. No GitHub, gravar o token como secret do repositório com o nome `CODACY_PROJECT_TOKEN`.
3. No Codacy, em Settings → General → Repository analysis, ligar "Run analysis through build server".

Com o passo 3 ligado, a análise de cada commit só começa quando o CI chama `resultsFinal`. Por isso o job roda em todo push, sem filtro de caminho, e o passo final roda mesmo quando uma ferramenta falha. Sem o secret, o job avisa e não envia nada; ligar o passo 3 antes do 2 deixa as análises paradas esperando.

### O que esperar

Quando Aligncheck ou Deadcode falham, o Codacy Analysis CLI sai com código 0 e resultado vazio (`[]`): o job ficaria verde e o Codacy receberia zero issues dessas ferramentas sem nenhum aviso. Por isso a etapa usa `--fail-if-incomplete`, que transforma a falha da ferramenta em falha do passo; o `resultsFinal` é chamado do mesmo jeito.

Aligncheck e Deadcode não foram validados localmente. No Mac arm64 a imagem amd64 do CLI roda emulada: o Aligncheck caiu com código 139 (falha de segmentação) e o Deadcode estourou o limite de 930 segundos, sintomas da emulação e não do código. O comportamento real só aparece no primeiro envio, num runner amd64. Se as duas falharem lá, a saída é tirá-las do workflow e desligá-las no Codacy, mantendo GolangCI-Lint, Gosec e Staticcheck.

O Gosec avulso não entende `//nolint:gosec`, só `#nosec`. As linhas que o golangci-lint aceita com `//nolint:gosec` justificado (G304, G204, G115) voltam a aparecer pelo Gosec: na dev, 40 apontamentos em 19 arquivos. GolangCI-Lint e Staticcheck saem limpos.

## Issues ignoradas pela interface

### `github.com/docker/docker` v28.5.x (Trivy, `go.mod`)

- **Origem:** o módulo entra no `go.mod` só pelo grafo de teste do `golang-migrate` (`migrate/v4/database/pgx/v5.test` → `github.com/dhui/dktest` → `github.com/docker/docker`). `go mod why -m github.com/docker/docker` mostra esse caminho.
- **Não entra no build:** nenhum pacote deste projeto importa `docker/docker`. O `testcontainers-go` v0.45.0 já usa `github.com/moby/moby/api` e `github.com/moby/moby/client`.
- **Não há versão corrigida nesse caminho:** o Trivy sugere 29.3.1, mas não existe `docker/docker` v29. A partir do Moby 29 o projeto é publicado como `github.com/moby/moby`. Até o `golang-migrate` v4.20.1 ainda declara `docker/docker` v28.5.2.
- **Quando revisar:** quando o `golang-migrate` trocar o `dhui/dktest` ou passar a declarar `github.com/moby/moby`; aí `go get -u github.com/golang-migrate/migrate/v4 && go mod tidy` tira o módulo do `go.mod`.

### `os.Chmod` com `0o1777` em `cmd/sandbox/main.go:111` (Opengrep, duas regras)

Padrões `go.lang.correctness.permissions.file_permission.incorrect-default-permission` e `go_file-permissions_rule-fileperm`, que apontam o mesmo trecho:

```go
if err := os.Chmod(filepath.Dir(caminho), permissaoDoPaiCompartilhado); err != nil {
```

- **O que o trecho faz:** antes de criar a pasta do código do PR, `prepararParaOCodigoDoPR` abre o diretório pai com `0o1777`. O `mkdir` seguinte roda como o usuário sem privilégio do PR (uid 10001), que precisa escrever no pai.
- **Por que `1777`:** é a permissão do `/tmp`. Todos escrevem, mas o sticky bit impede apagar ou renomear o que é de outro usuário.
- **Onde vale:** o pai é o `/tmp` do container (para `/tmp/aprova`, o `Chmod` não muda nada) ou a raiz do volume de preparo de uma única execução, dentro de um container isolado e sem rede na etapa de teste. Não há outro usuário a proteger ali.
- **Por que a regra não se aplica:** ela compara com `0600`, a permissão de um arquivo comum, e não com um diretório compartilhado de propósito.

### URL do Postgres em `.github/workflows/ci.yml:103` (Checkov `CKV_SECRET_4`)

```yaml
URL_BANCO: postgres://aprova:aprova@localhost:5432/aprova?sslmode=disable
```

- **O que é:** a conexão do job `banco` com o Postgres que o próprio job sobe como serviço. As credenciais são as mesmas de `POSTGRES_USER` e `POSTGRES_PASSWORD`, poucas linhas acima.
- **Por que não é segredo:** o banco só existe dentro do runner e é destruído no fim do job; a senha não protege dado nenhum.
- **Por que não virar secret do GitHub:** esconderia a senha sem ganho de segurança e deixaria o job dependente de configuração externa.

## Trabalho para depois de 24/11/2026

### Pasta do código do PR sem diretório pai aberto para todos

Hoje o pai da pasta do PR recebe `0o1777` para o uid 10001 conseguir criar a pasta. A alternativa mais fechada:

1. Criar a pasta como root.
2. Passar a posse para o uid 10001 com permissão `0o700`.
3. Deixar o pai em `0o755`.

O pai deixa de ser gravável por todos e só o usuário do PR acessa a própria pasta. Muda o comportamento do sandbox (`prepararParaOCodigoDoPR` em `cmd/sandbox/main.go` e as etapas em `internal/sandbox/conteiner/etapas.go`), então precisa de teste antes e da integração com Docker (`tests/integration`) para validar. Quando entrar, as duas issues do Opengrep acima deixam de existir e saem desta lista.

## Testes de integração e Docker

O pacote `tests/ambiente` decide o que acontece quando o Docker não é encontrado: com `CI=true` o teste reprova; fora do CI ele é pulado com a instrução de como ligar o Docker. Com o Colima, o testcontainers precisa de:

```sh
export DOCKER_HOST=unix://$HOME/.colima/default/docker.sock
export TESTCONTAINERS_DOCKER_SOCKET_OVERRIDE=/var/run/docker.sock
```
