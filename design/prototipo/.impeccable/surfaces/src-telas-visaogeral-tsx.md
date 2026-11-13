---

version: 1
slug: "src-telas-visaogeral-tsx"
primary_target: "src/telas/VisaoGeral.tsx"
related_targets: []
---

# Visão geral (/)

Modo: Operate. Quem chega: tech lead ou revisor, no começo do dia ou entre tarefas, querendo saber se algo espera por ele.

- **Tarefa:** ver o que está aguardando aprovação, há quanto tempo e por quê, e abrir a análise.
- **Conteúdo real:** fila de aguardando_aprovacao com tempo de espera; análises dos últimos 30 dias por nível; taxa de bloqueio; falhas de sandbox por tipo (teste, compilação, infraestrutura).
- **Estados:** com dados, vazio, carregando, erro (`?estado=`); fila vazia com o resto preenchido.
- **Limites:** todo número com unidade, período e denominador; cores de risco só em nível de risco; nenhuma coluna deixa buraco.
- **Estrutura escolhida depois de uma rodada de três conceitos:** o Pedro escolheu a fila do conceito A com a faixa de métricas do conceito B em cima. Os conceitos B e C e a versão antiga foram removidos.

## Direction contract

THESIS: a fila continua sendo o corpo da página, na largura toda e com linhas de 36px. A faixa de quatro números fica acima dela por decisão de quem pediu o protótipo, e cada número vem com barra de proporção e denominador escrito.

OWN-WORLD: livro de registro. Tinta e papel, três tons de fundo, bordas de 1px, Inter para falar e Plex Mono para identificar. Linguagem base adaptada do Linear: superfícies quase pretas, Inter em pesos leves, limão só no foco e no item ativo. Cor de risco só no selo e no mapa de calor. Tema claro com os cinzas do MONOLITH.

STORY: a pessoa lê quantos PRs esperam por ela e há quanto tempo, vê o motivo e a evidência de cada um na própria linha, filtra por repositório ou busca um PR, e abre a análise mais antiga.

FIRST VIEWPORT: título e período no topo. Faixa de quatro métricas unidas por emenda de 1px (aguardando aprovação, taxa de bloqueio, falhas no sandbox, análises recebidas). Fila na largura toda com busca, filtro de repositório e ordenação por espera; o nível aparece uma vez, no título. Embaixo, três painéis da mesma altura: bloqueadas por desfecho, falhas por tipo e o mapa de calor de análises por dia e nível.

FORM: faixa de métricas, fila de registro e três painéis; escolhida entre três conceitos (A, B e C). seed: none (unavailable)

FINISH: unreviewed and undocumented is unfinished; this build ends with the finish review, the verdict, DESIGN.md, and every shipping raster carrying its provenance
