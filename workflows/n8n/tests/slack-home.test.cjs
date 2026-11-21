const test = require('node:test');
const assert = require('node:assert/strict');
const crypto = require('node:crypto');
const eventos = require('../slack-consulta-metricas.json');
const interacoes = require('../central-slack.json');
const porNome = (w, prefixo) => w.nodes.find(n => n.name.startsWith(prefixo));
const AsyncFunction = Object.getPrototypeOf(async function () {}).constructor;

async function receber(w, payload, adulterar = false) {
  const evento = w === eventos;
  const raw = Buffer.from(evento ? JSON.stringify(payload, null, 2) :
    new URLSearchParams({ payload: JSON.stringify(payload) }).toString());
  const timestamp = String(Math.floor(Date.now() / 1000));
  const assinatura = 'v0=' + crypto.createHmac('sha256', 'fixture')
    .update('v0:' + timestamp + ':').update(raw).digest('hex');
  const headers = { 'x-slack-request-timestamp': timestamp, 'x-slack-signature': assinatura };
  const item = { binary: { data: {} }, json: { headers, body: { payload: JSON.stringify(payload) } } };
  const verificar = porNome(w, evento ? '02 | Verificar' : '06 | Verificar');
  const executar = new AsyncFunction('require', '$input', '$env', 'Buffer', verificar.parameters.jsCode);
  const [saida] = await executar.call({ helpers: { getBinaryDataBuffer: async () => adulterar ? Buffer.from('x') : raw } },
    require, { first: () => item }, { SLACK_SIGNING_SECRET: 'fixture', APROVA_SLACK_TEAM_ID: 'T123', APROVA_SLACK_APP_ID: 'A123' }, Buffer);
  return { saida: saida.json, original: { corpo: raw.toString(), timestamp, assinatura } };
}
const abertura = { type: 'event_callback', team_id: 'T123', api_app_id: 'A123', event_id: 'Ev123',
  event: { type: 'app_home_opened', user: 'U123', tab: 'home' } };
const clique = { api_app_id: 'A123', type: 'block_actions', team: { id: 'T123' }, user: { id: 'U123' },
  actions: [{ action_id: 'aprova_home_atualizar' }], view: { type: 'home' } };

test('importação CLI mantém os caminhos estáticos dos webhooks do Slack', () => {
  const ids = new Set();
  for (const [w, caminho] of [[eventos, 'aprova/slack/consultas'], [interacoes, 'aprova/slack/interacoes']]) {
    const webhook = w.nodes.find(n => n.type === 'n8n-nodes-base.webhook');
    assert.equal(webhook.parameters.path, caminho);
    assert.match(webhook.webhookId, /^[0-9a-f-]{36}$/);
    ids.add(webhook.webhookId);
  }
  assert.equal(ids.size, 2);
});

for (const [w, entrada, tipo, legado] of [
  [eventos, abertura, 'application/json', '04 | Preparar Consulta do Slack'],
  [interacoes, clique, 'application/x-www-form-urlencoded', '07F | É Feedback de Aprovação'],
]) {
  test('Home preserva assinatura e bytes até o adaptador após ACK: ' + tipo, async () => {
    const { saida, original } = await receber(w, entrada);
    assert.deepEqual(saida.home, original);
    await assert.rejects(receber(w, entrada, true), /assinatura/i);
    const ack = porNome(w, w === eventos ? '03A | Confirmar' : '07 | Confirmar');
    const seletor = porNome(w, 'H01 |'), relay = porNome(w, 'H02 |');
    assert.equal(w.connections[ack.name].main[0][0].node, seletor.name);
    assert.equal(seletor.parameters.conditions.conditions[0].leftValue, '={{ !!$json.home }}');
    assert.deepEqual(w.connections[seletor.name].main.map(r => r.map(n => n.node)), [[relay.name], [legado]]);
    assert.equal(relay.retryOnFail, false);
    assert.equal(relay.parameters.options.redirect.redirect.followRedirects, false);
    assert.equal(relay.parameters.rawContentType, tipo);
    assert.equal(relay.parameters.options.response.response.fullResponse, true);
    assert.equal(relay.parameters.options.response.response.responseFormat, 'file');
    assert.equal(relay.parameters.body, '={{ $json.home.corpo }}');
    assert.equal(relay.parameters.url, '={{ $env.APROVA_SLACK_HOME_URL }}');
    assert.deepEqual(relay.parameters.headerParameters.parameters.map(h => h.value),
      ['={{ $json.home.timestamp }}', '={{ $json.home.assinatura }}']);
  });
}
test('roteamento preserva mensagens, challenge, decisões e feedback já publicados', async () => {
  for (const evento of ['message', 'app_mention']) {
    assert.equal((await receber(eventos, { ...abertura, event: { type: evento } })).saida.home, undefined);
  }
  assert.equal((await receber(eventos, { type: 'url_verification', challenge: 'ok' })).saida.home, undefined);
  for (const id of ['aprova_aprovar', 'aprova_feedback_risco_aceito']) {
    assert.equal((await receber(interacoes, { ...clique, view: undefined, actions: [{ action_id: id }] })).saida.home, undefined);
  }
  for (const [id, home] of [['aprova_home_navegacao', true], ['aprova_aprovacao', false], ['aprova_revisao', false]]) {
    const envio = { ...clique, type: 'view_submission', actions: undefined, view: { type: 'modal', callback_id: id } };
    assert.equal(Boolean((await receber(interacoes, envio)).saida.home), home);
  }
});
