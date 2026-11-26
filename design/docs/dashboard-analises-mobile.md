# Análises: apresentação mobile e navegação

Ciclo de front na branch `feat/web-dashboard`, base `4436f88`, em 18/11/2026.
Escopo de F4A relacionado aos cards #66/#68. Sem alterações em Go, API, SQL,
migrations, workflows, dependências ou contratos; nenhum card movimentado.

## Entrega

A lista autenticada de análises usa cards abaixo de 768 px, conforme o Figma,
com repositório/PR, estado e link “Abrir análise”. O nome acessível de cada link
identifica seu PR. O risco, instante UTC e SHA da mesma resposta permanecem
disponíveis para leitores de tela. Desktop mantém a tabela e a ordenação existentes.
Somente uma apresentação participa da árvore de acessibilidade em cada largura.

Os cards usam a mesma página devolvida por `GET /api/analises`, sem uma segunda
consulta nem recálculo de autorização. Abrir o detalhe usa o UUID, a instalação
selecionada e os parâmetros de retorno existentes. Paginação, total e snapshot
continuam vindo do servidor. Reload e volta do detalhe preservam o recorte.
Vazio, erro e falta de acesso não viram cards fictícios.

No mobile, a expansão “Filtros e instalação” fica depois dos resultados e da
paginação; permite usar os filtros e trocar a instalação sem ocupar o início
da lista do Figma. A expansão fechada conserva a busca escrita. No desktop,
os mesmos controles ficam abertos antes da tabela. A ordem no HTML acompanha
a ordem visual; redimensionar preserva os resultados sem repetir sua consulta.

O Figma mobile não desenha os filtros, a paginação ou o aviso de snapshot. Esses
controles existentes foram preservados abaixo dos cards para manter o uso da
coleção real. A página pode rolar até eles, em vez de cortar os resultados na
altura do frame. As diferenças desktop preexistentes, como o seletor de
instalação e as ações de atualizar/limpar, não foram redesenhadas neste ciclo.

## Fontes e evidências

Contextos completos e screenshots consultados com a skill `figma-design-to-code`:

- Mobile escuro/claro: `88:653`, `88:1546`, 390×844.
- Desktop escuro/claro: `9:385`, `15:2152`, 1440×900.
- Estados desktop consultados como contexto: filtros `14:909`, página 2 `14:1060`.

Arquivo Figma `VtHNXer6wzJj37QdpN3Lb9`. Reutilizados tokens, fontes,
componentes de risco/estado, botões, seletor e ativos locais da fundação.
Nenhum ativo novo, URL temporária ou imagem de tela entrou na interface.

Capturas ignoradas em `.capturas-mcp/analises-mobile/`:
`figma-<nó>.png` e `local-<largura>-<tema>.png`. Comparação visual nos dois temas
em 390, 360 e 320 px. Cards começam em x=16/y=204, têm altura de 120 px,
intervalo de 16 px e botão de 180×32 px no cenário de referência. Nomes longos
podem aumentar a altura sem sair da largura disponível.
Regressão desktop/detalhe em `.capturas-mcp/f4a/local-*`.

## Validação e limites

Arquivos: `web/src/paginas/Analises.tsx`, `web/src/estilos/analises.css`,
`web/e2e/analises.spec.ts`, índice e plano do dashboard, além deste registro.
Biome e TypeScript/build aprovados. Mantido o aviso existente de bundle acima
de 500 kB, sem alterar regras ou limites. Os 21 cenários de análises passaram,
incluindo medidas, temas, teclado, filtros, paginação, retorno/reload, resize,
vazio, 403 e 503. Regressão completa: 121 testes aprovados.

As referências do Figma são fixtures exclusivamente dos testes HTTP do navegador.
Não foram adicionados mocks à lista real nem modo de demonstração automático.
O servidor isolado continua cobrindo login/leituras com handlers reais e provedor
controlado, sem representar homologação com uma conta GitHub real.

Continua pendente executar a API local atualizada e configurar OAuth para validar
leituras com a conta real. As funcionalidades sem contrato continuam nos mocks
separados dos ciclos anteriores. O guia de ambiente
`dashboard-api-desenvolvimento.md` permanece local e ignorado pelo Git.
