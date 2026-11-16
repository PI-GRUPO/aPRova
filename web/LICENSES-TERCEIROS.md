# Licenças de terceiros

Este protótipo tem componentes derivados de bibliotecas de código aberto. O
código não foi instalado como dependência: foi lido, reescrito em Tailwind com
os tokens do `DESIGN.md` e guardado em `src/componentes/`. Como é obra
derivada, o aviso de copyright e o texto da licença de cada origem ficam aqui.

As dependências instaladas pelo npm (React, Radix, Tailwind e as demais do
`package.json`) trazem a própria licença dentro de `node_modules/` e não entram
nesta lista.

## Arc Library

- Site: https://uiarc.dev
- Repositório: https://github.com/kuratlielia/arc-library
- Licença: MIT, conferida no arquivo `LICENSE` do repositório
- Parte usada: só a gratuita, que está no repositório. Nada do Arc Pro.
- Como o código foi lido: pelo MCP `ddl`, biblioteca `arc-library`

### Componentes derivados

| Arquivo no protótipo | Origem no Arc (id no ddl) | O que veio de lá | O que foi alterado |
|---|---|---|---|
| `src/componentes/SeloDeRisco.tsx` | `arc-library/badge` | Fundo tingido pela cor do tom, ponto de estado antes do texto, altura fixa | Cores trocadas pelas três de risco; texto na cor primária sobre a tinta; raio de 4px; sem troca animada de texto nem largura em mola |
| `src/componentes/Tabela.tsx` | `arc-library/sortable-data-table` | Cabeçalho ordenável como botão dentro do `th`, `aria-sort`, seta que vira com o sentido, seta antes do rótulo em coluna alinhada à direita | Linhas de 32px em vez de 52px; sem seleção de linhas, sem faixa de coluna e sem linhas deslizando ao ordenar; seta com transição de 120ms |
| `src/componentes/FaixaDeMetricas.tsx` | `arc-library/stats-band` (layout `divided`) | Faixa de números em `dl`, células separadas por fio de 1px, número grande tabular com a unidade menor na mesma linha de base | Sem contagem animada, sem entrada escalonada e sem os minigráficos; número em 32px; barra de proporção no lugar do minigráfico |
| `src/componentes/Proporcao.tsx` | `arc-library/usage-meter` | Trilha com preenchimento proporcional | Um segmento só, em cor neutra; 6px de altura e raio de 2px; sem legenda interativa nem números rolando |
| `src/componentes/CampoDeBusca.tsx` | `arc-library/search-field` | Lupa que escurece quando há texto, botão de limpar em espaço reservado, foco de volta ao campo depois de limpar | Altura de 32px; contorno na cor do texto secundário; sem animação de entrada do botão |
| `src/componentes/Selecao.tsx` | `arc-library/select` | Estrutura sobre `@radix-ui/react-select`: gatilho, lista em `popper`, marca no item escolhido | Sem valor rolando e sem animação de abertura; sem sombra; raio de 6px; fundo elevado com borda de 1px |
| `src/telas/FilaDeAprovacao.tsx` (barra de ferramentas) | `arc-library/filter-toolbar` | Filtros junto da coleção, contagem anunciada por `role="status"`, ação de limpar os filtros | Sem chips e sem o menu que cresce a partir do botão; filtros são um campo de busca e um select |
| `src/componentes/MapaDeCalor.tsx` | `arc-library/bar-chart` | `figure` com tabela alternativa para leitor de tela | Virou mapa de calor de dias por nível de risco, em CSS puro; sem barras, sem cursor de leitura, sem média e sem animação |

### Texto da licença

```
MIT License

Copyright (c) 2026 Elia Kuratli

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
```

## Amicro

Nenhum componente do Amicro entrou no protótipo até aqui. Se o retorno de
"copiado" das telas de Detalhe e de Recibos for inspirado em
`amicro/button-copy-hash`, o item entra nesta seção com o texto da licença MIT
do repositório https://github.com/Subhan-code/Amicro--Micro-transitions-
(Copyright (c) 2026 Syed Subhan Uddin).

## ARK CN

Nenhum código da ARK CN foi copiado: o repositório não tem arquivo `LICENSE` e
a base dela é Ark UI, não Radix. Ela serviu só de referência de padrão para a
estrutura da barra lateral (cabeçalho, conteúdo que rola e rodapé numa coluna
da altura da janela), escrita do zero em `src/componentes/Casca.tsx`.

## Referências de estilo

Quatro documentos de estilo orientaram o sistema visual. São descrições de
estilo, não código: nada foi copiado deles e nenhum texto de licença se aplica.
O que cada um trouxe está no `DESIGN.md`.

- Linear, style reference ("midnight precision instrument"): linguagem base.
- MONOLITH, system status: cinzas do tema claro.
  https://designmd.app.br/library/monolith-status
- Heat Map & Heatmap Style: mapa de calor da Visão geral.
  https://designmd.app.br/library/heat-map-heatmap-style
- Deconstructivism (Graphic): composição da página não encontrada.
  https://designmd.app.br/library/deconstructivism
