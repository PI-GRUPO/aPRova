const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const crypto = require('node:crypto');

const diretorio = path.resolve(__dirname, '..');
const central = JSON.parse(fs.readFileSync(path.join(diretorio, 'central-slack.json')));
const entrada = JSON.parse(fs.readFileSync(path.join(diretorio, 'entrada-eventos.json')));
const codigo = (workflow, prefixo) => workflow.nodes.find(n => n.name.startsWith(prefixo)).parameters.jsCode;
const rodar = (prefixo, json, origem = json) => new Function('$json', '$', codigo(central, prefixo))(
  json, () => ({ item: { json: origem } }),
);
const AsyncFunction = Object.getPrototypeOf(async function () {}).constructor;
const verificarEvento = new AsyncFunction('require', '$input', '$env', codigo(entrada, '01B |'));

function formulario(tipo, justificativa = 'Risco avaliado com testes.\nSem pendências.') {
  return {
    meta: { assinatura_verificada: true },
    slack: {
      type: 'view_submission', team: { id: 'T123' }, user: { id: 'U123', name: 'revisor' },
      view: {
        id: 'V123', callback_id: tipo === 'aprovar' ? 'aprova_aprovacao' : 'aprova_revisao',
        private_metadata: JSON.stringify({
          tipo, usuario_slack: 'U123', installation_id: 123, repositorio: 'equipe/projeto',
          numero_pr: 42, canal_slack: 'C123', task_run_id: 'c41d7e9a-2f3b-4c8d-a5e6-0b1c2d3e4f5a',
        }),
        state: { values: { justificativa: { texto: { value: justificativa } } } },
      },
    },
  };
}

for (const tipo of ['aprovar', 'solicitar_alteracao']) {
  test(`formulário ${tipo}: prepara contrato e publica resultado conforme o núcleo`, () => {
    const original = formulario(tipo);
    const pedido = rodar('11 |', original).json;
    assert.equal(pedido.pedido_acao.tipo, tipo);
    assert.equal(pedido.pedido_acao.task_run_id, JSON.parse(original.slack.view.private_metadata).task_run_id);
    assert.equal(pedido.pedido_acao.justificativa, original.slack.view.state.values.justificativa.texto.value);
    assert.equal(pedido.pedido_acao.usuario_slack, 'U123');
    assert.equal(pedido.pedido_acao.login_github, undefined);
    const assinado = new Function('require', '$json', '$env', codigo(central, '12 |'))(
      require, pedido, { N8N_SEGREDO_ACOES: 'segredo-de-teste' },
    ).json.requisicao_nucleo;
    assert.equal(assinado.corpo, JSON.stringify(pedido.pedido_acao));
    const timestamp = assinado.headers['X-Aprova-Timestamp'];
    const esperada = 'sha256=' + crypto.createHmac('sha256', 'segredo-de-teste')
      .update(timestamp + '.').update(assinado.corpo).digest('hex');
    assert.equal(assinado.headers['X-Aprova-Assinatura'], esperada);

    for (const status of [202, 400, 401, 403, 409, 500]) {
      const resultado = rodar('14 |', { statusCode: status, body: { status: 'aceito' } }, pedido).json;
      assert.equal(resultado.resultado_nucleo.confirmada, status === 202);
      assert.equal(resultado.slack_payload.user, original.slack.user.id);
      assert.equal(resultado.slack_payload.channel, 'C123');
      const [retorno] = central.connections['14 | Preparar Retorno ao Usuário'].main[0];
      assert.equal(central.nodes.find(n => n.name === retorno.node).parameters.url, 'https://slack.com/api/chat.postEphemeral');
      assert.match(resultado.slack_payload.text, /equipe\/projeto #42/);
      assert.match(resultado.slack_payload.text, /https:\/\/github.com\/equipe\/projeto\/pull\/42/);
      assert.ok(resultado.slack_payload.text.includes(pedido.pedido_acao.justificativa));
      assert.ok(resultado.slack_payload.blocks.some(b => b.text.text.includes('<@U123>')));
      assert.ok(resultado.slack_payload.blocks.every(b => b.text.text.length <= 3000));
      if (status !== 202) assert.doesNotMatch(resultado.slack_payload.text, /O núcleo confirmou/);
    }
    for (const resposta of [{ error: 'ECONNREFUSED' }, { statusCode: 202, body: 'invalid json' }, { statusCode: 202, body: { status: 'pendente' } }]) {
      const resultado = rodar('14 |', resposta, pedido).json;
      assert.equal(resultado.resultado_nucleo.confirmada, false);
      assert.match(resultado.slack_payload.text, /não está confirmada/);
    }
  });

  test(`formulário ${tipo}: recusa identidade trocada e justificativa vazia`, () => {
    const trocado = formulario(tipo);
    trocado.slack.user.id = 'U456';
    assert.throws(() => rodar('11 |', trocado), /identidade incompatível/);
    assert.throws(() => rodar('11 |', formulario(tipo, ' \n ')), /Informe uma justificativa/);
  });

  test(`formulário ${tipo}: preserva justificativa máxima sem interpretar marcação`, () => {
    const texto = '<@U456>'.repeat(428) + 'fim';
    const pedido = rodar('11 |', formulario(tipo, texto)).json;
    const resultado = rodar('14 |', pedido).json;
    assert.ok(resultado.slack_payload.text.includes(texto));
    const blocos = resultado.slack_payload.blocks.filter(b => b.text.type === 'plain_text');
    assert.ok(blocos.every(b => b.text.text.length <= 3000));
    assert.ok(blocos.map(b => b.text.text).join('').includes(texto));
  });
}

test('clique antigo em reexecução informa indisponibilidade sem chegar ao núcleo', () => {
  const destino = central.connections['09 | Identificar Botão'].main[2][0].node;
  const no = central.nodes.find(n => n.name === destino);
  assert.equal(no.parameters.url, 'https://slack.com/api/views.open');
  assert.match(no.parameters.jsonBody, /Nenhum pedido foi enviado/);
  assert.equal(central.connections[destino], undefined);
  assert.equal(no.retryOnFail, false);
});

test('aprovação reúne motivo e justificativa no contrato existente', () => {
  const entrada = formulario('aprovar');
  entrada.slack.view.callback_id = 'aprova_aprovacao_motivo';
  for (const [valor, texto] of [['risco_aceito', 'Risco aceito'], ['falso_positivo', 'Falso positivo'], ['regra_conservadora', 'Regra conservadora']]) {
    entrada.slack.view.state.values.motivo = { escolha: { selected_option: { value: valor } } };
    const pedido = rodar('11 |', entrada).json.pedido_acao;
    assert.equal(pedido.justificativa, 'Motivo: ' + texto + '\n\n' + entrada.slack.view.state.values.justificativa.texto.value);
    assert.equal(pedido.feedback_reason, undefined);
  }
  entrada.slack.view.state.values.motivo.escolha.selected_option.value = 'externo';
  assert.throws(() => rodar('11 |', entrada), /motivo/);
});

function eventoAssinado(corpo, segundos = Math.floor(Date.now() / 1000), segredo = 'segredo-de-teste') {
  const bruto = Buffer.from(corpo);
  const assinatura = 'sha256=' + crypto.createHmac('sha256', segredo).update(segundos + '.').update(bruto).digest('hex');
  return { bruto, item: { json: { headers: { 'x-aprova-timestamp': String(segundos), 'x-aprova-assinatura': assinatura } }, binary: { data: {} } } };
}

function executarVerificacao(evento, segredo = 'segredo-de-teste') {
  return verificarEvento.call({ helpers: { getBinaryDataBuffer: async () => evento.bruto } },
    require, { first: () => evento.item }, { N8N_SEGREDO_EVENTOS: segredo });
}

test('assinatura usa corpo bruto e marca a origem como verificada', async () => {
  const evento = eventoAssinado('{ "tenant_id": 123, "tipo_evento": "aguardando_aprovacao" }');
  const resultado = await executarVerificacao(evento);
  assert.equal(resultado[0].json.body.tenant_id, 123);
  assert.equal(resultado[0].json.meta.origem_verificada, true);
});

test('assinatura recusa adulteração, segredo errado, ausência e replay', async () => {
  const evento = eventoAssinado('{"tenant_id":123}');
  await assert.rejects(executarVerificacao(evento, 'outro-segredo'), /Assinatura do núcleo inválida/);
  evento.bruto = Buffer.from('{"tenant_id":456}');
  await assert.rejects(executarVerificacao(evento), /Assinatura do núcleo inválida/);
  await assert.rejects(executarVerificacao(evento, ''), /não configurado/);
  for (const diferenca of [-360, 360]) {
    await assert.rejects(executarVerificacao(eventoAssinado('{}', Math.floor(Date.now() / 1000) + diferenca)), /janela de cinco minutos/);
  }
  delete evento.item.json.headers['x-aprova-assinatura'];
  await assert.rejects(executarVerificacao(evento), /Cabeçalhos/);
});

test('exports não carregam credenciais, segredos ou dados fixados', () => {
  for (const workflow of [entrada, central]) {
    const texto = JSON.stringify(workflow);
    assert.doesNotMatch(texto, /https:\/\/hooks\.slack\.com\/services\//);
    assert.deepEqual(workflow.pinData, {});
    assert.ok(workflow.nodes.every(n => !n.credentials));
  }
});

for (const [nome, prefixo] of [['aguardando_aprovacao', '03A |'], ['falha_recuperavel', '03B |']]) {
  test(`envelope ${nome} do repositório chega à mensagem correta sem recalcular risco`, () => {
    const original = JSON.parse(fs.readFileSync(path.join(diretorio, '..', 'exemplos', nome + '.json')));
    const normalizado = new Function('$json', '$execution', codigo(entrada, '04 |'))({
      original, meta: { recebido_em: new Date().toISOString(), origem_verificada: true },
    }, { id: 'teste' }).json;
    assert.equal(normalizado.meta.origem_verificada, true);
    assert.equal(normalizado.meta.validacao.valido, true);
    const mensagem = rodar(prefixo, normalizado).json.slack_payload;
    assert.ok(mensagem.text.includes(original.dados.repositorio));
    const acoes = mensagem.blocks.find(b => b.type === 'actions')?.elements ?? [];
    assert.deepEqual(acoes.map(a => a.action_id), nome === 'aguardando_aprovacao'
      ? ['aprovar', 'solicitar_alteracao'] : []);
    if (acoes.length) assert.equal(JSON.parse(acoes[0].value).task_run_id, original.dados.task_run_id);
    const texto = mensagem.blocks.filter(b => b.type === 'section').map(b => b.text.text).join('\n');
    if (nome === 'falha_recuperavel') assert.ok(texto.includes(original.dados.diagnostico.slice(0, 2800)));
    else assert.ok(texto.includes(original.dados.autor_pr));
  });
}

function alertaComRecibo(url) {
  const original = JSON.parse(fs.readFileSync(path.join(diretorio, '..', 'exemplos', 'aguardando_aprovacao.json')));
  if (url === undefined) delete original.dados.url_recibo;
  else original.dados.url_recibo = url;
  const evento = new Function('$json', '$execution', codigo(entrada, '04 |'))({
    original, meta: { origem_verificada: true },
  }, { id: 'teste' }).json;
  return { evento, resultado: rodar('03A |', evento).json };
}

test('alerta usa a URL pública do recibo emitido pelo worker e preserva decisões e risco', () => {
  const id = '8e0a2c4d-6f1b-4d3e-a5c7-9b0d1e2f3a4b';
  for (const base of ['https://aprova.example.com', 'https://aprova.example.com/api', 'http://localhost:8080']) {
    for (const sufixo of ['', '/verify', '/verify/']) {
      const url = `${base}/receipts/${id}`;
      const { evento, resultado } = alertaComRecibo(url + sufixo);
      const grupos = resultado.slack_payload.blocks.filter(b => b.type === 'actions');
      assert.deepEqual(grupos[0].elements.map(b => b.action_id), ['aprovar', 'solicitar_alteracao']);
      assert.deepEqual(JSON.parse(grupos[0].elements[0].value), {
        installation_id: evento.meta.tenant_id, repositorio: evento.repositorio.full_name,
        numero_pr: evento.pr.numero, id_evento: evento.meta.id_evento,
        task_run_id: evento.meta.execution_id,
        osv_resumo: 'OSV: resultado ainda não disponível para esta análise. Isso não indica ausência de vulnerabilidades.',
      });
      assert.deepEqual(grupos[1].elements.map(b => [b.action_id, b.url]), [
        ['abrir_pr', `https://github.com/${evento.repositorio.full_name}/pull/${evento.pr.numero}`],
        ['abrir_recibo', url], ['verificar_recibo', url + '/verify'],
      ]);
      assert.deepEqual(resultado.risco, evento.risco);
      assert.equal(resultado.slack_payload.unfurl_links, false);
      assert.ok(resultado.slack_payload.text.includes(url + '/verify'));
      assert.doesNotMatch(resultado.slack_payload.text, /recibo (válido|verificado)/i);
    }
  }
});

test('recibo ausente ou incompatível não gera link de verificação nem bloqueia o alerta', () => {
  const id = '8e0a2c4d-6f1b-4d3e-a5c7-9b0d1e2f3a4b';
  for (const url of [undefined, '', '/receipts/' + id, 'javascript:alert(1)',
    `https://usuario:senha@aprova.example.com/receipts/${id}/verify`,
    `https://aprova.example.com/receipts/${id}/verify/verify`,
    `https://aprova.example.com/receipts/${id}/verify?token=segredo`,
    `https://aprova.example.com/receipts/${id}#trecho`,
    'https://aprova.example.com/receipts/invalido/verify']) {
    const { resultado } = alertaComRecibo(url);
    const grupos = resultado.slack_payload.blocks.filter(b => b.type === 'actions');
    assert.equal(grupos[0].elements.length, 2);
    assert.deepEqual(grupos[1].elements.map(b => b.action_id), ['abrir_pr']);
    assert.match(resultado.slack_payload.text, /Recibo indisponível/);
    assert.doesNotMatch(resultado.slack_payload.text, /token=|usuario:senha|javascript:/);
  }
});

test('erro na entrega privada não retorna ao caminho de ação', () => {
  assert.throws(() => rodar('16 |', {ok: false, error: 'channel_not_found'}), /não confirmou/);
  assert.equal(rodar('16 |', {ok: true}).json.entrega_privada, 'confirmada');
  const validar = central.nodes.find(n => n.name.startsWith('16 |'));
  const atualiza = central.connections[validar.name].main[0][0].node;
  assert.equal(central.connections[atualiza].main[0][0].node, 'H02 | Encaminhar à App Home');
  const roteia = new Function('$', '$env', codigo(central, 'H03 |'));
  const resultado = { resultado_nucleo: {confirmada: true}, atualizacao_home: {corpo: 'fixture assinada'} };
  const refs = () => ({first: () => ({json: resultado})});
  assert.deepEqual(roteia(refs, {}), []);
  assert.deepEqual(roteia(refs, {APROVA_SLACK_HOME_URL:'http://home.test'}), [{json:{home:resultado.atualizacao_home}}]);
  resultado.resultado_nucleo.confirmada = false;
  assert.deepEqual(roteia(refs, {APROVA_SLACK_HOME_URL:'http://home.test'}), []);
});
