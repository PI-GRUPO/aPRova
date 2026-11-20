# Implantação da API e do worker na Azure

O sandbox já roda na Azure (`aprova-sandbox-teste`, imagens em `aprovasandbox`).
Este guia publica o resto do núcleo: a **API**, que recebe os webhooks do GitHub
e precisa de uma URL HTTPS pública, e o **worker**, que leva cada análise pelas
etapas e dispara o job do sandbox.

```text
GitHub ──webhook──▶ aprova-api (Container App, HTTPS público)
                        │
                        ▼
                Postgres (Neon) ◀── aprova-worker (Container App, sem ingress)
                        ▲                 │
                Redis (Upstash)           └──▶ aprova-sandbox-teste (Container Apps Job)
```

As duas imagens saem do mesmo Dockerfile, `imagens/nucleo/Dockerfile`, com os
alvos `api` e `worker`. A base é `distroless/static` fixada por digest: só o
binário, os certificados HTTPS e um usuário sem privilégios (UID 65532), sem
shell. A API escuta na porta 8080.

## Quem faz cada passo

| Passo | Permissão necessária |
| --- | --- |
| Banco e Redis | conta no Neon e no Upstash |
| Publicar as imagens | `AcrPush` no registro (o workflow já tem) |
| Criar os Container Apps, segredos e variáveis | **Colaborador** no grupo `rg-aprova` |
| Dar à identidade dos apps o papel `AcrPull` | **Proprietário** ou Administrador de Acesso de Usuário |
| Trocar a URL do webhook | dono do **GitHub App** |
| Alerta de orçamento | Colaborador ou acima na assinatura |

## 0. Variáveis do terminal

Os passos usam a Azure CLI (`az login` antes). Ajuste os valores se mudarem:

```sh
GRUPO=rg-aprova
AMBIENTE=$(az containerapp env list -g "$GRUPO" --query "[0].name" -o tsv)
REGISTRO=aprovasandbox
SERVIDOR="$REGISTRO.azurecr.io"
SHA=$(git rev-parse HEAD)   # o commit que será publicado
```

## 1. Banco e Redis

1. **Postgres (Neon):** crie o projeto e copie a string de conexão com
   `sslmode=require`. A primeira migration cria a extensão `vector`, que o Neon
   oferece. Rode as migrations a partir de uma cópia do repositório:

   ```sh
   make migrate-up DATABASE_URL='postgres://...neon.tech/...?sslmode=require'
   ```

2. **Redis (Upstash):** crie o banco e copie a URL `rediss://...` (com dois
   `s`, de TLS).

## 2. Publicar as imagens

**Pelo workflow:** `imagem-nucleo` publica `aprova-api` e `aprova-worker` com a
tag do commit a cada push na `main` que mude o núcleo, e pode ser disparado à
mão em **Actions → imagem-nucleo → Run workflow**. A credencial OIDC do registro
só aceita a `main`, a mesma do workflow do sandbox. Uma tag já publicada nunca é
sobrescrita.

**Sem passar pela main**, de qualquer cópia do repositório, com `AcrPush`:

```sh
az acr build -r "$REGISTRO" -f imagens/nucleo/Dockerfile --target api    -t "aprova-api:$SHA" .
az acr build -r "$REGISTRO" -f imagens/nucleo/Dockerfile --target worker -t "aprova-worker:$SHA" .
```

## 3. Container App da API

```sh
az containerapp create -n aprova-api -g "$GRUPO" --environment "$AMBIENTE" \
  --image "$SERVIDOR/aprova-api:$SHA" \
  --registry-server "$SERVIDOR" --registry-identity system \
  --ingress external --target-port 8080 \
  --min-replicas 0 --max-replicas 1 --cpu 0.25 --memory 0.5Gi

az containerapp secret set -n aprova-api -g "$GRUPO" --secrets \
  banco='postgres://...?sslmode=require' \
  redis='rediss://...' \
  webhook='<GITHUB_WEBHOOK_SECRET>' \
  chave-app='<GITHUB_PRIVATE_KEY_BASE64>' \
  n8n='<N8N_SEGREDO_ACOES>'

az containerapp update -n aprova-api -g "$GRUPO" --set-env-vars \
  DATABASE_URL=secretref:banco REDIS_URL=secretref:redis \
  GITHUB_WEBHOOK_SECRET=secretref:webhook GITHUB_PRIVATE_KEY_BASE64=secretref:chave-app \
  N8N_SEGREDO_ACOES=secretref:n8n \
  GITHUB_APP_ID='<id do app>' APROVA_CHAVES_PUBLICAS_RECIBO='<chave pública>' LOG_LEVEL=info

URL_API="https://$(az containerapp show -n aprova-api -g "$GRUPO" --query properties.configuration.ingress.fqdn -o tsv)"
echo "$URL_API"
```

`--registry-identity system` cria a identidade do app e dá a ela o papel
`AcrPull` no registro. Atribuir papel exige ser Proprietário: com Colaborador o
comando falha nesse ponto, e quem é Proprietário roda depois:

```sh
az role assignment create --role AcrPull \
  --assignee "$(az containerapp show -n aprova-api -g "$GRUPO" --query identity.principalId -o tsv)" \
  --scope "$(az acr show -n "$REGISTRO" --query id -o tsv)"
```

Com `--min-replicas 0` a API dorme sem tráfego e acorda no primeiro pedido,
como pede o card; a primeira resposta depois de um tempo parado é mais lenta.

## 4. Container App do worker

O worker não recebe tráfego: ele busca trabalho no banco. Por isso fica sem
ingress e com uma réplica sempre ligada, senão nenhuma análise anda.

```sh
az containerapp create -n aprova-worker -g "$GRUPO" --environment "$AMBIENTE" \
  --image "$SERVIDOR/aprova-worker:$SHA" \
  --registry-server "$SERVIDOR" --registry-identity system \
  --min-replicas 1 --max-replicas 1 --cpu 0.25 --memory 0.5Gi

az containerapp secret set -n aprova-worker -g "$GRUPO" --secrets \
  banco='postgres://...?sslmode=require' \
  redis='rediss://...' \
  chave-app='<GITHUB_PRIVATE_KEY_BASE64>' \
  chave-recibo='<APROVA_CHAVE_RECIBO>' \
  azure='<AZURE_CLIENT_SECRET>'

az containerapp update -n aprova-worker -g "$GRUPO" --set-env-vars \
  DATABASE_URL=secretref:banco REDIS_URL=secretref:redis \
  GITHUB_PRIVATE_KEY_BASE64=secretref:chave-app APROVA_CHAVE_RECIBO=secretref:chave-recibo \
  AZURE_CLIENT_SECRET=secretref:azure \
  GITHUB_APP_ID='<id do app>' GITHUB_APP_SLUG='<slug do app>' \
  APROVA_URL_PUBLICA="$URL_API" \
  SANDBOX_EXECUTOR=azure SANDBOX_IMAGEM_GO="$SERVIDOR/aprova-sandbox-go:<sha publicado>" \
  AZURE_SUBSCRIPTION_ID='<...>' AZURE_TENANT_ID='<...>' AZURE_CLIENT_ID='<...>' \
  AZURE_RESOURCE_GROUP="$GRUPO" AZURE_SANDBOX_JOB_NAME=aprova-sandbox-teste LOG_LEVEL=info
```

O worker confere a configuração ao subir e para com uma mensagem que nomeia o
que falta (por exemplo, sem `GITHUB_APP_SLUG`). Opcionais, conforme o que for
ligado: `SANDBOX_IMAGEM_PYTHON`, `SANDBOX_LINGUAGENS`, `SANDBOX_DIRETORIOS`,
`N8N_WEBHOOK_URL` com `N8N_SEGREDO_EVENTOS`, `CALIBRACAO_LIGADA` com
`GEMINI_API_KEY`, `SARIF_LIGADO`. A imagem Python ainda não tem workflow que a
publique no registro, e a Java só roda no executor local.

O papel `AcrPull` da identidade do worker segue a mesma regra do passo 3.

## 5. Webhook do GitHub App

Quem é dono do GitHub App troca, em **Settings → Developer settings → GitHub
Apps → (o app) → Webhook URL**, para:

```text
<URL_API>/webhooks/github
```

O segredo do webhook continua o mesmo que está em `GITHUB_WEBHOOK_SECRET`.

## 6. Conferir

```sh
make smoke-test URL="$URL_API"
```

Com `SMOKE_RECIBO_ID`, `N8N_SEGREDO_ACOES` e `SMOKE_INSTALLATION_ID` ele também
verifica um recibo e as rotas assinadas. Os logs ficam em **Container App →
Monitoramento → Fluxo de logs**, em JSON.

## 7. Alerta de orçamento

No portal, **Gerenciamento de Custos → Orçamentos → Adicionar**, com um valor
mensal e alerta por e-mail em 50%, 80% e 100%. O worker fica ligado o tempo
todo, então é o principal custo fixo do núcleo.

## Publicar uma versão nova

Publique as imagens do commit novo (passo 2) e aponte os apps para ele:

```sh
az containerapp update -n aprova-api    -g "$GRUPO" --image "$SERVIDOR/aprova-api:$SHA"
az containerapp update -n aprova-worker -g "$GRUPO" --image "$SERVIDOR/aprova-worker:$SHA"
```

Antes de publicar um commit que traga migration nova, rode `make migrate-up`
contra o banco (passo 1). O worker recebe o sinal de parada da Azure ao trocar
de versão e encerra o que estava fazendo antes de sair.
