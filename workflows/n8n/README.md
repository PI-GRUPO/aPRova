# Workflows do n8n

Esta pasta contém os workflows exportados como JSON, sem segredos, credenciais
pessoais ou dados fixados:

- `entrada-eventos.json`: autenticação HMAC, normalização, deduplicação e
  roteamento dos eventos do núcleo.
- `central-slack.json`: notificações, formulários, pedido assinado à API e
  retorno da ação no canal de notificações.
- `dependencias-osv.json`: consulta pública ao OSV por dependência e versão;
  enriquece o alerta Slack sem modificar o risco definido pelo núcleo.

O inventário dos nós, funções, testes e pendências está em
[docs/central-slack-implementacao.md](../../docs/central-slack-implementacao.md).

`../exemplos/` contém exemplos de eventos do aPRova. Esses arquivos não são
workflows e não devem ser importados no editor do n8n.

## Executar em uma máquina nova

Na raiz do projeto, com Docker e Docker Compose instalados:

```sh
mkdir -p .n8n
docker compose up -d n8n-aprova
```

Abra <http://localhost:5978/> e crie sua própria conta na primeira execução.
Cada máquina mantém seus dados em uma pasta `.n8n/` independente, ignorada
pelo Git. Em Linux, essa pasta precisa permitir escrita pelo usuário `node`
do container (UID/GID `1000:1000`). Se ocorrer `EACCES`, ajuste a propriedade
da pasta para esse usuário.

O `.env` é opcional para abrir o editor com os valores padrão. Para
personalizar porta, URL e fuso, use as variáveis documentadas no
[README principal](../../README.md#n8n-opcional).

## Exportar para o repositório

1. Abra o workflow no seu n8n e use o menu do workflow para baixar o JSON
   (opção **Download**).
2. Revise uma cópia do JSON antes de colocá-la nesta pasta. Remova tokens,
   senhas, headers de autenticação, segredos usados em nós Code, dados
   pessoais e dados de teste fixados nos nós (`pinData`). Revise também
   nomes e referências das credenciais pessoais.
3. Substitua valores específicos da sua instalação por exemplos e documente
   as credenciais, URLs e configurações necessárias para aquele workflow.
4. Salve com um nome descritivo, como `notificacao-slack.json`, e inclua o
   arquivo no commit.

O JSON exportado pode conter nomes e IDs de credenciais e valores inseridos
diretamente nos nós. A revisão precisa acontecer mesmo sem exportar o banco
de credenciais. Veja a
[documentação de exportação do n8n](https://docs.n8n.io/workflows/export-import/).

Não inclua nesta pasta o banco SQLite, o arquivo `config`, exports de
credenciais ou cópias da pasta `.n8n/`.

## Importar e configurar

1. Abra sua própria instância do n8n.
2. Use **Import from File** no menu do workflow e selecione o JSON desta
   pasta.
3. Configure a credencial `slackApi` nos nós `10` e `10B` da Central.
4. No `.env`, configure `SLACK_SIGNING_SECRET`, `SLACK_INCOMING_WEBHOOK_URL`,
   `N8N_SEGREDO_ACOES`, `N8N_SEGREDO_EVENTOS` e `APROVA_API_URL`.
   Os segredos de eventos e ações são distintos; cada um deve ter o mesmo
   valor no emissor e no receptor daquela direção.
5. Copie o ID atribuído à Central para `APROVA_SLACK_WORKFLOW_ID` no `.env`.
   O nó `08 | Executar Central do Slack` do roteador usa essa variável.
   Importe também `dependencias-osv.json` e preencha `APROVA_OSV_WORKFLOW_ID`
   com o ID dessa importação. A rota `dependencia_nova` do export do Core usa
   essa variável, assim como a consulta complementar da Central, sem depender
   do ID de outra instalação.
   No mesmo projeto do n8n, crie uma Data Table chamada
   `aprova_osv_por_analise`, com colunas `chave` e `contexto`, ambas String.
   A definição está em `recursos/tabela-contexto-osv.json`; esse arquivo
   descreve a tabela e não é um workflow para importar. Os nós localizam a
   tabela pelo nome, sem um ID fixo de outra instalação.
6. Recrie apenas o serviço para carregar as variáveis:
   `docker compose --profile automacao up -d --no-deps n8n-aprova`.
7. Ajuste os endereços e teste antes de publicar ou ativar o workflow,
   conforme a versão do n8n.

As URLs públicas de produção são `/webhook/aprova/events` para eventos e
`/webhook/aprova/slack/interacoes` para interatividade do Slack. Configure a
segunda em **Interactivity & Shortcuts → Request URL** do Slack App. Durante
testes manuais, o prefixo é `/webhook-test/` e o editor precisa estar escutando
novamente a cada nova requisição, incluindo o envio do formulário.

A Central devolve mensagens pelo Incoming Webhook do canal configurado.
O bot usado para abrir formulários não precisa publicar via `chat.postMessage`.
O webhook fixa o destino dos retornos; para atender vários canais será preciso
outra configuração de envio.

O workflow OSV não exige token nem credencial. Ele consulta
`https://api.osv.dev/v1/query` com o nome, ecossistema e versão informados pelo
núcleo e publica um alerta por dependência usando `SLACK_INCOMING_WEBHOOK_URL`.
As consultas têm timeout de 30 segundos e limite de dez páginas. Resultados
incompletos e falhas são identificados no texto, sem anunciar ausência de
vulnerabilidades. A severidade só é exibida quando informada textualmente
pelas fontes. O risco do núcleo permanece inalterado.

Desde a integração de 2026-11-02, o OSV também guarda um resumo informativo
por tenant, análise, repositório e PR na Data Table do n8n. Isso não dá acesso
ao banco do núcleo. A Central lê esse contexto antes do alerta com botões e
leva o mesmo resumo aos formulários de aprovação e revisão. SHA incompatível,
quando informado nos dois eventos, impede reutilização.

O resumo é reutilizado por até 24 horas. Se estiver vencido, ou se o próprio
evento de aprovação trouxer dependências e não houver contexto compatível,
a Central chama o OSV e espera o retorno antes de publicar o alerta. Essa
consulta complementar limita-se a dez pacotes, duas páginas e cinco segundos
por requisição; esses limites podem produzir resultados parciais. Sem
dependências conhecidas, sem resultado ou em falha, o alerta declara
indisponibilidade e não presume ausência de vulnerabilidades.

A consulta não ocorre depois do clique: os formulários recebem o resumo do
próprio alerta. O resumo tem até 1000 caracteres e detalha até três pacotes.
Ele não é enviado em `pedido_acao` nem acrescentado à justificativa humana.
Publicar esse contexto na review do GitHub continua pendente no núcleo.

Se a aprovação chegar antes do evento de dependências e não trouxer a lista
no próprio payload, o alerta indicará contexto ainda indisponível. A chegada
posterior do evento de dependências alimenta a tabela e o alerta OSV separado;
a mensagem de aprovação já publicada não é atualizada automaticamente.

Os exports são importados inativos. Na instância revisada em 2026-11-02,
Core e OSV estavam publicados e ligados por seleção direta do workflow no
editor. A variável `APROVA_OSV_WORKFLOW_ID` é necessária para importar os
exports portáteis. A chamada da Central ao OSV foi publicada na instância
local; não houve reinício de container nessa integração.

A API recebe o corpo JSON bruto assinado em `POST /acoes-remotas`.
O nó `13` está habilitado e tem timeout de 15 segundos. Só `202` com
`status=aceito` confirma a ação; falhas HTTP e de comunicação não confirmam
execução. A autoridade continua sendo o núcleo e a permissão do GitHub.

## Testes

Com Node 22 ou superior, sem dependências npm e sem chamadas externas:

```sh
node --test workflows/n8n/tests/*.test.cjs
```

Na raiz do repositório, com Go e Docker disponíveis:

```sh
go test ./internal/... ./cmd/... ./scripts/... ./tests/functional/... -race
go test ./tests/integration/postgres/... -race -count=1
```

Os testes do PostgreSQL criam um container temporário. Os testes funcionais
da API usam um servidor GitHub simulado; eles não aprovam PRs reais.

Para chamar a API do aPRova em execução na máquina host, use
`http://host.docker.internal:8080` nos nós HTTP (ajustando a porta da API se
necessário). A API precisa estar em execução com suas dependências.

## Levar ao servidor

No servidor, importe os JSONs revisados, configure as credenciais e os
segredos daquele ambiente e ajuste as URLs para os serviços acessíveis de
lá. Depois de testar, publique ou ative os workflows.

A conta, as credenciais e os dados da sua instalação local permanecem na
sua máquina. Não copie o banco pessoal para distribuir os workflows.

O Compose do repositório é para desenvolvimento local. A publicação em um
servidor exige configuração própria de acesso, persistência e credenciais.
A esteira atual não importa workflows nem faz deploy do n8n.
