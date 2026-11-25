const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const dir = path.resolve(__dirname, '..');
const central = JSON.parse(fs.readFileSync(process.env.APROVA_TEST_CENTRAL || path.join(dir, 'central-slack.json')));
const osv = JSON.parse(fs.readFileSync(process.env.APROVA_TEST_OSV || path.join(dir, 'dependencias-osv.json')));
const core = JSON.parse(fs.readFileSync(path.join(dir, 'entrada-eventos.json')));
const node = (p, w = central) => w.nodes.find(n => n.name.startsWith(p));
const copy = v => JSON.parse(JSON.stringify(v));
const env = { APROVA_SLACK_CORRELACAO_HABILITADA: 'true', APROVA_SLACK_CANAL_ALERTAS_ID: 'C00000000TESTE',
  APROVA_SLACK_MENSAGENS_TABELA_ID: 'tabela-teste', APROVA_SLACK_CONTROLE_TABELA_ID: 'controle-teste' };
function execute(p, rows, sources = {}, ambiente = env, w = central) {
  return new Function('$json', '$input', '$', '$env', '$execution', node(p, w).parameters.jsCode)(
    rows[0], { all: () => rows.map(json => ({ json })), first: () => ({ json: rows[0] }) },
    name => ({ isExecuted: name in sources, first: () => ({ json: sources[name] }), item: { json: sources[name] } }),
    ambiente, { id: 'teste' },
  );
}
function run(...args) { const r = execute(...args); return Array.isArray(r) ? r[0]?.json : r.json; }
const source = (p, value) => ({ [node(p).name]: value });
const prepare = e => run('C01 |', [e]);
const plan = (e, rows) => run('C06 |', rows, source('C01 |', e));
const packageInfo = { nome: 'pacote', ecossistema: 'PyPI', versao: '1.0.0' };
function event(tipo = 'aguardando_aprovacao') {
  const original = JSON.parse(fs.readFileSync(path.join(dir, '..', 'exemplos', tipo + '.json')));
  if (tipo === 'aprovacao_registrada') original.dados.decisao = 'aprovada';
  return new Function('$json', '$execution', node('04 |', core).parameters.jsCode)(
    { original, meta: { origem_verificada: true } }, { id: 'teste' },
  ).json;
}
const initial = () => run('03A |', [event()]);
const alert = () => prepare(initial());
function callback(status = 'concluida', age = 1000) {
  const e = event(); e.evento.tipo = 'dependencia_nova'; e.entrega_osv = { versao: 1, fonte: 'workflow_osv' };
  e.dependencias = [packageInfo];
  e.contexto_osv_snapshot = { versao: 1, identidade: [e.meta.tenant_id, e.meta.execution_id, e.repositorio.full_name, e.pr.numero],
    sha: e.pr.sha, status, consultado_em: new Date(Date.now() - age).toISOString(), total_dependencias: 1,
    dependencias: [{ ...packageInfo, status, resumo: status === 'concluida' ? 'Nenhum aviso conhecido encontrado no OSV para a versão consultada.'
      : status === 'incompleta' ? 'Consulta incompleta. Os resultados são parciais: 2 avisos OSV observados.'
        : 'Não foi possível concluir a consulta ao OSV. A ausência de resultados não indica ausência de vulnerabilidades.' }] };
  return e;
}
const delivery = e => prepare(run('O01 |', [e || callback()]));
function record(a = alert(), state = 'confirmado') {
  return { id: 1, chave: a.correlacao_alerta.chave, contexto: a.correlacao_alerta.contexto, estado: state,
    canal: a.correlacao_alerta.canal, ts: state === 'confirmado' ? '1794691000.000001' : '', executor: 'teste' };
}
function persist(p, state = 'confirmado') {
  const c = p.correlacao_alerta, ctx = JSON.parse(c.contexto);
  if (c.operacao === 'osv') ctx.atualizacao_osv = state;
  else if (c.operacao === 'decisao') ctx.atualizacao_decisao = state;
  return { ...record(), contexto: JSON.stringify(ctx) };
}
const actions = p => p.blocks.filter(b => b.type === 'actions').flatMap(b => b.elements);
const block = (p, id) => p.blocks.find(b => b.block_id === id);

test('OSV posterior localiza o mesmo canal/ts sem publicar outro alerta', () => {
  const a = alert(), before = copy(a), d = delivery(), p = plan(d, [record(a)]);
  assert.equal(p.correlacao_alerta.acao, 'atualizar_osv'); assert.equal(p.correlacao_alerta.enviar, false);
  assert.equal(p.correlacao_alerta.chave, a.correlacao_alerta.chave);
  assert.equal(p.correlacao_alerta.ts, '1794691000.000001'); assert.deepEqual(a, before);
  assert.match(p.slack_payload.text, /Nenhum aviso conhecido/);
  assert.deepEqual(block(p.slack_payload, 'aprova_recibo_verificacao'), block(a.slack_payload, 'aprova_recibo_verificacao'));
  assert.deepEqual(p.slack_payload.blocks[1], a.slack_payload.blocks[1]);
});
test('contrato real dependencia_nova sem SHA ou risco não inventa essas informações', () => {
  const e = event('dependencia_nova');
  e.entrega_osv = { versao: 1, fonte: 'workflow_osv' };
  e.contexto_osv_snapshot = { versao: 1, identidade: [e.meta.tenant_id, e.meta.execution_id, e.repositorio.full_name, e.pr.numero],
    sha: '', status: 'concluida', consultado_em: new Date().toISOString(), total_dependencias: e.dependencias.length,
    dependencias: e.dependencias.map(d => ({ ...d, status: 'concluida', resumo: 'Nenhum aviso conhecido encontrado no OSV para a versão consultada.' })) };
  const d = delivery(e), a = event();
  a.meta.execution_id = e.meta.execution_id; a.pr.numero = e.pr.numero; a.pr.sha = '';
  const row = record(prepare(run('03A |', [a])));
  const p = plan(d, [row]);
  assert.equal(p.correlacao_alerta.acao, 'atualizar_osv');
  assert.equal(p.risco.nivel, e.risco.nivel);
  assert.deepEqual(p.slack_payload.blocks[1], JSON.parse(row.contexto).slack_payload.blocks[1]);
});
for (const status of ['concluida', 'incompleta', 'falha']) {
  test(`${status} mantém seu significado e o resumo nos formulários ainda disponíveis`, () => {
    const p = plan(delivery(callback(status)), [record()]);
    const ctx = JSON.parse(p.correlacao_alerta.contexto);
    assert.equal(ctx.osv.status, status);
    for (const a of actions(p.slack_payload).filter(a => ['aprovar', 'solicitar_alteracao'].includes(a.action_id))) {
      assert.equal(JSON.parse(a.value).osv_resumo, ctx.osv.resumo); assert.ok(a.value.length <= 2000);
    }
    if (status !== 'concluida') assert.doesNotMatch(p.slack_payload.text, /Nenhum aviso conhecido encontrado/);
  });
}
for (const choice of ['aprovada', 'alteracoes_solicitadas']) {
  test(`decisão ${choice} antes do OSV permanece fechada, com links e recibo preservados`, () => {
    const d = event('aprovacao_registrada'); d.aprovacao.decisao = choice; d.evento.acao = choice;
    const decision = plan(prepare(d), [record()]);
    const p = plan(delivery(), [persist(decision)]);
    assert.equal(p.correlacao_alerta.acao, 'atualizar_osv');
    assert.ok(actions(p.slack_payload).every(a => !['aprovar', 'solicitar_alteracao'].includes(a.action_id)));
    assert.ok(actions(p.slack_payload).some(a => a.action_id === 'verificar_recibo'));
    assert.match(p.slack_payload.text, choice === 'aprovada' ? /Aprovação registrada/ : /Alteração solicitada/);
    assert.equal(p.slack_payload.blocks.filter(b => b.text?.text.startsWith(choice === 'aprovada' ? 'Aprovação registrada pelo núcleo' : 'Pedido de alterações registrado pelo núcleo')).length, 1);
  });
}
test('decisão depois do OSV usa o resumo novo e remove os botões sem apagar recibos', () => {
  const osvPlan = plan(delivery(), [record()]);
  const p = plan(prepare(event('aprovacao_registrada')), [persist(osvPlan)]);
  assert.match(p.slack_payload.text, /Nenhum aviso conhecido/);
  assert.match(p.slack_payload.text, /Aprovação registrada/);
  assert.ok(actions(p.slack_payload).every(a => !['aprovar', 'solicitar_alteracao'].includes(a.action_id)));
});
for (const order of ['osv-decisao-alerta', 'decisao-osv-alerta']) {
  test(`${order}: guarda contexto mínimo e publica uma única mensagem já enriquecida e fechada`, () => {
    const o = delivery(), d = prepare(event('aprovacao_registrada'));
    const seq = order.startsWith('osv') ? [o, d] : [d, o];
    let row;
    for (const e of seq) {
      const p = plan(e, row ? [row] : [{}]);
      assert.equal(p.correlacao_alerta.acao, 'aguardar_alerta');
      row = { ...record(e, 'aguardando_alerta'), contexto: p.correlacao_alerta.contexto };
      assert.equal(JSON.parse(row.contexto).slack_payload, undefined);
    }
    const p = plan(alert(), [row]);
    assert.equal(p.correlacao_alerta.enviar, true); assert.match(p.slack_payload.text, /Nenhum aviso conhecido/);
    assert.match(p.slack_payload.text, /Aprovação registrada/);
    assert.ok(actions(p.slack_payload).every(a => !['aprovar', 'solicitar_alteracao'].includes(a.action_id)));
  });
}
test('reprocessamento confirmado não executa novo chat.update', () => {
  const d = delivery(), p = plan(d, [record()]);
  const repeated = plan(d, [persist(p)]);
  assert.equal(repeated.correlacao_alerta.status, 'osv_ja_aplicado'); assert.equal(repeated.correlacao_alerta.acao, 'encerrar');
});
test('resultado anterior ou de qualidade menor não apaga contexto recente e completo', () => {
  const p = plan(delivery(callback('concluida', 2000)), [record()]);
  for (const e of [callback('concluida', 5000), callback('incompleta', 100), callback('falha', 100)]) {
    const r = plan(delivery(e), [persist(p)]); assert.equal(r.correlacao_alerta.status, 'osv_desatualizado');
    assert.equal(r.correlacao_alerta.acao, 'encerrar');
  }
});
test('resultados diferentes no mesmo instante são conflito e não editam Slack', () => {
  const e = callback(), p = plan(delivery(e), [record()]);
  e.contexto_osv_snapshot.dependencias[0].resumo = '1 aviso OSV conhecido.';
  const r = plan(delivery(e), [persist(p)]); assert.equal(r.correlacao_alerta.status, 'osv_conflitante');
});
for (const state of ['reservado', 'incerto', 'falha']) {
  test(`${state}: OSV e decisão aguardam recuperação de atualização anterior`, () => {
    const d = delivery(), p = plan(d, [record()]), row = persist(p, state);
    assert.equal(plan(d, [row]).correlacao_alerta.status, 'osv_aguardando_recuperacao');
    assert.equal(plan(prepare(event('aprovacao_registrada')), [row]).correlacao_alerta.status, 'decisao_aguardando_recuperacao');
    const decided = persist(plan(prepare(event('aprovacao_registrada')), [record()]), state);
    assert.equal(plan(d, [decided]).correlacao_alerta.status, 'osv_aguardando_recuperacao');
  });
}
test('tenant, análise, repositório, PR, SHA ou canal incompatíveis não selecionam o alerta', () => {
  for (const field of ['installation_id', 'task_run_id', 'repositorio', 'numero_pr', 'sha']) {
    const row = record(), ctx = JSON.parse(row.contexto); ctx[field] = typeof ctx[field] === 'number' ? ctx[field] + 1 : 'outro';
    row.contexto = JSON.stringify(ctx); assert.equal(plan(delivery(), [row]).correlacao_alerta.status, 'correlacao_incompativel');
  }
  const row = record(); row.canal = 'COUTRO';
  assert.equal(plan(delivery(), [row]).correlacao_alerta.status, 'correlacao_incompativel');
});
test('snapshot vencido, futuro, outra identidade/SHA, incompleto ou duplicado é recusado', () => {
  for (const edit of [s => s.consultado_em = new Date(Date.now() - 25 * 3600000).toISOString(),
    s => s.consultado_em = new Date(Date.now() + 600000).toISOString(), s => s.identidade[0]++,
    s => s.sha = 'b'.repeat(40), s => s.total_dependencias = 2, s => s.dependencias[0].status = 'falha',
    s => s.dependencias = [], s => { s.dependencias.push(copy(s.dependencias[0])); s.total_dependencias = 2; },
    s => s.dependencias[0].resumo = null]) {
    const e = callback(); edit(e.contexto_osv_snapshot); assert.throws(() => delivery(e));
  }
});
test('envelope não autenticado ou entrega sem marcador interno é recusado', () => {
  for (const edit of [e => e.meta.origem_verificada = false, e => e.meta.validacao.valido = false,
    e => e.entrega_osv = {}, e => e.modo_osv = 'contexto_aprovacao']) {
    const e = callback(); edit(e); assert.throws(() => delivery(e));
  }
});
test('múltiplos eventos na mesma execução não são descartados silenciosamente', () => {
  assert.throws(() => execute('O01 |', [callback(), callback()]), /um evento/);
});
test('contexto anterior vencido não bloqueia a primeira publicação autenticada', () => {
  const p = plan(delivery(), [{}]), row = record(delivery(), 'aguardando_alerta');
  const ctx = JSON.parse(p.correlacao_alerta.contexto);
  ctx.osv.consultado_em = new Date(Date.now() - 25 * 3600000).toISOString();
  ctx.osv.impressao = JSON.stringify([ctx.osv.status, ctx.osv.consultado_em, ctx.osv.resumo]);
  row.contexto = JSON.stringify(ctx);
  const r = plan(alert(), [row]); assert.equal(r.correlacao_alerta.enviar, true);
  assert.doesNotMatch(r.slack_payload.text, /Nenhum aviso conhecido encontrado/);
});
test('bloco antigo sem block_id recebe OSV sem duplicação', () => {
  const a = alert(), row = record(a), ctx = JSON.parse(row.contexto);
  delete ctx.slack_payload.blocks.find(b => b.block_id === 'aprova_contexto_osv').block_id;
  row.contexto = JSON.stringify(ctx);
  const p = plan(delivery(), [row]); assert.equal(p.correlacao_alerta.acao, 'atualizar_osv');
  assert.equal(p.slack_payload.blocks.filter(b => b.block_id === 'aprova_contexto_osv').length, 1);
});
test('flag desligada impede encaminhamento, processamento e fallback no Incoming Webhook', () => {
  const e = callback(); assert.deepEqual(execute('O01 |', [e], {}, {}), []);
  assert.deepEqual(execute('06E |', [e], {}, {}, osv), []);
  assert.deepEqual(execute('C00 |', [run('C01 |', [e], {}, {})]), []);
});
test('consulta complementar síncrona nunca retorna à Central em ciclo', () => {
  const e = callback(); e.modo_osv = 'contexto_aprovacao';
  assert.deepEqual(execute('06E |', [e], {}, env, osv), []);
  const forward = run('06E |', [callback()], {}, env, osv);
  assert.equal(forward.entrega_osv.fonte, 'workflow_osv'); assert.equal(forward.slack_payload, undefined);
  assert.equal(node('06F |', osv).parameters.options.waitForSubWorkflow, false);
  assert.equal(node('06F |', osv).parameters.mode, 'each');
});
test('reserva compartilhada exige titularidade e tempo antes de qualquer escrita ou edição', () => {
  const p = plan(delivery(), [record()]);
  for (const lock of [{ executor: 'outra', bloqueado_ate: Date.now() + 900000 },
    { executor: 'teste', bloqueado_ate: Date.now() + 1000 }]) {
    assert.throws(() => run('D01 |', [{}], { ...source('C06 |', p), ...source('C03 |', lock) }));
  }
  const lock = { executor: 'teste', bloqueado_ate: Date.now() + 900000 };
  assert.deepEqual(run('D01 |', [{}], { ...source('C06 |', p), ...source('C03 |', lock) }), p);
});
test('classificação de Slack confirma somente o mesmo canal e ts, sem confirmar decisão por OSV', () => {
  const p = plan(delivery(), [record()]);
  for (const [response, state] of [[{ ok: true, channel: p.correlacao_alerta.canal, ts: p.correlacao_alerta.ts }, 'confirmado'],
    [{ ok: false }, 'falha'], [{ error: 'timeout' }, 'incerto'], [{ ok: true, channel: 'OUTRO', ts: p.correlacao_alerta.ts }, 'incerto']]) {
    const r = run('D05 |', [response], source('C06 |', p)), ctx = JSON.parse(r.correlacao_alerta.contexto);
    assert.equal(ctx.atualizacao_osv, state); assert.equal(ctx.atualizacao_decisao, undefined);
  }
});
test('retorno distingue atualização OSV de uma decisão humana confirmada', () => {
  const d = delivery(), p = plan(d, [record()]);
  const r = plan(d, [persist(p)]);
  const released = [{ chave: 'publicacao', executor: '', bloqueado_ate: 0 }];
  const result = run('C13 |', released, source('C06 |', r)).correlacao_alerta;
  assert.equal(result.osv_atualizado, true); assert.equal(result.decisao_atualizada, false);
  const beforeAlert = plan(d, [{}]);
  const early = run('C13 |', released, source('C06 |', beforeAlert)).correlacao_alerta;
  assert.equal(early.publicada, false); assert.equal(early.osv_atualizado, false);
});
test('recibo e estado de decisão permanecem após duas versões de OSV sem duplicar seções', () => {
  const decided = persist(plan(prepare(event('aprovacao_registrada')), [record()]));
  const partial = plan(delivery(callback('incompleta', 3000)), [decided]);
  const complete = plan(delivery(callback('concluida', 1000)), [persist(partial)]);
  assert.equal(complete.slack_payload.blocks.filter(b => b.block_id === 'aprova_contexto_osv').length, 1);
  assert.equal(complete.slack_payload.blocks.filter(b => b.text?.text.startsWith('Aprovação registrada pelo núcleo')).length, 1);
  assert.deepEqual(block(complete.slack_payload, 'aprova_recibo_verificacao'), block(initial().slack_payload, 'aprova_recibo_verificacao'));
  assert.ok(actions(complete.slack_payload).every(a => !['aprovar', 'solicitar_alteracao'].includes(a.action_id)));
});
