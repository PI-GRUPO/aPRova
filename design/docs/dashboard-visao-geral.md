# Visão geral — front e mocks separados

Lote em 16/11/2026 na branch `feat/web-dashboard`, base `12c8ee5`.
Escopo definido pela usuária: construir o front conforme o Figma, consumir APIs
existentes e manter dados/contratos ausentes em mocks. Nenhuma alteração em
API, handlers, Go, queries, migrations ou workflows foi feita neste lote.

## Fontes visuais

Figma `VtHNXer6wzJj37QdpN3Lb9`: contextos completos e screenshots dos nós
`9:22` (desktop escuro), `15:1789` (desktop claro) e `16:2` (mobile).
Página desktop `5:2` inspecionada para localizar os frames atuais. Componentes,
fontes, tokens, ícones e primitives existentes de `web/` foram reutilizados.
O ícone external-link foi exportado em ambos os temas, com dimensões originais.

Capturas de referência e da aplicação em `.capturas-mcp/dashboard/`, ignoradas
pelo Git. Exemplos: `figma-overview-escuro.png`, `figma-overview-claro.png`,
`figma-overview-mobile.png`, `local-overview-1440-escuro.png`,
`local-overview-390-claro.png` e `local-overview-360-escuro.png`.

## Entrega

- `/`: composição da visão geral do Figma, com faixa de métricas, fila de
  aprovação, distribuições, mapa de calor e referência de último recibo.
- Filtros da fila por texto e repositório refletidos na URL. Busca por PR,
  branch, motivo ou evidência. Modal de prévia explicita a origem simulada e
  não oferece mutações reais; IDs fictícios não são enviados à API de detalhe.
- Mobile com métricas em duas colunas, contextos compactos, fila em cards e
  mapa com rolagem própria. Temas claro/escuro e primitives Radix preservados.
- `/relatorio-semanal`: mantém a consulta autenticada existente de sete dias,
  com autorização por instalação, erros, retry, 401 e logout. A tela anterior
  foi extraída para `RelatorioSemanal.tsx`, sem alterar o contrato da API.
- Fixtures de dados em `web/src/mocks/visao-geral.ts`, separadas do componente.
  Contrato e ponto de troca em `web/src/servicos/visao-geral.ts`. O mock é
  explícito, não responde por HTTP nem substitui erros de consultas reais.
  Os dados simulados são identificados no rodapé da tela e na navegação desktop.

A API de métricas de 30 dias/fila agregada não existe nesta base. Não derivamos
esses números do relatório semanal nem criamos endpoint no backend. Os demais
serviços existentes de sessão, análises e recibos mantêm consultas reais.

## Validações

- Instalação por `npm ci --prefix web`.
- `npm --prefix web run verificar`: aprovado.
- `npm --prefix web run build`: aprovado.
- `npm --prefix web run teste:visual`: seis cenários aprovados, cobrindo
  1440×900, 390×844 e 360×844 em claro/escuro, filtro, reload, modais,
  navegação por teclado e ausência de requisições de escrita.
- Regressão `npm --prefix web run teste`: 51 testes aprovados, incluindo
  sessão/relatório, lista/detalhe de análises, recibos e servidores de teste.
  Os handlers reais nos testes usam dependências isoladas; isso não comprova
  login OAuth ou consulta aos dados reais da instalação da usuária.
- O servidor do Playwright passou a usar a porta exclusiva 5183, sem reutilizar
  o servidor de desenvolvimento. A primeira rodada reutilizou servidor sem
  harness de teste e falhou; após isolamento e atualização das URLs do relatório,
  os 51 cenários passaram.

## Limites e próximos lotes

Este lote cobre a visão geral; não conclui todo o dashboard. A série/contagem
por dia ainda é uma fixture visual, e seu modal explica o recorte simulado;
não apresenta UUIDs inventados como análises reais. O ciclo seguinte conectou a [fila ao detalhe e à decisão de demonstração](dashboard-fila-detalhe.md).
O [seletor do mapa](dashboard-mapa.md) agora abre listas de referências locais,
identificadas como simuladas. A coleção real por célula permanece pendente.
A última referência de recibo simulada leva à área de recibos, sem forjar uma
verificação criptográfica. Nenhuma aprovação, review ou configuração é enviada.

Continuar pelas telas e modais do inventário ainda não implementados, usando
contexto Figma antes de editar cada composição. Substituir mocks por adapters
quando os respectivos contratos forem disponibilizados, sem alterar a API
como parte deste trabalho de front. A fidelidade e a integração de cada tela
seguem critérios próprios; os 51 testes não representam validação de telas
que ainda estão como indisponíveis.
