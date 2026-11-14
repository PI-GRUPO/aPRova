const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const crypto = require('node:crypto');
const dir = path.resolve(__dirname, '..');
const central = JSON.parse(fs.readFileSync(path.join(dir, 'central-slack.json')));
const core = JSON.parse(fs.readFileSync(path.join(dir, 'entrada-eventos.json')));
const node = p => central.nodes.find(n => n.name.startsWith(p));
const copy = v => JSON.parse(JSON.stringify(v));
const env = { APROVA_API_URL: 'http://api-interna.test:8080' };
function run(p, input, source = {}, ambiente = env) {
  return new Function('$json', '$', '$env', 'require', 'Buffer', node(p).parameters.jsCode)(
    input, name => ({ item: { json: source[name] } }), ambiente, require, Buffer,
  ).json;
}
const from = (p, input, origin) => run(p, input, { [node(p === 'V04 |' ? 'V01 |' : 'V04 |').name]: origin });
const response = body => ({ statusCode: 200, body });
const names = ['hash', 'chave', 'assinatura', 'conteudo', 'elo'];
const failures = ['hash', 'chave_desconhecida', 'assinatura', 'conteudo', 'elo'];
function fixture() {
  const original = JSON.parse(fs.readFileSync(path.join(dir, '..', 'exemplos/aguardando_aprovacao.json')));
  const event = new Function('$json', '$execution', core.nodes.find(n => n.name.startsWith('04 |')).parameters.jsCode)(
    { original, meta: { origem_verificada: true } }, { id: 'teste' },
  ).json;
  const id = '00000000-0000-4000-8000-000000000031', key = '0123456789abcdef';
  const payload = { versao: 1, id, task_run_id: event.meta.execution_id,
    tenant_id: event.meta.tenant_id, repositorio: event.repositorio.full_name,
    numero_pr: event.pr.numero, head_sha: event.pr.sha, nivel: event.risco.nivel.toLowerCase(), key_id: key };
  const bytes = Buffer.from(JSON.stringify(payload));
  const hash = crypto.createHash('sha256').update(bytes).digest('hex');
  event.recibo = { url: `https://publico.example/prefixo/receipts/${id}/verify`, hash };
  const document = { id, task_run_id: payload.task_run_id, key_id: key, hash_sha256: hash,
    payload_base64: bytes.toString('base64'), payload, assinatura_base64: Buffer.alloc(64).toString('base64') };
  const verdict = { id, key_id: key, hash_sha256: hash, valido: true,
    verificacoes: names.map(nome => ({ nome, resultado: 'ok' })) };
  return { event, document, verdict };
}
function identified(f = fixture()) { return from('V04 |', response(f.document), run('V01 |', f.event)); }
function classify(f = fixture(), origin = identified(f)) { return from('V07 |', response(f.verdict), origin); }
function failure(f, index) {
  f.verdict.valido = false; f.verdict.falha = failures[index];
  f.verdict.verificacoes.forEach((v, i) => v.resultado = i < index ? 'ok' : i === index ? 'falhou' : 'nao_verificada');
}

test('consulta somente a base configurada e extrai o UUID da URL pública', () => {
  const f = fixture(); f.event.recibo.url = f.event.recibo.url.replace('publico.example', 'host-nao-confiavel.example');
  const r = run('V01 |', f.event);
  assert.equal(r.consulta_recibo.url_documento, `${env.APROVA_API_URL}/receipts/${f.document.id}`);
  assert.equal(r.consulta_recibo.url_verificacao, r.consulta_recibo.url_documento + '/verify');
  assert.equal(r.recibo.url, f.event.recibo.url);
});
test('recusa contexto não autenticado ou evento diferente', () => {
  for (const edit of [e => e.meta.origem_verificada = false, e => e.meta.validacao.valido = false,
    e => e.meta.versao_evento = 2, e => e.evento.tipo = 'aprovacao_registrada']) {
    const f = fixture(); edit(f.event); assert.throws(() => run('V01 |', f.event));
  }
});
test('referência ausente, malformada ou com credenciais/query/fragmento não dispara GET', () => {
  const f = fixture();
  for (const url of ['', null, '/receipts/id', f.event.recibo.url + '?x=1', f.event.recibo.url + '#x',
    f.event.recibo.url.replace('https://', 'https://u:p@'), f.event.recibo.url.replace('https:', 'file:'),
    f.event.recibo.url.replace('/prefixo', '/\\prefixo')]) {
    const r = run('V01 |', { ...f.event, recibo: { ...f.event.recibo, url } });
    assert.equal(r.consulta_recibo.consultar, false); assert.equal(r.verificacao_recibo.status, 'indisponivel');
  }
});
test('identidade incompleta e base inválida impedem consultas', () => {
  for (const edit of [e => e.meta.tenant_id = 0, e => e.meta.execution_id = 'invalido',
    e => e.pr.sha = '', e => e.pr.numero = '14', e => e.recibo.hash = '',
    e => e.repositorio.full_name = '../repo', e => e.risco.nivel = 'inventado']) {
    const f = fixture(); edit(f.event); assert.equal(run('V01 |', f.event).consulta_recibo.consultar, false);
  }
  for (const base of ['', 'file:///tmp', 'http://u:p@api', 'http://api?x=1', 'http://api#x']) {
    assert.equal(run('V01 |', fixture().event, {}, { APROVA_API_URL: base }).consulta_recibo.consultar, false);
  }
});
test('veredito injetado no evento é descartado', () => {
  const f = fixture(); f.event.verificacao_recibo = { status: 'valido', fonte: 'nucleo', correlacao_confirmada: true };
  assert.equal(run('V01 |', f.event).verificacao_recibo.status, 'indisponivel');
});
test('documento e cinco conferências coerentes produzem resultado válido', () => {
  const r = classify(); assert.equal(r.verificacao_recibo.status, 'valido');
  assert.equal(r.verificacao_recibo.fonte, 'nucleo'); assert.equal(r.consulta_recibo, undefined);
});
for (const field of ['tenant_id', 'task_run_id', 'repositorio', 'numero_pr', 'head_sha', 'nivel', 'key_id', 'id']) {
  test(`bytes assinados com ${field} diferente não são correlacionados, mesmo com payload legível forjado`, () => {
    const f = fixture(), p = copy(f.document.payload);
    p[field] = typeof p[field] === 'number' ? p[field] + 1 : 'diferente';
    f.document.payload_base64 = Buffer.from(JSON.stringify(p)).toString('base64');
    const r = identified(f); assert.equal(r.consulta_recibo.verificar, false);
    assert.equal(r.verificacao_recibo.correlacao_confirmada, false);
  });
}
test('wrapper de documento deve coincidir com evento, tarefa e chave', () => {
  for (const field of ['id', 'task_run_id', 'key_id', 'hash_sha256']) {
    const f = fixture(); f.document[field] = 'outro'; assert.equal(identified(f).consulta_recibo.verificar, false);
  }
});
test('ignora payload de conveniência e usa os bytes canônicos', () => {
  const f = fixture(); f.document.payload = { tenant_id: 999 }; assert.equal(classify(f).verificacao_recibo.status, 'valido');
});
test('base64/UTF8/JSON inválidos e payload acima do limite não chegam ao verificador', () => {
  for (const value of ['%%%%', 'Zg==\n', '/x==', Buffer.from([255]).toString('base64'),
    Buffer.from('invalid json').toString('base64'), 'a'.repeat(262148)]) {
    const f = fixture(); f.document.payload_base64 = value;
    assert.equal(identified(f).consulta_recibo.verificar, false);
  }
});
for (let i = 0; i < failures.length; i++) {
  test(`HTTP 200 com falha ${failures[i]} significa inválido`, () => {
    const f = fixture(); failure(f, i);
    const r = classify(f).verificacao_recibo; assert.equal(r.status, 'invalido'); assert.equal(r.falha, failures[i]);
  });
}
test('recalcula hash dos bytes e exige confirmação coerente do núcleo', () => {
  const f = fixture(), p = copy(f.document.payload); p.campo_extra = true;
  const bytes = Buffer.from(JSON.stringify(p)); f.document.payload_base64 = bytes.toString('base64');
  f.verdict.hash_sha256 = crypto.createHash('sha256').update(bytes).digest('hex');
  assert.equal(classify(f).verificacao_recibo.status, 'indisponivel');
  failure(f, 2); assert.equal(classify(f).verificacao_recibo.status, 'indisponivel');
  failure(f, 0); assert.equal(classify(f).verificacao_recibo.status, 'invalido');
});
test('respostas incoerentes, incompletas e falso booleano nunca viram recibo válido', () => {
  for (const edit of [r => r.valido = 'true', r => r.id = 'outro', r => r.key_id = 'outro',
    r => r.hash_sha256 = '0'.repeat(64), r => r.verificacoes.pop(), r => r.verificacoes.reverse(),
    r => r.verificacoes[1].nome = 'hash', r => r.verificacoes[2].resultado = 'falhou',
    r => r.falha = 'assinatura', r => { r.valido = false; r.falha = ['hash']; },
    r => { r.valido = false; r.falha = '__proto__'; }, r => { r.valido = false; r.falha = 'desconhecida'; }]) {
    const f = fixture(); edit(f.verdict); assert.equal(classify(f).verificacao_recibo.status, 'indisponivel');
  }
});
test('404, 503, timeout, simulação e JSON inválido degradam para indisponível', () => {
  const f = fixture(), prepared = run('V01 |', f.event), origin = identified(f);
  for (const r of [{ statusCode: 404 }, { statusCode: 503 }, { error: 'SEGREDO_TESTE' },
    { statusCode: 200, body: '{' }, { statusCode: 200, body: { simulado: true } }]) {
    assert.equal(from('V04 |', r, prepared).verificacao_recibo.status, 'indisponivel');
    assert.equal(from('V07 |', r, origin).verificacao_recibo.status, 'indisponivel');
  }
});
test('HTTP GET público não segue redirects nem repete chamadas automaticamente', () => {
  for (const p of ['V03 |', 'V06 |']) {
    const n = node(p); assert.equal(n.parameters.method, 'GET'); assert.equal(n.parameters.authentication, 'none');
    assert.equal(n.parameters.options.timeout, 5000);
    assert.equal(n.parameters.options.redirect.redirect.followRedirects, false);
    assert.equal(n.parameters.options.response.response.fullResponse, true);
    assert.equal(n.parameters.options.response.response.neverError, true);
    assert.equal(n.onError, 'continueRegularOutput'); assert.equal(n.retryOnFail, false); assert.equal(n.credentials, undefined);
  }
});
test('alerta preserva risco, PR, OSV e ações para todos os estados e não aceita fonte forjada', () => {
  for (const status of ['valido', 'invalido', 'indisponivel']) {
    const f = fixture(); f.event.verificacao_recibo = { status, falha: 'assinatura', fonte: 'nucleo', correlacao_confirmada: true };
    const r = run('03A |', f.event);
    assert.deepEqual(r.risco, f.event.risco); assert.deepEqual(r.pr, f.event.pr);
    const actions = r.slack_payload.blocks.filter(b => b.type === 'actions').flatMap(b => b.elements);
    assert.ok(['aprovar', 'solicitar_alteracao', 'abrir_recibo', 'verificar_recibo'].every(id => actions.some(a => a.action_id === id)));
    assert.ok(r.slack_payload.blocks.some(b => b.text?.text.includes('OSV')));
    const block = r.slack_payload.blocks.find(b => b.block_id === 'aprova_recibo_verificacao');
    assert.match(block.text.text, status === 'valido' ? /Recibo: válido/ : status === 'invalido' ? /Recibo: inválido/ : /indisponível/);
    assert.doesNotMatch(JSON.stringify(r.slack_payload), /payload_base64|assinatura_base64|api-interna/);
    delete f.event.verificacao_recibo.fonte;
    assert.match(run('03A |', f.event).slack_payload.text, /verificação indisponível/);
  }
});
test('a decisão posterior preserva o bloco de verificação no snapshot original', () => {
  const f = fixture(), event = run('03A |', classify(f));
  const sources = {};
  const exec = (p, rows) => new Function('$input', '$', '$env', '$execution', node(p).parameters.jsCode)(
    { all: () => rows.map(json => ({ json })), first: () => ({ json: rows[0] }) },
    name => ({ isExecuted: name in sources, first: () => ({ json: sources[name] }) }),
    { APROVA_SLACK_CORRELACAO_HABILITADA: 'true', APROVA_SLACK_CANAL_ALERTAS_ID: 'C00000000TESTE',
      APROVA_SLACK_MENSAGENS_TABELA_ID: 'teste', APROVA_SLACK_CONTROLE_TABELA_ID: 'controle' }, { id: 'teste' },
  )[0].json;
  const alert = exec('C01 |', [event]);
  const original = JSON.parse(fs.readFileSync(path.join(dir, '..', 'exemplos/aprovacao_registrada.json')));
  original.dados.decisao = 'aprovada';
  const decision = new Function('$json', '$execution', core.nodes.find(n => n.name.startsWith('04 |')).parameters.jsCode)(
    { original, meta: { origem_verificada: true } }, { id: 'teste' },
  ).json;
  sources[node('C01 |').name] = exec('C01 |', [decision]);
  const updated = exec('C06 |', [{ id: 1, chave: alert.correlacao_alerta.chave, estado: 'confirmado',
    contexto: alert.correlacao_alerta.contexto, canal: alert.correlacao_alerta.canal, ts: '1794692000.000001', executor: 'anterior' }]);
  const block = p => p.blocks.find(b => b.block_id === 'aprova_recibo_verificacao');
  assert.deepEqual(block(updated.slack_payload), block(event.slack_payload));
});
