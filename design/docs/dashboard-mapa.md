# Mapa diário → seleção → detalhe de demonstração

Ciclo de front em `feat/web-dashboard`, base `9d4b27d`, em 18/11/2026.
Escopo visual de F4B, relacionado aos cards #66/#67. Nenhuma alteração em API,
Go, SQL, migrations ou workflows; nenhuma movimentação de cards ou publicação.

## Entrega

Cada célula da visão geral abre um seletor de análises do mesmo dia e nível.
O modal reproduz os seletores desktop/mobile do Figma, com data/hora em UTC,
links de largura completa, fechamento pelo rodapé/X/Escape e retorno de foco
à célula. O rodapé identifica os dados simulados. No mobile, a lista omite
os horários, conforme a referência visual; a descrição permanece acessível.

O recorte fica em `mapa_dia` e `mapa_nivel` na URL. Reload reabre a seleção;
retornar do detalhe preserva o recorte e os filtros independentes da fila.
Células vazias exibem estado vazio, sem links; células com uma referência
mostram uma opção. Parâmetros inválidos mostram uma orientação e podem ser
limpos sem remover os filtros da fila. A volta continua restrita à visão geral,
sem aceitar destinos arbitrários.

`MapaDeAnalises.tsx` reutiliza o mapa existente e as primitivas Radix. A consulta
ocorre só para a célula aberta, com cancelamento e tratamento de erro. A falha
na carga do módulo de fixtures permite retry explícito por reload, preservando
a URL; não vira lista vazia nem ativa consulta à API real.

## Dados e integração

Fixtures em `web/src/mocks/mapa.ts`; contrato e adapter substituível em
`web/src/servicos/mapa.ts`. As referências de 8/11 e 11/11 reproduzem os PRs e
horários dos seletores do Figma. As demais células têm cenários públicos fictícios
para exercitar riscos alto/médio/baixo e quantidades zero/um/várias. Em todo
recorte, a contagem corresponde à lista e ao dia/nível do detalhe selecionado.
Não representam coleção real nem reconciliação das métricas agregadas do núcleo.

IDs de demonstração usam `mapa-<dia>-<nível>-<índice>`. A rota SPA existente
`/analises/demonstracao/:id` resolve essas referências por import dinâmico;
IDs desconhecidos não são enviados à API. Nova referência tem análise e SHA
próprios, sem reaproveitar uma decisão de outro contexto. O detalhe e os modais
simulados do ciclo anterior são reutilizados. Aprovar continua sendo uma
solicitação local: nenhum review, merge, Check Run, recibo ou fila real é alterado.

Os serviços reais de sessão, análises e recibos mantêm seus contratos. Não há
POST, credencial de serviço, HMAC, alteração de política, persistência ou polling
novo. Dependências e lockfile foram mantidos. Capturas e orientações locais
continuam ignoradas pelo Git.

## Figma e evidências

Contextos completos e screenshots consultados antes da implementação:

- Desktop escuro: `82:5826`, `82:5846`.
- Desktop claro: `82:6286`, `82:6306`.
- Mobile escuro: `88:470`, `88:489`.
- Mobile claro: `88:1528`, `88:1536`.

O arquivo é `VtHNXer6wzJj37QdpN3Lb9`, o mesmo do link de fundamentos `0:1`
reafirmado pela usuária. A conexão Figma foi reautenticada nesta retomada.
Tokens, tipografia e ícone de fechar existentes foram reutilizados. Não há ativo
novo nem URL temporária no código. A indicação visível de simulação e o foco
acessível distinguem as capturas da aplicação do protótipo estático.

Referências e comparações em `.capturas-mcp/mapa/`, ignoradas pelo Git:
`figma-<nó>.png`, `local-<dia>-<largura>-<tema>.png`. Comparação em 1440×900,
390×844 e 360×844, nos dois temas. A porta de desenvolvimento 5173 foi mantida.

## Validação e pendências

Biome e TypeScript/build aprovados; mantém-se o aviso conhecido de bundle acima
de 500 kB, sem desabilitar regras ou aumentar limites. A rodada específica de
mapa/visão geral passou com 16 testes. `mapa.spec.ts` cobre teclado/foco, URL,
retorno/reload, vazio/uma/várias referências, parâmetros inválidos, correspondência
de todos os recortes/detalhes, cancelamento, erro/retry e ausência de escritas
ou consultas reais de análises/recibos. A regressão completa passou com
99 testes, incluindo os contratos reais já existentes.

A agregação e leitura reais por célula continuam dependentes dos contratos de
métricas do núcleo, com sessão, escopo autorizado e fuso definidos no servidor.
A entrega conclui o fluxo de demonstração do mapa; não conclui a integração
real, a homologação completa ou os cards correspondentes.
