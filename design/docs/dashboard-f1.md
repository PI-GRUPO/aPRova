# Fundação do dashboard — F1

Implementação em `feat/web-dashboard`, a partir de `c75c7d7` e da reorganização
solicitada para `design/docs/`. Este documento acompanha F1A–F1E e o card
[#64](https://trello.com/c/aqBohAcP). Não altera o estado do card.

## Escopo e comportamento

- F1A: aplicação React/TypeScript/Vite em `web/`, lockfile, Node 24, fontes locais,
  tokens, casca desktop/mobile, tema persistido, acessibilidade inicial e página 404.
- F1B: todas as rotas do inventário registradas; telas de fases futuras mostram
  indisponibilidade explícita. Campo com erro associado, botões, modal Radix,
  seleção Radix, tabela acessível e fronteira de erro reutilizáveis.
- F1C: `fetch` centralizado com erros tipados, prazo, cancelamento, validação de
  JSON/DTOs de saúde, 401 privado para `/entrar`, sem repetição automática.
  Proxy local com alvo fixo; origem pública opcional validada.
- F1D: harness e fixtures somente no modo de teste do Vite; Playwright cobre
  navegação, foco, formulário, filtros, ordenação, temas e falhas de HTTP.
- F1E: Go incorpora o build com `embed`; fallback somente para navegação HTML,
  cache de ativos versionados, testes Go e job adicional no CI.

As rotas de login, listas, análises/recibos por ID e gestão estão reservadas,
sem chamar contratos ausentes. A F1 não conclui o layout nem a integração das
páginas previstas em F2–F7. Nenhum usuário, instalação ou contador fictício
aparece na execução normal. O rodapé informa ausência de sessão; autenticação
GitHub será implementada em F3. Os dados não são calculados no navegador.

## Referência visual e adaptações funcionais

Arquivo Figma `VtHNXer6wzJj37QdpN3Lb9`, sessão de implementação F1.

| Referência | Nós consultados |
| --- | --- |
| Página 404 e navegação, escuro/claro | `86:5942`, `86:6451` |
| Composição decorativa 404 | `86:6054` (captura específica) |
| Navegação desktop ativa | `9:23` |
| Cabeçalho mobile nos dois temas | `16:2`, `88:826` |
| Menu mobile | `16:215` |
| Botões e campos | `5:135`, `5:144` |
| Estrutura de modal, escuro/claro | `10:1263`, `15:4170` |

A casca usa lateral de 240px com menu rolável e rodapé ancorado; no mobile,
marca, tema e quatro abas. A marca mobile abre o menu completo acessível para
alcançar todas as rotas, preservando o espaço do cabeçalho de referência.
Esse acesso às demais seções é uma adaptação funcional registrada; o modal
reutiliza o padrão visual e acrescenta as seções que o menu curto do Figma não lista.
O rodapé de conta é adaptado para a sessão ainda ausente, sem copiar identidades
e instalação simuladas. Não há ação de logout sem sessão implementada.

Os SVGs fornecidos pelo Figma são ativos locais em `web/src/ativos/`, com nomes
por função e tema. O item ativo usa o próprio SVG como máscara CSS de 16px,
aplicando o token de acento sem alterar os arquivos. A composição 404 é código e usa seus vetores originais; a
captura Figma não é incorporada como imagem do produto. Raízes SVG preservadas,
com dimensionamento do contêiner para adaptação mobile. Fontes locais Inter
Variable e IBM Plex Mono; pesos 590/510 e cores semânticas do design.

Foram reaproveitados tokens, lógica de tema, função de classes e primitivas de
seleção/tabela do protótipo, com inspeção e adaptação ao contexto consultado.
Avisos MIT da Arc preservados em `web/LICENSES-TERCEIROS.md`. TanStack permanece
na stack para as tabelas integradas futuras; a fundação usa as primitivas
semânticas existentes, sem supor a API v8 em uma instalação v9. Dialog adiciona
uma primitiva à base Radix existente. DevLibs ainda não está disponível como
ferramenta nesta sessão; não foi usado como fonte visual ou de componentes.

## Desenvolvimento e build

Requer Node 24 e Go conforme `go.mod`. Na raiz:

```sh
npm --prefix web ci
npm --prefix web run dev
```

Vite atende em `http://127.0.0.1:5173`. Os prefixos `/health`, `/ready`,
`/receipts` e `/api` encaminham para `http://127.0.0.1:8080`; não há proxy
arbitrário por parâmetro do navegador. O backend local é iniciado por
`make run-api`, com suas dependências e ambiente próprios. Sem API, o cliente
apresenta erro de conexão nos fluxos de consulta quando usados.

Produção, na raiz:

```sh
make build-api-web
```

A ordem é `npm ci` → build Vite → `go build -o bin/api ./cmd/api`. Os ativos são
incorporados no mesmo binário e origem da API. Reiniciar o binário em um
ambiente de homologação/deploy é uma ação separada; não foi feito deploy.
**Não executar build Node e compilação Go simultaneamente:** o Vite substitui
os arquivos que o Go lê para incorporar.

Um checkout limpo contém apenas `web/dist/.gitkeep`; Go compila normalmente,
mas navegação HTML responde 503 até compilar o front. O build limpa os ativos
anteriores e preserva o marcador. `dist` gerado, dependências, relatórios e caches
são ignorados. Não versionar o build nem usar `git add -f`.

HTML tem `Cache-Control: no-store`. Arquivos sob `/assets/` têm nome derivado do
conteúdo e cache imutável. Ausência de API, ativo ou caminho reservado não cai no
index. O servidor responde a GET/HEAD e não usa o fallback para POST, JSON ou
caminhos com segmentos ocultos/travessia. O 404 visual continua sendo rota da SPA.

Por padrão o cliente usa a mesma origem. `VITE_API_ORIGEM`, quando necessário
para um ambiente específico, aceita somente origem HTTP(S), sem credenciais,
caminho, query ou fragmento. É configuração pública incorporada no build;
**nenhum segredo deve usar prefixo `VITE_`**. Outra origem exigiria política CORS
e cookies apropriados no servidor; isso não é necessário para o deploy de mesma
origem definido pelo projeto. Segredos n8n/GitHub/recibos continuam só no Go.

## Testes reproduzíveis

```sh
npm --prefix web run verificar
npm --prefix web run build
cd web
npx playwright install --with-deps chromium
npm run teste
```

Playwright inicia Vite no modo `teste` e um servidor Go isolado em 18080. Este
último usa o roteador/handlers reais e uma dependência declaradamente indisponível
para exercitar `/ready` 503; não usa banco de desenvolvimento ou credenciais.
A navegação normal usa build de produção incorporado ao Go. Portanto, compilar
Vite antes dos testes. Em execução normal de Vite e no build de produção,
`/__teste` não existe como tela de harness; as fixtures não entram no bundle.

Novamente na raiz:

```sh
go test ./web ./internal/api -race
go build -o bin/api ./cmd/api
```

No ambiente restrito desta sessão, caches temporários foram selecionados com
`GOCACHE=/tmp/aprova-f0-go-cache`, `GOPATH=/tmp/aprova-go` e
`PLAYWRIGHT_BROWSERS_PATH=/tmp/aprova-playwright`. Portas locais, download de
ativos/dependências e bibliotecas de sistema do Chromium exigiram as permissões
do ambiente. Esses caminhos não são exigências do produto nem do CI.

As interceptações Playwright validam estados do cliente. O smoke HTTP do servidor
isolado valida handlers reais e a SPA incorporada, com dependência controlada.
Nenhum deles comprova Postgres/Redis de homologação, sessão ou recibo real.

## Evidências e aceite

Capturas/contextos Figma e screenshots locais em `.capturas-mcp/f1/`, ignorados
pelo Git. Prefixo `local-404-`: 1440×900, 390×844 e 360×844, nos dois temas.
`local-composicao-` facilita comparar a composição com `404-composicao-figma.png`.
O Playwright verifica carregamento e dimensões dos ativos visíveis, com tolerância
de um pixel para arredondamento do browser na escala mobile, e ausência de
rolagem horizontal. A comparação visual é inspecionada; não há alegação de
comparação automática pixel a pixel.

Resultados finais e status dos sublotes são registrados no plano após a validação.
Homologação e UUID de recibo real seguem pendentes da F0; não bloqueiam a fundação,
mas impedem o aceite integrado de F2.

## Resultado final — 16/11/2026

F1A–F1E concluídos no escopo da fundação do card #64.

| Validação | Resultado |
| --- | --- |
| `npm --prefix web ci --offline --cache /tmp/aprova-npm-cache` | Lockfile reinstalado com sucesso; zero vulnerabilidades reportadas pelo npm |
| `npm --prefix web run verificar` | Biome sem apontamentos |
| `npm --prefix web run build` | TypeScript e Vite aprovados |
| `npm --prefix web run teste -- --workers=2` (com os caches da sessão) | **12 testes passaram em 55,4s**, saída 0 |
| `go test ./internal/... ./cmd/... ./web/... -race` (com os caches da sessão) | Regressão aprovada |
| `golangci-lint run ./web/... ./internal/api/...` | Zero apontamentos |
| `go build -o /tmp/aprova-f1-api ./cmd/api` | Binário com SPA compilado |
| Conferência dos ativos e bundle | 34 SVGs locais não vazios e com dimensões válidas; sem harness/fixtures/URLs temporárias no JS de produção |
| Documentos e patch | Links locais existentes; `git diff --check` sem erros |

A validação de navegador incluiu timeout antes e durante a leitura do corpo,
abrangendo o cancelamento da decodificação JSON. Capturas dos dois temas e dos
três viewports foram inspecionadas. As adaptações de sessão e acesso ao menu
mobile estão descritas acima; não são dados ou ações simulados.

A regressão Go ampliada precisou de rede para uma dependência de testes existente
e de permissão para portas HTTP locais. A última execução separada passou;
os testes de navegador também encerraram seus servidores corretamente. O job CI
foi configurado, mas não foi disparado remotamente nesta execução. Não houve
integração com banco de homologação, publicação, deploy ou alterações no Slack.

Próximo lote de produto: F2 (recibo por UUID). Para o aceite integrado, permanecem
necessários a URL da API de teste e um UUID real acessível. A ausência do AGENTS
local continua registrada na F0.
