# Gestão, recibos e vínculo Slack — ciclos de front

Branch `feat/web-dashboard`, base `12c8ee5`, execução em 16/11/2026.
Nenhuma alteração em Go, API, SQL, migrations ou workflows neste trabalho.
Escopos F4C/F6/F7 do plano; não representam aceite real dos contratos do núcleo.

## Gestão

Telas `/repositorios`, `/regras`, `/auditoria`, `/configuracao`, `/sugestoes` e
`/sla`, em claro/escuro e 1440, 390 e 360 px. Filtros ficam na URL; tabelas largas
têm rolagem própria. Em regras, o mobile usa os cards do Figma e oferece as
estatísticas em uma expansão.

O catálogo autenticado usa `GET /api/repositorios?installation_id=...`.
Falhas reais mostram erro e retry, sem substituir dados por mocks. Proteção,
última análise e permissões sem contrato ficam não confirmadas. Sem sessão,
a tela identifica o catálogo demonstrativo.

Fixtures: `web/src/mocks/gestao.ts`; adapter: `web/src/servicos/gestao.ts`.
Configuração tem rascunho, descarte, validação de caminhos, revisão das diferenças,
confirmação de redução de proteção e histórico de versões. Sugestões podem ser
recusadas ou encaminhadas à configuração; só a confirmação registra uma versão
simulada. SLA valida inteiros positivos e escalonamento posterior ao primeiro
aviso, com valores independentes por repositório. Nada habilita avisos reais.
Alterações existem em memória; reload restaura as fixtures.

## Coleção de recibos e conta

A coleção usa `mocks/recibos.ts` e `servicos/colecao-recibos.ts`.
Filtros, prévia, cenário de cadeia quebrada e exportações JSON/CSV funcionam
sobre metadados simulados. Os arquivos identificam a simulação; não geram
assinaturas, hashes ou comprovação criptográfica. Consulta pública por UUID,
chaves e verificação individual preservam os serviços reais.

O vínculo Slack usa `mocks/conta.ts`, `servicos/vinculo-slack.ts` e
`componentes/VinculoSlack.tsx`. Conectar/desvincular são demonstrações locais,
sem OAuth, identidade Slack inventada para a API ou requisição de escrita.
A identidade GitHub exibida continua sendo a da sessão real. O vínculo local
é descartado ao desmontar a área da conta.

## Fontes e evidências

Contextos e capturas Figma consultados antes da construção:

- Repositórios: `9:1185`, `15:2952`.
- Regras: `9:1351`, `15:3118`, mobile `88:741`.
- Auditoria: `9:1565`, `15:3332`.
- Configuração: `10:616`, `15:3523`; modais `10:1361`, `10:1388`, `452:8465`.
- Sugestões: `10:791`, `15:3698`; revisão `10:1397`.
- SLA: `10:1078`, `15:3985`.
- Coleção: `9:829`, `15:2596`; cadeia/exportação `10:1345`, `10:1354`.
- Conta: `10:943`, `15:3850`; vínculo/desvínculo `10:1409`, `10:1426`.

Referências e comparações em `.capturas-mcp/dashboard/`, ignoradas pelo Git:
`figma-<nó>.png`, `figma-regras-mobile.png`, `local-<tela>-<largura>-<tema>.png`.
Componentes, fontes e tokens da fundação reaproveitados; ícones primários de
configuração exportados do Figma. Adaptações mobile sem frame específico seguem
a fundação existente.

## Validação e limites

`gestao.spec.ts`: 15 cenários aprovados, incluindo sete telas em seis combinações
de viewport/tema, conta nos três viewports, formulários, filtros/reload,
catálogo autenticado com erro/retry, ausência de escritas e exportação sem assinatura.
Build TypeScript/Vite e Biome aprovados. Regressão final: 66 testes aprovados,
incluindo os contratos existentes de sessão, análises e recibos. O build mantém
o aviso de bundle acima de 500 kB; nenhuma regra ou limite foi desabilitado.

OAuth real do Slack, configuração persistida, coleção/cadeia real e gestão de SLA
dependem dos contratos do núcleo. As ações humanas estão em rascunho, fora destes
ciclos. Aprovação e reexecução reais permanecem indisponíveis; nenhum resultado
simulado muda análises reais.
