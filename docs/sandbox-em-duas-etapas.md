# Sandbox em duas etapas

O executor local roda o sandbox em dois contêineres, um depois do outro, sobre
a mesma imagem da linguagem:

| Etapa | Rede | O que pode fazer | O que nunca faz |
| --- | --- | --- | --- |
| `preparar` | `bridge` | clonar o PR no SHA pedido e baixar as dependências declaradas | executar qualquer código vindo do PR |
| `testar` | `--network none` | copiar o código preparado, instalar a partir do cache e rodar os testes | usar a rede, inclusive o host |

O código do PR só roda na etapa `testar`, que não tem rede nenhuma: nem DNS,
nem IP público, nem o gateway do host, onde ficam o Postgres e o Redis.

## Por que

Antes, um único contêiner clonava, baixava as dependências e rodava os testes.
Para analisar um PR real do GitHub, era preciso ligar a rede com
`SANDBOX_REDE_LOCAL=bridge`, e com ela o código do PR alcançava a rede inteira.

O teste `TestSandboxGoBaixaDependenciasNoPreparoETestaSemRede` (e o equivalente
em Python) usa um teste do PR que tenta resolver um nome por DNS e abrir
conexões TCP para `1.1.1.1:443` e para o gateway do host. Com `bridge`, antes
deste card, ele alcançou os três:

```
ataque: alcançou dns proxy.golang.org, tcp 1.1.1.1:443, tcp gateway 172.17.0.1:5432
```

Agora o mesmo teste falha com `rede inalcançável`.

## Como as etapas se ligam

1. O executor cria um volume Docker com nome próprio da execução
   (`aprova-preparo-<aleatório>`).
2. `preparar` sobe com o volume montado com escrita em `/aprova-preparo`,
   clona o PR em `/aprova-preparo/codigo` e grava as dependências em
   `/aprova-preparo/cache`. Publica no stdout `{"pronto":true}` ou a falha
   classificada.
3. Se o preparo não ficou pronto, a análise termina com essa falha e os testes
   não rodam.
4. `testar` sobe com `--network none` e o volume montado **somente leitura**.
   Confere com `git rev-parse` que o código preparado está no SHA pedido,
   copia o código para o tmpfs com o dono uid 10001, instala só a partir do
   cache e roda os testes como hoje.
5. O volume é removido ao fim, com sucesso, falha ou timeout.

O volume, e não um diretório do host, por dois motivos:

- os arquivos gravados pelo uid 10001 dentro do contêiner ficariam com dono
  10001 no host, e o usuário do runner do CI não conseguiria apagá-los;
- no macOS com Colima, o diretório temporário do sistema não é compartilhado
  com a VM do Docker.

`docker volume rm` sempre funciona.

As garantias do contêiner valem nas duas etapas:

- usuário: o root só troca para o uid 10001, que roda tudo o que toca o PR;
- capacidades: `--cap-drop ALL`, só `SETUID` e `SETGID`, `no-new-privileges`;
- limites: 1 CPU, 1 GB de memória, 256 processos, raiz somente leitura e tmpfs
  de 1 GB em `/tmp`;
- prazo: um único, dividido entre as duas etapas.

## Medição com o PR #1

O executor local foi rodado direto contra o PR #1 do aPRova-TESTES (head
`57e36c5`, diretório `go`), duas rodadas em cada versão, na mesma máquina:

| Versão | Rede dos testes | Rodada 1 | Rodada 2 | Resultado |
| --- | --- | --- | --- | --- |
| antes, um contêiner | `bridge` | 24,6 s | 23,7 s | `teste`: 10 passaram, 1 falhou |
| depois, duas etapas | `none` | 24,1 s | 24,1 s | `teste`: 10 passaram, 1 falhou |

O PR #1 não tem dependências, então o preparo é basicamente o clone, que
antes também acontecia. Num PR com dependências, o tempo de baixá-las só muda
de contêiner.

## Go

| Etapa | Comando | Ambiente |
| --- | --- | --- |
| `preparar` | `go mod download` | `GOPROXY=https://proxy.golang.org` (sem `,direct`), `GOVCS=*:off`, `GOTOOLCHAIN=local`, `GOFLAGS=-mod=readonly`, módulos em `cache/gomod` |
| `testar` | `go build ./...`, `go test -c`, `go test -json` | `GOPROXY=off`, `GOFLAGS=-mod=readonly`, módulos lidos de `cache/gomod` |

`go mod download` só baixa e confere hashes: não compila, não roda `init()`,
não roda `go generate` e não executa `-toolexec`. Sem `,direct`, nenhum módulo
é buscado no VCS.

| Situação | Classificação |
| --- | --- |
| proxy ou DNS fora do ar, timeout | `infraestrutura` (o motor tenta de novo) |
| módulo ou versão inexistente no proxy (404, 410, `unknown revision`) | `compilacao`, com o módulo na evidência |
| `checksum mismatch` contra o `go.sum` | `compilacao`, com o módulo na evidência |
| `go.mod` exige Go mais novo que a imagem | `infraestrutura`, como antes |
| módulo privado ou só disponível no VCS | `compilacao`: não está no proxy público |

## Python

**Preparar** lê, como dado, os `requirements*.txt` da raiz do projeto e a
tabela `[project]` do `pyproject.toml`:

- `dependencies` e os grupos `dev`, `test`, `tests` e `testing` de
  `optional-dependencies`;
- o `pyproject.toml` é lido pelo Python da imagem com `python -I` e `tomllib`.
  O `-I` ignora o diretório atual, então nenhum módulo do PR é importado.

A lista resultante vai para `cache/requisitos.txt`, e então roda
`pip download --only-binary=:all:`. Só wheels são baixadas, e uma wheel não
executa nada ao ser baixada nem instalada. O próprio projeto nunca é
construído.

**Testar** cria o venv, instala com
`pip install --no-index --find-links cache/wheels -r cache/requisitos.txt` e
roda o pytest importando o projeto pela raiz, como sempre.

| Requisito | Classificação |
| --- | --- |
| `-e` / `--editable` | `compilacao`: instalação editável exige construir o projeto |
| URL, `git+`, caminho local, `nome @ url` | `compilacao`: exige buscar ou construir fora do índice |
| `-i`, `--index-url`, `--extra-index-url`, `-f`, `--find-links`, `--trusted-host` | `compilacao`: só o PyPI é aceito |
| `-r` e `-c` (arquivo aninhado) | `compilacao`: só os `requirements*.txt` da raiz são lidos |
| `dynamic = ["dependencies"]` | `compilacao`: só se descobrem executando o build |
| pacote inexistente ou só em sdist | `compilacao`: "não existe no índice ou só existe como sdist" |
| versões incompatíveis entre si, hash divergente | `compilacao` |
| PyPI ou DNS fora do ar, timeout | `infraestrutura` |

O pip dá a mesma mensagem para "não existe" e para "só existe em sdist"
(`from versions: none`). As duas são falha do PR e também protegem contra
dependência alucinada (slopsquatting).

**Não suportado e por quê:**

- Dependências declaradas só no `setup.py` ou em tabelas do Poetry e do PDM não
  são lidas, porque ler exigiria executar código ou formatos próprios. A falta
  aparece no testar como erro de importação, ou seja, `compilacao`.

**Melhoria futura:** instalar o próprio projeto na etapa `testar`, baixando no
preparo as ferramentas de `build-system.requires` como wheels. Hoje o projeto
não é instalado, e o pytest importa pela raiz. Fica fora deste card para não
criar um modo de falha novo perto da apresentação.

## Java

O sandbox Java veio da branch `test` (Sara Rodrigues) e foi adaptado às duas
etapas. Só Maven é suportado.

| Etapa | Comando | Ambiente |
| --- | --- | --- |
| `preparar` | nenhum: só o clone e a conferência do `pom.xml` | rede `bridge`, mas nenhum `mvn` roda |
| `testar` | `mvn -o -B -ntp test-compile`, depois `mvn -o -B -ntp test` | `--network none`, uid 10001, repositório local no tmpfs encadeado ao cache da imagem, só leitura |

No Java o preparo não baixa nada: tudo o que o Maven pode usar já está na
imagem. Assim nenhuma resolução do Maven acontece com rede a partir do PR. O
Maven é sempre o `/usr/share/maven/bin/mvn` da imagem; o `mvnw` do repositório
nunca roda.

**O que é aquecido no cache.** No build da imagem, um projeto próprio
(`imagens/java/aquecimento/`) roda `mvn clean test` para cada combinação de:

| | Versões |
| --- | --- |
| JUnit Jupiter | 5.10.2, 5.11.4, 5.14.4, 6.1.3 |
| JUnit 4 | 4.13.2 |
| maven-surefire-plugin | 3.2.5, 3.5.2, 3.5.4 (padrão do Maven 3.9.16), 3.6.1 |

Rodar os testes, e não só `dependency:go-offline`, é o que traz os providers
do surefire e o `junit-platform-launcher`: o surefire só os resolve na hora do
teste, e o `go-offline` não os baixa. Isso foi medido: com o `go-offline` de um
projeto JUnit 5.10.2, o `test-compile` passa e o `mvn -o test` falha em
`junit-platform-launcher:1.10.2`. O surefire 3.6 não publica mais o
`surefire-junit4`; o JUnit 4 roda pela plataforma, com a vintage engine. As 20
combinações somam uns 5 minutos ao build e 38 MB à imagem.

| Situação | Classificação |
| --- | --- |
| sem `pom.xml` na raiz do projeto | `infraestrutura`, como "sem `go.mod`" |
| Java do projeto mais novo que o JDK 21 | `infraestrutura`, como "go.mod exige Go mais novo" |
| dependência, plugin, extensão ou provider fora do cache | `compilacao`, com o artefato na evidência |
| erro de compilação | `compilacao` |
| relatórios do surefire | `nenhuma` ou `teste`, cruzados com o código de saída do `mvn test` |
| projeto sem testes | `nenhuma` com 0/0, como no Python; o gate trata pela regra `codigo_sem_teste` |

Os `target/surefire-reports` que vierem versionados no repositório são apagados
antes do `mvn`, para não entrarem na contagem.

**Medição com o aPRova-TESTES.** O executor local foi rodado direto contra a
pasta `java/` da `main` do aPRova-TESTES (commit `2ce4ba4`; JUnit 5.10.2 com
surefire 3.2.5, compilado para Java 8), duas rodadas: `nenhuma`, 13 testes
passando (9 `@Test` e um `@ParameterizedTest` com 3 valores, contados como o
surefire conta), em 16,6 s e 11,7 s.

**O `.mvn/` não é brecha.** `.mvn/extensions.xml`, `.mvn/jvm.config` e
`.mvn/maven.config` carregam código ou opções do PR no Maven. Como o Maven só
roda na etapa `testar`, sem rede e como uid 10001, eles têm o mesmo alcance que
o próprio código do PR, e não são recusados.

**Não suportado e por quê:**

- **Gradle**: resolver dependências executa o build script do PR, e o sandbox
  exige `pom.xml`.
- **Versões e artefatos fora do cache**: a análise não tem rede. JUnit, surefire
  ou plugin em outra versão, dependência de produção (guava, por exemplo) ou
  plugin além dos usados até a fase `test` (jacoco, por exemplo) viram
  `compilacao` com o nome do artefato que falta.
- **Teste que trava**: é encerrado no prazo do executor e vira
  `infraestrutura`; não há timeout por teste, como o `-timeout` do Go.

## Azure: até o card 61

O executor da Azure continua com o fluxo de uma etapa só: o job roda o binário
sem subcomando, e ele clona, baixa e testa no mesmo contêiner, com a rede que o
job tiver. O Java não roda na Azure: a imagem Java só entra no executor local,
e o binário não tem fluxo de uma etapa para Java. As duas etapas na Azure ficam
para o card 61 (retorno do resultado do sandbox na Azure), que depende da
implantação.

O fluxo sem subcomando existe só para isso. O executor local sempre chama
`preparar` e depois `testar`, e não há opção de rede.

## Configuração

`SANDBOX_REDE_LOCAL` deixou de existir. Se ela estiver definida no ambiente
ou no `.env`, mesmo vazia, o worker recusa subir com a mensagem: a rede do
sandbox local agora é fixa por etapa.

## Risco que sobra

A etapa `preparar` tem rede `bridge`, que alcança o gateway do host. Nenhum
código do PR roda nela, e as ferramentas só falam com o proxy do Go e com o
PyPI. Os endereços vêm do aPRova; índices próprios do PR são recusados.
