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

O Gosec avulso não entende `//nolint:gosec`, só `#nosec`. As linhas que o golangci-lint aceita com `//nolint:gosec` justificado (G304, G204, G115) voltam a aparecer pelo Gosec: na dev, 40 apontamentos em 19 arquivos. GolangCI-Lint e Staticcheck saem limpos.

## Issues ignoradas pela interface

### `github.com/docker/docker` v28.5.x (Trivy, `go.mod`)

- **Origem:** o módulo entra no `go.mod` só pelo grafo de teste do `golang-migrate` (`migrate/v4/database/pgx/v5.test` → `github.com/dhui/dktest` → `github.com/docker/docker`). `go mod why -m github.com/docker/docker` mostra esse caminho.
- **Não entra no build:** nenhum pacote deste projeto importa `docker/docker`. O `testcontainers-go` v0.45.0 já usa `github.com/moby/moby/api` e `github.com/moby/moby/client`.
- **Não há versão corrigida nesse caminho:** o Trivy sugere 29.3.1, mas não existe `docker/docker` v29. A partir do Moby 29 o projeto é publicado como `github.com/moby/moby`. Até o `golang-migrate` v4.20.1 ainda declara `docker/docker` v28.5.2.
- **Quando revisar:** quando o `golang-migrate` trocar o `dhui/dktest` ou passar a declarar `github.com/moby/moby`; aí `go get -u github.com/golang-migrate/migrate/v4 && go mod tidy` tira o módulo do `go.mod`.

## Testes de integração e Docker

O pacote `tests/ambiente` decide o que acontece quando o Docker não é encontrado: com `CI=true` o teste reprova; fora do CI ele é pulado com a instrução de como ligar o Docker. Com o Colima, o testcontainers precisa de:

```sh
export DOCKER_HOST=unix://$HOME/.colima/default/docker.sock
export TESTCONTAINERS_DOCKER_SOCKET_OVERRIDE=/var/run/docker.sock
```
