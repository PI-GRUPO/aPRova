# Imagem do sandbox Java

Imagem que roda os testes de repositórios Maven no executor local. O executor
de dentro (`cmd/sandbox`) é o mesmo das imagens Go e Python; as variáveis de
entrada estão em [imagens/go/README.md](../go/README.md), com
`APROVA_LINGUAGEM=java`. Veio do sandbox Java da Sara Rodrigues (branch `test`),
adaptado às duas etapas.

## Construir

```sh
make imagem-sandbox-java
```

A base `maven:3.9.16-eclipse-temurin-21` é fixada por digest: Maven 3.9.16 e
JDK 21. O worker escolhe a imagem por `SANDBOX_IMAGEM_JAVA`. A Azure não roda
Java até o card 61, e não há workflow que publique esta imagem no registro.

## O cache aquecido

A análise não tem rede, então o Maven só encontra o que a imagem já trouxe. No
build, os projetos de [aquecimento/](aquecimento/) rodam `mvn clean test` para
cada combinação desta matriz e deixam tudo em `/opt/aprova-cache/m2`:

| | Versões |
| --- | --- |
| JUnit Jupiter | 5.10.2, 5.11.4, 5.14.4, 6.1.3 |
| JUnit 4 | 4.13.2 |
| maven-surefire-plugin | 3.2.5, 3.5.2, 3.5.4 (o padrão do Maven 3.9.16), 3.6.1 |

Rodar os testes de verdade, e não `dependency:go-offline`, é o que garante os
providers do surefire e o `junit-platform-launcher`, que o surefire só resolve na
hora do teste. O `go-offline` não os baixa, e o primeiro `mvn -o test` falharia.
Os plugins que o Maven usa sem versão no pom (resources, compiler, surefire)
entram nas versões padrão do Maven 3.9.16. O surefire 3.6 não publica mais o
`surefire-junit4`: o JUnit 4 roda pela plataforma, com a vintage engine, e essa
combinação também está aquecida.

Para mudar a matriz, edite `VERSOES_DO_SUREFIRE`, `VERSOES_DO_JUNIT5` e
`VERSOES_DO_JUNIT4` no `Dockerfile` e rode os testes de integração. Uma versão
que não existe faz o build falhar.

## O que roda

1. `preparar`, com rede: só o clone e a conferência de que há `pom.xml` na raiz
   do projeto. Sem ele, o resultado é `infraestrutura`, como o "sem `go.mod`" do
   Go. Nenhum `mvn` roda nesta etapa.
2. `testar`, com `--network none`, como uid 10001 e com o código copiado para o
   tmpfs:
   1. apaga os `target/surefire-reports` que tenham vindo no repositório, para
      que um relatório versionado não entre na contagem;
   2. `mvn -o -B -ntp test-compile`;
   3. `mvn -o -B -ntp test`, e apura os `TEST-*.xml` de todos os
      `target/surefire-reports` do projeto.

O Maven é sempre o `/usr/share/maven/bin/mvn` da imagem, por caminho absoluto.
O `mvnw` do repositório nunca roda: ele baixaria outro Maven, e sem rede
falharia de qualquer jeito.

O repositório local é encadeado: a cabeça, gravável, fica no tmpfs
(`-Dmaven.repo.local`), e o cache da imagem entra como cauda só de leitura
(`-Dmaven.repo.local.tail`, recurso do Maven 3.9). O cache pertence ao root e a
raiz do contêiner é somente leitura, então nenhum PR altera o que o próximo vai
usar.

## Como o desfecho é classificado

| Situação | Resultado |
| --- | --- |
| sem `pom.xml` na raiz do projeto | `infraestrutura` |
| `release`, `source` ou `target` mais novo que o JDK 21 | `infraestrutura`, como o "go.mod exige Go mais novo" |
| dependência, plugin ou provider fora do cache | `compilacao`, com o artefato na evidência |
| erro de compilação | `compilacao` |
| Maven que não inicia ou prazo esgotado | `infraestrutura` |
| `mvn test` com saída 0 e nenhuma falha nos relatórios, inclusive sem testes | `nenhuma` |
| `mvn test` com saída diferente de 0 e ao menos uma falha nos relatórios | `teste` |
| qualquer outra combinação | `infraestrutura` |

Passou é `tests − failures − errors − skipped`; falhou é `failures + errors`.
O código de saída vem do processo e os relatórios são escritos pelo uid do
código do PR: os dois precisam concordar. Os relatórios são lidos como entrada
hostil, sem seguir link simbólico e com tamanho limitado.

Um teste que trava é encerrado no prazo do executor e vira `infraestrutura`.

## O `.mvn/` do repositório

`.mvn/extensions.xml`, `.mvn/jvm.config` e `.mvn/maven.config` são lidos pelo
Maven e podem carregar código ou opções do PR. Aqui não são brecha nem são
recusados: o Maven só roda na etapa `testar`, sem rede e como uid 10001, as
mesmas condições do código do PR. Uma extensão fora do cache é `compilacao`,
como qualquer artefato fora do cache.

## Não suportado

- **Gradle.** Resolver dependências executa o build script do PR, e o projeto
  precisa de `pom.xml`.
- **Versões fora da matriz.** O primeiro artefato que falta vira `compilacao`
  com o nome dele, para o autor saber o que trocar.
- **Plugins além dos usados até a fase `test`** (por exemplo jacoco, spotless)
  e dependências de produção (por exemplo guava) que não estejam no cache.
