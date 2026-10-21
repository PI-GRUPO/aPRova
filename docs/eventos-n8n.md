# Eventos do aPRova para o n8n

Este guia é para quem constrói os workflows no n8n. Ele descreve os eventos que
o núcleo do aPRova envia, como conferir que um evento veio mesmo do núcleo e
como testar um workflow sem o núcleo rodando.

## Como o n8n se encaixa

O núcleo analisa o pull request, decide o risco e guarda tudo no banco. O n8n
fica na periferia: **recebe** eventos e avisa pessoas (Slack, Trello, SLA,
digest). Ele nunca decide risco e nunca lê o banco.

- O núcleo envia cada evento por `POST` para a URL do webhook do n8n
  (`N8N_WEBHOOK_URL`).
- O envio é *fire-and-forget*: o núcleo não espera o workflow terminar e não
  falha se o n8n estiver fora do ar. Um evento perdido não trava nenhuma
  análise, e por isso o workflow não pode ser a única fonte de nada.
- Tudo o que o n8n quiser **fazer** num PR volta pelo `POST /acoes-remotas`
  (veja o README), que confere a permissão no GitHub antes de agir.

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
| `id_evento` | texto (UUID) | sim | `0f8e6c1a-3b2d-4c5e-8f7a-9b0c1d2e3f40` | novo a cada envio; use para descartar repetidos |
| `tipo_evento` | texto | sim | `analise_concluida` | um dos seis tipos abaixo |
| `ocorrido_em` | texto (RFC 3339, UTC) | sim | `2026-10-19T14:32:08Z` | quando o fato aconteceu no núcleo |
| `tenant_id` | inteiro | sim | `58291734` | `installation_id` do GitHub App; identifica a conta |
| `dados` | objeto | sim | | campos do tipo, descritos abaixo |

O mesmo evento pode chegar duas vezes, por exemplo quando o núcleo reenvia
depois de um timeout. O `id_evento` se repete nesse caso. O nó **Remove
Duplicates**, com a opção de lembrar itens de execuções anteriores e a chave
`id_evento`, resolve isso.

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
| `motivos[].nome` | texto | sim | `sem_teste_para_alteracao` | identificador estável da regra |
| `motivos[].descricao` | texto | sim | `go/frete/frete.go mudou e nenhum teste Go cobre o caminho alterado` | frase curta para mostrar a pessoas |
| `explicacao` | texto | sim | `O PR muda a faixa de peso em frete.Calcular...` | explicação em português, até 4000 caracteres |
| `receipt_hash` | texto | sim | `9c4e2a7f...0e2f` | SHA-256 do recibo assinado, 64 caracteres hexadecimais minúsculos |
| `url_recibo` | texto (URL) | sim | `https://aprova.example.com/recibos/2b6d8f0a-...` | onde o recibo pode ser conferido |

O domínio `aprova.example.com` dos exemplos é provisório: o endereço público
do núcleo ainda não existe. Quando existir, ele vai na variável
`APROVA_URL_PUBLICA`, e só a base da URL muda; o caminho `/recibos/<id>`
continua o mesmo, então nenhum workflow precisa ser alterado.

### `aguardando_aprovacao`

Emitido quando o risco é alto e o PR fica bloqueado até uma decisão humana. Tem
**exatamente os mesmos campos** de `analise_concluida`, com `nivel_risco`
sempre `alto`. Uma análise de risco alto gera os dois eventos: o
`analise_concluida` para o histórico e este para a Central no Slack e para o
SLA.

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

Use este evento para encerrar o SLA do `task_run_id` e atualizar a mensagem do
alerta. Quando `nivel_risco_original` é `alto`, ele também dispara o pedido de
feedback ao revisor.

### `dependencia_nova`

Emitido quando o PR adiciona dependências.

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
2. **Path**: o caminho que vai em `N8N_WEBHOOK_URL`, por exemplo `aprova`.
3. **Respond**: `Using 'Respond to Webhook' Node`, para responder logo depois
   de conferir a assinatura.
4. Em **Options**, ligue **Raw Body**. Com ele, o corpo exato fica no binário
   `data` do item, além do JSON já interpretado em `body`. Os cabeçalhos ficam
   em `headers`, com os nomes em minúsculas.

O workflow principal fica assim:

```
Webhook → Code (confere a assinatura) → Respond to Webhook (200) → Remove Duplicates → Switch (tipo_evento)
```

Se a assinatura não conferir, o nó Code lança um erro antes do **Respond to
Webhook**, e o n8n responde 500. Assim o script de envio mostra que a
assinatura foi recusada.

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

- `NODE_FUNCTION_ALLOW_BUILTIN=crypto` libera o `require('crypto')` no nó
  Code. Sem ela, o n8n bloqueia a importação. Com os task runners em modo
  externo (imagem `n8nio/runners`), a variável vai no `env-overrides` do runner
  JavaScript, no arquivo `n8n-task-runners.json`, e não no contêiner do n8n.
- `N8N_SEGREDO_EVENTOS` com o mesmo valor do núcleo.
- `N8N_BLOCK_ENV_ACCESS_IN_NODE` precisa continuar `false`, que é o padrão,
  para o nó Code conseguir ler `$env`.

## Testar sem o núcleo

O script `scripts/emitir-evento.sh` manda um evento de exemplo para qualquer
URL. Ele lê o exemplo do tipo em `workflows/exemplos/`, troca o `id_evento`
por um UUID novo e o `ocorrido_em` pela hora atual, assina como o núcleo assina
e envia por `POST`.

```sh
scripts/emitir-evento.sh <tipo> <url> [opções]
```

O script é só para desenvolvimento: não use com o segredo de produção.

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
num evento `instalacao`. O n8n de teste local costuma escutar em
`http://localhost:5678`. A URL de teste do nó Webhook usa `/webhook-test/` e só
recebe enquanto o workflow está aberto no editor esperando um evento. A de
produção usa `/webhook/`.

Um exemplo por tipo:

```sh
URL=http://localhost:5678/webhook-test/aprova

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
