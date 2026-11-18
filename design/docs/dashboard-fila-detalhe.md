# Fila → detalhe → decisão de demonstração

Ciclo na branch `feat/web-dashboard`, base `bf0f65e`, em 16/11/2026.
Escopo de front de F4B/F5 e dos cards #66/#69/#74. Nenhum contrato de backend,
handler, código Go, query, migration ou workflow alterado; cards não movimentados.

## Entrega

Os cinco links da fila, em desktop e mobile, abrem o detalhe completo na rota
SPA `/analises/demonstracao/:id`. A origem simulada é explícita e essa rota
não consulta a API de análises. Os filtros de busca/repositório são preservados
no retorno à visão geral; o destino de volta é fixo e só aceita esses parâmetros.
IDs desconhecidos mostram referência não encontrada. Falha no carregamento do
módulo de fixtures é distinta de referência ausente e permite recarregar a página.
O cancelamento impede respostas obsoletas após troca de referência.

A composição do detalhe foi reutilizada de `DetalheAnalise.tsx`, com parâmetros
opcionais de apresentação da demonstração. O caminho real `/analises/:id` mantém
sessão, autorização, consulta da API, erros e verificação de recibos existentes.
Um 403/404/503 real não cai no fluxo simulado.

Fixtures públicas e identidade de demonstração em `web/src/mocks/analises-demonstracao.ts`;
adapter e modelo em `web/src/servicos/analises-demonstracao.ts`. Os IDs da fila
são a identidade das referências, sem convertê-los em IDs reais. Instalação e
revisor simulados têm escopo próprio; não se infere identidade Slack nem se muda
a sessão GitHub. Dados entram por import dinâmico e são clonados antes do uso.

O primeiro caso reproduz as evidências de falha de teste do Figma, com saída de
sandbox, calibração e linha do tempo de referência. Os demais usam os motivos
e evidências já apresentados na fila; histórico de calibração, tentativas e
recibo ausentes não são preenchidos com resultados supostamente reais.

Os modais do ciclo anterior funcionam também nesse contexto, sem escrita HTTP.
Uma solicitação permanece consultável ao voltar à fila e abrir novamente a
mesma referência na SPA. O estado continua aguardando aprovação, pois a
simulação só recebe o pedido: não executa o núcleo nem inventa sua conclusão.
Reload restaura o ambiente de demonstração.

Links ao PR/Check Run ficam desabilitados nas referências. O painel de recibo
mostra somente metadados e acesso à coleção de demonstração; não chama consulta
pública com um ID fictício nem produz resultado criptográfico. Dados reais de
recibos continuam nos seus fluxos existentes.

## Figma, componentes e evidências

Contextos completos e capturas: `9:608` e `15:2375` (desktop), `16:58` e
`88:1168` (mobile). Tokens, fontes, painel, tabela, risco, estado, ícones e
modais Radix existentes reutilizados. A linha do tempo visual de referência
fica em `LinhaDoTempoDemonstracao.tsx`, sem alterar a semântica dos eventos reais.
Mobile preserva a composição do Figma, incluindo título compacto, contexto do
commit e painéis de evidências; o aviso de dados simulados substitui dados pessoais do
protótipo. Botões externos desabilitados e nota de recibo distinguem as limitações.

Capturas ignoradas em `.capturas-mcp/fila/`: `figma-<nó>.png` e
`local-detalhe-<largura>-<tema>.png`, com comparação em 1440, 390 e 360 px,
nos temas claro/escuro. Nenhum ativo novo ou URL temporária entra na aplicação.

## Validação e limites

Build TypeScript/Vite e Biome aprovados. Testes em `fila-demonstracao.spec.ts`
cobrem fila → detalhe → decisão → retorno, filtros na URL, resultado preservado
na SPA, cinco referências, ID desconhecido, retorno restrito, erro de fixture e
retry, ausência de chamadas à API de análises/recibos e ausência de escritas.
A regressão completa passou com 89 testes, incluindo os caminhos reais.
A suíte específica passou novamente com nove testes após o ajuste visual final no mobile.
Permanece o aviso já conhecido de bundle acima de 500 kB.

O ciclo seguinte implementou a [seleção de referências simuladas por célula](dashboard-mapa.md),
com lista e navegação ao detalhe. A coleção real por dia/nível continua pendente. OAuth e contratos de métricas, ações web, coleção
real e gestão continuam sendo pendências de integração, fora do escopo autorizado
de front. A entrega não é homologação real nem conclusão integral do dashboard.
