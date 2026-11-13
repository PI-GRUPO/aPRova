# Como verificar um recibo do aPRova

Cada análise do aPRova termina com um **recibo assinado**: o registro de que,
num dado momento, o aPRova analisou um commit exato de um pull request, obteve
um resultado de sandbox e decidiu um nível de risco pelos motivos listados.

O recibo é **assinado e verificável**. O conteúdo é público e qualquer pessoa
pode lê-lo; o que a assinatura garante é que ele não foi forjado nem alterado
depois. Para verificar, basta a **chave pública**. Não é preciso acesso ao
banco, nem conta, nem confiar no servidor do aPRova: quem verifica faz a conta
na própria máquina.

Este guia mostra três caminhos, do mais cômodo ao mais independente:

1. perguntar ao próprio aPRova (`GET /receipts/{id}/verify`);
2. o executável `verificar-recibo`, que não acessa banco nem rede;
3. `openssl` puro, sem nenhum código do aPRova.

Os exemplos usam o recibo real da análise do PR #1 do repositório
`Azulpasta/aPRova-TESTES`, emitido num ambiente local. Troque `$APROVA` pelo
endereço do núcleo e `$ID` pelo id do recibo, que vem no campo `url_recibo` dos
eventos.

```sh
APROVA=http://localhost:8080
ID=6e626559-146f-4b61-b486-a05cd64d2b25
```

## O que o recibo contém

O conteúdo assinado é um JSON em UTF-8, com os campos sempre na mesma ordem e
sem espaços. Este é o do exemplo, quebrado em linhas só para leitura:

```json
{"versao":1,
 "id":"6e626559-146f-4b61-b486-a05cd64d2b25",
 "task_run_id":"4e241c77-bfc6-4ad7-a216-0308f1e516a4",
 "tenant_id":161192271,
 "repositorio":"Azulpasta/aPRova-TESTES",
 "numero_pr":1,
 "head_sha":"d7ee920e4f9f8a2f413630afbe6c25435149a302",
 "diff_sha256":"eb988b7001848021e8fcaa95935067459229de32a4be3df52159ba15de24c972",
 "diff_truncado":false,
 "sandbox":{"compilou":true,"testes_passaram":10,"testes_falharam":1,"tipo_falha":"teste","tentativa":1},
 "nivel_deterministico":"alto",
 "nivel":"alto",
 "motivos":[{"id":"falha_sandbox","evidencias":["1 de 11 testes falharam"]}],
 "emitido_em":"2026-11-10T02:46:20.517956Z",
 "anterior":null,
 "key_id":"5ae5b1f862c3be22"}
```

| Campo | O que registra |
| --- | --- |
| `versao` | versão do formato; hoje, 1 |
| `id`, `task_run_id`, `tenant_id` | o recibo, a análise e a instalação do GitHub App |
| `repositorio`, `numero_pr`, `head_sha` | o commit exato que foi analisado |
| `diff_sha256` | SHA-256 do diff analisado; o diff em si não entra, porque o hash prova a mesma coisa e mantém o recibo pequeno |
| `diff_truncado` | verdadeiro quando o diff passou de 5 MiB e só o começo foi lido |
| `sandbox` | resultado dos testes e a tentativa que o produziu |
| `nivel_deterministico`, `nivel` | o nível dado pelas regras e o nível final |
| `motivos` | regras que dispararam, com as evidências |
| `emitido_em` | instante da emissão, em UTC |
| `anterior` | id e SHA-256 do recibo anterior da mesma instalação, ou `null` no primeiro |
| `key_id` | chave que assinou: os 16 primeiros caracteres hexadecimais do SHA-256 da chave pública |

A assinatura é Ed25519 sobre esses bytes, exatamente como estão. Por isso a API
entrega o conteúdo em base64 (`payload_base64`): copiar o JSON de uma tela pode
trocar um espaço ou a ordem de um campo, e um único byte diferente invalida a
assinatura.

## Caminho 1: perguntar ao aPRova

```sh
curl -s $APROVA/receipts/$ID/verify
```

```json
{"id":"6e626559-146f-4b61-b486-a05cd64d2b25","valido":true,
 "verificacoes":[{"nome":"hash","resultado":"ok"},{"nome":"chave","resultado":"ok"},
                 {"nome":"assinatura","resultado":"ok"},{"nome":"conteudo","resultado":"ok"},
                 {"nome":"elo","resultado":"ok"}],
 "hash_sha256":"6146756177c1e88eab1218c5032b4b698ad5c20b28ffe1ec81ac6086d88ada4f",
 "key_id":"5ae5b1f862c3be22"}
```

O servidor recalcula o hash dos bytes guardados, confere a assinatura com a
chave do `key_id`, confere que o conteúdo assinado é mesmo o daquele registro e
confere o elo com o recibo anterior. Quando algo falha, `valido` é `false`,
`falha` diz qual verificação reprovou (`hash`, `chave_desconhecida`,
`assinatura`, `conteudo` ou `elo`) e `detalhe` explica.

Esse caminho é cômodo, mas ainda é o aPRova dizendo que o aPRova está certo. Os
dois caminhos seguintes não dependem dele.

## Caminho 2: o executável `verificar-recibo`

Baixe o recibo e a chave pública uma vez. Depois disso, a verificação não usa
rede nem banco: pode desligar o aPRova.

```sh
curl -s $APROVA/receipts/$ID > recibo.json
curl -s $APROVA/receipts/keys
```

```json
{"chaves":[{"key_id":"5ae5b1f862c3be22","algoritmo":"Ed25519",
            "chave_publica_base64":"oxyRUCyPXF5VI60ZSAIszVUau7XIK44n/XOaOSOhkyM="}]}
```

```sh
go build -o verificar-recibo ./cmd/verificar-recibo
./verificar-recibo -chave 'oxyRUCyPXF5VI60ZSAIszVUau7XIK44n/XOaOSOhkyM=' recibo.json
```

```
VÁLIDO: a assinatura confere com o conteúdo e com a chave informada
recibo:       6e626559-146f-4b61-b486-a05cd64d2b25
key_id:       5ae5b1f862c3be22
hash sha256:  6146756177c1e88eab1218c5032b4b698ad5c20b28ffe1ec81ac6086d88ada4f
análise:      Azulpasta/aPRova-TESTES, PR #1, commit d7ee920e4f9f8a2f413630afbe6c25435149a302
diff sha256:  eb988b7001848021e8fcaa95935067459229de32a4be3df52159ba15de24c972
sandbox:      10 testes passaram, 1 falharam, falha teste, tentativa 1
nível:        alto (determinístico: alto)
motivo:       falha_sandbox: 1 de 11 testes falharam
emitido em:   2026-11-10 02:46:20 UTC
anterior:     nenhum, é o primeiro recibo da cadeia
```

O recibo também pode vir da entrada padrão (`... | ./verificar-recibo -chave ...`).
O programa sai com 0 quando o recibo é válido, 1 quando é inválido e 2 quando a
chave ou o arquivo não puderam ser lidos. Tudo o que ele mostra vem dos bytes
assinados; o campo `payload` do JSON, que existe só para leitura, é ignorado.

Com um único byte do conteúdo alterado:

```
INVÁLIDO: a verificação de hash falhou
o SHA-256 do conteúdo não é o hash informado
```

**De onde vem a chave pública importa.** Se a chave vier do mesmo servidor que
entregou o recibo, um servidor comprometido poderia entregar um recibo falso e
uma chave falsa que combinam. Para uma auditoria, guarde a chave pública (ou só
o `key_id`) por um canal separado, como o README do projeto ou um contrato, e
compare.

## Caminho 3: `openssl` puro

Nenhum código do aPRova: só `curl`, `jq`, `xxd` e OpenSSL 3. O LibreSSL que vem
com o macOS (`/usr/bin/openssl`) não verifica Ed25519; no macOS use o do
Homebrew (`brew install openssl@3`).

**1. Separar os bytes assinados e a assinatura.**

```sh
jq -r .payload_base64    recibo.json | openssl base64 -d -A > payload.bin
jq -r .assinatura_base64 recibo.json | openssl base64 -d -A > assinatura.bin
```

**2. Colocar a chave pública no formato que o OpenSSL lê.** A API entrega os 32
bytes da chave. O OpenSSL espera o envelope padrão (SubjectPublicKeyInfo), que
para Ed25519 é sempre o mesmo prefixo de 12 bytes seguido da chave:

```sh
CHAVE='oxyRUCyPXF5VI60ZSAIszVUau7XIK44n/XOaOSOhkyM='
{ printf '302a300506032b6570032100' | xxd -r -p; printf '%s' "$CHAVE" | openssl base64 -d -A; } \
  | openssl pkey -pubin -inform DER -out chave-publica.pem
```

**3. Verificar a assinatura.**

```sh
openssl pkeyutl -verify -pubin -inkey chave-publica.pem -rawin -in payload.bin -sigfile assinatura.bin
```

```
Signature Verified Successfully
```

`-rawin` diz ao OpenSSL para usar os bytes como estão, sem resumi-los antes: é
assim que o Ed25519 funciona. Com um byte de `payload.bin` alterado, a resposta
é `Signature Verification Failure`.

**4. Conferir o hash e o `key_id`.**

```sh
openssl dgst -sha256 -r payload.bin
jq -r .hash_sha256 recibo.json
```

```
6146756177c1e88eab1218c5032b4b698ad5c20b28ffe1ec81ac6086d88ada4f
6146756177c1e88eab1218c5032b4b698ad5c20b28ffe1ec81ac6086d88ada4f
```

```sh
printf '%s' "$CHAVE" | openssl base64 -d -A | openssl dgst -sha256 -r | cut -c1-16
jq -r .key_id recibo.json
```

```
5ae5b1f862c3be22
5ae5b1f862c3be22
```

**5. Ler o que foi assinado.**

```sh
jq . payload.bin
```

## Conferir que o recibo fala do diff certo

O `diff_sha256` é o SHA-256 do diff do pull request, como o GitHub o entrega.
Enquanto o PR continuar no mesmo commit, qualquer pessoa recalcula:

```sh
curl -sL https://github.com/Azulpasta/aPRova-TESTES/pull/1.diff | openssl dgst -sha256 -r
```

```
eb988b7001848021e8fcaa95935067459229de32a4be3df52159ba15de24c972
```

É o mesmo valor que está no recibo. Depois de um commit novo o diff do PR muda,
e o que identifica a análise passa a ser o `head_sha`.

## A cadeia de recibos

Cada recibo traz, dentro do conteúdo assinado, o id e o SHA-256 do recibo
anterior da mesma instalação. O primeiro aponta para nada (`"anterior":null`).

Isso torna detectável apagar um recibo do meio: o recibo seguinte continua
apontando para um hash que não existe mais, e a verificação do elo reprova. E
não dá para consertar o seguinte, porque o apontador está sob a assinatura.

Para conferir o elo por conta própria, baixe o recibo anterior pelo id que está
em `anterior.id`, extraia o `payload.bin` dele e compare o SHA-256 com
`anterior.hash`.

A cadeia nunca bifurca: a emissão é serializada por instalação dentro da
transação que grava o recibo, e um índice único impede dois recibos de terem o
mesmo anterior.

## O que o recibo não prova

- **Não prova que a decisão estava certa.** Prova que foi essa a decisão, com
  esses dados, e que ninguém a mudou depois.
- **Não protege contra quem tem a chave privada.** Quem a tiver assina o que
  quiser. Ela fica só na variável `APROVA_CHAVE_RECIBO` do worker e nunca
  aparece em log, erro ou resposta; a API verifica só com chaves públicas.
- **Apagar o último recibo da cadeia não é detectável pela cadeia**, porque
  nenhum recibo aponta para ele ainda. O banco recusa `UPDATE`, `DELETE` e
  `TRUNCATE` em `receipts`, mas quem administra o banco pode desligar essa
  trava. O que fecha essa brecha é guardar o hash do recibo fora do banco, e
  ele já sai no campo `receipt_hash` dos eventos enviados ao n8n.
- **O resultado do sandbox é o que o sandbox mediu.** Os limites dele estão em
  [imagens/go/README.md](../imagens/go/README.md) e
  [imagens/python/README.md](../imagens/python/README.md).

## Troca de chave

`GET /receipts/keys` devolve uma lista, e não uma chave só. Ao trocar a chave
que assina, a chave pública antiga continua na lista
(`APROVA_CHAVES_PUBLICAS_RECIBO`), e os recibos antigos continuam verificáveis:
cada recibo diz, no `key_id`, com qual chave conferir.

## Por que Ed25519 e não HMAC

Com HMAC, verificar exige o mesmo segredo usado para assinar. Só quem emitiu
conseguiria conferir, e quem conseguisse conferir conseguiria forjar. Com
Ed25519, a chave pública verifica e não serve para assinar. É isso que permite
a alguém de fora conferir sozinho.
