# Imagem do sandbox Python

Imagem que roda dentro do job do sandbox para repositórios Python. O executor
de dentro (`cmd/sandbox`) é o mesmo da imagem Go: recebe o pedido por variáveis
de ambiente, clona o repositório no SHA exato e entrega um `ResultadoSandbox`
como uma única linha JSON no stdout. As variáveis de entrada estão em
[imagens/go/README.md](../go/README.md), com `APROVA_LINGUAGEM=python`.

## Construir

```sh
make imagem-sandbox-python
```

A base `python:3.12-slim` é fixada por digest, e `pytest`, `pytest-json-report`
e `pytest-timeout` por versão. Para atualizar, troque-os e rode os testes de
integração. O worker escolhe a imagem por `SANDBOX_IMAGEM_PYTHON`.

Ainda não há workflow que publique esta imagem no registro da Azure; o
`imagem-sandbox-go` publica só a imagem Go.

## Como um repositório vira Python

A linguagem vem da configuração do aPRova (`SANDBOX_LINGUAGENS`), nunca do
conteúdo do PR. Se dependesse de detectar arquivos, um PR poderia escolher a
linguagem em que é testado apenas adicionando um `go.mod`.

## O que roda

1. Conferir que o diretório do projeto tem ao menos um arquivo `.py` regular,
   sem contar `.git` e sem seguir link simbólico. Sem isso, o resultado é
   `infraestrutura` e nada do PR roda. Essa conferência é o equivalente ao "sem
   `go.mod`" da imagem Go: sem ela, um diretório configurado errado daria zero
   testes, e zero testes não reprova.
2. No executor local, a etapa `preparar` lê os `requirements*.txt` da raiz e o
   `[project]` do `pyproject.toml` como dado e baixa só as wheels para o cache;
   a etapa `testar` cria um venv com `--system-site-packages` e instala só a
   partir desse cache (`pip install --no-index --find-links`). O pytest e os
   plugins vêm da imagem; o venv os enxerga e só acrescenta o que foi pedido.
   Requisito inexistente ou só em sdist é `compilacao`. Sem subcomando, o fluxo
   da Azure até o card 61 ainda roda `pip install -r requirements.txt` direto,
   e lá uma instalação que falha é `infraestrutura`.
3. Rodar `python -m pytest` com `--json-report` num arquivo com sufixo aleatório
   fora do clone, e `--timeout` com o tempo que resta menos a margem de entrega.

`--timeout` limita cada teste, não a suíte. Teste que trava é `teste`, culpa do
PR. Quem trava fora de um teste, como na coleta, é encerrado no prazo do
executor e vira `infraestrutura`.

## Como o desfecho é classificado

O código de saída vem do processo; o relatório é escrito pelo uid do código do
PR. Os dois precisam concordar, e qualquer divergência vira `infraestrutura`.

| Saída do pytest | Relatório | Resultado |
| --- | --- | --- |
| 0 | nenhuma falha | `nenhuma` |
| 1 | ao menos uma falha | `teste` |
| 2 | um coletor falhou | `compilacao` |
| 4 | não há; a saída diz `while loading conftest` | `compilacao` |
| 5 | zero testes | `nenhuma` |
| qualquer outra combinação | | `infraestrutura` |

Passou é `passed` mais `xpassed`. Falhou é `failed` mais `error`, porque erro em
fixture também reprova. `xfailed` e `skipped` não entram em nenhum dos dois.

O arquivo do relatório é entrada hostil: link simbólico é recusado
(`O_NOFOLLOW`), fifo não prende a leitura (`O_NONBLOCK`) e o tamanho é limitado.

## Por que o código do PR não forja o resultado entregue

Valem as mesmas barreiras da imagem Go: o código do PR roda como uid 10001, não
abre o stdout do executor e tudo o que imprime vira texto de `saida_bruta`. O
teste `TestSandboxPythonTesteMaliciosoNaoForjaOResultado` tenta esses ataques.

## Limites conhecidos

- **O PR consegue falsificar o próprio veredito do pytest.** O pytest roda no
  mesmo processo que o `conftest.py` e os testes do PR, e o caminho do relatório
  está em `sys.argv`. Um `conftest.py` que grava um relatório com 99 aprovados e
  chama `os._exit(0)` é aceito como `nenhuma`, escondendo um teste que falha:
  isso foi medido com a imagem real. O sufixo aleatório só impede plantar o
  arquivo antes da execução. O ganho para quem ataca é o mesmo de apagar o teste
  que falha no próprio PR, e nenhum dos dois é barrado pelo sandbox. O sandbox
  detecta erro honesto; não é defesa contra autor mal-intencionado.
- **No executor local, o código do PR não tem rede.** A etapa `preparar` baixa só
  wheels (`pip download --only-binary=:all:`), que não executam nada, e a etapa
  `testar` instala só a partir delas, com `--network none`. Dependência
  inexistente ou só em sdist é `compilacao`. Veja
  [docs/sandbox-em-duas-etapas.md](../../docs/sandbox-em-duas-etapas.md).
- Na Azure, até o card 61, o job ainda roda tudo numa etapa só, com a rede que o
  job tiver, e lá o `pip install` pode executar código de pacote em sdist. Não há
  segredo no container.
