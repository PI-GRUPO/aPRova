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
