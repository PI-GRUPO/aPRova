# Product

<!-- impeccable:product-schema 1 -->

## Platform

web

## Stack

Definida por quem pediu o protótipo, igual à que o front-end real vai usar,
para o protótipo poder ser aproveitado: React, Vite e TypeScript; Tailwind;
React Router; TanStack Table; Recharts; Lucide; date-fns com locale pt-BR;
Biome; Playwright. Fontes via `@fontsource`, sem CDN.

Componentes: a base de primitivas é uma só, Radix. As fontes de componentes
permitidas são shadcn/ui, Arc (parte gratuita), Amicro e ARK CN, e nenhuma
outra. Tudo entra como código-fonte reescrito com os tokens do DESIGN.md; a
origem e a licença de cada peça ficam em `LICENSES-TERCEIROS.md`. A ARK CN vale
só como referência de padrão, porque é feita sobre Ark UI e não tem arquivo de
licença.

## Users

**Usuário principal:** tech lead ou revisor de um time que usa agentes de
código. Abre o dashboard para saber o que está bloqueado e por quê, para
aprovar ou reexecutar uma análise e para provar a terceiros o que foi decidido.

**Segunda audiência:** a banca da apresentação acadêmica, que vê o mesmo
dashboard, sem versão própria.

## Product Purpose

O aPRova é um GitHub App que audita pull requests gerados por agentes de IA.
Para cada PR ele roda os testes num sandbox, classifica o risco com regras
determinísticas, emite um recibo assinado com Ed25519 e, quando o risco é alto,
bloqueia o merge pelo Check Run até uma pessoa decidir.

O dashboard é onde essa pessoa enxerga as análises, os motivos de cada decisão
e os recibos. Dá certo quando ela descobre em poucos segundos o que espera por
ela e consegue mostrar a alguém de fora, com o recibo, o que foi decidido.

## Positioning

A decisão de cada análise sai com um recibo que qualquer pessoa verifica só com
a chave pública, sem acesso ao banco e sem confiar no servidor do aPRova. O
nível de risco vem de regras determinísticas com motivos nomeados e evidências,
não da opinião de um modelo.

## Operating Context

- O trabalho começa no GitHub: o PR, o Check Run e o review ficam lá. O
  dashboard acompanha e explica; não substitui o GitHub.
- Cada análise (`task_run`) percorre estados fixos: `pendente`,
  `mapeando_contexto`, `testando_sandbox`, `avaliando_risco`, depois
  `aguardando_aprovacao` ou `aplicando`, até `concluido`, `rejeitado` ou
  `falhou`.
- O gate tem sete regras: `falha_sandbox`, `muitos_arquivos`,
  `configuracao_ci`, `dependencia_nova`, `segredo_versionado` e `area_sensivel`
  (alto) e `codigo_sem_teste` (médio). O nível final é o maior entre os motivos.
- Os recibos de uma instalação formam uma cadeia: cada um cita o id e o hash do
  anterior.
- A verificação independente de um recibo é feita no terminal, com `openssl` ou
  com o executável `verificar-recibo`, como em `docs/verificacao-recibo.md`.

## Capabilities and Constraints

**Este projeto é um protótipo navegável**, feito para validar a interface antes
da fundação do front-end real.

- Cinco telas com dados fictícios: Visão geral, Análises, Detalhe da análise,
  Recibos (lista e detalhe) e Regras do gate.
- Sem backend, sem login e sem chamada a API. Os dados vêm de arquivos JSON
  locais, no formato que a API real deve devolver, com os mesmos nomes de
  estado, de motivo e de campo do banco.
- Aprovar e Reexecutar aparecem desabilitados, com a explicação de que só
  funcionam no produto real.
- Toda tela tem quatro estados (com dados, vazio, carregando e erro),
  alternáveis por `?estado=` na URL.
- Tema escuro por padrão; o claro existe e é alternável.
- Interface em português do Brasil.
- Desktop primeiro. No celular as telas precisam funcionar, mas não são o foco.
- O protótipo vive isolado em `design/prototipo/` e não se mistura com o código
  do aPRova.

**Em aberto:** se o dashboard mostra uma instalação do GitHub App por vez ou
várias, com seletor. O protótipo é construído com uma instalação fixa, sem
seletor.

## Brand Commitments

- Nome: aPRova. Logo e ícone em `design/marca/`, que não é alterada.
- Tom: ferramenta técnica e sóbria, mais perto de um terminal ou de um painel
  de observabilidade do que de uma landing page de startup.
- O aPRova nunca faz merge sozinho. Nenhum texto pode sugerir o contrário.
- O recibo é **assinado e verificável**. Nunca "criptografado": o conteúdo é
  público, só não pode ser forjado.
- As regras do gate não usam modelo de IA. A interface não as apresenta como
  "inteligentes".
- É um Projeto Integrador do UNIPAM. A interface não sugere produto comercial
  em operação: sem preços, planos ou clientes.
- A identidade visual (cores, tipografia, forma e movimento) já foi definida
  por quem pediu o protótipo e está registrada no DESIGN.md.

## Evidence on Hand

- Caso real para os dados fictícios: PR #1 de `Azulpasta/aPRova-TESTES`, branch
  `teste-peso-zero`, risco alto por `falha_sandbox` (1 de 11 testes falhou),
  aguardando aprovação.
- Passo a passo real de verificação de recibo em
  `docs/verificacao-recibo.md`.
- Nomes de estado, de motivo e de campo nas migrations e em `internal/` do
  repositório.
- Não existem clientes, depoimentos, métricas de uso nem benchmarks. Os números
  do protótipo são fictícios e não podem ser apresentados como reais.

## Product Principles

1. **Primeiro o que espera por uma pessoa.** O que está bloqueado, há quanto
   tempo e por quê vem antes de qualquer estatística.
2. **Todo número com contexto.** Unidade e período sempre presentes; nada de
   métrica solta.
3. **A decisão se explica sozinha.** Cada nível de risco aparece junto dos
   motivos e das evidências que o produziram.
4. **A prova sai do dashboard.** O recibo é mostrado de modo que alguém de fora
   consiga verificá-lo sem o aPRova.
5. **Nada finge funcionar.** O que o protótipo não faz está desabilitado e diz
   por quê.

## Accessibility & Inclusion

WCAG 2.2 nível AA. Navegação completa só com teclado, com foco sempre visível.
O nível de risco nunca é comunicado só pela cor: vem sempre com o nome do nível
em texto.
