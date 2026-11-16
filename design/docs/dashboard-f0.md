# Dashboard — levantamento F0

Revisão de 16/11/2026. Este registro atualiza a base histórica de
[implementacao-dashboard.md](implementacao-dashboard.md). Em caso de divergência
sobre a disponibilidade do código, vale esta auditoria no SHA abaixo.

## Base e instruções

- Branch: `feat/web-dashboard`, criada de `dev`.
- SHA auditado: `a28725daaf15968f45fbcb64ae463bbd9bb14c67`; `dev` e
  `upstream/dev` locais apontavam para o mesmo SHA. Não foi feito pull/fetch.
- Checkout limpo ao iniciar; arquivos deste levantamento são as únicas alterações.
- A branch `feat/slack-app-home` não foi incorporada. Comparação somente de leitura:
  22 arquivos adicionais, incluindo `internal/slackhome/` e `cmd/slack-home/`.
  As consultas de análises descritas abaixo já pertencem à `dev` escolhida.
- `AGENTS.md` não existe neste ambiente; `.agents/` está vazia. Não foi possível
  revalidar as orientações locais unificadas mencionadas no plano. Nenhum texto
  anterior foi reconstruído por suposição. O arquivo continua ignorado pelo Git.
- Convenções aplicáveis: plano de dashboard, PRODUCT, licenças e instruções da
  sessão. A usuária definiu o Figma como fonte visual: o protótipo local não é fiel.
  Reaproveitar código somente após conferir sua adequação ao nó correspondente.
- Workflows, flags Slack/SLA, deploy e cards não foram modificados.

## Contratos confirmados no núcleo

Fonte: `internal/api/servidor.go`, handlers, `cmd/api/main.go`,
`internal/receipt/exportacao.go` e `queries/consulta_analises.sql`.
Há **10 rotas**, em vez das oito da revisão histórica. Implementação em código
não comprova disponibilidade no ambiente implantado.

| Método e caminho existente | Autenticação / campos e limites | Uso web |
| --- | --- | --- |
| `GET /health` | Público; `{status: "ok"}` | Smoke F1C |
| `GET /ready` | Público; `status`, `dependencias`; 200/503, verifica Postgres/Redis | Estado de prontidão F1C |
| `POST /webhooks/github` | Assinatura GitHub do corpo | Entrada de eventos; não é formulário web |
| `POST /acoes-remotas` | HMAC n8n, timestamp; identidade `usuario_slack`, instalação, repo, PR, ação e justificativa | Exige identidade/autorização/origem web em F5; reexecução usa reenfileirador indisponível |
| `GET /reports/weekly` | HMAC n8n da query; `installation_id`, `inicio`; janela de sete dias UTC | Reusar no servidor após autorizar sessão F3 |
| `GET /api/analises` | HMAC n8n de método + caminho + query; instalação obrigatória | Resumo de serviço existente; adaptação web em F3/F4A |
| `GET /api/analises/{id}` | Mesmo HMAC; UUID + instalação; repo opcional; 404 fora do escopo | Resumo existente; falta detalhe agregado e autorização por sessão |
| `GET /receipts/keys` | Público; `chaves[]`: `key_id`, `algoritmo`, `chave_publica_base64` | F2 / seção pública de chaves |
| `GET /receipts/{id}` | Público por UUID; envelope exportável | F2, sem inventar coleção |
| `GET /receipts/{id}/verify` | Público; `valido`, `falha?`, `detalhe?`, `verificacoes[]`, hash e chave | F2; 200 pode ser inválido, verifica recibo/elo e não a cadeia inteira |

Consultas de análises usam `N8N_SEGREDO_ACOES`, assinatura de
`timestamp + "." + método + "\n" + caminho escapado + "\n" + query bruta`,
com janela de cinco minutos. O relatório assina `timestamp + "." + query bruta`.
Não compartilhar a credencial com o navegador. O escopo de serviço é informado
na query; isso **não implementa** isolamento por usuário GitHub autenticado.

Lista de análises:

- Query permitida: `installation_id`, `repositorio`, `estado`, `numero_pr`,
  `pagina`, `limite`; parâmetros desconhecidos/duplicados são recusados.
- Padrões: estado `em_andamento`, página 1, limite 20. Limite 1–50; página
  1–201; deslocamento máximo 10000. `estado=todos` inclui terminais.
- Resposta: `installation_id`, `consultado_em`, `estado`, `repositorio`,
  `numero_pr`, `pagina`, `limite`, `tem_mais`, `itens[]`.
- Cada item: `id`, `repositorio`, `numero_pr`, `sha_head`, `estado`,
  `recebido_em`, `criado_em`, `atualizado_em`. Detalhe retorna
  `installation_id`, `consultado_em`, `analise` com os mesmos campos.
- Ordenação `created_at DESC, id DESC`, paginação por offset com item adicional
  para `tem_mais`; não é snapshot imutável sob inserções concorrentes.
- Ainda faltam filtros de nível/período, motivos, timeline, tentativas, saídas,
  autoria, decisão, permissões e vínculo com recibo exigidos pelo card #66.
- Erros: 401 sem assinatura válida; 400 parâmetros inválidos; 503 fonte ausente;
  500 falha da fonte; detalhe 404 fora do escopo ou inexistente.

Recibo: `id`, `task_run_id`, `key_id`, `hash_sha256`, `payload_base64`,
`assinatura_base64`, `recibo_anterior_id` anulável e `payload` de leitura.
Preservar os bytes assinados; não usar a resserialização de `payload` para
verificar. Conferências: `hash`, `chave`, `assinatura`, `conteudo`, `elo`, cada
uma com `ok`, `falhou` ou `nao_verificada`. UUID inválido/inexistente: 404;
falha de leitura: 500; fonte ausente: 503. Chave pública vazia não prova validade.

Relatório: instalação, início/fim, totais, concluídas, rejeitadas, falhas,
em andamento, taxa de sucesso anulável, exigiram aprovação humana, risco médio
e revertidas. Não equivale às quatro métricas de 30 dias do desenho. Tempo médio
do card #67 continua obrigatório: apresentar em contexto próprio da visão geral
em F4B, preservando as quatro métricas do Figma e o contrato real.

## Matriz tela × endpoint × campos × permissão

Os nós desktop escuro/claro estão no inventário da seção 4 do plano e foram
reencontrados no arquivo. [Inventário completo](dashboard-inventario-figma.md)
inclui mobile, modais, variantes e dimensões efetivamente consultados.

| Rota / cards | Nós escuro / claro; mobile quando existente | Campos necessários / fonte existente | Permissão e contrato pendente |
| --- | --- | --- | --- |
| `/entrar` #65 | `9:2` / `15:1769` | Identidade e estado da autorização; sem endpoint | Público; C01 sessão OAuth GitHub |
| `/` #67 | `9:22` / `15:1789`; `16:2` / `88:826` | Período, fila/idade, riscos, tentativas, séries, tempo médio; semanal é parcial | Usuário/repositórios autorizados; C01/C02/C05 |
| `/analises` #68 | `9:385` / `15:2152`; `88:653` / `88:1546` | Lista/filtros/paginação; resumo HTTP existe | C01 e evolução C03 para filtros/dados web |
| `/analises/:id` #69/#74 | `9:608` / `15:2375`; `16:58` / `88:1168` | SHA, timeline, sandbox, gate, decisão, recibo; apenas resumo existe | Leitura autorizada C03; ações C09, sem autoaprovação |
| `/recibos` #70 | `9:829` / `15:2596`; `16:119` / `88:1244` | Coleção, cadeia, exportação; chaves públicas existentes | Coleção privada C06; chaves públicas F2 |
| `/recibos/:id` #70 | `9:1024` / `15:2791` | Documento, cinco conferências e chave; três rotas públicas existentes | Público por UUID F2; não autoriza consultar instalação privada |
| `/repositorios` #71 | `9:1185` / `15:2952` | Catálogo, permissões, check obrigatório, última análise | Leitura GitHub autorizada C04; 403 = não confirmado |
| `/regras` #72 | `9:1351` / `15:3118`; `88:741` / `88:1632` | Catálogo, limites, acionamentos e denominadores | Leitura autorizada C07; domínio interno não é API |
| `/auditoria` #73 | `9:1565` / `15:3332` | Ator, origem, tentativa/resultado e justificativa | Leitura escopada C08; origem web exige evolução |
| `/configuracao` #75 | `10:616` / `15:3523` | Efetiva, versão, histórico/diff, caminhos, versão assinada | Admin conferido no GitHub C10; sem CRUD atual |
| `/sugestoes` #76 | `10:791` / `15:3698` | Proposta, evidência, estado e versão alvo | Revisão humana administrativa C11; sem API atual |
| `/conta` #77 | `10:943` / `15:3850` | Sessão e prova de propriedade Slack | Própria conta C01/C12; resolvedor atual em memória |
| `/sla` #78 | `10:1078` / `15:3985` | Dois limites em minutos desde início da espera, versão/efeito | Admin por repo C13 e consumidor n8n; sem CRUD atual |
| `*` #64 | `86:5942` / `86:6451` | Caminho não encontrado | Sem API; F1A |

Coleções, sessão, métricas, detalhe agregado e operações administrativas continuam
propostas C01–C14 conforme seção 7 do plano. A existência dos caminhos C03 não
conclui o contrato nem o card #66. Recibos completos de instalação exigem C06.

## Levantamento visual e reaproveitamento

Conexão MCP confirmada nas três páginas. Contextos com captura consultados:
fundamentos `5:58`, visão desktop `9:22` e mobile `16:2`. Evidências locais em
`.capturas-mcp/f0/`: três PNGs, contextos e XML das páginas `0:1`, `5:2`, `5:3`.
São arquivos ignorados, sem inclusão forçada no Git. As capturas mostram o frame;
conteúdo rolável abaixo da dobra será capturado no lote de cada tela.

- Referência visual: Figma atual. PRODUCT orienta produto/stack; DESIGN local
  é candidato a tokens, sujeito à conferência com o Figma.
- Fundamentos conferidos: Inter para interface, Plex Mono para identificadores;
  fundos `#08090A`, `#0F1011`, `#161718`, borda `#23252A`, títulos brancos,
  corpo `#D0D6E0`, secundário `#8A8F98`, acento `#E4F222`; três cores semânticas
  de risco. Controles 32px, raios 6px/12px/4px, bordas 1px, sem sombras.
- O Figma usa estilos de peso 590/510; a exportação React arredonda classes para
  `font-semibold`/`font-medium`. A implementação deve preservar os pesos reais.
- Desktop: lateral 240px, conteúdo com padding 32px; mobile 390×844 com conteúdo
  rolável, margens 16px e métricas em duas colunas. Validar também 360px em F1.
- Diferenças confirmadas por inspeção: `Casca.tsx` local tem quatro links e não
  contém as seções completas de navegação/gestão e rodapé do desktop Figma;
  `FaixaDeMetricas.tsx` põe unidade junto do valor, enquanto o Figma a separa em
  linha própria e reserva espaço para a proporção. Não portar esses layouts sem
  adaptação. Não foi feita comparação pixel a pixel do protótipo no navegador.
- Reutilização candidata: tokens CSS/fontes, `cn`, formatação, lógica de tema,
  acessibilidade de busca/select/tabela e estados básicos. Revisar cada peça com
  o contexto Figma do lote; fixtures e `?estado=` ficam fora do runtime real.
- Marca em `design/marca/` é candidata somente após conferir ativo e geometria
  com o Figma; URLs temporárias não entram no produto.
- Componentes derivados da Arc gratuita exigem preservar o aviso MIT de
  `LICENSES-TERCEIROS.md`; ARK CN serviu apenas como referência e não autoriza
  copiar código sem licença. Uma base de primitivas Radix. DevLibs não substitui
  o design nem autoriza novas bibliotecas fora das fontes permitidas em PRODUCT.

Estados reencontrados: análises vazias/carregando/erro, sem instalação, sessão
expirada, acesso negado, autoaprovação, detalhe por infraestrutura/aprovado;
visão geral vazia/erro/carregando; filtros/página 2; modais de ação e retorno,
verificação válida/inválida, cadeia/exportação, configuração/histórico/caminhos,
sugestão, Slack e SLA. Os IDs de todas as variantes estão no inventário.
Só há frames mobile específicos para parte dos fluxos; adaptar os demais pelos
componentes e padrões confirmados, registrando a ausência e validando o resultado.

## Trello e automação

[Snapshot dos critérios #63–#78](dashboard-cards-f0.md), leitura de 16/11/2026:
#63 Pronto, #64 Em andamento, #65–#78 A Fazer. Nenhum card modificado.
O texto antigo azul/violeta de #63 diverge da orientação visual atual da usuária
e do Figma; usar os tokens atuais. A fundação #64 ainda não existe em `web/`.
Os critérios de cada lote permanecem na matriz da seção 2.2 do plano.

O commit histórico `9c0fd55676895ea7f2506a4b1ce788dd761d7d5c` existe localmente.
A `dev` auditada avançou: a documentação da Central foi alterada em `e5c8fa9`,
com consultas e comparação semanal. Reusar fronteiras de serviço e eventos sem
incorporar a App Home. O plano/checklist de automação mantém seu escopo próprio;
flags, timer e persistência n8n não são APIs autenticadas de usuário.

## Ambiente, validação e próximo lote

| Item | Situação |
| --- | --- |
| Origem local proposta | `http://localhost:8080`, conforme `.env.example`; smoke `/health` retornou conexão recusada, sem servidor disponível |
| Homologação | URL e instalação/repositório de teste aguardam confirmação da usuária |
| Recibo real | UUID acessível ainda não fornecido/validado; bloqueia aceite integrado F2 |
| Credenciais | Somente no processo Go via ambiente local ignorado ou segredos da hospedagem; nenhum valor documentado; nada em `VITE_*` |
| Node | Local `v24.22.2`; Vite 8.3.8/plugin React 6.3.0 no lock exigem `^20.19.0 || >=22.12.0`; Playwright exige >=20. Adotar Node 24 em F1/CI do front |
| CI atual | Go/n8n/Gitleaks existentes; Node 22 para workflows. Ainda sem pipeline do front; preservar checks ao adicionar F1E |
| API/recibos | `GOCACHE=/tmp/aprova-f0-go-cache GOPATH=/tmp/aprova-go go test ./internal/api ./internal/receipt` passou (cache). Testes locais de handlers/domínio; não é integração com API implantada |
| Evidência visual | Capturas Figma preservadas; ainda sem aplicação web para comparação local |

Primeiro lote implementável: **F1A**, card #64 — `web/`, scripts/lockfile,
tokens/fontes/licenças, casca desktop/mobile, tema e 404. Consultar os nós da
casca e `86:5942`/`86:6451` com captura antes de implementar. Não exige endpoint
novo. Depois F1B–F1E, antes de F2. Esta execução permanece no escopo F0.

Checklist F0:

- [x] Branch/SHA/checkout auditados; exclusão da App Home preservada.
- [x] Ausência do AGENTS local registrada, com fontes e conflitos explícitos.
- [x] Matriz de contratos revalidada; propostas e limites separados.
- [x] Trello, tokens, componentes/licenças e inventário das três páginas consultados.
- [x] MCP de Figma confirmado; primeiro lote implementável escolhido.
- [ ] Ambiente de homologação e recibo real confirmados (item 5 da F0).

Estado: levantamento técnico concluído; **F0 em andamento** pela pendência de
ambiente. Essa pendência não bloqueia F1A, mas impede afirmar integração real ou
fechamento integral da F0. AGENTS adicional pode ser fornecido depois sem bloquear
a parte independente, conforme o plano.
