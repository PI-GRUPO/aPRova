const test = require('node:test');
const assert = require('node:assert/strict');
const crypto = require('node:crypto');
const workflow = require('../digest-semanal.json');
const codigo = prefix => workflow.nodes.find(n => n.name.startsWith(prefix)).parameters.jsCode;
const periodo = { installation_id: 123, inicio: '2026-10-26', fim: '2026-11-02' };
const metricas = () => ({
  installation_id: 123, inicio: '2026-10-26T00:00:00Z', fim: '2026-11-02T00:00:00Z',
  total_execucoes: 10, concluidas: 6, rejeitadas: 1, falhas: 1, em_andamento: 2,
  taxa_sucesso: 0.75, exigiram_aprovacao_humana: 3, risco_medio: 1, revertidas: 1,
});
const formatar = resposta => new Function('$json', '$', codigo('05 |'))(
  resposta, () => ({ item: { json: periodo } }),
).json;
const resposta = body => ({ statusCode: 200, body });

test('Digest: export inativo tem somente o caminho real e nenhuma credencial ou dado fixado', () => {
  assert.equal(workflow.active, false);
  assert.deepEqual(workflow.pinData, {});
  assert.equal(workflow.nodes.length, 7);
  assert.equal(/simula|TESTE|enviar_slack/i.test(JSON.stringify(workflow)), false);
  assert.equal(workflow.settings.timezone, 'America/Sao_Paulo');
  const names = workflow.nodes.map(n => n.name);
  workflow.nodes.forEach((n, i) => {
    assert.equal(n.credentials, undefined);
    if (i < names.length - 1) assert.equal(workflow.connections[n.name].main[0][0].node, names[i + 1]);
  });
  assert.equal(workflow.nodes[0].parameters.rule.interval[0].expression, '0 9 * * 1');
  assert.equal(workflow.nodes[5].retryOnFail, undefined);
});

test('Digest: configuração vazia bloqueia consulta e semana anterior funciona na virada do ano', () => {
  assert.throws(() => new Function(codigo('02 |'))(), /Preencha installation_ids/);
  class RelogioFixo extends Date {
    constructor(...args) { super(...(args.length ? args : ['2027-01-04T12:00:00Z'])); }
  }
  const executar = ids => new Function('Date', codigo('02 |').replace('installation_ids: []', `installation_ids: ${JSON.stringify(ids)}`))(RelogioFixo);
  assert.deepEqual(executar([123, 123, 456]).map(i => i.json), [123, 456].map(installation_id => ({
    installation_id, inicio: '2026-12-28', fim: '2027-01-04',
  })));
  for (const ids of [[0], [-1], [1.5], ['123'], Array.from({ length: 51 }, (_, i) => i + 1)]) {
    assert.throws(() => executar(ids), /Preencha installation_ids/);
  }
});

test('Digest: assinatura HMAC cobre exatamente a query e exige configurações', () => {
  const executar = env => new Function('require', '$json', '$env', codigo('03 |'))(require, periodo, env).json;
  const env = { APROVA_API_URL: 'http://api:8080/', N8N_SEGREDO_ACOES: 'segredo-teste', SLACK_INCOMING_WEBHOOK_URL: 'https://slack.invalid/' };
  const r = executar(env).requisicao;
  const url = new URL(r.url);
  assert.equal(url.origin, 'http://api:8080');
  assert.equal(url.pathname, '/reports/weekly');
  assert.equal(url.search, '?installation_id=123&inicio=2026-10-26');
  assert.equal(r.headers['X-Aprova-Assinatura'], 'sha256=' + crypto.createHmac('sha256', env.N8N_SEGREDO_ACOES)
    .update(r.headers['X-Aprova-Timestamp'] + '.' + url.search.slice(1)).digest('hex'));
  for (const key of ['N8N_SEGREDO_ACOES', 'SLACK_INCOMING_WEBHOOK_URL']) assert.throws(() => executar({ ...env, [key]: '' }));
});

test('Digest: valida métricas, exibe período inclusivo e preserva taxa nula em semana vazia', () => {
  const r = formatar(resposta(metricas()));
  assert.match(r.slack_payload.text, /75,0%/);
  assert.match(r.slack_payload.text, /26\/10\/2026 a 01\/11\/2026 \(UTC\)/);
  assert.ok(r.slack_payload.text.includes('\n'));
  assert.equal(r.slack_payload.blocks[0].text.type, 'plain_text');
  const vazio = metricas();
  for (const key of Object.keys(vazio)) if (typeof vazio[key] === 'number' && key !== 'installation_id') vazio[key] = 0;
  vazio.taxa_sucesso = null;
  assert.match(formatar(resposta(vazio)).slack_payload.text, /nenhuma execução finalizada/);
});

test('Digest: recusa outra instalação, período, contagens incoerentes, taxa e HTTP inválidos', () => {
  for (const alteracao of [
    { installation_id: 999 }, { inicio: '2026-10-25T00:00:00Z' }, { fim: '2026-11-03T00:00:00Z' },
    { total_execucoes: 11 }, { falhas: -1 }, { em_andamento: 0.5 }, { taxa_sucesso: 0.9 },
    { taxa_sucesso: null }, { exigiram_aprovacao_humana: 11 },
  ]) assert.throws(() => formatar(resposta({ ...metricas(), ...alteracao })), /Relatório semanal inválido/);
  for (const entrada of [{ statusCode: 500, body: metricas() }, resposta(null), resposta('inválido')]) {
    assert.throws(() => formatar(entrada), /Relatório semanal inválido/);
  }
});

test('Digest: confirmação aceita body/data com ok e recusa falha sem anunciar envio', () => {
  const executar = entrada => new Function('$json', '$', codigo('07 |'))(
    entrada, () => ({ item: { json: periodo } }),
  ).json;
  for (const campo of ['body', 'data']) assert.equal(executar({ statusCode: 200, [campo]: 'ok' }).status, 'enviado');
  for (const entrada of [{ statusCode: 200, data: 'erro' }, { statusCode: 500, data: 'ok' }, { error: 'timeout' }]) {
    assert.throws(() => executar(entrada), /Slack não confirmou/);
  }
});
