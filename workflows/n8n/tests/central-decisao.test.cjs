const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const dir = path.resolve(__dirname, '..');
const central = JSON.parse(fs.readFileSync(path.join(dir, 'central-slack.json')));
const core = JSON.parse(fs.readFileSync(path.join(dir, 'entrada-eventos.json')));
const node = p => central.nodes.find(n => n.name.startsWith(p));
const env = { APROVA_SLACK_CORRELACAO_HABILITADA: 'true', APROVA_SLACK_CANAL_ALERTAS_ID: 'C00000000TESTE',
  APROVA_SLACK_MENSAGENS_TABELA_ID: 'mensagens-teste', APROVA_SLACK_CONTROLE_TABELA_ID: 'controle-teste' };
function executar(p, rows, sources = {}, ambiente = env) {
  return new Function('$input', '$', '$env', '$execution', node(p).parameters.jsCode)(
    { all: () => rows.map(json => ({ json })), first: () => ({ json: rows[0] }) },
    name => ({ isExecuted: name in sources && sources[name] !== null,
      first: () => { assert.ok(name in sources, name); return { json: sources[name] }; } }),
    ambiente, { id: 'teste' },
  );
}
const run = (...args) => executar(...args)[0].json;
const source = (p, value) => ({ [node(p).name]: value });
function normalizar(tipo, alterar = () => {}) {
  const original = JSON.parse(fs.readFileSync(path.join(dir, '..', 'exemplos', tipo + '.json')));
  if (tipo === 'aprovacao_registrada') original.dados.decisao = 'aprovada';
  alterar(original);
  return new Function('$json', '$execution', core.nodes.find(n => n.name.startsWith('04 |')).parameters.jsCode)(
    { original, meta: { origem_verificada: true } }, { id: 'teste' },
  ).json;
}
function alerta() {
  const e = normalizar('aguardando_aprovacao');
  return new Function('$json', node('03A |').parameters.jsCode)(e).json;
}
const prepararAlerta = () => run('C01 |', [alerta()]);
const prepararDecisao = alterar => run('C01 |', [normalizar('aprovacao_registrada', alterar)]);
const selecionar = (e, rows) => run('C06 |', rows, source('C01 |', e));
const registro = (e, estado = 'confirmado') => ({ id: 1, chave: e.correlacao_alerta.chave, estado,
  contexto: e.correlacao_alerta.contexto, canal: e.correlacao_alerta.canal,
  ts: estado === 'confirmado' ? '1794692000.000001' : '', executor: 'anterior' });
function comDecisao(row, evento = prepararDecisao(), estado) {
  const ctx = JSON.parse(row.contexto);
  ctx.decisao = JSON.parse(evento.correlacao_alerta.contexto).decisao;
  if (estado) ctx.atualizacao_decisao = estado;
  return { ...row, contexto: JSON.stringify(ctx) };
}
const planejarAtualizacao = () => selecionar(prepararDecisao(), [registro(prepararAlerta())]);
const lock = () => ({ executor: 'teste', bloqueado_ate: Date.now() + 900000 });

test('evento do worker normalizado identifica exatamente o alerta da mesma análise', () => {
  const a = prepararAlerta(), d = prepararDecisao();
  assert.equal(d.correlacao_alerta.chave, a.correlacao_alerta.chave);
  assert.equal(d.correlacao_alerta.operacao, 'decisao');
  assert.equal(JSON.parse(d.correlacao_alerta.contexto).decisao.id_aprovacao, 37);
  assert.deepEqual(d.risco, normalizar('aprovacao_registrada').risco);
});

test('decisão exige origem verificada, aprovação, revisor e enumerações válidos', () => {
  for (const alterar of [
    e => e.meta.origem_verificada = false,
    e => e.aprovacao.id = 0,
    e => e.aprovacao.login_github = '<@alguem>',
    e => e.aprovacao.origem = 'ia',
    e => e.aprovacao.decisao = 'mergeada',
    e => e.evento.acao = 'alteracoes_solicitadas',
    e => e.risco.nivel_original = 'inventado',
  ]) {
    const e = normalizar('aprovacao_registrada'); alterar(e);
    assert.throws(() => executar('C01 |', [e]));
  }
});

test('decisões de risco original baixo ou médio não atualizam alertas ALTO', () => {
  for (const nivel of ['baixo', 'medio']) {
    const e = normalizar('aprovacao_registrada', o => o.dados.nivel_risco_original = nivel);
    assert.deepEqual(executar('C01 |', [e]), []);
  }
});

test('correlação desabilitada nunca encaminha uma decisão ao Incoming Webhook', () => {
  const d = run('C01 |', [normalizar('aprovacao_registrada')], {}, {});
  assert.deepEqual(executar('C00 |', [d]), []);
  const a = alerta();
  assert.deepEqual(executar('C00 |', [a]), [{ json: a }]);
});

for (const decisao of ['aprovada', 'alteracoes_solicitadas']) {
  test(`${decisao}: altera o mesmo alerta e preserva risco, OSV e recibos`, () => {
    const a = prepararAlerta(), row = registro(a);
    const original = JSON.parse(JSON.stringify(a));
    const d = prepararDecisao(o => o.dados.decisao = decisao);
    const p = selecionar(d, [row]);
    assert.equal(p.correlacao_alerta.acao, 'atualizar_decisao');
    assert.equal(p.correlacao_alerta.ts, row.ts);
    const actions = p.slack_payload.blocks.filter(b => b.type === 'actions').flatMap(b => b.elements);
    assert.ok(actions.every(b => !['aprovar', 'solicitar_alteracao'].includes(b.action_id)));
    assert.ok(actions.some(b => b.action_id === 'verificar_recibo'));
    assert.ok(p.slack_payload.blocks.some(b => b.text?.text.includes('OSV')));
    assert.match(p.slack_payload.text, decisao === 'aprovada' ? /Aprovação registrada/ : /Alterações solicitadas/);
    assert.doesNotMatch(p.slack_payload.blocks[0].text.text, /aguardando aprovação/);
    assert.deepEqual(a, original);
    assert.deepEqual(JSON.parse(p.correlacao_alerta.contexto).slack_payload, a.slack_payload);
  });
}

test('decisão repetida confirmada não chama chat.update novamente', () => {
  const d = prepararDecisao();
  const row = comDecisao(registro(prepararAlerta()), d, 'confirmado');
  const p = selecionar(d, [row]).correlacao_alerta;
  assert.equal(p.status, 'decisao_ja_aplicada');
  assert.equal(p.acao, 'encerrar');
});

test('aprovação, tipo ou revisor conflitante preservam a primeira decisão', () => {
  const row = comDecisao(registro(prepararAlerta()), prepararDecisao(), 'confirmado');
  for (const alterar of [
    o => o.dados.id_aprovacao = 38,
    o => o.dados.decisao = 'alteracoes_solicitadas',
    o => o.dados.login_github = 'outro-revisor',
  ]) {
    const p = selecionar(prepararDecisao(alterar), [row]).correlacao_alerta;
    assert.equal(p.status, 'decisao_conflitante');
    assert.equal(p.acao, 'encerrar');
    assert.equal(p.contexto, row.contexto);
  }
});

test('outra instalação, análise, PR, repositório, canal ou SHA não reutiliza a mensagem', () => {
  const d = prepararDecisao();
  for (const campo of ['installation_id', 'task_run_id', 'numero_pr', 'repositorio']) {
    const row = registro(prepararAlerta()), ctx = JSON.parse(row.contexto);
    ctx[campo] = typeof ctx[campo] === 'number' ? 999 : 'incompativel';
    row.contexto = JSON.stringify(ctx);
    assert.equal(selecionar(d, [row]).correlacao_alerta.status, 'correlacao_incompativel');
  }
  assert.equal(selecionar(d, [{ ...registro(prepararAlerta()), canal: 'COUTROTESTE' }]).correlacao_alerta.status,
    'correlacao_incompativel');
  const outroSha = prepararDecisao(o => o.dados.sha = 'b'.repeat(40));
  assert.equal(selecionar(outroSha, [registro(prepararAlerta())]).correlacao_alerta.status, 'correlacao_incompativel');
});

test('decisão antes do alerta guarda somente a referência e fecha a publicação futura', () => {
  const d = prepararDecisao(), p = selecionar(d, [{}]);
  assert.equal(p.correlacao_alerta.acao, 'aguardar_alerta');
  assert.equal(p.correlacao_alerta.estado_persistir, 'aguardando_alerta');
  assert.equal(JSON.parse(p.correlacao_alerta.contexto).slack_payload, undefined);
  const row = registro(p, 'aguardando_alerta');
  const a = selecionar(prepararAlerta(), [row]);
  assert.equal(a.correlacao_alerta.enviar, true);
  assert.ok(JSON.parse(a.correlacao_alerta.contexto).decisao);
  assert.ok(a.slack_payload.blocks.filter(b => b.type === 'actions').flatMap(b => b.elements)
    .every(b => !['aprovar', 'solicitar_alteracao'].includes(b.action_id)));
  const c = run('C10 |', [{ ok: true, channel: env.APROVA_SLACK_CANAL_ALERTAS_ID, message_timestamp: '1794692000.000001' }], {
    ...source('C06 |', a), ...source('C08 |', { ...registro(a, 'reservado'), executor: 'teste' }),
  }).correlacao_alerta;
  assert.equal(JSON.parse(c.contexto).atualizacao_decisao, 'confirmado');
});

for (const estado of ['reservado', 'incerto', 'falha']) {
  test(`decisão preserva publicação ${estado} e não autoriza republicação`, () => {
    const row = registro(prepararAlerta(), estado);
    const c = selecionar(prepararDecisao(), [row]).correlacao_alerta;
    assert.equal(c.status, 'decisao_aguardando_recuperacao');
    assert.equal(c.estado_persistir, estado);
    assert.equal(c.enviar, false);
    assert.deepEqual(JSON.parse(c.contexto).slack_payload, JSON.parse(row.contexto).slack_payload);
  });
  test(`atualização ${estado} não é repetida por um evento duplicado`, () => {
    const d = prepararDecisao();
    const row = comDecisao(registro(prepararAlerta()), d, estado);
    const c = selecionar(d, [row]).correlacao_alerta;
    assert.equal(c.status, 'decisao_aguardando_recuperacao');
    assert.equal(c.acao, 'encerrar');
  });
}

test('reserva e lease válidos são obrigatórios antes de chat.update', () => {
  const p = planejarAtualizacao(), c = p.correlacao_alerta;
  const row = { ...registro(p), contexto: c.contexto, executor: 'teste' };
  const sources = { ...source('C06 |', p), ...source('C03 |', lock()) };
  assert.deepEqual(run('D03 |', [row], sources), p);
  for (const bad of [{}, { ...row, ts: '1794692000.999999' }, { ...row, executor: 'outro' }, { ...row, contexto: '{}' }]) {
    assert.throws(() => run('D03 |', [bad], sources), /Reserva da atualização não confirmada/);
  }
  assert.throws(() => run('D03 |', [row], { ...sources, ...source('C03 |', { ...lock(), bloqueado_ate: 0 }) }),
    /Reserva da atualização não confirmada/);
});

test('confirmação de atualização exige exatamente o canal e o timestamp reservados', () => {
  const p = planejarAtualizacao(), c = p.correlacao_alerta;
  const ok = { ok: true, channel: c.canal, message_timestamp: c.ts };
  const confirmar = r => run('D05 |', [r], source('C06 |', p));
  assert.equal(confirmar(ok).correlacao_alerta.status, 'decisao_atualizada');
  for (const r of [{ error: 'timeout' }, { ...ok, channel: 'COUTROTESTE' }, { ...ok, message_timestamp: '1794692000.999999' },
    { ...ok, simulado: true }]) {
    const saida = confirmar(r);
    assert.equal(saida.correlacao_alerta.status, 'atualizacao_incerta');
    assert.equal(JSON.parse(saida.correlacao_alerta.contexto).atualizacao_decisao, 'incerto');
  }
  assert.equal(confirmar({ ok: false, error: 'message_not_found' }).correlacao_alerta.status, 'atualizacao_falhou');
});

test('atualização só termina após persistência exata; resultado incerto exige recuperação', () => {
  const p = planejarAtualizacao();
  const incerto = run('D05 |', [{ error: 'timeout' }], source('C06 |', p));
  const row = { ...registro(incerto), executor: 'teste' };
  assert.deepEqual(run('D07 |', [row], source('D05 |', incerto)), incerto);
  assert.throws(() => run('D07 |', [{ ...row, contexto: '{}' }], source('D05 |', incerto)), /não persistido/);
  const fim = run('C13 |', [{ chave: 'publicacao', executor: '', bloqueado_ate: 0 }], source('D05 |', incerto)).correlacao_alerta;
  assert.equal(fim.publicada, true);
  assert.equal(fim.decisao_atualizada, false);
  assert.equal(fim.exige_recuperacao_manual, true);
});

test('Core encaminha decisões à Central preservando feedback e SLA', () => {
  const targets = core.connections['07H | aprovacao_registrada | Conectar workflow'].main[0].map(c => c.node);
  for (const target of ['07I | Solicitar Motivo da Aprovação', '07M | Acompanhar SLA de Aprovação', '07N | Atualizar alerta após decisão']) {
    assert.ok(targets.includes(target));
  }
  const forward = core.nodes.find(n => n.name === '07N | Atualizar alerta após decisão');
  assert.equal(forward.parameters.mode, 'each');
  assert.equal(forward.parameters.options.waitForSubWorkflow, false);
});
