# Validação final da interface

Branch `feat/web-dashboard`, base `3f51497`, revisão em 18–19/11/2026.
Escopo: interface e fluxos do dashboard, relacionados aos cards #64–#78.
Esta revisão cobre a interface em ambiente isolado; não conclui a homologação
integrada de F8. Nenhum card foi movimentado e não houve push, merge ou deploy.

## Cobertura

Auditoria transversal de 17 variantes de tela em 1440, 1024, 768, 390 e 320 px,
nos temas escuro e claro: 170 combinações. Capturas próprias em 1440×900,
390×844 e 320×844; as larguras intermediárias também executam as verificações
de interface. Os estados completos continuam cobertos pelas suítes de cada fluxo.

| Tela/variante auditada | Fonte visual e comportamento |
| --- | --- |
| Login | `9:2`, `15:1769`; disponibilidade e autenticação preservadas |
| Visão geral | `9:22`, `15:1789`, `16:2`, `88:826`; mocks explícitos, mapa e fila |
| Análises | `9:385`, `15:2152`, `88:653`, `88:1546`; coleção da API, filtros e cards mobile |
| Detalhe real e demonstração | `9:608`, `15:2375`, `16:58`, `88:1168`; evidências, ações e ausência de dados |
| Coleção de recibos | `9:829`, `15:2596`, `16:119`, `88:1244`; metadados simulados e consulta real por UUID |
| Recibo individual e verificação | `9:1024`, `15:2791`, `16:159`, `88:1299`; cinco conferências, download e chave |
| Repositórios demonstrativos e catálogo real | `9:1185`, `15:2952`; os estados sem contrato continuam não confirmados |
| Regras | `9:1351`, `15:3118`, `88:741`, `88:1632`; cards e expansão de estatísticas |
| Auditoria de ações | `9:1565`, `15:3332`; filtros e coleção simulada |
| Configuração | `10:616`, `15:3523`; rascunho, revisão e versões simuladas |
| Sugestões | `10:791`, `15:3698`; revisão e encaminhamento à configuração |
| Conta | `10:943`, `15:3850`; identidade GitHub e demonstração de vínculo Slack |
| SLA | `10:1078`, `15:3985`; formulário e validação de prazos simulados |
| Relatório semanal | Complemento que consome o contrato existente de sete dias; não equivale à visão de 30 dias |
| Página não encontrada | `86:5942`, `86:6451`; navegação de recuperação e arte responsiva |

As 40 referências atuais foram obtidas no arquivo Figma
`VtHNXer6wzJj37QdpN3Lb9`. As correções de composição também consultaram o
contexto estruturado, seguindo a skill `figma-design-to-code`. Telas sem frame
mobile próprio mantêm a adaptação dos componentes e tokens existentes; não são
apresentadas como reprodução de um frame mobile inexistente.

## Correções realizadas

**Coleção mobile de recibos.** A tabela larga foi substituída por cards com
identificador, repositório/PR e abertura de prévia. Cabeçalho, espaçamento,
seletor e link de retorno seguem a referência mobile. Cards começam em x=16,
y=310, têm 150 px de altura e botão de 210×32 px no cenário de referência.
Período, cadeia e exportação permanecem acessíveis em expansão abaixo da lista.
Filtros continuam na URL e sobrevivem a reload. Desktop mantém a tabela.

Os dados da coleção são metadados simulados: o botão diz “Ver prévia do recibo”,
e a explicação informa ausência de assinatura. Um ID `*-demo` nunca é enviado
ao verificador real. O formulário de UUID e as chaves públicas continuam abaixo
da demonstração, consumindo os serviços existentes.

**Verificação mobile.** Depois de receber uma resposta válida do contrato da API,
o resultado passa à apresentação própria do Figma, com heading, ajuda e tabela
das cinco conferências. O topo da tabela fica em x=16/y=276. Resultado inválido
mantém falha e etapas não verificadas, inclusive quando o HTTP é 200.
O título recebe foco; “Voltar ao recibo” devolve foco ao botão de verificação,
permitindo nova consulta e exportação no detalhe. Reload limpa o resultado
transitório, sem apresentar uma conferência anterior como nova. Não há rota
ou requisição de API adicional. Desktop mantém a apresentação do detalhe.

O retorno aponta ao recibo consultado, pois a entrada por UUID público não exige
sessão ou contexto de análise. O verdict e o horário da consulta são exibidos
além da tabela para deixar claros o resultado e sua validade temporal.

**Regras mobile.** Ajustados os 4 px de diferença do link de retorno e o espaço
do subtítulo oculto. A expansão de estatísticas e o aviso de simulação foram
preservados. O hook do breakpoint foi compartilhado entre Análises e Recibos,
sem duplicar consultas ao redimensionar.

**Ambiente de validação.** Os caches do Vite foram separados por modo:
desenvolvimento (`development`), regressão (`teste`) e teste de proxy
(`teste-proxy`). Os três servidores podem funcionar simultaneamente sem usar
o mesmo diretório de dependências otimizadas. O destino e os contratos da API
não foram alterados.

## Verificações

- Um título principal e uma região `main` por tela, IDs únicos, nomes dos
  controles, rótulos de campos e protocolos de links.
- Ausência de rolagem horizontal no documento e de imagens ausentes/deformadas.
  A arte 404 usa zoom responsivo e SVG com dimensões fracionárias; a auditoria
  considera arredondamento nativo de até 1 px, sem alterar os ativos.
- Modais de caminho sensível, aprovação, pedido de alteração, exportação e cadeia
  em 1440×900 e 320×568, nos dois temas: posição, rolagem interna, Tab,
  Shift+Tab, Escape, foco contido e retorno ao gatilho. Ações não foram confirmadas
  contra serviços externos. Os demais fluxos/modais têm cobertura nas suítes próprias.
- Contraste dos tokens de texto e risco sobre a superfície elevada: menor razão
  medida de 5,15:1 para texto secundário e 4,99:1 para risco, nos dois temas.
  Isso não substitui uma certificação WCAG ou teste com leitor de tela real.
- Fluxos anteriores: sessão expirada, escopo, erros HTTP/rede/timeout, ausência
  de dados, filtro/paginação, decisão simulada, recibo inválido, bytes exportados,
  consulta independente e prevenção de escritas indevidas.

Biome, árvore de dependências instalada e TypeScript/build aprovados. O aviso
preexistente de bundle acima de 500 kB permanece, sem alteração de limites.
Regressão final: **142 testes aprovados** em Chromium, em 2,9 minutos, com
`npm --prefix web run teste -- --workers=4`. As dependências de navegador e
o cache Go usados na execução são locais ao ambiente de teste.
Navegador: Chromium via Playwright; não foi executada uma matriz Safari/Firefox.
Após o isolamento dos caches, os 26 cenários de sessão, proxy e auditoria
transversal passaram em conjunto. As falhas da rodada anterior foram revisadas
antes de repetir a regressão completa, sem remover verificações ou aumentar
os tempos limite dos testes existentes.

## Evidências e limites

Capturas em `.capturas-mcp/validacao-final/`, ignoradas pelo Git:
`figma-<nó>.png`, `local-<tela>-<largura>-<tema>.png`,
`modal-<nome>-<largura>-<tema>.png`, `verificacao-<largura>-<tema>.png`
e folhas de comparação `comparacao-*.jpg`. Os fluxos existentes também atualizam
as capturas em `dashboard/`, `f2/`, `f4a/`, `acoes/`, `mapa/` e `login/`.

As diferenças de conteúdo necessárias foram preservadas: usuário autenticado
real, campos não disponíveis, controles de escopo, avisos de simulação, estados
do verificador e capacidades sem contrato. Não foram introduzidos resultados,
assinaturas, identidades ou autorização para imitar os números do protótipo.

Arquivos de implementação: `web/src/app/responsivo.ts`, páginas de Análises e
Recibos, `ColecaoRecibos.tsx`, estilos de Recibos/Gestão e testes de interface,
gestão e recibos, além de `web/vite.config.ts`. Nenhum código Go, API, endpoint, SQL, migration, workflow,
dependência ou lockfile foi alterado.

Continuam pendentes a execução da API atualizada, a configuração OAuth e a
validação com conta/instalação e dados reais. Métricas de 30 dias, operações
web e gestão sem contrato continuam nas simulações já documentadas. O guia
`dashboard-api-desenvolvimento.md` permanece local e ignorado pelo Git.
O dashboard respondeu HTTP 200 em `http://127.0.0.1:5173/entrar`; a consulta
`/api/sessao` pela mesma origem continuou respondendo 404 na API local.
