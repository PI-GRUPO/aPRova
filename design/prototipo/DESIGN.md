---
name: aPRova
description: Dashboard de auditoria de pull requests gerados por agentes de IA
colors:
  fundo-base: "#08090A"
  fundo-superficie: "#0F1011"
  fundo-elevado: "#161718"
  borda-padrao: "#23252A"
  texto-titulo: "#FFFFFF"
  texto-primario: "#D0D6E0"
  texto-secundario: "#8A8F98"
  acento: "#E4F222"
  risco-baixo: "#27A644"
  risco-medio: "#E0A43B"
  risco-alto: "#EB5757"
  claro-fundo-base: "#E9EAEC"
  claro-fundo-superficie: "#FFFFFF"
  claro-fundo-elevado: "#F1F2F4"
  claro-borda-padrao: "#D3D5DA"
  claro-texto-titulo: "#0F1011"
  claro-texto-primario: "#1C1E24"
  claro-texto-secundario: "#62666D"
  claro-acento: "#5F6600"
  claro-risco-baixo: "#1F764E"
  claro-risco-medio: "#915C08"
  claro-risco-alto: "#B42F28"
typography:
  display:
    fontFamily: "Inter Variable, sans-serif"
    fontSize: "32px"
    fontWeight: 590
    lineHeight: "40px"
    letterSpacing: "-0.022em"
    fontFeature: "'cv01', 'ss03', 'zero', 'tnum'"
  headline:
    fontFamily: "Inter Variable, sans-serif"
    fontSize: "24px"
    fontWeight: 590
    lineHeight: "32px"
    letterSpacing: "-0.012em"
    fontFeature: "'cv01', 'ss03', 'zero', 'tnum'"
  title:
    fontFamily: "Inter Variable, sans-serif"
    fontSize: "20px"
    lineHeight: "28px"
    letterSpacing: "-0.011em"
  titulo-de-estado:
    fontFamily: "Inter Variable, sans-serif"
    fontSize: "16px"
    fontWeight: 590
    lineHeight: "24px"
    letterSpacing: "-0.011em"
  titulo-de-painel:
    fontFamily: "Inter Variable, sans-serif"
    fontSize: "14px"
    fontWeight: 590
    lineHeight: "20px"
    letterSpacing: "-0.011em"
  body:
    fontFamily: "Inter Variable, sans-serif"
    fontSize: "14px"
    fontWeight: 400
    lineHeight: "20px"
    letterSpacing: "-0.011em"
    fontFeature: "'cv01', 'ss03', 'zero', 'tnum'"
  label:
    fontFamily: "Inter Variable, sans-serif"
    fontSize: "12px"
    fontWeight: 510
    lineHeight: "16px"
    letterSpacing: "-0.011em"
  apoio:
    fontFamily: "Inter Variable, sans-serif"
    fontSize: "12px"
    fontWeight: 400
    lineHeight: "16px"
    letterSpacing: "-0.011em"
  identificador:
    fontFamily: "IBM Plex Mono, monospace"
    fontSize: "14px"
    fontWeight: 400
    lineHeight: "20px"
rounded:
  selo: "4px"
  controle: "6px"
  painel: "12px"
spacing:
  "0.5": "2px"
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
    textColor: "{colors.texto-titulo}"
  barra-lateral:
    backgroundColor: "{colors.fundo-superficie}"
    width: "240px"
  faixa-de-metricas:
    rounded: "{rounded.painel}"
  metrica:
    backgroundColor: "{colors.fundo-superficie}"
    padding: "12px 16px"
  metrica-valor:
    textColor: "{colors.texto-titulo}"
    typography: "{typography.display}"
  proporcao:
    backgroundColor: "{colors.texto-secundario}"
    rounded: "2px"
    height: "6px"
  painel:
    backgroundColor: "{colors.fundo-superficie}"
    rounded: "{rounded.painel}"
  painel-cabecalho:
    backgroundColor: "{colors.fundo-elevado}"
    textColor: "{colors.texto-titulo}"
    typography: "{typography.titulo-de-painel}"
    padding: "6px 12px"
  mapa-de-calor-celula-vazia:
    backgroundColor: "{colors.fundo-elevado}"
    rounded: "2px"
    height: "20px"
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
    textColor: "{colors.texto-titulo}"
    typography: "{typography.titulo-de-estado}"
    rounded: "{rounded.painel}"
    padding: "24px"
  esqueleto:
    backgroundColor: "{colors.fundo-elevado}"
    rounded: "{rounded.selo}"
  esqueleto-lista:
    backgroundColor: "{colors.fundo-superficie}"
    rounded: "{rounded.painel}"
---

# Design System: aPRova

## Overview

**Creative North Star: "O livro de registro"**

O dashboard do aPRova é um livro de registro: cada linha é uma decisão que
alguém pode ter de provar depois. A interface é sóbria como um terminal ou um
painel de observabilidade. Nada nela pede atenção por conta própria; o que
chama o olho é o que espera por uma pessoa.

O livro agora é desenhado com uma paleta mais escura, adaptada da referência de
estilo do Linear: fundo quase preto, três tons de superfície muito próximos,
título em branco, texto em cinza frio e um único acento, um lima ácido. A
referência foi adaptada, não copiada. Do aPRova ficaram a ausência total de
sombra, o tema claro, as cores de risco como cores semânticas, a linha densa
da fila (36px) e as regras do identificador e do denominador. Do Linear não vieram o fio de
0,5px, as sombras, as pílulas, os tamanhos de 13, 15 e 17px, a escala de texto
de marketing (48 a 72px) nem a Berkeley Mono.

Este arquivo registra o que o código construído faz. Até agora existem a casca
(navegação, moldura de página e os estados de carregamento, erro e vazio), a
tela Visão geral e a tela Página não encontrada. Análises, Recibos e Regras
aparecem na navegação, mas ainda não têm tela: os três endereços caem em
"Página não encontrada". Nenhum componente dessas telas está descrito aqui, e
os que elas trouxerem entram quando forem construídos.

O tema escuro é o padrão. O claro existe e é alternável pelo botão de tema, e a
escolha persiste no navegador.

**Key Characteristics:**

- Escuro por padrão, denso, sem ornamento nas telas de trabalho.
- Superfícies separadas por borda de 1px e por três tons de fundo, nunca por
  sombra.
- Duas famílias com papéis fixos: Inter fala, Plex Mono identifica.
- Título e corpo têm cores diferentes: título em `texto-titulo`, corpo em
  `texto-primario`.
- Cor reservada: o lima é de foco, item ativo e seleção, e as três cores de
  risco só aparecem onde se fala de risco. Link não tem cor própria.
- Todo número vem com unidade, período e denominador.
- Dois elementos decorativos permitidos em todo o sistema: a borda inferior
  serrilhada do bloco do recibo, que ainda não foi construída, e a composição
  tipográfica da Página não encontrada.

## Colors

Uma paleta de tinta e papel, quase sem cor, em que cada cor tem um único
significado. O tema escuro tem fundo quase preto, texto em cinza frio e título
em branco. O tema claro tira os cinzas da referência "MONOLITH, system status":
base cinza frio, superfície branca, elevado um cinza muito claro. Do MONOLITH
veio só a paleta de cinzas; blur, vidro, neumorfismo, gradiente, 3D e sombra
continuam proibidos aqui.

Os valores vivem em `src/estilo.css`: o escuro em `:root` e o claro em
`[data-tema="claro"]`, com os mesmos nomes de variável. As escalas padrão do
Tailwind (cor, raio, tamanho de texto, peso, sombra, fonte e ponto de quebra)
foram zeradas no `@theme`, então só existem as cores deste arquivo.

### Primary

- **Lima ácido** (`acento`; no claro, `claro-acento`): anel de foco, item ativo
  da navegação e seleção de texto, e mais nada até aqui. O anel de foco é um
  contorno de 2px afastado 2px. No item ativo, o lima pinta o ícone na barra
  lateral e, no celular, uma linha de 2px embaixo da aba. No tema claro o
  acento é um oliva escuro, porque o lima não tem contraste sobre branco.

### Tertiary

As três cores de risco. São semânticas, não decorativas.

- **Risco baixo** (`risco-baixo` / `claro-risco-baixo`): selo e texto do nível
  baixo, linha do nível baixo no mapa de calor.
- **Risco médio** (`risco-medio` / `claro-risco-medio`): o mesmo, para médio.
- **Risco alto** (`risco-alto` / `claro-risco-alto`): o mesmo, para alto.

No selo tingido a cor de risco entra misturada: 13% sobre o fundo como tinta e
32% como borda. No mapa de calor ela entra em três passos fixos, 76% para uma
análise, 88% para duas e 100% para três ou mais, misturada na cor da
superfície.

### Neutral

- **Base** (`fundo-base` / `claro-fundo-base`): o fundo da página.
- **Superfície** (`fundo-superficie` / `claro-fundo-superficie`): painéis,
  tabela, navegação, campo de busca e select.
- **Elevado** (`fundo-elevado` / `claro-fundo-elevado`): o que está sobre uma
  superfície: linha em hover, item ativo da navegação, faixa de cabeçalho de
  painel e de tabela, lista do select, bloco de esqueleto e célula vazia do
  mapa de calor.
- **Borda** (`borda-padrao` / `claro-borda-padrao`): a linha de 1px que separa
  superfícies, e a emenda de 1px entre as métricas.
- **Texto de título** (`texto-titulo` / `claro-texto-titulo`): `h1`, `h2`, `h3`,
  o valor da métrica e o rótulo do item ativo da navegação.
- **Texto primário** (`texto-primario` / `claro-texto-primario`): corpo, células
  de tabela, controles e links.
- **Texto secundário** (`texto-secundario` / `claro-texto-secundario`):
  rótulos, unidades, datas, apoio, contorno de controle e o preenchimento
  neutro da barra de proporção. A trilha da barra é essa mesma cor a 28%.

A seleção de texto usa o acento: fundo na cor do acento, texto na cor da base.

### Named Rules

**A regra do acento.** O lima é só para o anel de foco, o item ativo da
navegação (ícone na barra lateral, linha de 2px embaixo da aba no celular) e a
seleção de texto. Um botão principal pode usá-lo quando existir; nenhum existe
ainda. Não é cor de link, de destaque nem de gráfico. Não há mais azul no
sistema.

**A regra do link.** Link não tem cor própria: é texto primário com sublinhado
de 1px, afastado 3px, que engrossa para 2px no hover. O que diz "isto é um
link" é o sublinhado, não a cor. Na fila, o link do pull request ganha peso
510, porque é ele que abre a análise.

**A regra do risco.** Verde, âmbar e vermelho aparecem só onde se fala de risco:
selo de nível, motivo, linha do mapa de calor. Nunca como decoração, sucesso
genérico ou erro de formulário. Um estado `concluido` não é verde; um erro de
carregamento não é vermelho.

**A regra do nome.** O nível de risco nunca é dito só pela cor: vem sempre com
"baixo", "médio" ou "alto" escrito. Quando o selo mostra só o nome, a palavra
"risco" continua lá para leitor de tela. No mapa de calor, cada linha tem o
nome do nível escrito à esquerda.

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
| texto de título                         | 19,9 / 19,0 / 17,9                  | 15,8 / 19,0 / 17,0                 |
| texto primário                          | 13,6 / 13,0 / 12,2                  | 13,8 / 16,6 / 14,8                 |
| texto secundário                        | 6,1 / 5,8 / 5,5                     | 4,7 / 5,7 / 5,1                    |
| acento                                  | 16,1 / 15,4 / 14,5                  | 5,1 / 6,1 / 5,5                    |
| risco baixo                             | 6,2 / 6,0 / 5,6                     | 4,6 / 5,5 / 4,9                    |
| risco médio                             | 9,0 / 8,6 / 8,1                     | 4,6 / 5,6 / 5,0                    |
| risco alto                              | 5,7 / 5,4 / 5,1                     | 5,1 / 6,2 / 5,5                    |
| borda de superfície                     | 1,2 / 1,2 / 1,1                     | 1,2 / 1,4 / 1,3                    |
| contorno de controle (texto secundário) | 6,1 / 5,8 / 5,5                     | 4,7 / 5,7 / 5,1                    |

O preenchimento da barra de proporção contra a própria trilha, sobre a
superfície, dá 3,8:1 nos dois temas.

O selo tingido põe o texto primário sobre a tinta de 13% da cor de risco. Nos
três fundos, o pior caso dá 9,8:1 no escuro (risco médio sobre o elevado) e
11,4:1 no claro (risco alto sobre a base).

A seleção de texto, base sobre acento, dá 16,1:1 no escuro e 5,1:1 no claro.

A borda de superfície não precisa de 3:1: ela não é o contorno de um controle,
e as superfícies também se distinguem pelo tom do fundo.

Toda célula preenchida do mapa de calor fica em 3:1 ou mais contra a
superfície e contra a célula vazia, nos dois temas: o passo mais fraco (76%)
dá 3,0:1 no pior caso, o médio sobre o branco. O passo cheio chega a 5:1.
A célula vazia quase não se separa da superfície (1,0 no escuro, 1,1 no
claro), e de propósito: zero não é um dado a destacar. A célula nunca é a
única fonte do número: a dica de cada célula, o número escrito nas células de
duas análises ou mais, o total da linha e a escala dizem o mesmo.

## Typography

**Body Font:** Inter Variable (com `sans-serif`)
**Label/Mono Font:** IBM Plex Mono (com `monospace`)

**Character:** uma só voz para texto e outra para tudo o que é copiado e
conferido. As duas vêm de `@fontsource`, sem CDN: a Inter do pacote
`@fontsource-variable/inter`, variável, e a Plex Mono no peso 400. A
Inter é configurada com `font-feature-settings: "cv01", "ss03", "zero"` (o zero
é cortado) e algarismos tabulares em tudo. A Plex Mono ficou no lugar da
Berkeley Mono da referência, que é paga.

### Hierarchy

A escala é 32 / 24 / 20 / 16 / 14 / 12, cada tamanho com sua altura de linha:
32/40, 24/32, 20/28, 16/24, 14/20 e 12/16. O texto base da página é 14/20, com
espaçamento entre letras de -0,011em, que os outros tamanhos herdam; o de 24px
usa -0,012em e o de 32px, -0,022em. São três pesos, os únicos que existem no
tema: 400 no texto, 510 (`font-medium`) em rótulos, cabeçalhos de coluna, selo
e item ativo da navegação, 590 (`font-semibold`) em títulos e no valor da
métrica.

- **Display** (590, 32px, 40px, -0,022em): o valor de cada métrica, na cor de
  título.
- **Headline** (590, 24px, 32px, -0,012em): título da página.
- **Title** (20px, 28px): o passo existe na escala, mas nenhuma tela construída
  o usa. Na Visão geral os títulos de seção são de 14px, como descrito abaixo.
- **Título de estado** (590, 16px, 24px): título do bloco de erro e do bloco
  vazio. É o único uso do passo de 16px até aqui.
- **Título de painel** (590, 14px, 20px): título da fila e dos painéis da Visão
  geral. É uma escolha de painel denso: o título fica do tamanho do texto e se
  distingue pelo peso, pela cor de título e pela faixa de cabeçalho, não pelo
  tamanho.
- **Body** (400, 14px, 20px): texto, tabelas, listas e controles. Texto corrido
  não passa de 65ch de largura.
- **Label** (510, 12px, 16px): rótulo de métrica, cabeçalho de coluna, selo.
- **Apoio** (400, 12px, 16px): linha de apoio da métrica, resumo do painel,
  contagem da fila, rótulos, totais e eixo do mapa de calor. Em texto
  secundário, salvo o total da linha do mapa, que é primário.
- **Identificador** (Plex Mono, no tamanho do texto em volta): hash,
  `head_sha`, `key_id`, id de motivo, nome de branch, número de PR e caminho de
  arquivo.

Títulos quebram com `text-wrap: balance`; parágrafos, com `pretty`.

### Named Rules

**A regra do identificador.** Tudo o que alguém pode copiar para um terminal ou
comparar caractere a caractere é Plex Mono, e isso inclui caminho de arquivo.
Todo o resto é Inter. Mono não é usada para dar ar técnico a um rótulo. Na
fila, a evidência só vai em Mono quando é um caminho; quando é uma frase, fica
em Inter.

**A regra do denominador.** Todo número vem com unidade, período e denominador.
A métrica escreve a porcentagem com "das avaliadas" ao lado e, embaixo, a
fração e o período, no formato "N de M análises em D dias". A barra de
proporção nunca fala sozinha: a fração está sempre escrita ao lado dela. O mapa
de calor traz a escala "0 · 1 · 2 · 3+ por dia" e o total de cada linha.

**A regra do peso.** Na interface os pesos são 400, 510 e 590. Nenhum texto de
tela usa 700 ou mais. A única exceção é a composição decorativa da Página não
encontrada, que desenha os algarismos em 800.

## Layout

Grade de 4px, com os passos 4, 8, 12, 16, 20, 24, 32 e 64. Há um meio passo de
6px, usado no respiro vertical do cabeçalho de tabela e de painel e no espaço
entre o ponto e o texto do selo. A linha de tabela tem 36px: texto de 14/20 com
8px em cima e embaixo. Há também um passo de 2px, usado só entre as células e
as linhas do mapa de calor. Desktop primeiro, com quatro pontos de quebra: `md`
em 768px, `lg` em 1100px, `xl` em 1280px, `2xl` em 1440px e `3xl` em 1600px.

**Casca.** A partir de 768px, a navegação é uma barra lateral de 240px sobre o
fundo de superfície, com borda de 1px à direita e a altura do documento
inteiro. Dentro dela, uma coluna presa ao topo com a altura da janela tem três
partes: cabeçalho com a logo (28px de altura), navegação que rola sozinha se
não couber, e rodapé com a instalação e o botão de tema. O conteúdo fica ao
lado e ocupa o resto.

Abaixo de 768px a barra lateral vira uma barra no topo, com borda embaixo: a
logo e o botão de tema numa linha, e as quatro seções em quatro abas de largura
igual, sem rolagem horizontal. As abas perdem o ícone e o complemento do rótulo
("do gate" fica só para leitor de tela), a aba ativa ganha a linha de 2px na
cor do acento, e o rodapé da instalação some.

**Página.** Margem de 16px nos lados, 24px no topo e 64px embaixo; a partir de
768px, 32px nos lados e no topo. O cabeçalho tem o título e, na mesma linha de
base, o contexto (período e hora da atualização) em texto secundário; quando
não cabem, o contexto desce. Abaixo do cabeçalho, 32px. O contexto não afirma
o que a tela não sabe: no carregando ele diz "Atualizando…", e no erro, "Última
atualização às HH:mm", a da última leitura que deu certo.

**Visão geral.** Uma coluna com 16px entre os blocos, nesta ordem: faixa de
métricas, fila de aprovação, painéis do período e a linha do último recibo. A
faixa tem 2 colunas abaixo de 1280px e 4 a partir daí. Os painéis são três:
"Bloqueadas" (resumo "N no período, por desfecho", com as linhas Aguardando,
Aprovadas e Alterações pedidas), "Falhas por tipo" e "Análises por dia e
nível", que é o mapa de calor e ocupa sozinho a linha de baixo. Ficam em 1 coluna no celular, 2 colunas entre 768px e 1280px (o mapa
ocupa as duas) e 3 colunas a partir de 1280px, na proporção 1 : 1 : 2.

**Tabela em tela estreita.** A fila perde colunas em vez de rolar: abaixo de
1600px some Commit e abaixo de 1440px some Branch, para a Evidência caber;
entre 768px e 1100px os motivos se empilham e o respiro lateral das células
cai para 8px; abaixo de 1100px a Evidência desce, inteira, para dentro da
célula do pull request, e abaixo de 768px os Motivos também. Quando a
Evidência é um caminho e não cabe, a pasta corta com reticências e o nome do
arquivo fica sempre inteiro ("migration…000031_parcelas.down.sql"); o caminho
completo fica no `title`. Na versão empilhada, o caminho quebra só depois das
barras. A rolagem horizontal existe só como reserva.

**Página não encontrada.** A mensagem e o link vêm primeiro, como texto normal
de até 65ch. A composição decorativa vem 32px abaixo e ocupa a largura inteira
do conteúdo, encolhendo em proporção.

## Elevation & Depth

Sem sombra, em nenhum lugar. A profundidade vem da troca entre base, superfície
e elevado e de uma borda de 1px. A escala de sombra do Tailwind foi zerada,
então não há classe de sombra disponível. A lista do select, que flutua sobre a
página, usa fundo elevado e borda de 1px, sem sombra. As sombras da referência
do Linear e do MONOLITH não foram adotadas.

O movimento é um só: o utilitário `transicao-de-estado`, de 120ms com a curva
`cubic-bezier(0.16, 1, 0.3, 1)`, aplicado a fundo, borda, cor, rotação e
opacidade. Com `prefers-reduced-motion: reduce` a duração é zero. Não há
animação de entrada nem mola. O esqueleto de carregamento é parado, e as
células do mapa de calor não se movem nem crescem no hover.

### Named Rules

**A regra da borda.** Superfícies se separam por borda de 1px e pela troca de
fundo. Sombra não separa nada aqui, e o fio é sempre de 1px, nunca de 0,5px.

## Shapes

Quatro raios, do menor para o maior:

- **2px**, sem token próprio: barra de proporção, células do
  mapa de calor e amostras da escala.
- **4px** (`selo`): selos e peças pequenas dentro de um controle: botão de
  limpar a busca, item do select, bloco de esqueleto.
- **6px** (`controle`): campo de busca, select e sua lista, botão de tema, item
  da navegação.
- **12px** (`painel`): os recipientes: faixa de métricas, fila, painéis, bloco
  de erro e de vazio, lista de esqueleto.

12px é o máximo, e nenhum controle tem formato de pílula. O ponto do selo é um
círculo de 6px, a única forma redonda.

Os ícones são da Lucide, em traço de 1,75: 16px ao lado de um rótulo, 14px
dentro do botão de limpar e na seta de ordenação. São sempre decorativos e vêm
junto de um texto ou de um rótulo acessível.

São dois os elementos decorativos permitidos em todo o sistema. O bloco do
recibo pode ter a borda inferior serrilhada, como um cupom; ainda não foi
construído. E a Página não encontrada tem uma composição tipográfica de canto
reto, descrita em Components.

### Named Rules

**A regra do caminho de leitura.** Em tela de trabalho nada se sobrepõe, nada
gira e o caminho de leitura nunca é ambíguo. Sobreposição e rotação existem só
na composição da Página não encontrada, que é decorativa e fica escondida de
leitor de tela.

## Components

Peças densas e quietas: 32px de altura para controle e 36px para linha, fundo de
superfície, borda de 1px, e estado mostrado pela troca para o fundo elevado.
Tudo fica em `src/componentes/`, salvo a barra de ferramentas da fila e a
composição da Página não encontrada.

### Selo de risco (`SeloDeRisco.tsx`)

- **Ponto:** um ponto de 6px na cor do nível e o nome do nível na mesma cor, em
  12px peso 510. A palavra "risco" vem antes do nome; pode ficar visível ou só
  para leitor de tela. É a variante em uso, no título da fila, onde leva um
  `title` que explica o que o risco alto faz: o Check Run bloqueia o merge até
  uma pessoa aprovar ou pedir alterações.
- **Tingido:** 20px de altura, raio de 4px, 6px de respiro lateral, fundo com
  13% da cor do nível, borda de 1px com 32%, ponto de 6px e texto na cor
  primária. Está pronto no componente, mas nenhuma tela construída o usa ainda.
- **Amostra do nível:** quadrado de 8px com raio de 2px na cor do nível. Está
  pronta no componente, mas ficou sem uso desde que o gráfico de barras saiu.

### Faixa de métricas (`FaixaDeMetricas.tsx`)

- **Forma:** uma lista de definições com borda de 1px e raio de 12px. As
  células se juntam por uma emenda de 1px na cor da borda (o utilitário
  `emenda`), sem espaço entre elas. São 2 colunas abaixo de 1280px e 4 acima.
- **Célula:** respiro de 12px por 16px (20px por 20px a partir de 1280px).
  Rótulo em 12px peso 510 na cor do texto primário; valor em 32px peso 590 tabular, na cor de
  título, com a unidade em 14px secundário na mesma linha de base; barra de
  proporção só quando o valor é uma porcentagem; linha de apoio em 12px
  secundário com a fração e o período.
- **As quatro células:** "Espera mais antiga" (o valor é a espera, e o apoio
  diz qual PR é e a posição dele na fila, sem repetir a contagem da fila),
  "Taxa de bloqueio" e "Falhas no sandbox" (porcentagens, com barra, e no apoio
  das falhas "cada tentativa conta", porque há mais execuções que análises) e
  "Análises recebidas" (com quantas passaram pelo gate e quantas estão em
  andamento ou pararam antes dele).

### Proporção (`Proporcao.tsx`)

- **Forma:** barra de 6px de altura e raio de 2px, na largura inteira do
  espaço.
- **Cor:** preenchimento neutro na cor do texto secundário sobre a trilha (a
  mesma cor a 28%). Não usa cor de risco nem o acento.
- **Papel:** decorativa e escondida de leitor de tela. A fração está sempre
  escrita ao lado.

### Tabela (`Tabela.tsx`)

- **Cabeçalho:** faixa de fundo elevado com borda de 1px embaixo; rótulos em
  12px peso 510 secundário.
- **Linha:** 36px (texto de 14/20 com 8px em cima e embaixo, 12px nos lados),
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
  de 4px, fundo elevado no hover), que devolve o foco ao campo. No foco, o anel
  de 2px na cor do acento.

### Seleção (`Selecao.tsx`, sobre Radix Select)

- **Gatilho:** igual ao campo de busca em altura, raio, fundo e contorno, com
  uma seta de 16px secundária à direita. Aberto, o fundo passa a elevado.
- **Lista:** abre colada ao gatilho, 4px abaixo, pelo menos da largura dele.
  Fundo elevado, borda de 1px, raio de 6px, 4px de respiro, sem sombra e sem
  animação de abertura.
- **Item:** 32px, raio de 4px. O item sob o cursor ou o teclado passa ao fundo
  de superfície; o escolhido tem uma marca de 16px à direita.

### Fila de aprovação e barra de ferramentas (`src/telas/FilaDeAprovacao.tsx`)

- **Forma:** bloco de superfície com borda de 1px e raio de 12px, com a tabela
  dentro.
- **Anatomia:** na mesma faixa do título da fila, à direita: campo de busca
  (224px), select de repositório (208px) e a contagem "N de M análises" em 12px
  secundário, anunciada por `role="status"`. Abaixo de 768px a busca ocupa a
  largura inteira e o select divide a linha com a contagem.
- **Busca:** procura em repositório, número do PR, branch, commit, id dos
  motivos e evidências. O texto é dividido por espaços e todos os termos
  precisam aparecer, então "falha sandbox" encontra `falha_sandbox` e
  "migrations" encontra a evidência.
- **Limpar filtros:** aparece só quando o filtro não devolve nenhuma linha, na
  frase que explica o vazio, com aparência de link: texto primário sublinhado.
- **Fila vazia:** se não há nada à espera, a tabela dá lugar a uma frase em
  texto secundário que diz quando um PR aparece ali.

### Painel (`Painel.tsx`)

- **Forma:** bloco de superfície com borda de 1px e raio de 12px. O cabeçalho é
  uma faixa de fundo elevado com borda embaixo: título em 14px peso 590, na cor
  de título, à esquerda, e resumo em 12px secundário à direita (o denominador,
  ou a escala do mapa de calor).
- **Tabela de contagem:** cada linha tem o rótulo, a barra de proporção
  ocupando o meio e o número à direita, com borda de 1px entre linhas.

### Mapa de calor (`MapaDeCalor.tsx`)

Segue a referência "Heat Map & Heatmap Style", da qual vieram só a matriz de
células, a escala visível, o valor por célula e a alternativa que não depende
de cor. O raio de 24px, as sombras, a escala no hover, a mola e as animações de
entrada da referência não são usados.

- **Matriz:** 3 linhas, uma por nível, na ordem alto, médio, baixo, e uma
  coluna por dia do período (30). Cada célula tem 24px de altura, pelo menos
  24px de largura e raio de 2px, com 2px entre células e entre linhas. O painel
  ocupa uma linha inteira, abaixo dos outros dois; quando as 30 colunas não
  cabem (abaixo de 900px de largura do mapa), o mapa rola na horizontal dentro
  do painel, abre já mostrando hoje e mantém a coluna do nível fixa à
  esquerda.
- **Intensidade:** a célula vazia tem o fundo elevado. As outras têm a cor de
  risco da linha misturada na superfície em três passos fixos pela contagem:
  76% para uma análise, 88% para duas e 100% para três ou mais. Passos fixos,
  e não relativos ao pico, porque com uma análise por dia na maior parte do
  período uma escala relativa prometia uma gradação que não existia. A partir
  de 768px, a célula com duas análises ou mais traz o número escrito, em 11px
  peso 590 na cor da superfície.
- **Linha:** o nome do nível à esquerda, em 12px secundário, e o total da linha
  à direita, em 12px primário. O matiz nunca é o único portador do nível.
- **Escala:** fica no cabeçalho do painel, no lugar do resumo: quatro amostras
  de 10px com borda de 1px, rotuladas "0", "1", "2" e "3+", e "por dia". A do
  zero tem a cor da célula vazia; as outras usam o texto secundário nos mesmos
  passos das células.
- **Eixo:** três marcas de data embaixo (primeiro dia, dia do meio e "hoje"),
  em 12px secundário.
- **Leitura:** a matriz é uma `<table role="grid">`, no padrão "Data Grid" do
  WAI-ARIA: o nível é um `<th scope="row">` que também diz o total, e cada dia
  é um `<td>` com rótulo ("9 nov, alto: 1 análise"). A tabela é o único ponto
  de tabulação; a célula ativa vem por `aria-activedescendant` e ganha
  contorno de 2px no acento. As setas andam entre dias e níveis, Home e End vão
  ao primeiro e ao último dia, Escape fecha a dica.
- **Destino:** a célula leva às análises que ela conta. Com uma análise, a
  célula inteira é um link para `/analises/<id>`; com duas ou mais, abre uma
  lista (Radix Popover, `ListaDaCelula.tsx`) com cada análise, repositório,
  número do PR e hora de chegada, cada linha um link de 32px; a vazia não é
  clicável. Pelo teclado, Enter ou espaço na célula ativa faz o mesmo, e ao
  fechar a lista o foco volta à grade. Os links e botões dentro das células têm
  `tabindex="-1"`, para a grade continuar com um único ponto de tabulação. No
  hover, a célula clicável ganha contorno de 1px na cor do título. No foco ou no ponteiro aparece uma dica
  (`role="tooltip"`, ligada por `aria-describedby`) com o mesmo texto, em 12px
  sobre fundo elevado com borda de 1px; na linha de cima ela abre para baixo,
  para não cobrir a escala, e nas pontas se alinha à borda da grade. O
  `biome.json` desliga três regras de a11y só para `MapaDeCalor.tsx`, porque
  elas não reconhecem esse padrão.

### Link do motivo (`LinkDoMotivo.tsx`)

- O id do motivo em Plex Mono, como link para `/regras#id`: texto primário
  sublinhado, como todo link. Vários motivos ficam lado a lado com 12px entre
  eles.

### Navegação e casca (`Casca.tsx`)

- **Item:** 32px, raio de 6px, ícone de 16px e rótulo em 14px secundário, sem
  sublinhado. No hover, fundo elevado e texto primário. O item ativo tem fundo
  elevado, texto na cor de título, peso 510 e o ícone na cor do acento. No
  celular, sem ícone, a aba ativa tem uma linha de 2px na cor do acento
  embaixo. O anel de foco do item fica para dentro.
- **Botão de tema:** 32px, raio de 6px, fundo de superfície, borda de 1px na
  cor da borda padrão, ícone de sol ou lua e o rótulo "Tema claro" ou "Tema
  escuro" (o tema para o qual ele leva). No hover, fundo elevado. A escolha
  fica guardada no navegador e é aplicada antes da primeira pintura.
- **Rodapé da instalação:** conta em 14px primário, número de repositórios e o
  aviso de protótipo em 12px secundário.
- **Pular para o conteúdo:** primeiro elemento focável, visível só com foco.

### Último recibo (`UltimoRecibo.tsx`)

Uma linha discreta no fim da Visão geral, para o diferencial do aPRova aparecer
na tela de entrada sem disputar com a fila.

- **Forma:** linha com borda de 1px em cima e 12px de respiro, sem fundo
  próprio, em 12px secundário.
- **Conteúdo:** "Último recibo assinado" em peso 510 primário, o repositório e
  o PR, o nível em selo de ponto, os 16 primeiros caracteres do hash em Plex
  Mono, "Ed25519, encadeado a" com os 8 primeiros do hash anterior, a data e
  hora da assinatura e o link "Ver os recibos", alinhado à direita. Os hashes
  completos ficam no `title`.

### Página e estados (`Pagina.tsx`, `Esqueleto.tsx`)

- **Carregando:** blocos de esqueleto com fundo elevado, parados, com a forma
  e a altura do que vai chegar, para a tela não pular. A faixa é um bloco só,
  com emenda, e quatro células com rótulo, valor, barra e duas linhas de apoio.
  A fila tem a faixa da barra de ferramentas, a do cabeçalho e cinco linhas de
  36px. Os painéis têm a altura dos reais e raio de 12px. Um texto para leitor
  de tela diz o que está carregando.
- **Erro:** bloco de superfície com borda de 1px, raio de 12px e 24px de
  respiro, até 65ch: título de 16px peso 590, explicação em texto secundário e
  o link "Tentar de novo". Não usa vermelho. A explicação diz só o que é
  verdade sem resposta do servidor: as decisões e os recibos ficam gravados no
  servidor, e o bloqueio no GitHub não depende desta tela.
- **Vazio:** o mesmo bloco, com título, texto e uma ação opcional.

### Composição da Página não encontrada (`src/telas/NaoEncontrada.tsx`)

O segundo elemento decorativo permitido no sistema, e só nesta página. Segue a
referência "Deconstructivism (Graphic)", sem o vermelho e o azul dela e sem o
lima.

- **Papel:** um único SVG embutido, com `aria-hidden` e sem seleção de texto. A
  mensagem legível e o link para a Visão geral ficam acima dele, como texto
  normal.
- **Forma:** uma faixa entre dois fios de 1px na cor de título, com canto reto.
  Dentro, os algarismos de "404" em Inter peso 800, fragmentados, girados e
  sobrepostos: um "404" grande ao fundo na cor do elevado, dois "4" na cor de
  título e um zero só em contorno, na cor secundária. Dois fios de 1px se
  cruzam por cima.
- **Texto:** o caminho pedido aparece duas vezes em Plex Mono, uma numa etiqueta
  na cor de título com o texto na cor da base e outra riscada, na cor
  secundária. Três rótulos pequenos em maiúsculas (12px, peso 590) completam a
  composição. Rótulo em maiúsculas existe só aqui, como parte do desenho; não é
  um estilo de rótulo da interface.
- **Cor:** só os tokens de título, secundário, elevado e base. Nenhuma cor de
  risco e nenhum acento.

### Componentes e origem

A fonte desta tabela é `LICENSES-TERCEIROS.md`, que também guarda o aviso de
copyright e o texto da licença. Oito peças derivam da Arc Library
(https://github.com/kuratlielia/arc-library, licença MIT, só a parte gratuita),
lida pelo MCP `ddl`.

| Componente                                    | Origem                         | Licença | O que foi alterado                                                                                                                         |
| --------------------------------------------- | ------------------------------ | ------- | ------------------------------------------------------------------------------------------------------------------------------------------ |
| `SeloDeRisco.tsx`                             | Arc, `badge`                   | MIT     | Cores trocadas pelas três de risco; texto primário sobre a tinta; raio de 4px; sem troca animada de texto nem largura em mola              |
| `Tabela.tsx`                                  | Arc, `sortable-data-table`     | MIT     | Linhas de 36px em vez de 52px; sem seleção de linhas, sem faixa de coluna e sem linhas deslizando ao ordenar; seta com transição de 120ms  |
| `FaixaDeMetricas.tsx`                         | Arc, `stats-band` (`divided`)  | MIT     | Sem contagem animada, sem entrada escalonada e sem minigráficos; número em 32px; barra de proporção no lugar do minigráfico                |
| `Proporcao.tsx`                               | Arc, `usage-meter`             | MIT     | Um segmento só, em cor neutra; 6px de altura e raio de 2px; sem legenda interativa nem números rolando                                     |
| `CampoDeBusca.tsx`                            | Arc, `search-field`            | MIT     | Altura de 32px; contorno na cor do texto secundário; sem animação de entrada do botão                                                      |
| `Selecao.tsx`                                 | Arc, `select`                  | MIT     | Sem valor rolando e sem animação de abertura; sem sombra; raio de 6px; fundo elevado com borda de 1px                                      |
| Barra de ferramentas em `FilaDeAprovacao.tsx` | Arc, `filter-toolbar`          | MIT     | Sem chips e sem o menu que cresce a partir do botão; os filtros são um campo de busca e um select                                          |
| `MapaDeCalor.tsx`                             | Arc, `bar-chart`               | MIT     | Do Arc ficou só a `figure` com tabela alternativa para leitor de tela. Virou mapa de calor de dias por nível, em CSS puro; sem barras, sem cursor de leitura, sem média e sem animação |
| `Casca.tsx`                                   | ARK CN, só como referência     | nenhuma | Nenhum código copiado. Serviu de referência para a estrutura da barra lateral (cabeçalho, conteúdo que rola e rodapé), escrita do zero     |

`BarrasPorDia.tsx` não existe mais. `Painel.tsx`, `Pagina.tsx`,
`Esqueleto.tsx`, `LinkDoMotivo.tsx`, `Identificador.tsx` e a composição de
`NaoEncontrada.tsx` foram escritos do zero. Do Amicro, nada foi adotado até
aqui. A ARK CN não tem arquivo `LICENSE` e é construída sobre Ark UI, não sobre
Radix; por isso serve de referência de estrutura e não de fonte de código.

#### Referências de estilo

Quatro documentos de estilo informaram o sistema. São descrições de estilo, não
código, então nenhum texto de licença se aplica. Os três últimos vêm da
biblioteca do designmd.app.

| Referência                   | Endereço                                               | O que veio                                                                                                                  | O que não veio                                                                                   |
| ---------------------------- | ------------------------------------------------------ | --------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------ |
| Linear, referência de estilo | sem endereço registrado                                | A linguagem de base: paleta escura, título separado do corpo, Inter com seus recursos e pesos 510 e 590, raio de 12px       | Fio de 0,5px, sombras, pílulas, tamanhos de 13, 15 e 17px, escala de marketing, Berkeley Mono    |
| MONOLITH, system status      | https://designmd.app.br/library/monolith-status        | Só a paleta de cinzas do tema claro                                                                                         | Blur, vidro, neumorfismo, gradiente, 3D e sombra                                                 |
| Heat Map & Heatmap Style     | https://designmd.app.br/library/heat-map-heatmap-style | Matriz de células, escala visível, valor por célula e alternativa que não depende de cor                                    | Raio de 24px, sombras, escala no hover, mola e animações de entrada                              |
| Deconstructivism (Graphic)   | https://designmd.app.br/library/deconstructivism       | Fragmentação, rotação e sobreposição de tipos, fios cruzados e canto reto, só na composição da Página não encontrada        | O vermelho e o azul da referência; qualquer uso em tela de trabalho                              |

Regras que valem para toda peça nova:

- Uma só base de primitivos: Radix.
- As fontes de código permitidas são shadcn/ui, Arc (parte gratuita), Amicro e
  ARK CN, e nenhuma outra.
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
- **Do** usar o lima só no anel de foco, no item ativo da navegação e na
  seleção de texto.
- **Do** escrever link como texto primário com sublinhado de 1px, que engrossa
  para 2px no hover.
- **Do** pôr títulos e o valor da métrica na cor de título, e o corpo na cor
  primária.
- **Do** escrever em Plex Mono todo hash, `head_sha`, `key_id`, id de motivo,
  nome de branch, número de PR e caminho de arquivo.
- **Do** escrever o nome do nível junto da cor de risco.
- **Do** escrever o nível uma vez no título quando a lista inteira tem o mesmo
  nível, em vez de repeti-lo numa coluna.
- **Do** dar unidade, período e denominador a todo número, e escrever a fração
  ao lado de toda barra de proporção.
- **Do** acompanhar toda célula de mapa de calor do rótulo da linha, da escala
  e do valor por célula.
- **Do** manter controle em 32px e linha de tabela em 36px de altura.
- **Do** usar raio de 12px nos recipientes, 6px nos controles, 4px nos selos e
  2px em barras e células.
- **Do** limitar o movimento à transição curta de estado (120ms) e zerá-la com
  `prefers-reduced-motion: reduce`.
- **Do** registrar em `LICENSES-TERCEIROS.md` toda peça derivada de outra
  biblioteca.
- **Do** usar a logo de `design/marca/svg/` no topo da navegação, na versão do
  tema, e `icone-app-fundo-escuro-32.png` como favicon.

### Don't:

- **Don't** usar o lima em link, destaque ou gráfico, nem trazer o azul de
  volta.
- **Don't** dar cor própria a link.
- **Don't** usar cor de risco fora de risco: nem como sucesso genérico, nem
  como erro de formulário, nem como decoração.
- **Don't** usar roxo, gradiente, glassmorphism, neumorfismo, 3D ou fundo com
  blur.
- **Don't** usar sombra, de nenhum tamanho.
- **Don't** usar fio de 0,5px.
- **Don't** usar peso 700 ou mais na interface.
- **Don't** usar Roboto ou fonte do sistema no lugar da Inter, nem outra mono no
  lugar da Plex Mono.
- **Don't** pôr card dentro de card.
- **Don't** pôr um ícone num quadradinho arredondado em cima de cada título.
- **Don't** escrever texto cinza sobre fundo colorido.
- **Don't** usar emoji na interface.
- **Don't** usar botão em pílula nem cantos acima de 12px.
- **Don't** deixar a cor de uma célula de mapa de calor ser a única fonte do
  número.
- **Don't** sobrepor, girar ou fragmentar nada fora da composição da Página não
  encontrada.
- **Don't** trocar informação por frase de efeito.
- **Don't** animar a entrada de página, nem usar mola.
- **Don't** manter cor, raio, sombra, fonte ou animação de uma biblioteca de
  origem.
- **Don't** alterar `design/marca/`.
