# CLAUDE.md

Este arquivo define as diretrizes obrigatórias para qualquer trabalho de código realizado neste projeto.

## Diretriz primária: projeto conduzido para aprendizado profundo

Este projeto é conduzido com foco primário em aprendizado profundo, não em produção rápida de código. O desenvolvedor quer entender o "porquê" de cada decisão técnica — arquitetura, trade-offs, alternativas descartadas e conceitos subjacentes — mais do que simplesmente obter uma solução funcionando. Sempre que possível, priorize explicações minuciosas antes/junto do código, trate cada entrega como uma oportunidade de ensino, e não avance para a próxima etapa sem garantir que o raciocínio por trás dela foi compreendido. Código deve vir acompanhado de justificativa pedagógica, não apenas de funcionalidade.

### Onde vive a justificativa pedagógica

A regra de "zero comentário no código" (ver abaixo) permanece válida e não é enfraquecida por esta diretriz. A explicação pedagógica é entregue na resposta ao desenvolvedor e na documentação do projeto (`docs/`), nunca em comentários dentro dos arquivos de código.

## Uso de Skills

Faça uso máximo das skills disponíveis no ambiente para qualquer tarefa aplicável. Antes de escrever código, criar arquivos ou executar qualquer ação, verifique se existe uma skill relevante e utilize-a. Não pule esse passo mesmo em tarefas que pareçam simples.

## Nomenclatura de arquivos

Nomes de arquivos devem seguir o padrão idiomático já usado no projeto (ex: convenção do Go, convenção do React/TypeScript), porém **em português**, exceto quando isso quebrar a compilação ou violar uma convenção obrigatória da linguagem/framework (ex: nomes reservados, arquivos que precisam de nome fixo como `main.go`, `index.tsx`, `package.json`).

Exemplos:

- Correto: `execucao_tarefa.go`, `PainelExecucoes.tsx`
- Evitar apenas se quebrar build/import: nomes que colidem com convenções obrigatórias da ferramenta

## Comentários no código

**NUNCA** adicione comentários no código, em nenhuma circunstância. O código deve ser autoexplicativo através de nomenclatura clara de variáveis, funções e estruturas — não através de comentários explicativos, TODOs, ou anotações de qualquer tipo.

## Código limpo

Siga estritamente as regras de código limpo (Clean Code) em todo o projeto:

- Funções pequenas, com responsabilidade única
- Nomes descritivos e sem abreviações desnecessárias
- Evitar duplicação de código
- Evitar efeitos colaterais ocultos
- Estrutura consistente entre módulos semelhantes
- Preferir clareza a "esperteza" — se uma solução precisa de explicação, ela está errada, deve ser reescrita de forma mais clara em vez de comentada

## Organização do projeto em Go

Siga esta estrutura de pastas para o backend:

- `go.mod` na raiz do projeto, junto do `Dockerfile` — nunca mover o Dockerfile para uma subpasta tipo `build/`, ele deve ficar no mesmo nível do `go.mod` para evitar problema de contexto de build do Docker
- `cmd/<nome-do-executavel>/main.go` — cada executável do projeto (ex: `cmd/api/main.go` para o servidor, `cmd/worker/main.go` para o worker) fica em sua própria subpasta dentro de `cmd/`. Não colocar `main.go` direto na raiz do projeto
- `internal/` — todo código que é específico deste projeto e não deve ser importado por nenhum outro projeto. É onde deve morar a maior parte do código de negócio (domain, repository, worker, sandbox, llm, receipt, etc.)
- `pkg/` — usar **somente** se algum pacote for genuinamente reutilizável como biblioteca por outro projeto/pessoa. Se não houver essa necessidade clara, não criar essa pasta — não usar `pkg/` por padrão ou "porque é convenção"
- `scripts/` — scripts de shell para apoiar o desenvolvimento (setup local, migrations, etc.), quando necessário
- Nomeação de pacotes: o nome do pacote é semanticamente importante em Go, pois aparece toda vez que o pacote é referenciado (`pacote.Funcao()`). Escolher nomes de pacote curtos e que descrevam o que ele contém, evitando nomes genéricos

### Testes

- Testes unitários ficam **dentro do mesmo pacote** do código testado (arquivo `nome_test.go` ao lado do arquivo `nome.go`), nunca em uma pasta separada
- Testes de integração ou end-to-end, que cruzam múltiplos pacotes/fluxos do projeto, ficam em uma pasta própria na raiz, ex: `tests/` — não devem ficar dentro de `internal/` junto dos testes unitários

## Exceção obrigatória: documentação de exports em Go

A única exceção à regra de "nunca comentar" é a documentação de identificadores exportados em Go (funções, tipos, structs, constantes públicas), exigida pela convenção `godoc` da linguagem. Nesse caso específico, e somente nesse caso:

- O comentário deve ser em **português**
- Deve seguir o formato padrão do Go (começar com o nome do identificador)
- Deve ser estritamente descritivo do contrato (o que a função faz, parâmetros, retorno) — nunca explicativo de lógica interna ou justificativa de decisão de implementação

Fora desse caso específico (godoc de identificador exportado em Go), a regra de zero comentário permanece absoluta em todas as linguagens e arquivos do projeto.
