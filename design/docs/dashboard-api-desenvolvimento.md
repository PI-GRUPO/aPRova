# API para desenvolvimento do dashboard

Ciclo na branch `feat/web-dashboard`, base `d1a255d`, em 18/11/2026.
Escopo de integração do front relacionado aos cards #64/#65/#66; não conclui
homologação OAuth nem altera o backend. Nenhum card foi movimentado.

## Configurar o destino das consultas

O Vite mantém o navegador em `http://127.0.0.1:5173` e encaminha `/health`,
`/ready`, `/receipts`, `/api` e `/auth` à API, incluindo o callback OAuth.
Por padrão o destino continua `http://127.0.0.1:8080`.

Para uma API local em outra porta, crie `web/.env.local` (ignorado pelo Git):

```dotenv
APROVA_API_PROXY=http://127.0.0.1:18081
```

Reinicie `npm --prefix web run dev`. A variável aceita somente uma origem
HTTP(S), sem credenciais, caminhos, consulta ou fragmento. A configuração é
lida no processo Node do Vite e não entra no bundle. `web/.env.example` é uma
referência sem segredos; não substitua o `.env` já existente no backend.

O proxy não adiciona autorização, não reescreve cookies/redirects e preserva
Origin, cookies e CSRF. A API deve aceitar a origem pública usada pelo navegador.
Um endereço remoto, sozinho, não resolve o login: se o backend estiver configurado
com uma origem HTTPS, seus cookies seguros e callback exigem que o front seja
acessado pela origem HTTPS correspondente, com encaminhamento de `/auth` e `/api`.
Não desabilite validação TLS nem altere cookies para contornar essa configuração.
O exemplo HTTP acima se destina ao desenvolvimento local.

## Preparar o login real com GitHub

O responsável pelo ambiente precisa disponibilizar uma versão atualizada da API,
com PostgreSQL/Redis acessíveis, e configurar no `.env` do backend:

- `WEB_URL_PUBLICA=http://127.0.0.1:5173` para este front local.
- `GITHUB_CLIENT_ID` e `GITHUB_CLIENT_SECRET` do GitHub App usado no ambiente.
- `WEB_CHAVE_SESSAO`: chave aleatória de 32 bytes codificada em base64,
  mantida estável enquanto as sessões desse ambiente forem utilizadas.

Cadastre no GitHub App o callback
`http://127.0.0.1:5173/auth/github/callback`. Use o mesmo host durante todo o
fluxo; não alterne entre `localhost` e `127.0.0.1`. O App deve estar instalado
no repositório de teste e o usuário autenticado deve ter acesso a ele.
As demais configurações necessárias ao núcleo continuam sendo as existentes
em `.env.example`. Não envie segredos em mensagens nem coloque valores em `VITE_*`.

Na raiz, `make run-api` executa o código atual. Se a porta já estiver ocupada,
o responsável deve substituir o processo antigo ou escolher outra porta usando
a configuração existente `PORT`, ajustando o proxy. Este ciclo não encerrou,
reiniciou ou substituiu a API em execução.

## Conferir o ambiente

Consulte pelo endereço do front, para exercitar o mesmo proxy do navegador:

```sh
curl -i http://127.0.0.1:5173/health
curl -i http://127.0.0.1:5173/ready
curl -i http://127.0.0.1:5173/api/sessao
```

`/health` confirma atendimento HTTP; `/ready` verifica as dependências.
Na versão atual do código, `/api/sessao` retorna JSON mesmo sem OAuth habilitado.
Sem autenticação deve informar `autenticado: false`; `login_disponivel: true`
indica que o fluxo está habilitado, sem comprovar credenciais válidas no GitHub.
404 nessa rota exige conferir a versão e o destino da API. Depois, abra `/entrar`,
autorize o App no GitHub e confira o escopo autorizado em análises e repositórios.
Só esse teste com a conta real confirma o fluxo completo.

Na verificação desta data, a porta 8080 atendia pelo executável
`/tmp/aprova-atualizacao-local-20261114/bin/api`; `/health` retornou 200 e
`/api/sessao` e `/auth/github` retornaram 404. As quatro configurações OAuth
estavam ausentes no processo e no `.env` local. Nenhum segredo foi exibido.
Esse diagnóstico substitui as descrições históricas do container em F3 para
este ambiente local; não representa o estado de outros servidores.

## O que fica disponível

Com sessão válida, o front já consulta o catálogo autorizado, lista/detalhe de
análises e relatório semanal. Recibos por UUID, verificação e chaves públicas
usam as rotas públicas existentes. Uma análise/recibo real precisa existir no
ambiente; autenticar não cria dados. Rotas n8n com HMAC não são login web.

Visão geral de 30 dias, mapa, fila demonstrativa, decisões/reexecuções simuladas,
gestão, coleção/cadeia e vínculo Slack continuam nos mocks separados quando não
há contrato correspondente. Uma falha de API real nunca ativa esses mocks.

## Validação do ciclo

Arquivos: `web/vite.config.ts`, `web/.env.example`, `web/e2e/proxy.spec.ts`,
`web/README.md` e documentação do dashboard. Sem mudanças em componentes,
estilos, ativos, dependências ou contratos HTTP. Os hashes dos arquivos de
interface produzidos pelo build permaneceram iguais aos da base; não há nova
composição visual ou nó Figma neste ciclo.

Os testes iniciam uma API HTTP isolada e um Vite com destino configurado,
validando encaminhamento, query string, OAuth/callback, redirect, HttpOnly,
cookies, Origin, CSRF e preservação de 404 sem fallback HTML. Configurações
inválidas são rejeitadas sem expor seu conteúdo. Essas fixtures não autenticam
uma conta real nem comprovam disponibilidade da API de desenvolvimento.

Biome e TypeScript/build aprovados; permanece o aviso existente de bundle acima
de 500 kB. Testes específicos: 2 aprovados. Regressão completa: 110 testes
aprovados, incluindo login nos dois temas, desktop/mobile, análises, recibos,
gestão e fluxos simulados. Confirmada a ausência de `APROVA_API_PROXY` nos
arquivos JavaScript do build. A porta 5173 respondeu 200 em `/entrar`.

Pendência: disponibilizar a API atualizada/configurada e validar OAuth com a
conta de teste. Nenhuma API, SQL, migration, Go ou workflow foi alterado.
