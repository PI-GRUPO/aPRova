const test = require('node:test');
const assert = require('node:assert/strict');
const crypto = require('node:crypto');
const chat = require('../slack-consulta-metricas.json');
const consulta = require('../consulta-relatorio-semanal.json');
const no = (workflow, prefixo) => workflow.nodes.find(n => n.name.startsWith(prefixo));
const codigo = (workflow, prefixo) => no(workflow, prefixo).parameters.jsCode;
const ambiente = {
  APROVA_SLACK_TEAM_ID: 'TTESTTEAM', APROVA_SLACK_APP_ID: 'ATESTAPP',
  APROVA_SLACK_CANAL_CONSULTAS_ID: 'CTESTCHANNEL',
};
const evento = (dm = false) => ({
  type: 'event_callback', team_id: ambiente.APROVA_SLACK_TEAM_ID,
  api_app_id: ambiente.APROVA_SLACK_APP_ID, event_id: 'EvTEST001',
  event: { type: dm ? 'message' : 'app_mention', channel_type: dm ? 'im' : 'channel',
    channel: dm ? 'DTEST001' : ambiente.APROVA_SLACK_CANAL_CONSULTAS_ID,
    user: 'UTESTUSER', text: dm ? 'Resumo semanal?' : '<@UTESTBOT> Resumo semanal?',
    ts: '1794070000.123456' },
});
const preparar = (e = evento(), env = ambiente, verificado = true) => new Function('$input', '$env', codigo(chat, '04 |'))(
  { first: () => ({ json: { meta: { assinatura_verificada: verificado }, envelope: e } }) }, env,
);
const periodo = { installation_id: 123, inicio: '2026-10-26', fim: '2026-11-02' };
const metricas = () => ({ ...periodo, inicio: periodo.inicio + 'T00:00:00Z', fim: periodo.fim + 'T00:00:00Z',
  total_execucoes: 10, concluidas: 6, rejeitadas: 1, falhas: 1, em_andamento: 2,
  taxa_sucesso: 0.75, exigiram_aprovacao_humana: 3, risco_medio: 2, revertidas: 1,
});
const validarRelatorio = resposta => new Function('$json', '$', codigo(consulta, '05 |'))(
  resposta, () => ({ item: { json: periodo } }),
).json;

test('consultas: exports portáteis não têm simulação, credenciais, dados fixados ou referências locais', () => {
  assert.equal(chat.nodes.length, 18);
  assert.equal(consulta.nodes.length, 6);
  for (const workflow of [chat, consulta]) {
    assert.equal(workflow.active, false);
    assert.deepEqual(workflow.pinData, {});
    assert.equal(/simulad|simula[çc][aã]o|APROVA_CHAT_SIMULACAO/i.test(JSON.stringify(workflow)), false);
    const nomes = new Set(workflow.nodes.map(n => n.name));
    for (const n of workflow.nodes) assert.equal(n.credentials, undefined);
    for (const [origem, tipos] of Object.entries(workflow.connections)) {
      assert.ok(nomes.has(origem));
      for (const saidas of Object.values(tipos)) for (const grupo of saidas) for (const destino of grupo) {
        assert.ok(nomes.has(destino.node));
      }
    }
  }
  const ferramenta = no(chat, 'consultar_relatorio_semanal');
  assert.equal(ferramenta.parameters.workflowId.value, '={{ $env.APROVA_CONSULTA_SEMANAL_WORKFLOW_ID }}');
  assert.deepEqual(Object.keys(ferramenta.parameters.workflowInputs.value), ['inicio']);
});

test('Slack: exige configuração externa e autenticação antes de processar a pergunta', () => {
  assert.throws(() => preparar(evento(), {}), /Configure APROVA_SLACK/);
  assert.throws(() => preparar(evento(), ambiente, false), /não autenticada/);
  for (const alterar of [e => e.team_id = 'TOUTRO', e => e.api_app_id = 'AOUTRO', e => e.event.channel = 'COUTRO']) {
    const e = evento(); alterar(e); assert.deepEqual(preparar(e), []);
  }
});

test('Slack: DM contínua sem menção e canal somente com app_mention', () => {
  const dm = preparar(evento(true))[0].json;
  assert.equal(dm.slack.tipo_conversa, 'dm');
  assert.equal(dm.chatInput, 'Resumo semanal?');
  const e = evento(true); e.event.ts = '1794070100.654321'; e.event.text = 'E as falhas?';
  assert.equal(preparar(e)[0].json.sessionId, dm.sessionId);
  const canal = preparar()[0].json;
  assert.equal(canal.slack.tipo_conversa, 'canal');
  assert.equal(canal.chatInput, 'Resumo semanal?');
  e.event.channel_type = 'mpim'; assert.deepEqual(preparar(e), []);
  const semMencao = evento(); semMencao.event.type = 'message';
  assert.deepEqual(preparar(semMencao), []);
});

test('Slack: memória separa DM, canal e usuários; deduplicação preserva a identidade do evento', () => {
  const dm = preparar(evento(true))[0].json;
  const canal = preparar()[0].json;
  assert.notEqual(dm.sessionId, canal.sessionId);
  const outro = evento(true); outro.event.user = 'UOUTRO';
  assert.notEqual(preparar(outro)[0].json.sessionId, dm.sessionId);
  assert.equal(preparar()[0].json.dedupe_key, canal.dedupe_key);
  assert.equal(no(chat, '05 |').parameters.operation, 'removeItemsSeenInPreviousExecutions');
  assert.equal(no(chat, '05 |').parameters.options.historySize, 10000);
  assert.match(no(chat, 'Simple Memory').parameters.sessionKey, /consultas-reais-v1:/);
});

test('Slack: bots, mensagens vazias e eventos inválidos não acionam a IA', () => {
  for (const alterar of [e => e.event.bot_id = 'BTEST', e => e.event.subtype = 'bot_message', e => e.event.text = '']) {
    const e = evento(true); alterar(e); assert.deepEqual(preparar(e), []);
  }
  for (const alterar of [e => e.event_id = '', e => e.event.ts = 'inválido', e => e.event.text = 'x'.repeat(6001)]) {
    const e = evento(true); alterar(e); assert.throws(() => preparar(e));
  }
});

test('Slack: HMAC verifica corpo bruto, segredo, replay e challenge sem acionar o agente', async () => {
  const AsyncFunction = Object.getPrototypeOf(async function () {}).constructor;
  const executar = new AsyncFunction('require', '$input', '$env', 'Buffer', codigo(chat, '02 |'));
  async function assinar(body, { segredo = 'segredo-teste', antigo = false, adulterar = false } = {}) {
    const bruto = Buffer.from(JSON.stringify(body));
    const ts = Math.floor(Date.now() / 1000 - (antigo ? 600 : 0)).toString();
    const assinatura = 'v0=' + crypto.createHmac('sha256', 'segredo-teste').update('v0:' + ts + ':').update(bruto).digest('hex');
    return executar.call({ helpers: { getBinaryDataBuffer: async () => adulterar ? Buffer.from('{}') : bruto } },
      require, { first: () => ({ json: { headers: { 'x-slack-request-timestamp': ts, 'x-slack-signature': assinatura } }, binary: { data: {} } }) },
      { SLACK_SIGNING_SECRET: segredo }, Buffer);
  }
  assert.equal((await assinar(evento()))[0].json.meta.assinatura_verificada, true);
  for (const opcoes of [{ segredo: 'outro' }, { antigo: true }, { adulterar: true }]) {
    await assert.rejects(() => assinar(evento(), opcoes));
  }
  assert.equal((await assinar({ type: 'url_verification', challenge: 'teste' }))[0].json.envelope.challenge, 'teste');
  assert.equal(chat.connections['03 | É Verificação de URL'].main[0][0].node, '03C | Responder Challenge do Slack');
  assert.equal(chat.connections['02 | Verificar Assinatura do Slack'].main[1][0].node, '02E | Recusar Requisição Não Autenticada');
});

test('Slack: resposta usa destino autenticado, escapa menções e não usa thread nos dois modos', () => {
  const origem = preparar(evento(true))[0].json;
  const r = new Function('$json', '$', codigo(chat, '06 |'))(
    { output: 'Resposta <@UOUTRO> & <!channel>', slack: { channel: 'COUTRO' } },
    () => ({ first: () => ({ json: origem }) }),
  )[0].json;
  assert.equal(r.slack.channel, 'DTEST001');
  assert.equal(r.resposta, 'Resposta &lt;@UOUTRO&gt; &amp; &lt;!channel&gt;');
  for (const prefixo of ['08A |', '08B |']) assert.equal(no(chat, prefixo).parameters.otherOptions.thread_ts, undefined);
});

test('agente: expõe só três consultas e nenhum destino HTTP controlado pelo modelo', () => {
  assert.deepEqual(chat.nodes.filter(n => chat.connections[n.name]?.ai_tool).map(n => n.name).sort(),
    ['consultar_prontidao_api', 'consultar_relatorio_semanal', 'consultar_saude_api']);
  for (const [nome, caminho] of [['consultar_saude_api', '/health'], ['consultar_prontidao_api', '/ready']]) {
    const p = no(chat, nome).parameters;
    assert.equal(p.method, 'GET'); assert.ok(p.url.includes(caminho));
    assert.equal(p.url.includes('$fromAI'), false);
    assert.equal(p.options.redirect.redirect.followRedirects, false);
  }
  assert.equal(no(consulta, '04 |').parameters.method, 'GET');
});

test('consulta: instalação ausente bloqueia e data inválida não chega à assinatura', () => {
  const validar = new Function('$json', codigo(consulta, '02 |'));
  for (const installation_id of [0, -1, '123', 1.5]) assert.throws(() => validar({ installation_id, inicio: '' }));
  for (const inicio of ['2026-02-30', '2026-10-26&installation_id=999', 123]) {
    assert.throws(() => validar({ installation_id: 123, inicio }));
  }
  assert.deepEqual(validar({ installation_id: 123, inicio: '2026-10-26' }).json, periodo);
  assert.equal(consulta.connections['01 | Configurar Instalação'].main[0][0].node, '02 | Validar Consulta');
});

test('consulta: assinatura cobre exatamente a query e não retorna o segredo', () => {
  const r = new Function('require', '$json', '$env', codigo(consulta, '03 |'))(require, periodo,
    { N8N_SEGREDO_ACOES: 'segredo-teste', APROVA_API_URL: 'http://api:8080/' }).json;
  const url = new URL(r.requisicao.url);
  const ts = r.requisicao.headers['X-Aprova-Timestamp'];
  assert.equal(url.search, '?installation_id=123&inicio=2026-10-26');
  assert.equal(r.requisicao.headers['X-Aprova-Assinatura'], 'sha256=' + crypto.createHmac('sha256', 'segredo-teste').update(ts + '.' + url.search.slice(1)).digest('hex'));
  assert.equal(JSON.stringify(r).includes('segredo-teste'), false);
});

test('consulta: valida contrato real, preserva semana vazia e distingue falha de zero', () => {
  const ok = validarRelatorio({ statusCode: 200, body: metricas() });
  assert.equal(ok.status, 'disponivel'); assert.equal(ok.dados.taxa_sucesso, 0.75);
  const vazio = metricas();
  for (const k of Object.keys(vazio)) if (typeof vazio[k] === 'number' && k !== 'installation_id') vazio[k] = 0;
  vazio.taxa_sucesso = null;
  assert.equal(validarRelatorio({ statusCode: 200, body: vazio }).status, 'disponivel');
  for (const r of [{ error: 'timeout' }, { statusCode: 503 }, { statusCode: 200, body: { ...metricas(), installation_id: 999 } },
    { statusCode: 200, body: { ...metricas(), total_execucoes: 11 } }, { statusCode: 200, body: { ...metricas(), taxa_sucesso: 0.9 } }]) {
    const erro = validarRelatorio(r); assert.equal(erro.status, 'indisponivel'); assert.equal(erro.dados, null);
  }
  assert.equal(ok.requisicao, undefined);
});
