const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');

const dir = path.resolve(__dirname, '..');
const central = JSON.parse(fs.readFileSync(path.join(dir, 'central-slack.json')));
const entrada = JSON.parse(fs.readFileSync(path.join(dir, 'entrada-eventos.json')));
const node = (prefixo) => central.nodes.find(n => n.name.startsWith(prefixo));
const env = {
  APROVA_SLACK_CORRELACAO_HABILITADA: 'true',
  APROVA_SLACK_CANAL_ALERTAS_ID: 'C00000000TESTE',
  APROVA_SLACK_MENSAGENS_TABELA_ID: 'tabela-de-teste',
  APROVA_SLACK_CONTROLE_TABELA_ID: 'controle-de-teste',
};
function run(prefixo, rows, sources = {}, ambiente = env) {
  return new Function('$input', '$', '$env', '$execution', node(prefixo).parameters.jsCode)(
    { all: () => rows.map(json => ({ json })), first: () => ({ json: rows[0] }) },
    name => {
      const presente = name in sources;
      const value = sources[name];
      const item = () => {
        if (!presente) throw new Error('Origem de teste ausente: ' + name);
        return { json: value };
      };
      return { first: item, all: () => [item()], isExecuted: presente && value !== null };
    }, ambiente, { id: 'execucao-de-teste' },
  )[0].json;
}
const name = prefixo => node(prefixo).name;
function alerta() {
  const original = JSON.parse(fs.readFileSync(path.join(dir, '..', 'exemplos', 'aguardando_aprovacao.json')));
  const normalizar = entrada.nodes.find(n => n.name.startsWith('04 |')).parameters.jsCode;
  const evento = new Function('$json', '$execution', normalizar)(
    { original, meta: { origem_verificada: true } }, { id: 'teste' },
  ).json;
  return new Function('$json', node('03A |').parameters.jsCode)(evento).json;
}
const preparar = () => run('C01 |', [alerta()]);
function registro(origem, estado = 'confirmado') {
  const c = origem.correlacao_alerta;
  return { id: 1, chave: c.chave, estado, contexto: c.contexto, canal: c.canal,
    ts: estado === 'confirmado' ? '1794691000.000001' : '', executor: c.executor };
}
const selecionar = (origem, rows) => run('C06 |', rows, { [name('C01 |')]: origem });
const lock = () => ({ chave: 'publicacao', executor: 'execucao-de-teste', bloqueado_ate: Date.now() + 900000 });

test('flag desabilitada preserva o alerta e o envio anterior sem exigir configuração nova', () => {
  const original = alerta();
  const saida = run('C01 |', [original], {}, {});
  assert.deepEqual(saida.slack_payload, original.slack_payload);
  assert.deepEqual(saida.correlacao_alerta, { habilitada: false });
});

test('correlação habilitada exige origem autenticada, contrato e destinos válidos', () => {
  for (const alterar of [
    e => e.meta.origem_verificada = false,
    e => e.meta.validacao.valido = false,
    e => e.meta.versao_evento = 2,
    e => e.meta.tenant_id = 0,
    e => e.meta.execution_id = 'invalido',
    e => e.evento.tipo = 'aprovacao_registrada',
    e => e.repositorio.full_name = 'equipe/..',
    e => e.pr.numero = 0,
    e => e.pr.sha = 'sha-invalido',
    e => e.slack_payload.blocks = null,
  ]) {
    const e = alerta(); alterar(e);
    assert.throws(() => run('C01 |', [e]));
  }
  for (const key of ['APROVA_SLACK_CANAL_ALERTAS_ID', 'APROVA_SLACK_MENSAGENS_TABELA_ID', 'APROVA_SLACK_CONTROLE_TABELA_ID']) {
    assert.throws(() => run('C01 |', [alerta()], {}, { ...env, [key]: '' }), /Configurar canal e tabelas/);
  }
  assert.throws(() => run('C01 |', [alerta(), alerta()]), /um evento/);
});

test('chave separa instalação, análise, repositório, PR e tipo do alerta', () => {
  const base = preparar();
  for (const alterar of [
    e => e.meta.tenant_id++,
    e => e.meta.execution_id = '00000000-0000-4000-8000-000000000001',
    e => e.repositorio.full_name = 'outra/equipe',
    e => e.pr.numero++,
    e => e.evento.tipo = 'falha_recuperavel',
  ]) {
    const e = alerta(); alterar(e);
    assert.notEqual(run('C01 |', [e]).correlacao_alerta.chave, base.correlacao_alerta.chave);
  }
  assert.deepEqual(base.risco, alerta().risco);
});

test('leitura vazia permite reservar; confirmação anterior suprime a duplicata', () => {
  const origem = preparar();
  assert.equal(selecionar(origem, [{}]).correlacao_alerta.enviar, true);
  const c = selecionar(origem, [registro(origem)]).correlacao_alerta;
  assert.equal(c.enviar, false);
  assert.equal(c.status, 'ja_publicado');
  assert.equal(c.existente.ts, '1794691000.000001');
});

for (const estado of ['reservado', 'incerto', 'falha']) {
  test(`estado ${estado} exige recuperação manual e impede repetição automática`, () => {
    const origem = preparar();
    const c = selecionar(origem, [registro(origem, estado)]).correlacao_alerta;
    assert.equal(c.enviar, false);
    assert.equal(c.status, 'recuperacao_manual');
  });
}

test('registros ambíguos ou leitura com erro nunca autorizam publicação', () => {
  const origem = preparar();
  const c = selecionar(origem, [registro(origem), { ...registro(origem), id: 2 }]).correlacao_alerta;
  assert.equal(c.enviar, false);
  assert.equal(c.status, 'correlacao_ambigua');
  assert.throws(() => selecionar(origem, [{ error: 'indisponivel' }]), /Leitura da correlação inválida/);
});

test('contexto incompatível, SHA divergente ou canal trocado não reutiliza mensagem', () => {
  const origem = preparar();
  for (const campo of ['installation_id', 'task_run_id', 'repositorio', 'numero_pr', 'tipo_alerta', 'sha']) {
    const r = registro(origem);
    const ctx = JSON.parse(r.contexto);
    ctx[campo] = campo === 'installation_id' || campo === 'numero_pr' ? 999 : 'incompativel';
    r.contexto = JSON.stringify(ctx);
    const c = selecionar(origem, [r]).correlacao_alerta;
    assert.equal(c.enviar, false);
    assert.equal(c.status, 'correlacao_incompativel');
  }
  assert.equal(selecionar(origem, [{ ...registro(origem), canal: 'COUTROTESTE' }]).correlacao_alerta.status,
    'correlacao_incompativel');
  assert.equal(selecionar(origem, [{ ...registro(origem), contexto: 'invalido' }]).correlacao_alerta.enviar, false);
});

test('controle ocupado, vazio ou ambíguo interrompe antes de qualquer envio', () => {
  const origem = preparar();
  const sources = { [name('C01 |')]: origem };
  assert.deepEqual(run('C04 |', [lock()], sources), origem);
  for (const rows of [[{}], [{ ...lock(), executor: 'outro' }], [lock(), lock()], [{ ...lock(), bloqueado_ate: 0 }]]) {
    assert.throws(() => run('C04 |', rows, sources), /Publicação ocupada ou controle inválido/);
  }
});

test('envio exige reserva persistida compatível e controle ainda válido', () => {
  const origem = selecionar(preparar(), [{}]);
  const sources = { [name('C06 |')]: origem, [name('C03 |')]: lock() };
  assert.deepEqual(run('C08B |', [registro(origem, 'reservado')], sources), origem);
  for (const r of [{}, { ...registro(origem, 'reservado'), executor: 'outro' }, registro(origem)]) {
    assert.throws(() => run('C08B |', [r], sources), /Reserva ou exclusão não confirmada/);
  }
  assert.throws(() => run('C08B |', [registro(origem, 'reservado')], {
    ...sources, [name('C03 |')]: { ...lock(), bloqueado_ate: Date.now() + 1000 },
  }), /Reserva ou exclusão não confirmada/);
});

function confirmar(resposta) {
  const origem = selecionar(preparar(), [{}]);
  const reserva = registro(origem, 'reservado');
  return run('C10 |', [resposta], { [name('C06 |')]: origem, [name('C08 |')]: reserva });
}

test('somente canal e timestamp confirmados registram a publicação', () => {
  for (const field of ['ts', 'message_timestamp']) {
    const c = confirmar({ channel: env.APROVA_SLACK_CANAL_ALERTAS_ID, [field]: '1794691000.000001' }).correlacao_alerta;
    assert.equal(c.estado, 'confirmado');
    assert.equal(c.ts, '1794691000.000001');
  }
  for (const resposta of [{ error: 'timeout' }, { ok: true },
    { channel: 'COUTROTESTE', ts: '1794691000.000001' },
    { channel: env.APROVA_SLACK_CANAL_ALERTAS_ID, ts: 'invalido' },
    { channel: env.APROVA_SLACK_CANAL_ALERTAS_ID, ts: '1794691000.000001', simulado: true }]) {
    const c = confirmar(resposta).correlacao_alerta;
    assert.equal(c.estado, 'incerto');
    assert.equal(c.ts, '');
  }
  assert.equal(confirmar({ ok: false, error: 'channel_not_found' }).correlacao_alerta.estado, 'falha');
});

test('persistência ausente ou incompatível impede confirmar a conclusão', () => {
  const origem = confirmar({ channel: env.APROVA_SLACK_CANAL_ALERTAS_ID, ts: '1794691000.000001' });
  const c = origem.correlacao_alerta;
  const row = { ...registro(origem), id: c.reserva_id, estado: c.estado, ts: c.ts };
  const sources = { [name('C10 |')]: origem };
  assert.deepEqual(run('C11B |', [row], sources), origem);
  for (const r of [{}, { ...row, estado: 'reservado' }, { ...row, ts: '' }, { ...row, id: 2 }]) {
    assert.throws(() => run('C11B |', [r], sources), /não persistido/);
  }
});

test('resultado final distingue recuperação manual de publicação e exige liberação', () => {
  const origem = confirmar({ error: 'timeout' });
  const sources = { [name('C10 |')]: origem };
  const released = { chave: 'publicacao', bloqueado_ate: 0, executor: '' };
  const c = run('C13 |', [released], sources).correlacao_alerta;
  assert.equal(c.publicada, false);
  assert.equal(c.exige_recuperacao_manual, true);
  assert.equal(c.ts, '');
  assert.throws(() => run('C13 |', [lock()], sources), /Liberação do controle não confirmada/);
});
