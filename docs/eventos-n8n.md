# Eventos do aPRova para o n8n

Este guia é para quem constrói os workflows no n8n. Ele descreve os eventos que
o núcleo do aPRova envia, como conferir que um evento veio mesmo do núcleo e
como testar um workflow sem o núcleo rodando.

## Como o n8n se encaixa

O núcleo analisa o pull request, decide o risco e guarda tudo no banco. O n8n
fica na periferia: **recebe** eventos e avisa pessoas (Slack, Trello, SLA,
digest). Ele nunca decide risco e nunca lê o banco.

- O worker do núcleo envia cada evento por `POST` para a URL do webhook do n8n
  (`N8N_WEBHOOK_URL`). Para testar os workflows sem o núcleo rodando, use o
  script descrito neste guia.
- O envio é *fire-and-forget*: o núcleo não espera o workflow terminar e não
  falha se o n8n estiver fora do ar. Um evento perdido não trava nenhuma
  análise, e por isso o workflow não pode ser a única fonte de nada.
- Tudo o que o n8n quiser **fazer** num PR volta pelo `POST /acoes-remotas`
  (veja o README), que confere a permissão no GitHub antes de agir.

Para iniciar uma instância local opcional e compartilhar os workflows como
JSON, veja [workflows/n8n/README.md](../workflows/n8n/README.md). Cada pessoa
usa sua própria conta e credenciais. Os JSONs em `workflows/exemplos/`
abaixo são payloads de eventos, não workflows importáveis no n8n.

As implementações importáveis de entrada e da Central do Slack já estão em
`workflows/n8n/`. O estado de implementação, a configuração e as pendências
da operação remota estão em [central-slack-implementacao.md](central-slack-implementacao.md).

### Estado do código integrado — 2026-11-13

O worker atual já conecta o emissor e produz `analise_concluida`,
`aguardando_aprovacao`, `aprovacao_registrada` e `falha_recuperavel`
(`cmd/worker/main.go` e `internal/worker/eventos_do_motor.go`). Isso é
implementação no código; não comprova validação de ponta a ponta com o
GitHub App real. O worker também emite `dependencia_nova` quando há
dependências identificadas; `instalacao` ainda não tem emissão automática.
O gate determinístico, os recibos assinados, a publicação no GitHub e a
calibração opcional por histórico estão implementados no núcleo.

O workflow `aPRova | Digest | Resumo Semanal` consulta `GET /reports/weekly`
com HMAC de `N8N_SEGREDO_ACOES`, sem acessar o banco. Tem agendamento próprio,
independente dos seis tipos de evento do núcleo. O Core também aceita o
pedido autenticado `relatorio_periodico` como extensão da automação; esse
pedido não é um sétimo evento emitido pelo worker e apenas guarda o relatório
na execução. Consulta semanal e Digest da instância leem a instalação por
`APROVA_CHAT_INSTALLATION_ID`; o valor real está no `.env` local e carregado
no container, sem ID fixo nos parâmetros. O Digest continua desativado até
validar o relatório real e o disparo agendado.

App ID, chave privada válida, webhook secret e slug GitHub já estão no
`.env` do núcleo. Falta validar instalação/permissões reais, configurar
`SLACK_GITHUB_VINCULOS` e validar API/worker. No teste do onboarding,
`/health` e `/ready` responderam 200 a partir do n8n; isso não comprova
os endpoints de negócio ou a emissão automática de eventos.
Configuração disponível não comprova integração real.

O Core encaminha `aprovacao_registrada` ao novo workflow de feedback.
Ele pergunta somente para decisão `aprovada` de risco original ALTO; sem
vínculo único do revisor Slack/GitHub, encerra sem pergunta. Os cliques passam
pela autenticação da Central. O envio à API está desabilitado: faltam endpoint
específico e integração do feedback com a calibração. Encerramento de SLA e atualização da mensagem
original ainda não foram implementados.

Os exports incluem feedback, suas ligações no Core/Central e leitura da
variável no Digest. As chamadas ao feedback usam
`APROVA_FEEDBACK_WORKFLOW_ID`, preenchida com o ID do workflow importado
em cada ambiente. Novas sincronizações dependem de
autorização explícita. Estado e pendências estão na
[documentação central](central-slack-implementacao.md) e no
[guia do n8n](../workflows/n8n/README.md).

## Envelope

Todo evento tem a mesma casca. Um único nó **Switch** em `tipo_evento` roteia
para o workflow certo.

```json
{
  "versao": 1,
  "id_evento": "0f8e6c1a-3b2d-4c5e-8f7a-9b0c1d2e3f40",
  "tipo_evento": "analise_concluida",
  "ocorrido_em": "2026-10-19T14:32:08Z",
  "tenant_id": 58291734,
  "dados": { }
}
```

| Campo | Tipo | Obrigatório | Exemplo | Observação |
| --- | --- | --- | --- | --- |
| `versao` | inteiro | sim | `1` | versão do contrato; hoje só existe a 1 |
| `id_evento` | texto (UUID) | sim | `0f8e6c1a-3b2d-4c5e-8f7a-9b0c1d2e3f40` | um por fato; o mesmo fato reenviado repete o id; use para descartar repetidos |
| `tipo_evento` | texto | sim | `analise_concluida` | um dos seis tipos abaixo |
| `ocorrido_em` | texto (RFC 3339, UTC) | sim | `2026-10-19T14:32:08Z` | quando o fato aconteceu no núcleo |
| `tenant_id` | inteiro | sim | `58291734` | `installation_id` do GitHub App; identifica a conta |
| `dados` | objeto | sim | | campos do tipo, descritos abaixo |

O mesmo evento pode chegar duas vezes, por exemplo quando um worker cai no
meio de uma análise e outro a retoma. O `id_evento` é calculado a partir do
fato (o tipo, o `task_run` e, na aprovação, o id da decisão), então se repete
nesse caso. O Core usa **Remove Duplicates** com memória entre execuções,
chave `tenant_id:id_evento` e histórico de 10.000 chaves. A retenção limita
essa deduplicação; ela não substitui idempotência de ações no núcleo.

A ordem de chegada também não é garantida: cada evento sai numa conexão
própria, e um `aguardando_aprovacao` pode chegar antes do `analise_concluida`
da mesma análise. Nenhum workflow deve depender de um evento ter chegado antes
de outro.

### Mudanças futuras

Campo novo em `dados` e tipo novo de evento são mudanças compatíveis: o
workflow que não os conhece continua funcionando, desde que o **Switch** tenha
uma saída padrão que ignore tipos desconhecidos. Remover ou renomear um campo,
ou mudar o significado de um valor, sobe o `versao`. Um workflow pode recusar
a versão que não conhece.

## Tipos de evento

| Tipo | Quando | Workflow que alimenta |
| --- | --- | --- |
| `analise_concluida` | ao fim de toda análise, em qualquer risco | resultado da análise no canal do time, comentário no card do Trello, monitor de anomalias |
| `aguardando_aprovacao` | risco alto, PR bloqueado | Central de aprovação no Slack (alerta com botões) e início do SLA |
| `aprovacao_registrada` | um humano decidiu sobre um PR bloqueado | fim do SLA, atualização do alerta no Slack e pedido de feedback quando o risco original era alto |
| `dependencia_nova` | o PR adiciona dependências | alerta de dependência nova |
| `falha_recuperavel` | a análise falhou por infraestrutura e pode ser reexecutada | alerta de operação com o botão de reexecutar |
| `instalacao` | o GitHub App foi instalado, removido ou mudou de repositórios | boas-vindas e cadastro da conta no n8n |

Na instância atual, `analise_concluida` chama
o workflow Trello: ele procura o link completo da PR na descrição de um
único card do quadro configurado, comenta sugerindo movimentação e replica
o comentário confirmado no Slack. Nunca move o card. Sem vínculo ou com
ambiguidade, registra o resultado na execução e não publica mensagem. A
inserção do link na descrição permanece uma prática manual da equipe na
versão atual. Configuração, inventário e limites estão em
[central-slack-implementacao.md](central-slack-implementacao.md), na seção
Trello. O export portátil e as instruções de importação estão em
[workflows/n8n/README.md](../workflows/n8n/README.md). Essa integração não
adiciona campos ao contrato. O worker já emite esse tipo; sua validação
integrada com o GitHub real continua pendente.

Os exemplos completos de cada tipo ficam em `workflows/exemplos/`, um arquivo
por tipo. São eles que o script de envio usa, e um teste do núcleo garante que
cada um é um evento válido.

### `analise_concluida`

Emitido ao fim de toda análise. Um PR de risco baixo também gera este evento:
quem só quer os casos graves filtra por `nivel_risco`.

| Campo | Tipo | Obrigatório | Exemplo | Observação |
| --- | --- | --- | --- | --- |
| `task_run_id` | texto (UUID) | sim | `7f3c2a1e-5b8d-4e6f-9a0b-1c2d3e4f5a6b` | a execução de análise no núcleo |
| `repositorio` | texto | sim | `Azulpasta/aPRova-TESTES` | `dono/nome` |
| `numero_pr` | inteiro | sim | `14` | |
| `url_pr` | texto (URL) | sim | `https://github.com/Azulpasta/aPRova-TESTES/pull/14` | |
| `autor_pr` | texto | sim | `wlaisw` | login do GitHub de quem abriu o PR |
| `sha` | texto | sim | `3f9a1c7e2b4d6f8091a3c5e7f9b1d3e5a7c9e1f3` | commit analisado, 40 caracteres hexadecimais minúsculos |
| `nivel_risco` | texto | sim | `medio` | `baixo`, `medio` ou `alto` |
| `motivos` | lista de objetos | sim | veja abaixo | regras acionadas; lista vazia quando nenhuma regra acionou |
| `motivos[].nome` | texto | sim | `codigo_sem_teste` | identificador estável da regra; a lista está abaixo |
| `motivos[].descricao` | texto | sim | `o PR altera código sem alterar nenhum teste` | frase curta para mostrar a pessoas |
| `explicacao` | texto | sim | `Risco medio: o PR altera código sem alterar nenhum teste.` | hoje é o nível e as descrições dos motivos; até 4000 caracteres |
| `receipt_hash` | texto | sim | `9c4e2a7f...0e2f` | SHA-256 do conteúdo assinado do recibo, 64 caracteres hexadecimais minúsculos; é o `hash_sha256` devolvido pela verificação |
| `url_recibo` | texto (URL) | sim | `https://aprova.example.com/receipts/2b6d8f0a-.../verify` | endereço público que verifica o recibo e responde `valido` verdadeiro ou falso |

Os valores possíveis de `motivos[].nome` são os identificadores das regras do
gate. O nível é o que a regra, sozinha, atribui ao PR; sem calibração, o nível
do evento é o maior entre os motivos. A descrição é o texto que vai em
`motivos[].descricao`. O evento não leva o nível de cada motivo, só o da
análise em `nivel_risco`.

| Identificador | Nível | Descrição enviada | Quando dispara |
| --- | --- | --- | --- |
| `falha_sandbox` | alto | os testes ou a compilação falharam no sandbox | os testes falharam ou o projeto não compilou |
| `muitos_arquivos` | alto | o PR altera N arquivos, acima do limite de M | o PR altera mais arquivos que o limite (20 por padrão) |
| `configuracao_ci` | alto | o PR altera configuração de build ou de CI | o PR altera workflow, Dockerfile, Makefile ou arquivo de build equivalente |
| `dependencia_nova` | alto | o PR acrescenta dependência nova, que precisa de revisão humana | o PR acrescenta dependência, ou o diff do manifesto não pôde ser lido |
| `segredo_versionado` | alto | o PR versiona um arquivo que costuma guardar segredo | o PR versiona `.env`, chave privada ou arquivo equivalente |
| `area_sensivel` | alto | o PR altera um caminho sensível | o PR altera um caminho sensível, como `auth/` ou `migrations/` |
| `codigo_sem_teste` | medio | o PR altera código sem alterar nenhum teste | o PR altera código sem alterar nenhum teste |
| `calibracao_historico` | informativo | risco alto pelas regras, reduzido a médio pelo histórico: 7 casos parecidos aprovados e mergeados, nenhum revertido | a calibração por histórico desceu o nível um degrau ([calibracao.md](calibracao.md)) |

Os exemplos em `workflows/exemplos/` usam esses identificadores. Versões
antigas dos exemplos traziam nomes inventados, como `falha_nos_testes` e
`sem_teste_para_alteracao`, que o núcleo nunca envia: um workflow que compare
`motivos[].nome` com texto fixo precisa usar a lista acima.

`calibracao_historico` não é uma regra de risco: ele explica por que
`nivel_risco` ficou um degrau abaixo do que as regras dariam. Com
`CALIBRACAO_LIGADA=true`, um PR com `area_sensivel` pode chegar com
`nivel_risco` `medio`, e um com `codigo_sem_teste` com `baixo`. Um workflow que
deduz o nível a partir dos nomes dos motivos erra nesses casos; o nível certo é
sempre o de `nivel_risco`. Com a calibração desligada, que é o padrão, esse
motivo nunca aparece.

`area_sensivel` é alto porque a calibração por histórico só pode abaixar o
risco, nunca subir. Com a base em médio, um PR em `auth/` com os testes passando
sairia sem aprovação humana, e nenhum dado futuro o faria voltar a exigi-la.

Quando o GitHub não devolve o diff de um arquivo de dependências, o gate
bloqueia o PR com o motivo `dependencia_nova` (evidência "não foi possível ler o
diff"), mas o evento `dependencia_nova` **não** é enviado: ele exige ao menos
uma dependência, e sem o diff não há o que listar. Nesse caso o PR aparece só
em `analise_concluida` e `aguardando_aprovacao`.

O domínio `aprova.example.com` dos exemplos é provisório: o endereço público
do núcleo ainda não existe. Quando existir, ele vai na variável
`APROVA_URL_PUBLICA`, e só a base da URL muda; o caminho `/receipts/<id>/verify`
continua o mesmo, então nenhum workflow precisa ser alterado.

O recibo agora é real: `url_recibo` abre a verificação pública do recibo, e
`receipt_hash` é o SHA-256 do conteúdo assinado. Até a versão anterior deste
contrato o caminho era `/recibos/<id>` e não respondia; um workflow que monte ou
desmonte essa URL por conta própria precisa passar a usar
`/receipts/<id>/verify`. Como verificar um recibo sem depender do núcleo está em
[verificacao-recibo.md](verificacao-recibo.md).

### `aguardando_aprovacao`

Emitido quando o risco é alto e o PR fica bloqueado até uma decisão humana. Tem
**exatamente os mesmos campos** de `analise_concluida`, com `nivel_risco`
sempre `alto`. Uma análise de risco alto gera os dois eventos: o
`analise_concluida` para o histórico e este para a Central no Slack e para o
SLA.

A Central pode enriquecer o alerta de decisão com uma consulta OSV da mesma
análise, sem alterar o risco. Para isso, usa o resultado de `dependencia_nova`
correlacionado por `tenant_id`, `task_run_id`, repositório e PR. O adaptador
também preserva uma lista opcional `dados.dependencias` quando o emissor a
fornece em `aguardando_aprovacao`, permitindo consulta antes do alerta mesmo
sem evento anterior. Essa lista não é obrigatória no contrato v1; esta
integração não alterou o emissor Go. Sem contexto disponível, o alerta deve
informar a indisponibilidade, nunca afirmar que não há vulnerabilidades.

### `aprovacao_registrada`

Emitido quando um humano aprova ou pede alterações num PR bloqueado, pelo
GitHub ou pelos botões do Slack.

| Campo | Tipo | Obrigatório | Exemplo | Observação |
| --- | --- | --- | --- | --- |
| `task_run_id` | texto (UUID) | sim | `c41d7e9a-2f3b-4c8d-a5e6-0b1c2d3e4f5a` | o mesmo do `aguardando_aprovacao` |
| `repositorio` | texto | sim | `Azulpasta/aPRova-TESTES` | |
| `numero_pr` | inteiro | sim | `14` | |
| `url_pr` | texto (URL) | sim | `https://github.com/Azulpasta/aPRova-TESTES/pull/14` | |
| `decisao` | texto | sim | `alteracoes_solicitadas` | `aprovada` ou `alteracoes_solicitadas`, os mesmos valores que o núcleo guarda para cada decisão |
| `login_github` | texto | sim | `Azulpasta` | quem decidiu |
| `origem` | texto | sim | `slack` | `github` ou `slack` |
| `nivel_risco_original` | texto | sim | `alto` | o risco que bloqueou o PR |
| `id_aprovacao` | inteiro | sim | `37` | identificador da decisão no núcleo |

O Core encaminha este evento ao feedback. Apenas `decisao=aprovada` com
`nivel_risco_original=alto` é elegível à pergunta; pedidos de alteração não
disparam feedback. Vínculo e API continuam pendentes, conforme a documentação
central. A instalação vem do tenant autenticado, sem substituição pelo ID
global das consultas.

Encerrar o SLA do `task_run_id` e atualizar a mensagem original são usos
previstos, ainda sem implementação. Exigem correlação e persistência do
estado/mensagem; não presumir que já ocorrem por receber este evento.

### `dependencia_nova`

Emitido quando o PR adiciona dependências, junto com o `analise_concluida`,
qualquer que seja o nível final. Mudar a versão de uma dependência que já existe
não conta. Quando o GitHub omite o diff de um manifesto, o gate bloqueia com o
motivo `dependencia_nova`, mas este evento não sai, porque não há como saber
quais são as dependências.

| Campo | Tipo | Obrigatório | Exemplo | Observação |
| --- | --- | --- | --- | --- |
| `task_run_id` | texto (UUID) | sim | `3b5d7f9a-1c2e-4a4b-8d6f-0a2c4e6b8d0f` | |
| `repositorio` | texto | sim | `Azulpasta/aPRova-TESTES` | |
| `url_pr` | texto (URL) | sim | `https://github.com/Azulpasta/aPRova-TESTES/pull/15` | o número do PR está no fim da URL |
| `dependencias` | lista de objetos | sim | veja abaixo | ao menos uma |
| `dependencias[].ecossistema` | texto | sim | `Go` | `Go`, `PyPI` ou `Maven`, os nomes usados pelo OSV.dev |
| `dependencias[].nome` | texto | sim | `github.com/shopspring/decimal` | no Maven, `groupId:artifactId`, como `org.javamoney:moneta` |
| `dependencias[].versao` | texto | sim | `v1.4.0` | como aparece no manifesto |

Os nomes de ecossistema são os do OSV.dev, então o workflow pode passar cada
item direto para `https://api.osv.dev/v1/query` e mostrar as vulnerabilidades
conhecidas. Esse resultado é informativo: quem decide o risco é o núcleo.

### `falha_recuperavel`

Emitido quando a análise falha por um problema de infraestrutura que pode ser
resolvido tentando de novo.

| Campo | Tipo | Obrigatório | Exemplo | Observação |
| --- | --- | --- | --- | --- |
| `task_run_id` | texto (UUID) | sim | `6d8f0a2c-4e1b-4c3d-b5a7-9e0f1a2b3c4d` | |
| `repositorio` | texto | sim | `Azulpasta/aPRova-TESTES` | |
| `numero_pr` | inteiro | sim | `15` | |
| `url_pr` | texto (URL) | sim | `https://github.com/Azulpasta/aPRova-TESTES/pull/15` | |
| `etapa` | texto | sim | `testando_sandbox` | onde falhou: `pendente`, `mapeando_contexto`, `gerando`, `testando_sandbox`, `avaliando_risco` ou `aplicando` |
| `diagnostico` | texto | sim | `a execução do job aprova-sandbox no Azure Container Apps não começou em 120 s...` | até 4000 caracteres |
| `tentativa` | inteiro | sim | `1` | quantas vezes a análise já foi tentada |

O botão de reexecutar envia ao `POST /acoes-remotas` um pedido do tipo
`reexecutar_analise`.

### `instalacao`

Emitido quando o GitHub App é instalado, removido ou ganha ou perde
repositórios. O `tenant_id` do envelope é o `installation_id` dessa instalação.

Na instância, a saída `instalacao` do Core chama o workflow funcional
`aPRova | Onboarding de Repositório`, pelos nós `07K → 07L`. O contrato
abaixo é o envelope do emissor; o subworkflow consome a saída do normalizador:
`evento.tipo`, `evento.acao`, `instalacao.acao`, `instalacao.conta`,
`instalacao.repositorios` e `meta` com autenticação/validação. A emissão
automática deste evento pelo backend ainda não foi implementada.

Somente `instalado` e `repositorios_adicionados` geram ações, um repositório
por vez. Consulta a branch padrão e sua proteção, cria/reutiliza a lista
`owner/repo` no Trello e resume no Slack. Não cria cards ou altera regras
no GitHub. Distingue proteção confirmada, Check obrigatório pendente e
verificação indisponível; apenas a primeira confirma prontidão. A consulta
real ainda retorna 401, sem confirmação de proteção. Detalhes, configuração,
testes e pendências estão na
[documentação central](central-slack-implementacao.md#onboarding-de-repositório).

O export funcional é `workflows/n8n/onboarding-repositorio.json`; configure
`APROVA_ONBOARDING_WORKFLOW_ID` com o ID da importação. Os exports ficam
inativos e sem credenciais ou IDs locais; configure e valide os destinos
antes de habilitar Trello/Slack. O workflow manual de testes não foi exportado
nem conectado à rota real. O envelope v1 permanece inalterado.

| Campo | Tipo | Obrigatório | Exemplo | Observação |
| --- | --- | --- | --- | --- |
| `acao` | texto | sim | `instalado` | `instalado`, `removido`, `repositorios_adicionados` ou `repositorios_removidos` |
| `conta` | texto | sim | `Azulpasta` | usuário ou organização do GitHub |
| `repositorios` | lista de textos | sim | `["Azulpasta/aPRova-TESTES"]` | os repositórios afetados, como `dono/nome`; pode vir vazia |

## Assinatura

A URL do webhook fica exposta na internet, então qualquer um pode mandar um
`POST` para ela. Por isso o núcleo assina cada evento, e o workflow só deve
seguir depois de conferir a assinatura.

```
X-Aprova-Timestamp: 1760889600
X-Aprova-Assinatura: sha256=<hex do HMAC-SHA256 de "<timestamp>.<corpo bruto>">
```

- O segredo é `N8N_SEGREDO_EVENTOS`, que precisa ter o mesmo valor no núcleo e
  no n8n.
- Ele é diferente do `N8N_SEGREDO_ACOES`, que assina o caminho contrário (o n8n
  pedindo uma ação ao núcleo). Há um segredo por direção. Assim, quem descobrir
  um deles consegue falsificar só uma direção.
- O texto assinado é o timestamp, um ponto e o corpo **exatamente como chegou**,
  byte a byte. Se o n8n converter o JSON em objeto e depois de volta em texto,
  a ordem das chaves e os espaços podem mudar, e a assinatura deixa de conferir.
- Recuse timestamp com mais de 5 minutos de diferença do relógio do n8n, para
  os dois lados. Isso impede que alguém capture um evento verdadeiro e o
  reenvie depois.

### Nó Webhook

1. **HTTP Method**: `POST`.
2. **Path**: o caminho que vai em `N8N_WEBHOOK_URL`, como `aprova/events`
   no workflow importável deste repositório.
3. **Respond**: `Using 'Respond to Webhook' Node`, para responder logo depois
   de conferir a assinatura.
4. Em **Options**, ligue **Raw Body**. Com ele, o corpo exato fica no binário
   `data` do item, além do JSON já interpretado em `body`. Os cabeçalhos ficam
   em `headers`, com os nomes em minúsculas.

O workflow principal fica assim:

```
Webhook → assinatura → normalização/validação → resposta 202 → deduplicação → roteamento
```

No workflow do projeto, o erro de autenticação tem uma
saída dedicada que responde **401** e encerra o caminho, sem normalizar nem
rotear. Contrato inválido recebe **400**. O **202** confirma a entrada validada,
não a conclusão das integrações. O exemplo de Code abaixo demonstra a
assinatura; sua exceção deve manter a rota de recusa do Core.

### Nó Code

Modo **Run Once for All Items**, linguagem JavaScript:

```js
const crypto = require('crypto');

const segredo = $env.N8N_SEGREDO_EVENTOS;
if (!segredo) {
  throw new Error('N8N_SEGREDO_EVENTOS não está definido no n8n');
}

const cabecalhos = $input.first().json.headers;
const timestamp = cabecalhos['x-aprova-timestamp'] ?? '';
const assinatura = cabecalhos['x-aprova-assinatura'] ?? '';
const corpoBruto = await this.helpers.getBinaryDataBuffer(0, 'data');

if (!/^\d+$/.test(timestamp)) {
  throw new Error('timestamp ausente ou inválido');
}
if (Math.abs(Date.now() / 1000 - Number(timestamp)) > 300) {
  throw new Error('timestamp fora da janela de 5 minutos');
}

const esperada = Buffer.from(
  'sha256=' + crypto.createHmac('sha256', segredo).update(timestamp + '.').update(corpoBruto).digest('hex'),
);
const recebida = Buffer.from(assinatura);
if (recebida.length !== esperada.length || !crypto.timingSafeEqual(recebida, esperada)) {
  throw new Error('assinatura não confere');
}

return [{ json: JSON.parse(corpoBruto.toString('utf8')) }];
```

O `timingSafeEqual` compara em tempo constante. Uma comparação com `===` para
no primeiro caractere diferente, e medir esse tempo ajuda um atacante a
adivinhar a assinatura aos poucos.

O nó devolve o evento já conferido, e os nós seguintes leem `tipo_evento`,
`dados` e o resto direto do item.

### Variáveis de ambiente do n8n

O Compose de desenvolvimento do repositório injeta os segredos da integração
a partir do ambiente, libera `crypto` e configura o acesso a variáveis nos
nós. Os valores devem ser definidos no `.env` local, que não é versionado.
Em outro ambiente ou com runners externos, confira a configuração equivalente
e as permissões da versão utilizada. Não coloque valores reais no JSON
versionado do workflow. Essa configuração local não substitui a revisão de
segredos e permissões necessária para produção.

- `NODE_FUNCTION_ALLOW_BUILTIN=crypto` libera o `require('crypto')` no nó
  Code. Sem ela, o n8n bloqueia a importação. Com os task runners em modo
  externo (imagem `n8nio/runners`), a variável vai no `env-overrides` do runner
  JavaScript, no arquivo `n8n-task-runners.json`, e não no contêiner do n8n.
- `N8N_SEGREDO_EVENTOS` com o mesmo valor do núcleo.
- `N8N_BLOCK_ENV_ACCESS_IN_NODE` precisa estar `false` para o nó Code
  conseguir ler `$env`; não presuma o valor padrão da versão instalada.

`APROVA_CHAT_INSTALLATION_ID` configura consultas iniciadas pelo Slack e
Digest; não é segredo de autenticação nem substitui o `tenant_id` do evento.
Seu valor fica no ambiente, não no JSON. Depois de alterar o `.env`, recrie
`n8n-aprova` para carregar a configuração. Credenciais do GitHub pertencem
à API/worker, sem necessidade de copiá-las ao container do n8n.

## Testar sem o núcleo

O script `scripts/emitir-evento.sh` manda um evento de exemplo para qualquer
URL. Ele lê o exemplo do tipo em `workflows/exemplos/`, troca o `id_evento`
por um UUID novo e o `ocorrido_em` pela hora atual, assina como o núcleo assina
e envia por `POST`.

```sh
scripts/emitir-evento.sh <tipo> <url> [opções]
```

O script é só para desenvolvimento: não use com o segredo de produção.
Os exemplos são artificiais, não evidência de métricas ou permissões reais.
O tenant de um exemplo não é a instalação configurada para consultas;
o Core preserva o tenant do envelope assinado.

Precisa de `bash`, `curl`, `jq` e `openssl`; se faltar algum, o script diz
qual. O segredo vem da variável `N8N_SEGREDO_EVENTOS` ou, se ela não estiver no
ambiente, do `.env` na raiz do projeto. O script nunca o imprime. Ele mostra o
código HTTP da resposta e termina com erro se não for 2xx.

A base da `url_recibo` vem de `APROVA_URL_PUBLICA`, do ambiente ou do `.env`.
Vazia, fica o endereço provisório `https://aprova.example.com`.

| Opção | Efeito |
| --- | --- |
| `--repositorio dono/nome` | troca `dados.repositorio` e ajusta `url_pr` |
| `--numero-pr N` | troca `dados.numero_pr` e ajusta `url_pr` |
| `--nivel-risco NIVEL` | troca `dados.nivel_risco` |
| `--dado campo=valor` | troca qualquer campo de `dados`; um valor que é JSON válido (`7`, `true`, `[]`) entra como JSON, o resto como texto |
| `--imprimir` | mostra cabeçalhos e corpo sem enviar nada |

O script recusa sobrescrever um campo que o tipo não tem, como `--numero-pr`
num evento `instalacao`. O n8n do Compose local fica em
`http://localhost:5978` por padrão. A URL de teste do nó Webhook usa
`/webhook-test/` e só recebe enquanto o workflow está aberto no editor
esperando um evento. A de produção usa `/webhook/`.

Um exemplo por tipo:

```sh
URL=http://localhost:5978/webhook/aprova/events

scripts/emitir-evento.sh analise_concluida "$URL" --nivel-risco baixo --dado 'motivos=[]'
scripts/emitir-evento.sh aguardando_aprovacao "$URL" --repositorio PI-GRUPO/aPRova --numero-pr 7
scripts/emitir-evento.sh aprovacao_registrada "$URL" --dado decisao=aprovada --dado origem=github
scripts/emitir-evento.sh dependencia_nova "$URL"
scripts/emitir-evento.sh falha_recuperavel "$URL" --dado tentativa=3
scripts/emitir-evento.sh instalacao "$URL" --dado acao=repositorios_adicionados
```

Para ver exatamente o que seria enviado:

```sh
scripts/emitir-evento.sh aguardando_aprovacao "$URL" --imprimir
```

## O que o n8n não pode fazer

- **Decidir risco.** O nível vem pronto em `nivel_risco`. Um workflow pode
  filtrar e enfeitar a mensagem, mas não pode rebaixar nem elevar um risco, nem
  transformar o resultado do OSV.dev num novo nível.
- **Acessar o banco.** O n8n não recebe credencial do PostgreSQL. O que ele
  precisa ler vem nos eventos ou nos endpoints do núcleo, como o
  `GET /reports/weekly` do digest.
- **Aprovar sem passar pelo `POST /acoes-remotas`.** Nenhum workflow chama a
  API do GitHub para aprovar ou pedir alterações. O botão do Slack envia o
  pedido ao núcleo, e o núcleo confere no GitHub, a cada pedido, se a pessoa
  tem permissão de escrita no repositório.

## Configuração e portabilidade — 2026-11-08

`N8N_WEBHOOK_URL` permanece o destino completo dos eventos emitidos pelo
backend, incluindo `/webhook/aprova/events`. Vazio desliga a emissão.
`N8N_URL_BASE` configura o editor; `N8N_WEBHOOK_URL_PUBLICA` configura a base
pública dos webhooks por `WEBHOOK_URL` nativo no Compose, com fallback para a
URL do editor. Não confundir essas bases com o destino completo do emissor
ou com `APROVA_URL_PUBLICA`, base dos recibos.

As chamadas de subworkflows usam variáveis `APROVA_*_WORKFLOW_ID`. Destinos
Trello/Slack e flags do onboarding também vêm do ambiente. O [guia de
configuração](../workflows/n8n/README.md#contrato-de-configuração--2026-11-08)
registra consumidores, padrões, origem, recarga e migração. API e feedback
compartilham `SLACK_GITHUB_VINCULOS`; não há associações pessoais nos nós.

Tokens/OAuth ficam nas credenciais do n8n (`aPRova | Serviço`); HMAC e
Signing Secret ficam no ambiente. Exports são inativos, sem credenciais
pessoais ou IDs locais de workflow. Esta padronização não valida fluxos com
serviços externos nem implementa persistência de feedback ou executor de
reexecução. Gate, recibos e publicação reais foram incorporados da `dev`.
