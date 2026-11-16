# aPRova no Slack — proposta visual

16/11/2026 · Versão visual 3 — identidade corrigida

## Comece aqui

Abra `previa.html` no navegador. O arquivo funciona sozinho, sem instalação,
sem servidor e sem acesso à internet. Use Home, Mensagens e Sobre. Os controles
superiores alternam estados e temas. O botão Baixar JSON exporta a Home ativa;
quando um modal estiver aberto, exporta esse modal.

Clique em Conversar com o bot e experimente as perguntas sugeridas. Os exemplos
são respostas fixas para demonstrar a experiência. Texto livre também pode ser
digitado, mas não é processado por IA. Filtros e formulários são simulações
locais e não alteram o GitHub. Os formulários exigem justificativa e mostram
aviso de demonstração ao confirmar.

`previa-visual.png` apresenta uma composição aproximada da Home, da conversa e
do painel de detalhes. O painel de detalhes é um modal no produto, separado da
aba Mensagens; a composição reúne essas superfícies para facilitar a revisão.

## Visualizar no Slack Block Kit Builder

1. Acesse https://app.slack.com/block-kit-builder.
2. Cole o objeto completo de `01-home-vinculada.json` no editor de payload.
3. Selecione a superfície Home, caso o Builder solicite.
4. Abra os demais JSONs para conferir estados, modais e mensagem de boas-vindas.

São objetos de view (Home/modal) ou corpo de mensagem, sem wrapper de chamada
HTTP. Para publicar, o backend precisará envolver a Home em views.publish com
user_id; para abrir modal, usar views.open com trigger_id. O Builder permite
revisar os blocos; não implementa o comportamento dos botões.

## Arquivos

| Arquivo | Conteúdo |
| --- | --- |
| 01-home-vinculada.json | Conversa em destaque, resumo, filtros, pendências e últimas análises |
| 02-home-sem-pendencias.json | Home sem decisões aguardando atendimento |
| 03-home-sem-analises.json | Primeiro acesso, ainda sem análises |
| 04-home-sem-vinculo.json | Orientação e integração opcional com GitHub |
| 05-home-falha-consulta.json | Consulta indisponível, sem inventar métricas |
| 06-modal-analise.json | Contexto, motivo do risco, testes, OSV e ações |
| 07-modal-aprovar.json | Aprovação com justificativa |
| 08-modal-revisao.json | Solicitação de revisão com justificativa |
| 09-modal-recibo.json | Apresentação da verificação do recibo |
| 10-modal-perguntas.json | Sugestões de consultas em linguagem natural |
| 11-modal-configuracoes.json | Resumo dos vínculos e acesso à configuração |
| 12-mensagem-boas-vindas.json | Conteúdo inicial da aba Mensagens |

## Escolhas da proposta

- A conversa ocupa o primeiro bloco útil da Home: é parte central do produto.
- Aprovação continua explícita, em formulário, e depende de autorização.
- Risco alto aparece com motivo e evidências. Testes aprovados e consulta OSV
  sem avisos não diminuem automaticamente o risco.
- Os números identificam período ou estado atual; não se misturam os recortes.
- A versão sem pendências omite métricas agregadas para não confundir ausência
  de aprovações pendentes com ausência de análises.
- Repositórios, nomes de PRs, métricas, tempos e recibos são fictícios.
- Os temas da prévia são aproximados. No Slack, tipografia, cores, dimensões,
  aparência dos botões e temas pertencem ao cliente Slack. O ícone da prévia usa o arquivo oficial
  design/marca/svg/icone-app-fundo-escuro.svg da branch dev: fundo #111214,
  símbolo #ECE9E2 e contorno #2A2C31. O ícone no Slack é configurado no perfil
  do app. Foram removidos o marcador aP e os destaques amarelos da proposta.

## Preparação para conectar ao backend

O comportamento é proposto. Nenhuma rota, workflow ou configuração foi alterada.

| action_id | Comportamento esperado |
| --- | --- |
| aprova_home_conversar | Abrir a conversa com o bot no workspace correto |
| aprova_home_perguntas | Abrir o modal de sugestões |
| aprova_home_repositorio | Atualizar a lista com filtro autorizado |
| aprova_home_atualizar | Consultar dados e republicar a Home |
| aprova_home_detalhes | Consultar a análise autorizada e abrir o modal |
| aprova_home_aprovar | Abrir formulário; encaminhar decisão ao núcleo após envio |
| aprova_home_revisar | Abrir formulário; encaminhar solicitação ao núcleo após envio |
| aprova_home_recibo | Consultar e verificar recibo antes de apresentar resultado |
| aprova_home_dashboard | Abrir a URL real do dashboard |
| aprova_home_configuracoes | Mostrar a configuração permitida para o usuário |
| aprova_home_vincular | Iniciar fluxo autorizado de vinculação GitHub/Slack |
| aprova_home_ajuda | Mostrar orientação inicial; implementar variante sem vínculo |
| aprova_recibo_documento | Abrir o documento real de recibo |
| aprova_config_gerenciar | Abrir configurações completas com autenticação |

Os valores `demo`, `exemplo-*`, o private_metadata de demonstração e IDs como
A00000000 e T00000000 são marcadores, não configurações reais. Os endereços
aprova.example.com também são marcadores. Trocar todos na integração; nunca
publicar esses objetos sem adaptação. O link de conversa usa app_redirect
com o app ID e workspace ID reais. A aba Mensagens também pode ser aberta por
slack://app?team=TEAM_ID&id=APP_ID&tab=messages no cliente nativo.

Os callbacks dos modais são novos e precisam ser mapeados ao contrato existente.
No exemplo de aprovação, o formulário usa o block_id justificativa e action_id
texto. Não copiar o formato diretamente para /acoes-remotas: o backend precisa
consultar o contexto real (instalação, análise, PR e commit) e validar identidade,
permissão, estado, autoria e repetição. A decisão deve continuar no núcleo.

Quando a lista mudar, gerar block_id adequado para a nova revisão da view.
Não confiar em value/private_metadata como autorização nem aceitar instalação
ou identidade fornecida pelo modelo. O backend deve resolver o vínculo correto
por instalação e workspace; a integração atual de workspace único não implementa
por si só esse isolamento entre equipes.

## Linguagem natural: capacidade e proposta

As consultas encontradas no projeto incluem listagem e detalhe de análises,
relatório semanal, comparação semanal e saúde/prontidão. A conversa em DM ainda
precisa de validação específica no ambiente, conforme a documentação consultada.

A explicação conversacional dos motivos do risco é uma ampliação futura: o
contrato de detalhe do assistente ainda não oferece esses motivos. Por isso,
as perguntas sugeridas priorizam consultas disponíveis. O modal de análise
mostra como os motivos poderão ser apresentados quando o backend os fornecer.

Orientação por conversa sem vínculo também é proposta futura. A Home sem
vínculo não configura o bot nem libera acesso a dados de repositórios.

## Validação desta entrega

- 12 JSONs parseados; conferidos limites de blocos, títulos e textos de botões.
- Script da prévia passou por verificação de sintaxe JavaScript.
- Composição PNG inspecionada visualmente.
- Não houve publicação no Slack nem validação do payload na API real.
- O navegador automatizado local não pôde iniciar neste ambiente. A prévia HTML
  não passou por teste de navegação renderizada; confira-a no seu navegador e
  use o Block Kit Builder para validar a apresentação nativa antes de publicar.

## Referências

Design consultado na branch dev: design/prototipo/DESIGN.md e PRODUCT.md.
Integração consultada: workflows/n8n/central-slack.json,
workflows/n8n/slack-consulta-metricas.json e docs/central-slack-implementacao.md.

- https://docs.slack.dev/block-kit/
- https://docs.slack.dev/reference/views/home-tab-views/
- https://docs.slack.dev/interactivity/deep-linking/
