# Ações humanas simuladas — ciclo de front

Branch `feat/web-dashboard`, base `3e88548`, execução em 16/11/2026.
Escopo visual do [card #74](https://trello.com/c/qzCbNMo5) e F5A/B do plano,
sob a autorização atual de front com mocks. Não conclui a integração real do card.
Nenhuma alteração em Go, API, SQL, migrations, regras de negócio ou workflows.

## Comportamento

Os botões existentes no detalhe abrem os modais Aprovar, Pedir alteração e
Reexecutar análise. Todos identificam a demonstração antes da confirmação.
Aprovar/pedir alteração exigem justificativa de até 4000 caracteres, conforme
o limite já existente em `internal/acaoremota/contrato.go`. O texto permanece
no formulário ao cancelar ou falhar. Há estado de envio, trava contra clique
simultâneo, identificador estável e consulta de uma solicitação já registrada.

A demonstração impede autoaprovação e ações em análise substituída. Reexecução
só fica disponível na simulação de uma análise falha cuja última tentativa seja
de infraestrutura; falha de teste/compilação não habilita o controle. Isso é uma
regra de apresentação da demonstração: autorização e elegibilidade reais devem
vir do núcleo, com revalidação no momento da ação. O front não afirma permissão
GitHub nem inventa vínculo Slack. Erros de leitura da API mantêm o tratamento
existente e nunca habilitam dados simulados como fallback.

Fixture de respostas: `web/src/mocks/acoes.ts`. Adapter tipado e estado local:
`web/src/servicos/acoes-simuladas.ts`. Componente: `AcoesSimuladas.tsx`, integrado
em `DetalheAnalise.tsx`; estilos em `estilos/acoes.css`.

Uma solicitação é correlacionada a usuário, instalação, repositório, PR, análise
e SHA. Repetir o mesmo ID/conteúdo devolve o resultado anterior; outra submissão
no mesmo contexto gera conflito. A UI oferece consultar o resultado já recebido,
sem apresentar a tentativa que falhou como aceita. Outro SHA requer nova
confirmação; trocar análise/commit/estado descarta o formulário anterior.
O adapter mantém somente memória e o reload restaura a demonstração.

O resultado explica que nenhum review, merge ou tentativa foi executado.
Risco, estado, histórico de decisões e recibo reais não são alterados. A tela
continua consumindo o detalhe real da API. Não há POST, HMAC, chamadas diretas
ao GitHub/n8n, persistência ou polling de uma execução fictícia.

## Figma e acessibilidade

Contextos completos consultados e reutilizados: `10:1235`, `10:1251`, `10:1263`;
claros `15:4142`, `15:4158`, `15:4170`; mobile `16:225`, `16:238`;
resultados `14:2325`, `14:2331`, `14:2337`.

Modais Radix, tokens, fontes e ícones locais da fundação reaproveitados. Capturas
do Figma e da aplicação em `.capturas-mcp/acoes/`, ignoradas pelo Git:
`figma-<nó>.png`, `local-aprovar-<largura>-<tema>.png`,
`local-alteracao-<largura>-<tema>.png`, `local-recebida-<largura>-<tema>.png`.
O aviso de demonstração acrescenta texto ao protótipo para distinguir a simulação
da operação real. Rótulos, erro anunciado, foco visível, Escape e retorno ao botão
que abriu o modal foram verificados. O conteúdo cabe em 1440, 390 e 360 px;
modais preservam rolagem quando necessário.

## Validação e pendências

Biome e build TypeScript/Vite aprovados; permanece o aviso de bundle acima de
500 kB. Testes em `web/e2e/acoes.spec.ts`: temas/viewport, rascunho, justificativa
vazia/excessiva, cancelamento/foco, autoaprovação, análise substituída, elegibilidade
de reexecução, idempotência, clique duplo, isolamento de contexto, outro SHA,
conflito e ausência de requisições de escrita. Regressão completa: 80 testes
aprovados. Após ampliar as capturas de reexecução e resultados, a suíte específica
de ações passou novamente com 14 testes.

Integração real depende do contrato de ações web, autorização por repositório,
reserva/idempotência no servidor e reenfileiramento. Não houve validação de review
ou reexecução reais, mudança de flags de automação, publicação nem aceite do card.
