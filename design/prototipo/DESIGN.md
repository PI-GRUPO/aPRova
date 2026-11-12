---
name: aPRova
description: Dashboard de auditoria de pull requests gerados por agentes de IA
colors:
  fundo-base: "#111214"
  fundo-superficie: "#17181B"
  fundo-elevado: "#1E2024"
  borda-padrao: "#2A2C31"
  texto-primario: "#ECE9E2"
  texto-secundario: "#9A9EA6"
  acao-link: "#7FA2C8"
  risco-baixo: "#3FB37F"
  risco-medio: "#E0A43B"
  risco-alto: "#E75D55"
  claro-fundo-base: "#F5F5F6"
  claro-fundo-superficie: "#FFFFFF"
  claro-fundo-elevado: "#ECECEE"
  claro-borda-padrao: "#DCDCE0"
  claro-texto-primario: "#111214"
  claro-texto-secundario: "#5F636B"
  claro-acao-link: "#3F6A96"
  claro-risco-baixo: "#1F764E"
  claro-risco-medio: "#915C08"
  claro-risco-alto: "#B42F28"
typography:
  display:
    fontFamily: "IBM Plex Sans, sans-serif"
    fontSize: "32px"
    fontWeight: 600
    lineHeight: "40px"
    fontFeature: "tnum"
  headline:
    fontFamily: "IBM Plex Sans, sans-serif"
    fontSize: "24px"
    fontWeight: 600
    lineHeight: "32px"
  title:
    fontFamily: "IBM Plex Sans, sans-serif"
    fontSize: "20px"
    lineHeight: "28px"
  titulo-de-estado:
    fontFamily: "IBM Plex Sans, sans-serif"
    fontSize: "16px"
    fontWeight: 600
    lineHeight: "24px"
  titulo-de-painel:
    fontFamily: "IBM Plex Sans, sans-serif"
    fontSize: "14px"
    fontWeight: 600
    lineHeight: "20px"
  body:
    fontFamily: "IBM Plex Sans, sans-serif"
    fontSize: "14px"
    fontWeight: 400
    lineHeight: "20px"
    fontFeature: "tnum"
  label:
    fontFamily: "IBM Plex Sans, sans-serif"
    fontSize: "12px"
    fontWeight: 500
    lineHeight: "16px"
  apoio:
    fontFamily: "IBM Plex Sans, sans-serif"
    fontSize: "12px"
    fontWeight: 400
    lineHeight: "16px"
  identificador:
    fontFamily: "IBM Plex Mono, monospace"
    fontSize: "14px"
    fontWeight: 400
    lineHeight: "20px"
rounded:
  selo: "4px"
  controle: "6px"
spacing:
  "1": "4px"
  "1.5": "6px"
  "2": "8px"
  "3": "12px"
  "4": "16px"
  "5": "20px"
  "6": "24px"
  "8": "32px"
  "16": "64px"
components:
  selo-de-risco-ponto:
    typography: "{typography.label}"
  selo-de-risco-tingido:
    textColor: "{colors.texto-primario}"
    typography: "{typography.label}"
    rounded: "{rounded.selo}"
    height: "20px"
    padding: "0 6px"
  campo-de-busca:
    backgroundColor: "{colors.fundo-superficie}"
    textColor: "{colors.texto-primario}"
    typography: "{typography.body}"
    rounded: "{rounded.controle}"
    height: "32px"
    padding: "0 32px"
  selecao-gatilho:
    backgroundColor: "{colors.fundo-superficie}"
    textColor: "{colors.texto-primario}"
    typography: "{typography.body}"
    rounded: "{rounded.controle}"
    height: "32px"
    padding: "0 8px 0 12px"
  selecao-gatilho-aberto:
    backgroundColor: "{colors.fundo-elevado}"
  selecao-lista:
    backgroundColor: "{colors.fundo-elevado}"
    textColor: "{colors.texto-primario}"
    rounded: "{rounded.controle}"
    padding: "4px"
  selecao-item:
    rounded: "{rounded.selo}"
    height: "32px"
    padding: "0 32px 0 8px"
  selecao-item-destacado:
    backgroundColor: "{colors.fundo-superficie}"
  botao-de-tema:
    backgroundColor: "{colors.fundo-superficie}"
    textColor: "{colors.texto-primario}"
    typography: "{typography.body}"
    rounded: "{rounded.controle}"
    height: "32px"
    padding: "0 8px"
  botao-de-tema-hover:
    backgroundColor: "{colors.fundo-elevado}"
  navegacao-item:
    textColor: "{colors.texto-secundario}"
    typography: "{typography.body}"
    rounded: "{rounded.controle}"
    height: "32px"
    padding: "0 8px"
  navegacao-item-hover:
    backgroundColor: "{colors.fundo-elevado}"
    textColor: "{colors.texto-primario}"
  navegacao-item-ativo:
    backgroundColor: "{colors.fundo-elevado}"
    textColor: "{colors.texto-primario}"
  barra-lateral:
    backgroundColor: "{colors.fundo-superficie}"
    width: "240px"
  metrica:
    backgroundColor: "{colors.fundo-superficie}"
    padding: "12px 16px"
  metrica-valor:
    textColor: "{colors.texto-primario}"
    typography: "{typography.display}"
  proporcao:
    backgroundColor: "{colors.texto-secundario}"
    rounded: "2px"
    height: "6px"
  painel:
    backgroundColor: "{colors.fundo-superficie}"
    rounded: "{rounded.controle}"
  painel-cabecalho:
    backgroundColor: "{colors.fundo-elevado}"
    typography: "{typography.titulo-de-painel}"
    padding: "6px 12px"
  tabela-cabecalho:
    backgroundColor: "{colors.fundo-elevado}"
    textColor: "{colors.texto-secundario}"
    typography: "{typography.label}"
    padding: "6px 12px"
  tabela-linha:
    textColor: "{colors.texto-primario}"
    typography: "{typography.body}"
    height: "32px"
    padding: "6px 12px"
  tabela-linha-hover:
    backgroundColor: "{colors.fundo-elevado}"
  estado-de-pagina:
    backgroundColor: "{colors.fundo-superficie}"
    typography: "{typography.titulo-de-estado}"
    rounded: "{rounded.controle}"
    padding: "24px"
  esqueleto:
    backgroundColor: "{colors.fundo-elevado}"
    rounded: "{rounded.selo}"
---

# Design System: aPRova

## Overview

**Creative North Star: "O livro de registro"**

O dashboard do aPRova é um livro de registro: cada linha é uma decisão que
alguém pode ter de provar depois. A interface é sóbria como um terminal ou um
painel de observabilidade. Nada nela pede atenção por conta própria; o que
chama o olho é o que espera por uma pessoa.

A identidade foi definida por quem pediu o protótipo e não é reinterpretada
aqui. Este arquivo registra o que o código construído faz. Até agora existem a
casca (navegação, moldura de página e os estados de carregamento, erro e vazio)
e a tela Visão geral. Análises, Recibos e Regras aparecem na navegação, mas
ainda não têm tela: os três endereços caem em "Página não encontrada". Nenhum
componente dessas telas está descrito aqui, e os que elas trouxerem entram
quando forem construídos.

O tema escuro é o padrão. O claro existe e é alternável pelo botão de tema, e a
escolha persiste no navegador.

**Key Characteristics:**

- Escuro por padrão, denso, sem ornamento.
- Superfícies separadas por borda de 1px e por três tons de fundo, nunca por
  sombra.
- Duas famílias com papéis fixos: Plex Sans fala, Plex Mono identifica.
- Cor reservada: o azul é de link e foco, e as três cores de risco só aparecem
  onde se fala de risco.
- Todo número vem com unidade, período e denominador.
- Um único elemento decorativo previsto: a borda inferior serrilhada do bloco
  do recibo, que ainda não foi construída.

## Colors

Uma paleta de tinta e papel, quase sem cor, em que cada cor tem um único
significado. O tema escuro tem um fundo de carvão e texto levemente quente. O
tema claro é um cinza neutro, sem tom de creme: fundos, borda e texto
secundário são cinzas frios bem próximos do neutro.

Os valores vivem em `src/estilo.css`: o escuro em `:root` e o claro em
`[data-tema="claro"]`, com os mesmos nomes de variável. As escalas padrão do
Tailwind (cor, raio, tamanho de texto, sombra, fonte e ponto de quebra) foram
zeradas no `@theme`, então só existem as cores deste arquivo.

### Primary

- **Azul-aço** (`acao-link`; no claro, `claro-acao-link`): link e anel de foco,
  e mais nada. O link é sublinhado com 1px, e o sublinhado engrossa para 2px no
  hover. O anel de foco é um contorno de 2px afastado 2px.

### Tertiary

As três cores de risco. São semânticas, não decorativas.

- **Risco baixo** (`risco-baixo` / `claro-risco-baixo`): selo e texto do nível
  baixo, segmento do nível baixo no gráfico de barras por dia.
- **Risco médio** (`risco-medio` / `claro-risco-medio`): o mesmo, para médio.
- **Risco alto** (`risco-alto` / `claro-risco-alto`): o mesmo, para alto.

No selo tingido a cor de risco entra misturada: 13% sobre o fundo como tinta e
32% como borda.

### Neutral

- **Base** (`fundo-base` / `claro-fundo-base`): o fundo da página.
- **Superfície** (`fundo-superficie` / `claro-fundo-superficie`): painéis,
  tabela, navegação, campo de busca e select.
- **Elevado** (`fundo-elevado` / `claro-fundo-elevado`): o que está sobre uma
  superfície: linha em hover, item ativo da navegação, faixa de cabeçalho de
  painel e de tabela, lista do select, bloco de esqueleto.
- **Borda** (`borda-padrao` / `claro-borda-padrao`): a linha de 1px que separa
  superfícies, e a emenda de 1px entre as métricas.
- **Texto primário** (`texto-primario` / `claro-texto-primario`): conteúdo.
- **Texto secundário** (`texto-secundario` / `claro-texto-secundario`):
  rótulos, unidades, datas, apoio, contorno de controle e o preenchimento
  neutro da barra de proporção. A trilha da barra é essa mesma cor a 28%.

A seleção de texto inverte o par: fundo na cor do texto primário, texto na cor
da base.

### Named Rules

**A regra do azul.** O azul-aço é só para link e anel de foco. Não é cor de
botão principal, de destaque nem de gráfico.

**A regra do risco.** Verde, âmbar e vermelho aparecem só onde se fala de risco:
selo de nível, motivo, barra de gráfico de risco. Nunca como decoração, sucesso
genérico ou erro de formulário. Um estado `concluido` não é verde; um erro de
carregamento não é vermelho.

**A regra do nome.** O nível de risco nunca é dito só pela cor: vem sempre com
"baixo", "médio" ou "alto" escrito. Quando o selo mostra só o nome, a palavra
"risco" continua lá para leitor de tela.

**A regra do nível único.** Quando todas as linhas de uma lista têm o mesmo
nível, ele é escrito uma vez no título da lista, e não repetido numa coluna. A
fila "Aguardando aprovação" traz "risco alto" ao lado do título e não tem
coluna de risco.

**A regra do contorno.** Campo de texto, select e caixa de seleção têm o
contorno na cor do texto secundário, que dá 3:1 ou mais contra os três fundos
nos dois temas. A borda padrão continua sendo a linha que separa superfícies e
não delimita controle.

### Contraste medido

Razões de contraste contra os três fundos de cada tema, truncadas em uma casa.
O alvo é WCAG 2.2 AA: 4,5:1 para texto e 3:1 para o contorno de um controle.
Todo texto fica em 4,5:1 ou mais, inclusive sobre o fundo elevado, que é a
linha em hover e o item ativo.

| Cor                                     | Escuro: base / superfície / elevado | Claro: base / superfície / elevado |
| --------------------------------------- | ----------------------------------- | ---------------------------------- |
| texto primário                          | 15,4 / 14,6 / 13,4                  | 17,2 / 18,7 / 15,8                 |
| texto secundário                        | 6,9 / 6,6 / 6,0                     | 5,5 / 6,0 / 5,1                    |
| link                                    | 7,0 / 6,6 / 6,1                     | 5,1 / 5,6 / 4,7                    |
| risco baixo                             | 7,1 / 6,7 / 6,1                     | 5,1 / 5,5 / 4,7                    |
| risco médio                             | 8,5 / 8,0 / 7,4                     | 5,1 / 5,6 / 4,7                    |
| risco alto                              | 5,4 / 5,1 / 4,7                     | 5,7 / 6,2 / 5,2                    |
| borda de superfície                     | 1,3 / 1,2 / 1,1                     | 1,2 / 1,3 / 1,1                    |
| contorno de controle (texto secundário) | 6,9 / 6,6 / 6,0                     | 5,5 / 6,0 / 5,1                    |

O selo tingido põe o texto primário sobre a tinta de 13% da cor de risco. Nos
três fundos, o pior caso dá 10,5:1 ou mais no escuro e 13,0:1 ou mais no claro.

A borda de superfície não precisa de 3:1: ela não é o contorno de um controle,
e as superfícies também se distinguem pelo tom do fundo.

## Typography

**Body Font:** IBM Plex Sans (com `sans-serif`)
**Label/Mono Font:** IBM Plex Mono (com `monospace`)

**Character:** uma só voz para texto e outra para tudo o que é copiado e
conferido. As duas vêm de `@fontsource`, sem CDN. A Plex Sans é carregada nos
pesos 400, 500 e 600; a Plex Mono, em 400 e 500.

### Hierarchy

A escala é 32 / 24 / 20 / 16 / 14 / 12, cada tamanho com sua altura de linha:
32/40, 24/32, 20/28, 16/24, 14/20 e 12/16. O texto base da página é 14/20, com
algarismos tabulares (`font-variant-numeric: tabular-nums`) em tudo. São três
pesos: 400 no texto, 500 em rótulos, cabeçalhos de coluna e no selo, 600 em
títulos e no valor da métrica.

- **Display** (600, 32px, 40px): o valor de cada métrica.
- **Headline** (600, 24px, 32px): título da página.
- **Title** (20px, 28px): o passo existe na escala, mas nenhuma tela construída
  o usa. Na Visão geral os títulos de seção são de 14px, como descrito abaixo.
- **Título de estado** (600, 16px, 24px): título do bloco de erro e do bloco
  vazio. É o único uso do passo de 16px até aqui.
- **Título de painel** (600, 14px, 20px): título da fila e dos painéis da Visão
  geral. É uma escolha de painel denso: o título fica do tamanho do texto e se
  distingue pelo peso e pela faixa de cabeçalho, não pelo tamanho.
- **Body** (400, 14px, 20px): texto, tabelas, listas e controles. Texto corrido
  não passa de 65ch de largura.
- **Label** (500, 12px, 16px): rótulo de métrica, cabeçalho de coluna, selo.
- **Apoio** (400, 12px, 16px): linha de apoio da métrica, resumo do painel,
  contagem da fila, eixos do gráfico, na cor do texto secundário.
- **Identificador** (Plex Mono, no tamanho do texto em volta): hash,
  `head_sha`, `key_id`, id de motivo, nome de branch, número de PR e caminho de
  arquivo.

Títulos quebram com `text-wrap: balance`; parágrafos, com `pretty`.

### Named Rules

**A regra do identificador.** Tudo o que alguém pode copiar para um terminal ou
comparar caractere a caractere é Plex Mono, e isso inclui caminho de arquivo.
Todo o resto é Plex Sans. Mono não é usada para dar ar técnico a um rótulo. Na
fila, a evidência só vai em Mono quando é um caminho; quando é uma frase, fica
em Sans.

**A regra do denominador.** Todo número vem com unidade, período e denominador.
A métrica escreve a porcentagem com "das avaliadas" ao lado e, embaixo, a
fração e o período, no formato "N de M análises em D dias". A barra de
proporção nunca fala sozinha: a fração está sempre escrita ao lado dela.

## Layout

Grade de 4px, com os passos 4, 8, 12, 16, 20, 24, 32 e 64. Há um meio passo de
6px, usado no respiro vertical das linhas densas (célula de tabela, cabeçalho
de painel) e no espaço entre o ponto e o texto do selo; é ele que dá a linha de
32px com texto de 14/20. Desktop primeiro, com quatro pontos de quebra: `md` em
768px, `lg` em 1100px, `xl` em 1280px e `2xl` em 1440px.

**Casca.** A partir de 768px, a navegação é uma barra lateral de 240px sobre o
fundo de superfície, com borda de 1px à direita e a altura do documento
inteiro. Dentro dela, uma coluna presa ao topo com a altura da janela tem três
partes: cabeçalho com a logo (28px de altura), navegação que rola sozinha se
não couber, e rodapé com a instalação e o botão de tema. O conteúdo fica ao
lado e ocupa o resto.

Abaixo de 768px a barra lateral vira uma barra no topo, com borda embaixo: a
logo e o botão de tema numa linha, e as quatro seções em quatro abas de largura
igual, sem rolagem horizontal. As abas perdem o ícone e o complemento do rótulo
("do gate" fica só para leitor de tela), e o rodapé da instalação some.

**Página.** Margem de 16px nos lados, 24px no topo e 64px embaixo; a partir de
768px, 32px nos lados e no topo. O cabeçalho tem o título e, na mesma linha de
base, o contexto (período e hora da atualização) em texto secundário; quando
não cabem, o contexto desce. Abaixo do cabeçalho, 32px.

**Visão geral.** Uma coluna com 16px entre os blocos, nesta ordem: faixa de
métricas, fila de aprovação, painéis do período. A faixa tem 2 colunas abaixo
de 1280px e 4 a partir daí. Os painéis ficam em 1 coluna no celular, 2 colunas
entre 768px e 1280px (o gráfico ocupa as duas) e 3 colunas a partir de 1280px,
na proporção 1 : 1 : 2.

**Tabela em tela estreita.** A fila perde colunas em vez de rolar: abaixo de
1440px somem Branch e Commit, para a Evidência caber inteira; entre 768px e
1100px os motivos se empilham e o respiro lateral das células cai para 8px;
abaixo de 768px, Motivos e Evidência descem para dentro da célula do pull
request. A Evidência corta com reticências quando não cabe e mostra o texto
inteiro no `title`. A rolagem horizontal existe só como reserva.

## Elevation & Depth

Sem sombra. A profundidade vem da troca entre base, superfície e elevado e de
uma borda de 1px. A escala de sombra do Tailwind foi zerada, então não há
classe de sombra disponível. A lista do select, que flutua sobre a página, usa
fundo elevado e borda de 1px, sem sombra.

O movimento é um só: o utilitário `transicao-de-estado`, de 120ms com a curva
`cubic-bezier(0.16, 1, 0.3, 1)`, aplicado a fundo, borda, cor, rotação e
opacidade. Com `prefers-reduced-motion: reduce` a duração é zero. Não há
animação de entrada nem mola. O esqueleto de carregamento é parado.

### Named Rules

**A regra da borda.** Superfícies se separam por borda de 1px e pela troca de
fundo. Sombra difusa não separa nada aqui.

## Shapes

Raio de 6px em controles e também nos blocos de superfície (painel, faixa de
métricas, fila, bloco de erro e de vazio). Raio de 4px em selos e nas peças
pequenas dentro de um controle: botão de limpar a busca, item do select, bloco
de esqueleto. Nada acima de 6px, e nenhum controle em formato de pílula.

Duas formas menores aparecem sem token próprio: a barra de proporção e a
amostra da legenda têm raio de 2px, e o ponto do selo é um círculo de 6px. As
barras do gráfico têm canto reto.

Os ícones são da Lucide, em traço de 1,75: 16px ao lado de um rótulo, 14px
dentro do botão de limpar e na seta de ordenação. São sempre decorativos e vêm
junto de um texto ou de um rótulo acessível.

O bloco do recibo pode ter a borda inferior serrilhada, como um cupom. É o
único elemento decorativo permitido em todo o sistema, e ainda não foi
construído.

## Components

Peças densas e quietas: 32px de altura para controle e para linha, fundo de
superfície, borda de 1px, e estado mostrado pela troca para o fundo elevado.
Tudo fica em `src/componentes/`, salvo a barra de ferramentas da fila.

### Selo de risco (`SeloDeRisco.tsx`)

- **Ponto:** um ponto de 6px na cor do nível e o nome do nível na mesma cor, em
  12px peso 500. A palavra "risco" vem antes do nome; pode ficar visível ou só
  para leitor de tela. É a variante em uso, no título da fila.
- **Tingido:** 20px de altura, raio de 4px, 6px de respiro lateral, fundo com
  13% da cor do nível, borda de 1px com 32%, ponto de 6px e texto na cor
  primária. Está pronto no componente, mas nenhuma tela construída o usa ainda.
- **Amostra do nível:** quadrado de 8px com raio de 2px na cor do nível, para
  legenda de gráfico. Vem sempre seguido do nome e da contagem.

### Faixa de métricas (`FaixaDeMetricas.tsx`)

- **Forma:** uma lista de definições com borda de 1px e raio de 6px. As células
  se juntam por uma emenda de 1px na cor da borda (o utilitário `emenda`), sem
  espaço entre elas. São 2 colunas abaixo de 1280px e 4 acima.
- **Célula:** respiro de 12px por 16px (16px por 20px a partir de 1280px).
  Rótulo em 12px peso 500 secundário; valor em 32px peso 600 tabular com a
  unidade em 14px secundário na mesma linha de base; barra de proporção; linha
  de apoio em 12px secundário com a fração e o período.

### Proporção (`Proporcao.tsx`)

- **Forma:** barra de 6px de altura e raio de 2px, na largura inteira do
  espaço.
- **Cor:** preenchimento neutro na cor do texto secundário sobre a trilha (a
  mesma cor a 28%). Não usa cor de risco nem o azul.
- **Papel:** decorativa e escondida de leitor de tela. A fração está sempre
  escrita ao lado.

### Tabela (`Tabela.tsx`)

- **Cabeçalho:** faixa de fundo elevado com borda de 1px embaixo; rótulos em
  12px peso 500 secundário.
- **Linha:** 32px (texto de 14/20 com 6px em cima e embaixo, 12px nos lados),
  borda de 1px entre linhas, sem quebra de texto. No hover o fundo passa a
  elevado.
- **Coluna ordenável:** o cabeçalho é um botão dentro do `th`, com `aria-sort`.
  A seta de 14px gira 180 graus conforme o sentido e some quando a coluna não
  ordena; em coluna alinhada à direita a seta vem antes do rótulo. A coluna que
  ordena tem o rótulo na cor primária.
- **Linha clicável:** o link do pull request cobre a linha inteira. Os
  identificadores e os links de motivo ficam acima dessa área, para continuarem
  selecionáveis e clicáveis.

### Campo de busca (`CampoDeBusca.tsx`)

- **Estilo:** 32px de altura, raio de 6px, fundo de superfície, contorno de 1px
  na cor do texto secundário, texto de 14px. Lupa de 16px à esquerda.
- **Estados:** no hover o contorno passa à cor primária. Com texto digitado, a
  lupa passa de secundária a primária e aparece o botão de limpar (24px, raio
  de 4px, fundo elevado no hover), que devolve o foco ao campo.

### Seleção (`Selecao.tsx`, sobre Radix Select)

- **Gatilho:** igual ao campo de busca em altura, raio, fundo e contorno, com
  uma seta de 16px secundária à direita. Aberto, o fundo passa a elevado.
- **Lista:** abre colada ao gatilho, 4px abaixo, pelo menos da largura dele.
  Fundo elevado, borda de 1px, raio de 6px, 4px de respiro, sem sombra e sem
  animação de abertura.
- **Item:** 32px, raio de 4px. O item sob o cursor ou o teclado passa ao fundo
  de superfície; o escolhido tem uma marca de 16px à direita.

### Barra de ferramentas da fila (`src/telas/FilaDeAprovacao.tsx`)

- **Anatomia:** na mesma faixa do título da fila, à direita: campo de busca
  (224px), select de repositório (208px) e a contagem "N de M análises" em 12px
  secundário, anunciada por `role="status"`. Abaixo de 768px a busca ocupa a
  largura inteira e o select divide a linha com a contagem.
- **Limpar filtros:** aparece só quando o filtro não devolve nenhuma linha, na
  frase que explica o vazio, com aparência de link.
- **Fila vazia:** se não há nada à espera, a tabela dá lugar a uma frase em
  texto secundário que diz quando um PR aparece ali.

### Painel (`Painel.tsx`)

- **Forma:** bloco de superfície com borda de 1px e raio de 6px. O cabeçalho é
  uma faixa de fundo elevado com borda embaixo: título em 14px peso 600 à
  esquerda e resumo em 12px secundário à direita (o denominador, ou a legenda
  do gráfico).
- **Tabela de contagem:** cada linha tem o rótulo, a barra de proporção
  ocupando o meio e o número à direita, com borda de 1px entre linhas.

### Barras por dia (`BarrasPorDia.tsx`)

- **Forma:** uma coluna por dia, em CSS puro, com 3px entre os dias. Cada
  coluna empilha os níveis de baixo para cima (baixo, médio, alto) nas cores de
  risco, com 1px entre os segmentos. Linha de base e linha de topo em borda de
  1px.
- **Eixos:** valores à direita (o pico e o zero) e três marcas de data embaixo
  (primeiro dia, dia do meio e "hoje"), em 12px secundário.
- **Leitura:** cada dia tem um `title` nativo com as três contagens. Para
  leitor de tela há uma legenda e uma tabela com os mesmos números; o desenho
  fica escondido.

### Link do motivo (`LinkDoMotivo.tsx`)

- O id do motivo em Plex Mono, como link para `/regras#id`. Vários motivos
  ficam lado a lado com 12px entre eles.

### Navegação e casca (`Casca.tsx`)

- **Item:** 32px, raio de 6px, ícone de 16px e rótulo em 14px secundário, sem
  sublinhado. No hover, fundo elevado e texto primário. O item ativo tem fundo
  elevado, texto primário e peso 500.
- **Botão de tema:** 32px, raio de 6px, fundo de superfície, borda de 1px na
  cor da borda padrão, ícone de sol ou lua e o rótulo "Tema claro" ou "Tema
  escuro" (o tema para o qual ele leva). No hover, fundo elevado. A escolha
  fica guardada no navegador e é aplicada antes da primeira pintura.
- **Rodapé da instalação:** conta em 14px primário, número de repositórios e o
  aviso de protótipo em 12px secundário.
- **Pular para o conteúdo:** primeiro elemento focável, visível só com foco.

### Página e estados (`Pagina.tsx`, `Esqueleto.tsx`)

- **Carregando:** blocos de esqueleto com fundo elevado e raio de 4px, nas
  medidas do conteúdo que vai chegar, parados. Um texto para leitor de tela diz
  o que está carregando.
- **Erro:** bloco de superfície com borda de 1px, raio de 6px e 24px de
  respiro, até 65ch: título de 16px peso 600, explicação em texto secundário e
  o link "Tentar de novo". Não usa vermelho.
- **Vazio:** o mesmo bloco, com título, texto e uma ação opcional.

### Componentes e origem

A fonte desta tabela é `LICENSES-TERCEIROS.md`, que também guarda o aviso de
copyright e o texto da licença. Oito peças derivam da Arc Library
(https://github.com/kuratlielia/arc-library, licença MIT, só a parte gratuita),
lida pelo MCP `ddl`.

| Componente                                  | Origem                         | Licença | O que foi alterado                                                                                                                         |
| ------------------------------------------- | ------------------------------ | ------- | ------------------------------------------------------------------------------------------------------------------------------------------ |
| `SeloDeRisco.tsx`                           | Arc, `badge`                   | MIT     | Cores trocadas pelas três de risco; texto primário sobre a tinta; raio de 4px; sem troca animada de texto nem largura em mola              |
| `Tabela.tsx`                                | Arc, `sortable-data-table`     | MIT     | Linhas de 32px em vez de 52px; sem seleção de linhas, sem faixa de coluna e sem linhas deslizando ao ordenar; seta com transição de 120ms  |
| `FaixaDeMetricas.tsx`                       | Arc, `stats-band` (`divided`)  | MIT     | Sem contagem animada, sem entrada escalonada e sem minigráficos; número em 32px; barra de proporção no lugar do minigráfico                |
| `Proporcao.tsx`                             | Arc, `usage-meter`             | MIT     | Um segmento só, em cor neutra; 6px de altura e raio de 2px; sem legenda interativa nem números rolando                                     |
| `CampoDeBusca.tsx`                          | Arc, `search-field`            | MIT     | Altura de 32px; contorno na cor do texto secundário; sem animação de entrada do botão                                                      |
| `Selecao.tsx`                               | Arc, `select`                  | MIT     | Sem valor rolando e sem animação de abertura; sem sombra; raio de 6px; fundo elevado com borda de 1px                                      |
| Barra de ferramentas em `FilaDeAprovacao.tsx` | Arc, `filter-toolbar`        | MIT     | Sem chips e sem o menu que cresce a partir do botão; os filtros são um campo de busca e um select                                          |
| `BarrasPorDia.tsx`                          | Arc, `bar-chart`               | MIT     | Barras empilhadas por nível de risco, em CSS puro; sem cursor de leitura, sem média e sem animação                                         |
| `Casca.tsx`                                 | ARK CN, só como referência     | nenhuma | Nenhum código copiado. Serviu de referência para a estrutura da barra lateral (cabeçalho, conteúdo que rola e rodapé), escrita do zero     |

`Painel.tsx`, `Pagina.tsx`, `Esqueleto.tsx`, `LinkDoMotivo.tsx` e
`Identificador.tsx` foram escritos do zero. Do Amicro, nada foi adotado até
aqui. A ARK CN não tem arquivo `LICENSE` e é construída sobre Ark UI, não sobre
Radix; por isso serve de referência de estrutura e não de fonte de código.

Regras que valem para toda peça nova:

- Uma só base de primitivos: Radix.
- As fontes permitidas são shadcn/ui, Arc (parte gratuita), Amicro e ARK CN, e
  nenhuma outra.
- Tudo entra como código-fonte reescrito com os tokens deste arquivo. Não fica
  nenhuma cor, raio, sombra ou fonte da origem.
- Componente animado entra com o movimento reduzido à transição curta de
  estado.
- Toda peça derivada é registrada em `LICENSES-TERCEIROS.md` com a origem, a
  licença e o que mudou.

## Do's and Don'ts

### Do:

- **Do** usar o tema escuro como padrão e manter o claro alternável, com a
  escolha persistida.
- **Do** separar superfícies com borda de 1px e com a troca entre base,
  superfície e elevado.
- **Do** escrever em Plex Mono todo hash, `head_sha`, `key_id`, id de motivo,
  nome de branch, número de PR e caminho de arquivo.
- **Do** escrever o nome do nível junto da cor de risco.
- **Do** escrever o nível uma vez no título quando a lista inteira tem o mesmo
  nível, em vez de repeti-lo numa coluna.
- **Do** dar unidade, período e denominador a todo número, e escrever a fração
  ao lado de toda barra de proporção.
- **Do** manter controle e linha de tabela em 32px de altura.
- **Do** limitar o movimento à transição curta de estado (120ms) e zerá-la com
  `prefers-reduced-motion: reduce`.
- **Do** registrar em `LICENSES-TERCEIROS.md` toda peça derivada de outra
  biblioteca.
- **Do** usar a logo de `design/marca/svg/` no topo da navegação, na versão do
  tema, e `icone-app-fundo-escuro-32.png` como favicon.

### Don't:

- **Don't** usar o azul-aço em botão principal, destaque ou gráfico.
- **Don't** usar cor de risco fora de risco: nem como sucesso genérico, nem
  como erro de formulário, nem como decoração.
- **Don't** usar roxo, gradiente, glassmorphism ou fundo com blur.
- **Don't** usar sombra difusa como separador, nem sombra grande.
- **Don't** usar Inter, Roboto ou fonte do sistema no lugar da Plex.
- **Don't** pôr card dentro de card.
- **Don't** pôr um ícone num quadradinho arredondado em cima de cada título.
- **Don't** escrever texto cinza sobre fundo colorido.
- **Don't** usar emoji na interface.
- **Don't** usar botão em pílula nem cantos acima de 6px.
- **Don't** trocar informação por frase de efeito.
- **Don't** animar a entrada de página, nem usar mola.
- **Don't** manter cor, raio, sombra, fonte ou animação de uma biblioteca de
  origem.
- **Don't** alterar `design/marca/`.
