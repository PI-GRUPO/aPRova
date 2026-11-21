const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const dir = path.resolve(__dirname, '..');
const chat = JSON.parse(fs.readFileSync(process.env.APROVA_TEST_CHAT || path.join(dir, 'slack-consulta-metricas.json')));
const digest = JSON.parse(fs.readFileSync(process.env.APROVA_TEST_DIGEST || path.join(dir, 'digest-semanal.json')));
const node = (w, p) => w.nodes.find(n => n.name.startsWith(p));
function origin(dm = false) {
  const environment = { APROVA_SLACK_TEAM_ID: 'TTESTTEAM', APROVA_SLACK_APP_ID: 'ATESTAPP', APROVA_SLACK_CANAL_CONSULTAS_ID: 'CTESTCHANNEL' };
  return new Function('$input', '$env', node(chat, '04 |').parameters.jsCode)({ first: () => ({ json: {
    meta: { assinatura_verificada: true }, envelope: { type: 'event_callback', team_id: 'TTESTTEAM', api_app_id: 'ATESTAPP', event_id: 'EvTEST001',
      event: { type: dm ? 'message' : 'app_mention', channel_type: dm ? 'im' : 'channel', channel: dm ? 'DTEST001' : 'CTESTCHANNEL',
        user: 'UTESTUSER', text: 'Resumo semanal?', ts: '1794070000.123456' } },
  } }) }, environment)[0].json;
}
const runChat = (prefix, input, source = origin()) => new Function('$json', '$', node(chat, prefix).parameters.jsCode)(
  input, () => ({ first: () => ({ json: source }) }),
)[0].json;

test('erros e saídas vazias do agente convergem em um caminho sem resposta duplicada', () => {
  const n = node(chat, 'AI Agent');
  assert.equal(n.onError, 'continueRegularOutput'); assert.equal(n.retryOnFail, false); assert.equal(n.alwaysOutputData, true);
  assert.ok(chat.connections[n.name].main.slice(1).every(saida => saida.length === 0));
  assert.deepEqual(chat.connections[n.name].main[0].map(n => n.node), [node(chat, '05R |').name]);
  assert.deepEqual(chat.connections[node(chat, '05R |').name].main[0].map(n => n.node), [node(chat, '05S |').name]);
  assert.deepEqual(chat.connections[node(chat, '05R |').name].main[1].map(n => n.node), [node(chat, '06 |').name]);
  assert.deepEqual(chat.connections[node(chat, '05S |').name].main[0].map(n => n.node), [node(chat, '06 |').name]);
});
for (const dm of [false, true]) {
  for (const [name, input] of [
    ['cota', { error: { statusCode: 429, message: 'RESOURCE_EXHAUSTED segredo-interno-test' } }],
    ['timeout', { error: { code: 'ETIMEDOUT', message: 'segredo-interno-test' } }],
    ['provedor', { error: { statusCode: 503, message: 'segredo-interno-test' } }],
  ]) {
    test(`${dm ? 'DM' : 'canal'}: ${name} recebe aviso seguro no destino autenticado`, () => {
      const o = origin(dm), r = runChat('06 |', { ...input, slack: { channel: 'COUTRO' }, output: 'segredo-interno-test' }, o);
      assert.deepEqual(r.slack, o.slack); assert.equal(r.consulta.status, 'indisponivel');
      assert.match(r.resposta, /Não foi possível responder/); assert.match(r.resposta, /Nenhuma métrica foi confirmada/);
      assert.doesNotMatch(JSON.stringify(r), /segredo-interno-test|RESOURCE_EXHAUSTED|ETIMEDOUT|COUTRO/);
    });
  }
}
test('saída vazia ou inválida e erro acompanhado de output nunca deixam conversa sem aviso', () => {
  for (const input of [{}, { output: '' }, { output: '   ' }, { output: null }, { output: {} },
    { output: ['segredo-interno-test'] }, { error: { message: 'segredo-interno-test' }, output: 'segredo-interno-test' }]) {
    const r = runChat('06 |', input); assert.equal(r.consulta.status, 'indisponivel');
    assert.match(r.resposta, /Tente novamente mais tarde/); assert.doesNotMatch(JSON.stringify(r), /segredo-interno-test/);
  }
});
test('texto válido mantém escape e limite sem permitir destino escolhido pelo modelo', () => {
  const r = runChat('06 |', { output: '<@UOUTRO> & <!channel>', slack: { channel: 'COUTRO' } }, origin(true));
  assert.equal(r.resposta, '&lt;@UOUTRO&gt; &amp; &lt;!channel&gt;'); assert.equal(r.slack.channel, 'DTEST001');
  assert.equal(r.consulta.status, 'resposta_preparada'); assert.equal(runChat('06 |', { output: 'a'.repeat(8000) }).resposta.length, 3500);
});
test('destino ausente, modo/tipo incompatível ou identidade inválida interrompem ambos os ramos', () => {
  for (const change of [o => delete o.slack, o => o.slack.channel = 'COUTRO', o => o.slack.tipo_conversa = 'grupo',
    o => o.slack.team_id = '', o => o.slack.app_id = '', o => o.slack.user = '<@UOUTRO>']) {
    const o = origin(true); change(o);
    assert.throws(() => runChat('06 |', { output: 'Olá' }, o), /Destino Slack/);
  }
});
test('rota de erro preserva ferramentas somente leitura, autenticação, memória e envio sem thread', () => {
  const tools = Object.entries(chat.connections).filter(([, c]) => c.ai_tool).map(([n]) => n).sort();
  assert.deepEqual(tools, ['consultar_prontidao_api', 'consultar_saude_api']);
  assert.equal(chat.connections[node(chat, '02 |').name].main[1][0].node, node(chat, '02E |').name);
  for (const p of ['08A |', '08B |']) {
    const n = node(chat, p); assert.equal(n.parameters.channelId.value, '={{ $json.slack.channel }}');
    assert.equal(n.parameters.otherOptions.thread_ts, undefined); assert.equal(n.retryOnFail, false);
  }
});

const period = { installation_id: 123, inicio: '2026-10-26', fim: '2026-11-02' };
const runDigest = (input, original = period) => new Function('$json', '$', node(digest, '07 |').parameters.jsCode)(
  input, () => ({ item: { json: original } }),
).json;
test('Digest: rede entrega falhas aos validadores, sem redirects ou retries automáticos', () => {
  for (const p of ['04 |', '06 |']) {
    const n = node(digest, p); assert.equal(n.onError, 'continueRegularOutput'); assert.equal(n.retryOnFail, false);
    assert.equal(n.parameters.options.response.response.neverError, true);
    assert.equal(n.parameters.options.redirect.redirect.followRedirects, false); assert.equal(n.parameters.options.timeout, 15000);
  }
  assert.equal(digest.active, false);
});
test('Digest: somente 200 com ok confirma, sem expor corpo ou erros internos', () => {
  for (const key of ['body', 'data']) {
    const r = runDigest({ statusCode: 200, [key]: 'ok' });
    assert.equal(r.status, 'enviado'); assert.equal(r.estado_envio, 'confirmado');
    assert.equal(r.reenviar_automaticamente, false); assert.equal(r.exige_conferencia_manual, false);
  }
});
test('Digest: recusa HTTP conhecida é falha; timeout, 5xx e resposta inesperada são incertos', () => {
  for (const [input, state] of [[{ statusCode: 400, body: 'segredo-interno-test' }, 'falha'],
    [{ statusCode: 403 }, 'falha'], [{ statusCode: 429 }, 'falha'], [{ statusCode: 302 }, 'falha'],
    [{ error: 'segredo-interno-test' }, 'incerto'], [{ statusCode: 408 }, 'incerto'], [{ statusCode: 500, body: 'ok' }, 'incerto'],
    [{ statusCode: 200, body: 'erro' }, 'incerto'], [{ statusCode: 200, body: {} }, 'incerto'],
    [{ statusCode: 200, body: 'ok', error: 'segredo-interno-test' }, 'incerto']]) {
    const r = runDigest(input); assert.equal(r.status, state); assert.equal(r.estado_envio, state);
    assert.equal(r.reenviar_automaticamente, false); assert.equal(r.exige_conferencia_manual, true);
    assert.doesNotMatch(JSON.stringify(r), /segredo-interno-test|requisicao|slack_payload/);
  }
});
test('Digest: relatório indisponível não chega ao envio e a classificação preserva instalação/período', () => {
  const validate = input => new Function('$json', '$', node(digest, '05 |').parameters.jsCode)(input, () => ({ item: { json: period } }));
  for (const input of [{ error: 'timeout' }, { statusCode: 503, body: {} }, { statusCode: 200, body: null }]) assert.throws(() => validate(input));
  const body = { installation_id: 123, inicio: period.inicio + 'T00:00:00Z', fim: period.fim + 'T00:00:00Z',
    total_execucoes: 1, concluidas: 1, rejeitadas: 0, falhas: 0, em_andamento: 0,
    exigiram_aprovacao_humana: 0, revertidas: 0, taxa_sucesso: 1 };
  assert.throws(() => validate({ statusCode: 200, body, error: 'timeout' }));
  const r = runDigest({ error: 'timeout' }); assert.equal(r.installation_id, period.installation_id); assert.equal(r.inicio, period.inicio);
  assert.throws(() => runDigest({ statusCode: 200, body: 'ok' }, {}), /Identidade do Digest/);
});

test('chamada de função malformada não é publicada como resposta útil', () => {
  const n = node(chat, '06 |');
  const origem = {slack:{tipo_conversa:'canal',channel:'C123',team_id:'T123',app_id:'A123',user:'U123'}};
  for (const output of [
    'Calling consultar_relatorio_semanal with input: {"inicio":"2026-11-02"}',
    'Calling consultar_detalhe_execucao{id:\nb390994b-b6b3-456c-832b-236a21ea02e9\n}',
  ]) {
    const [r] = new Function('$json','$',n.parameters.jsCode)({output},()=>({first:()=>({json:origem})}));
    assert.equal(r.json.consulta.status,'indisponivel');assert.match(r.json.resposta,/Não foi possível/);assert.ok(!r.json.resposta.includes('Calling'));
  }
});

test('resposta usa texto sem asteriscos duplos e datas legíveis sem mudar URLs ou IDs', () => {
  const input = '**Última atualização:** 2026-11-15T21:43:22.852007068Z (UTC)\n**Período:** de 2026-11-02 a 2026-11-09 (UTC; fim não incluído)\nID: b390994b-b6b3-456c-832b-236a21ea02e9\nhttps://api.example.test?inicio=2026-11-02';
  const r = runChat('06 |', {output:input});
  assert.match(r.resposta,/Última atualização: 15\/11\/2026 às 18h43 \(Brasília\)/);
  assert.match(r.resposta,/de 02\/11\/2026 a 09\/11\/2026 \(UTC; fim não incluído\)/);
  assert.ok(!r.resposta.includes('**'));assert.ok(!r.resposta.includes('(Brasília) (UTC)'));
  assert.ok(r.resposta.includes('b390994b-b6b3-456c-832b-236a21ea02e9'));
  assert.ok(r.resposta.includes('https://api.example.test?inicio=2026-11-02'));
});

test('conversão de horário respeita offset, virada de dia e calendário inválido', () => {
  for (const [iso,expected] of [
    ['2026-11-16T01:30:00Z','15/11/2026 às 22h30 (Brasília)'],
    ['2026-11-15T13:29:05.17749-03:00','15/11/2026 às 13h29 (Brasília)'],
  ]) assert.equal(runChat('06 |',{output:iso}).resposta,expected);
  assert.equal(runChat('06 |',{output:'2026-02-30'}).resposta,'2026-02-30');
});

test('recuperação ocorre somente para chamada malformada, sem repetir cota ou falha de rede', () => {
  const condition = node(chat,'05R |').parameters.conditions.conditions[0].leftValue;
  const recover = input => new Function('$json','return ('+condition.slice(3,-2)+');')(input);
  for (const output of ['Calling consultar_execucoes with input: {}','Calling comparar_relatorios_semanais{inicio_a:}']) assert.equal(recover({output}),true);
  for (const input of [{output:'Há uma análise pendente.'},{error:{statusCode:429}},{error:{code:'ETIMEDOUT'}},{output:''},{}, {output:null}]) assert.equal(recover(input),false);
});

test('segunda tentativa é isolada, usa a pergunta autenticada e somente as mesmas leituras', () => {
  const fallback=node(chat,'05S |');assert.equal(fallback.onError,'continueRegularOutput');assert.equal(fallback.alwaysOutputData,true);assert.equal(fallback.retryOnFail,false);
  assert.match(fallback.parameters.text,/04 \| Preparar Consulta do Slack/);assert.match(fallback.parameters.text,/chatInput/);assert.ok(!fallback.parameters.text.includes('$json.output'));
  const targets=kind=>Object.entries(chat.connections).filter(([,c])=>c[kind]?.flat().some(t=>t.node===fallback.name)).map(([n])=>n).sort();
  assert.deepEqual(targets('ai_memory'),[]);
  assert.deepEqual(targets('ai_tool'),['consultar_prontidao_api', 'consultar_saude_api']);
  assert.deepEqual(targets('ai_languageModel'),['Google Gemini Chat Model']);
  assert.match(fallback.parameters.options.systemMessage,/acesso individual/);
  assert.deepEqual(chat.connections[fallback.name].main[0].map(n=>n.node),[node(chat,'06 |').name]);
});
