const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');

const raiz = path.resolve(__dirname, '..');
const workflow = JSON.parse(fs.readFileSync(path.join(raiz, 'trello-sugestao-movimentacao.json')));
const core = JSON.parse(fs.readFileSync(path.join(raiz, 'entrada-eventos.json')));
const no = prefixo => workflow.nodes.find(n => n.name.startsWith(prefixo));
const codigo = prefixo => no(prefixo).parameters.jsCode;
const rodar = (prefixo, json, origem = json) => new Function('$json', '$', codigo(prefixo))(
  json, () => ({ first: () => ({ json: origem }), item: { json: origem } }),
).json;
const rodarPaginas = (prefixo, paginas, origem) => new Function('$input', '$', codigo(prefixo))(
  { all: () => paginas.map(json => ({ json })) }, () => ({ first: () => ({ json: origem }) }),
)[0].json;
const ids = new Map();
let sequencia = 0;
const configurado = codigo('02 |').replace(/CONFIGURE_ID_[A-Z_]+/g, marcador => {
  if (!ids.has(marcador)) ids.set(marcador, (++sequencia).toString(16).padStart(24, '0'));
  return ids.get(marcador);
});
const normalizar = new Function('$json', '$execution', core.nodes.find(n => n.name.startsWith('04 |')).parameters.jsCode);
function evento() {
  const original = JSON.parse(fs.readFileSync(path.join(raiz, '..', 'exemplos', 'analise_concluida.json')));
  return normalizar({ original, meta: { origem_verificada: true } }, { id: 'regressao' }).json;
}
const preparar = (entrada = evento(), code = configurado) => new Function('$input', code)(
  { all: () => [{ json: entrada }] },
)[0].json;
const resposta = body => ({ statusCode: 200, body });
function card(contexto, alteracoes = {}) {
  return {
    id: 'card-de-teste', idBoard: contexto.trello.quadro_id,
    idList: contexto.trello.configuracao.listas.backlog.id, closed: false,
    name: 'Tarefa de teste', url: 'https://trello.com/c/exemplo',
    desc: 'PR vinculada: [ver PR](' + contexto.trello.url_pr + ')', ...alteracoes,
  };
}
const vincular = (base = preparar()) => rodarPaginas('05 |', [resposta({ cards: [card(base)] })], base);
const revalidar = (base = vincular()) => rodar('08 |', resposta(base.trello.card), base);
const comentar = (base = revalidar()) => rodarPaginas('11 |', [resposta([])], base);
function confirmar(base = comentar()) {
  return rodar('14 |', {
    id: 'comentario-de-teste', type: 'commentCard', date: '2026-11-02T12:00:00Z',
    data: { card: { id: base.trello.card.id }, text: base.trello.comentario },
  }, base);
}

test('Trello: export bloqueia configuração pendente e aceita IDs sintéticos apenas no teste', () => {
  assert.throws(() => preparar(evento(), codigo('02 |')), /Configure o ID/);
  assert.equal(preparar().trello.status, 'pesquisar');
});

test('Trello: exige um evento por execução e ignora outros tipos', () => {
  for (const entradas of [[], [{ json: evento() }, { json: evento() }]]) {
    assert.throws(() => new Function('$input', configurado)({ all: () => entradas }), /um evento por execução/);
  }
  const outro = evento(); outro.evento.tipo = 'aguardando_aprovacao';
  assert.equal(preparar(outro, codigo('02 |')).trello.status, 'evento_ignorado');
});

test('Trello: origem não verificada, tenant, UUID e risco inválidos não chegam à pesquisa', () => {
  for (const alterar of [
    e => { e.meta.origem_verificada = false; }, e => { e.meta.tenant_id = 0; },
    e => { e.meta.id_evento = 'invalido'; }, e => { e.risco.nivel = 'inventado'; },
  ]) {
    const e = evento(); alterar(e);
    assert.throws(() => preparar(e));
  }
});

test('Trello: recusa divergência entre URL, PR e repositório', () => {
  for (const alterar of [
    e => { e.pr.numero = 140; }, e => { e.repositorio.full_name = 'outra/equipe'; },
    e => { e.pr.url = 'https://github.com.exemplo/owner/repo/pull/14'; },
  ]) {
    const e = evento(); alterar(e);
    assert.throws(() => preparar(e), /inconsistentes/);
  }
});

test('Trello: link Markdown identifica um único card e preserva o risco do núcleo', () => {
  const base = preparar();
  const resultado = vincular(base);
  assert.equal(resultado.trello.status, 'card_encontrado');
  assert.equal(resultado.trello.correspondencias, 1);
  assert.equal(resultado.risco, base.risco);
});

test('Trello: não confunde PR 14 com 140 nem usa card arquivado ou de outro quadro', () => {
  const base = preparar();
  for (const alteracoes of [
    { desc: base.trello.url_pr + '0' }, { desc: 'Sem link' }, { closed: true }, { idBoard: 'outro' },
  ]) {
    const resultado = rodarPaginas('05 |', [resposta({ cards: [card(base, alteracoes)] })], base);
    assert.equal(resultado.trello.status, 'sem_card_vinculado');
    assert.equal(resultado.trello.card, null);
  }
});

test('Trello: ambiguidade bloqueia escrita; repetição do mesmo ID na busca não cria ambiguidade', () => {
  const base = preparar(); const primeiro = card(base);
  assert.equal(rodarPaginas('05 |', [resposta({ cards: [primeiro, card(base, { id: 'outro' })] })], base)
    .trello.status, 'vinculo_ambiguo');
  assert.equal(rodarPaginas('05 |', [resposta({ cards: [primeiro, primeiro] })], base)
    .trello.status, 'card_encontrado');
});

test('Trello: falha ou busca incompleta não significa ausência de vínculo', () => {
  const base = preparar();
  for (const paginas of [[], [{ statusCode: 503 }], [resposta({ cards: null })],
    [resposta({ cards: Array.from({ length: 100 }, () => card(base)) })]]) {
    assert.throws(() => rodarPaginas('05 |', paginas, base));
  }
});

test('Trello: relê o card e bloqueia vínculo removido, arquivamento ou mudança de quadro', () => {
  const base = vincular();
  assert.equal(revalidar(base).trello.status, 'vinculo_confirmado');
  for (const alteracoes of [{ desc: '' }, { closed: true }, { idBoard: 'outro' }, { id: 'outro' }]) {
    assert.equal(rodar('08 |', resposta({ ...base.trello.card, ...alteracoes }), base)
      .trello.status, 'vinculo_alterado');
  }
  assert.throws(() => rodar('08 |', { statusCode: 503 }, base));
});

test('Trello: comentário prévio em página posterior impede nova publicação', () => {
  const base = revalidar();
  const recentes = Array.from({ length: 1000 }, (_, i) => ({
    id: 'acao-' + i, type: 'commentCard', data: { text: 'Comentário anterior' },
  }));
  const antigo = { id: 'ja-publicado', type: 'commentCard', data: { text: base.trello.marcador } };
  const resultado = rodarPaginas('11 |', [resposta(recentes), resposta([antigo])], base);
  assert.equal(resultado.trello.status, 'evento_ja_comentado');
  assert.equal(resultado.trello.comentario_id, antigo.id);
});

test('Trello: histórico inválido ou incompleto impede comentar', () => {
  const base = revalidar();
  for (const paginas of [[], [{ statusCode: 429 }], [resposta({})], [resposta([{}])],
    [resposta(Array.from({ length: 1000 }, () => ({ type: 'commentCard', data: { text: 'anterior' } })))]] ) {
    assert.throws(() => rodarPaginas('11 |', paginas, base));
  }
});

test('Trello: sugestão muda conforme a etapa e mantém risco e decisão humana', () => {
  for (const etapa of ['backlog', 'revisao', 'teste', 'concluido']) {
    const base = revalidar();
    base.trello.card.idList = base.trello.configuracao.listas[etapa].id;
    const resultado = comentar(base);
    assert.equal(resultado.risco, base.risco);
    assert.equal(resultado.trello.card.idList, base.trello.card.idList);
    assert.ok(resultado.trello.comentario.includes(base.trello.url_pr));
    assert.ok(resultado.trello.comentario.includes(base.trello.marcador));
    assert.match(resultado.trello.comentario, /não comprova aprovação, merge ou conclusão/);
    assert.equal(resultado.trello.status, 'pronto_para_comentar');
  }
});

test('Trello: só confirma comentário com ID, tipo, card e texto esperados', () => {
  const base = comentar();
  assert.equal(confirmar(base).trello.status, 'comentario_adicionado');
  for (const raw of [{}, { id: 'id', type: 'updateCard' },
    { id: 'id', type: 'commentCard', data: { card: { id: base.trello.card.id }, text: 'Outro texto' } }]) {
    assert.throws(() => rodar('14 |', raw, base), /não confirmou/);
  }
});

test('Trello: Slack exige comentário confirmado e preserva o texto em blocos simples', () => {
  assert.throws(() => rodar('15 |', comentar()), /comentários confirmados/);
  const base = confirmar();
  base.trello.comentario += '\n' + '😀'.repeat(3000) + '\n<@U123> não interpretar';
  const resultado = rodar('15 |', base);
  const blocos = resultado.slack_payload.blocks.slice(2);
  assert.equal(blocos.map(b => b.text.text).join(''), base.trello.comentario);
  assert.ok(blocos.every(b => b.text.type === 'plain_text' && Array.from(b.text.text).length <= 2800));
  assert.equal(resultado.slack_payload.mrkdwn, false);
});

test('Trello: resultado Slack aceita body/data e conserva comentário quando o envio falha', () => {
  const base = rodar('15 |', confirmar());
  for (const campo of ['body', 'data']) {
    const resultado = rodar('17 |', { statusCode: 200, [campo]: 'ok' }, base);
    assert.equal(resultado.notificacao_slack.status, 'enviado');
    assert.equal(resultado.trello, base.trello);
  }
  for (const raw of [{ statusCode: 500, body: 'erro' }, { error: 'ECONNREFUSED' },
    { statusCode: 200, body: 'not_ok' }]) {
    const resultado = rodar('17 |', raw, base);
    assert.equal(resultado.notificacao_slack.status, 'falha');
    assert.equal(resultado.trello.status, 'comentario_adicionado');
    assert.equal(resultado.trello.comentario_id, base.trello.comentario_id);
  }
});

test('Trello: paginação de comentários avança pelo cursor before da API, sem repetir a primeira página', () => {
  const paginacao = no('10 |').parameters.options.pagination.pagination;
  const parametro = paginacao.parameters.parameters.find(p => p.name === 'before');
  assert.ok(parametro, 'A API de actions pagina com before, não com page.');
  assert.ok(!paginacao.parameters.parameters.some(p => p.name === 'page'));
  const avaliar = (texto, pageCount, body) => new Function('$pageCount', '$response',
    'return (' + texto.slice(3, -2).trim() + ');')(pageCount, { statusCode: 200, body });
  assert.equal(avaliar(parametro.value, 0), undefined);
  assert.equal(avaliar(parametro.value, 1, [{ id: 'mais-novo' }, { id: 'mais-antigo' }]), 'mais-antigo');
  assert.equal(avaliar(paginacao.completeExpression, 0, Array(1000).fill({ id: 'acao' })), false);
  assert.equal(avaliar(paginacao.completeExpression, 1, []), true);
});

test('Trello: rota do Core mantém execução por evento e workflow não move cards', () => {
  const nome = '09 | Executar Sugestão no Trello';
  const chamada = core.nodes.find(n => n.name === nome);
  assert.equal(chamada.parameters.mode, 'each');
  assert.equal(chamada.parameters.options.waitForSubWorkflow, false);
  assert.ok(core.connections['07G | analise_concluida | Conectar workflow'].main[0].some(c => c.node === nome));
  const escritas = workflow.nodes.filter(n => n.type === 'n8n-nodes-base.trello');
  assert.equal(escritas.length, 1);
  assert.equal(escritas[0].parameters.resource, 'cardComment');
  assert.ok(!escritas[0].parameters.operation || escritas[0].parameters.operation === 'create');
});

test('exports: grafos consistentes, inativos e sem credenciais, dados fixados ou webhook Slack real', () => {
  for (const arquivo of fs.readdirSync(raiz).filter(f => f.endsWith('.json'))) {
    const w = JSON.parse(fs.readFileSync(path.join(raiz, arquivo)));
    assert.equal(w.active, false);
    assert.deepEqual(w.pinData, {});
    assert.ok(w.nodes.every(n => !n.credentials));
    assert.doesNotMatch(JSON.stringify(w), /https:\/\/hooks\.slack\.com\/services\//);
    const nomes = new Set(w.nodes.map(n => n.name));
    assert.equal(nomes.size, w.nodes.length);
    for (const [origem, tipos] of Object.entries(w.connections)) {
      assert.ok(nomes.has(origem));
      for (const saidas of Object.values(tipos)) for (const arestas of saidas) {
        for (const aresta of arestas) assert.ok(nomes.has(aresta.node));
      }
    }
  }
});
