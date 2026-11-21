# Login conforme o Figma

Correção de front em `feat/web-dashboard`, base `a185c1e`, em 18/11/2026.
Prioridade solicitada pela usuária antes de continuar as demais implementações.
Nenhuma alteração em backend, contratos HTTP, sessão ou credenciais.

## Alterações

Referências atuais consultadas por contexto completo e captura: `9:2` (escuro)
e `15:1769` (claro), arquivo `VtHNXer6wzJj37QdpN3Lb9`.
O cartão fica centralizado em 1440×900, com 480×369, padding de 32, raio 6
e os espaçamentos, textos e cores do Figma. A marca mantém o SVG original de
116×28 dentro do espaço de 145×35 do protótipo; o alinhamento específico de cada
tema também foi reproduzido. Os arquivos da marca foram conferidos por hash
como idênticos aos fornecidos pelo MCP, sem substituição dos ativos existentes.

O botão usa as duas variantes exatas de contraste do ícone GitHub, exportadas
para `*-github-login.svg`, com 16×16. A linha “Protótipo · dados simulados”
foi incluída como no desenho; ela não altera a autenticação real. Os controles
extras de tema e recibo abaixo do cartão foram retirados da tela de login.
O tema salvo continua respeitado; o controle de tema permanece dentro do painel.
As rotas públicas de recibos continuam disponíveis.

A quebra de texto automática foi ajustada somente dentro do cartão: o estilo
global `pretty` deslocava uma palavra para a segunda linha e diferia do Figma.
A adaptação mobile conserva o cartão e seus espaçamentos em 390, 360 e 320 px,
permitindo quebra natural do título sem overflow ou redimensionamento dos SVGs.

## Autenticação e testes

As consultas de `/api/sessao`, o destino `/auth/github`, o retorno ao painel
para uma sessão existente e os estados de carregamento, falha/retry e login
indisponível foram preservados. Nenhum mock de sessão foi introduzido no produto.
Os testes controlam as respostas da sessão para comparar os estados visuais;
o teste existente com handlers Go isolados continua cobrindo login, CSRF,
relatório e logout, sem representar um login OAuth real no GitHub.

`login.spec.ts` cobre oito combinações de viewport/tema, dimensões/centralização,
texto e elementos do protótipo, assets com dimensões originais, teclado, erro
recuperável, indisponibilidade e sessão autenticada. A primeira rodada de login
e sessão passou com 19 testes. Biome e TypeScript/build passaram; permanece o
aviso de bundle acima de 500 kB. A regressão completa passou com 108 testes
após o ajuste final das quebras de texto.

Capturas Figma e locais em `.capturas-mcp/login/`, ignoradas pelo Git:
`figma-9-2.png`, `figma-15-1769.png`, `local-<largura>-<tema>.png`.
Novos ativos têm a origem registrada em `web/LICENSES-TERCEIROS.md`.

## Ambiente local

`http://127.0.0.1:5173/entrar` respondeu HTTP 200. A consulta de sessão encaminhada
pelo Vite para a API local retornou HTTP 404, com `404 page not found`.
Assim, esse ambiente apresenta o estado de erro da sessão, e não o estado normal
com autenticação disponível. A falha não foi mascarada nem substituída por mocks.
Corrigir a disponibilidade do backend é uma tarefa separada; o backend não foi
alterado ou reiniciado como parte desta correção visual.
