# F2 — Recibos por UUID

Execução em 16/11/2026 na branch `feat/web-dashboard`, base `dbafee3`.
AGENTS.md local não encontrado. Escopo: F2A–F2D, parte do card #70;
nenhuma alteração no quadro Trello.

## Implementação

- `/recibos`: campo acessível para localizar UUID e chaves públicas. A coleção
  continua pendente de F4C, conforme o contrato atual.
- `/recibos/:id`: conteúdo, evidências da classificação, elo anterior,
  verificação explícita nesta sessão e exportação do documento retornado.
- Chave pública correspondente ao `key_id`, download do base64 sem alteração
  e instrução para verificação independente pelo executável do projeto.
- Cinco conferências retornadas pelo núcleo, incluindo `falhou` e
  `nao_verificada`. HTTP 200 com `valido:false` continua sendo uma reprovação.
- Data da consulta em UTC, estados de carregamento/falha, repetição manual,
  cancelamento e limpeza de estado ao trocar UUID. Identidade divergente ou
  resultado válido com hash/conferências inconsistentes é recusado.
- Payload ausente, chave desconhecida e elo ausente são representados
  explicitamente; o elo conhecido pode ser consultado. A verificação da cadeia
  completa permanece desabilitada com explicação.

Contratos existentes, sem novas APIs: `GET /receipts/keys`,
`GET /receipts/{id}` e `GET /receipts/{id}/verify`, auditados em
`internal/api/recibos.go`, `internal/receipt/exportacao.go` e
`internal/receipt/conteudo.go`. O envelope original é exportado como JSON;
`payload_base64` e `assinatura_base64` não são decodificados nem reserializados.
O campo `payload` é apresentado como conteúdo registrado, sem substituir a
verificação do núcleo. A interface não calcula nem fabrica um resultado válido.

## Referências visuais e adaptações

Figma consultado via MCP: `9:1024`, `15:2791`, `16:159`, `10:1272`, `10:1309`.
Componentes e tokens da F1 reutilizados. Ajuda e acabamento serrilhado foram
baixados do Figma para os dois temas; navegação e seta usam os ativos existentes.
Captura de referência: `.capturas-mcp/f2/figma-recibo-1440-escuro.png`.
Capturas locais: `.capturas-mcp/f2/local-{1440,390,360}-{escuro,claro}.png`.
Todas permanecem ignoradas pelo Git.

Adaptações funcionais: busca por UUID em vez de coleção ainda inexistente;
conferências no próprio detalhe, acessíveis também no mobile, em vez de exigir
modal; seção complementar com chave e instruções de verificação independente.
Não se inferem calibração ou acesso à instalação a partir do recibo público.
Datas mostram UTC e campos longos quebram linha em telas estreitas.

## Validação e limites

Os cenários Playwright em `web/e2e/recibos.spec.ts` separam interceptações de
contrato e integração HTTP local. O servidor exclusivo de testes cria um recibo
sintético com chave Ed25519 efêmera em memória, usa os handlers e o verificador
reais, e o teste confere hash e assinatura do download com `node:crypto`,
independentemente do verificador Go. Não há fixture no runtime de produção.
O guia completo continua em [verificação de recibo](../../docs/verificacao-recibo.md).

Homologação: API em `127.0.0.1:8080` sem conexão nesta execução; URL de teste e
UUID real solicitados à usuária, ainda não fornecidos. O recibo sintético não
substitui esse aceite. F2 permanece com aceite integrado pendente até abrir,
verificar, baixar e conferir um recibo real com chave pública conhecida.

Resultados finais:

- `npm --prefix web ci --include=optional`: instalação aprovada, sem mudança no lockfile.
- `npm --prefix web run verificar`: 29 arquivos, sem apontamentos.
- `npm --prefix web run build`: TypeScript e Vite aprovados.
- `npm --prefix web run teste`: 12 cenários de regressão F1/servidor aprovados;
  dois cenários novos detectaram o redimensionamento do ativo no mobile.
- Após a correção, `npm --prefix web run teste -- recibos.spec.ts`:
  **13/13 aprovados**, incluindo os três viewports nos dois temas.
- `go test ./internal/api ./internal/receipt ./web/...` e `go vet ./web/...`:
  aprovados. Não houve alteração em persistência nem necessidade de migrations.
- Capturas locais inspecionadas: hierarquia, painéis, tema, ativos e quebra
  de identificadores conferidos; sem rolagem horizontal. Diferenças funcionais
  em relação às fixtures do Figma estão descritas acima.

As primeiras tentativas de navegador foram bloqueadas pela ausência de bibliotecas
Chromium no ambiente. Dependências instaladas; execução posterior aprovada.
A suíte CI remota não foi executada nesta tarefa.
