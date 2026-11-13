const test = require('node:test');
const assert = require('node:assert/strict');
const crypto = require('node:crypto');
const workflow = require('../feedback-motivo-aprovacao.json');
const core = require('../entrada-eventos.json');
const central = require('../central-slack.json');
const env = {
  N8N_SEGREDO_ACOES: 'segredo-apenas-do-teste', APROVA_API_URL: 'http://api:8080',
  APROVA_SLACK_TEAM_ID: 'T123', APROVA_SLACK_APP_ID: 'A123',
  APROVA_SLACK_CANAL_CONSULTAS_ID: 'C123',
};
const rodar = (prefixo, json, referencias = {}) => {
  const codigo = workflow.nodes.find(n => n.name.startsWith(prefixo)).parameters.jsCode;
  return new Function('$json', '$env', 'require', 'Buffer', '$', codigo)(
    json, env, require, Buffer, nome => ({ first: () => ({ json: referencias[nome] }) }),
  ).json;
};
const evento = () => ({
  meta: { origem_verificada: true, validacao: { valido: true }, tenant_id: 123,
    execution_id: '11111111-1111-1111-1111-111111111111' },
  evento: { tipo: 'aprovacao_registrada' },
  aprovacao: { id: 7, decisao: 'aprovada', login_github: 'revisor' },
  risco: { nivel_original: 'ALTO' }, repositorio: { full_name: 'equipe/projeto' },
  pr: { numero: 42 }, dados: {},
});
function pergunta() {
  const e = rodar('02 |', evento());
  e.configuracao_feedback.vinculos_slack_github = { U123: 'revisor' };
  return rodar('07 |', rodar('04 |', e));
}
function clique(botao, configuracao) {
  return { meta: { assinatura_verificada: true }, configuracao_feedback: configuracao,
    slack: { type: 'block_actions', team: { id: 'T123' }, api_app_id: 'A123',
      channel: { id: 'C123' }, user: { id: 'U123' },
      actions: [{ ...botao, action_ts: '1234567890.123456' }] } };
}

test('Feedback: export portátil e chamadas preservam autenticação e rotas antigas', () => {
  assert.equal(workflow.active, false);
  assert.deepEqual(workflow.pinData, {});
  assert.equal(workflow.settings.callerPolicy, 'workflowsFromSameOwner');
  for (const n of workflow.nodes) {
    assert.equal(n.credentials, undefined);
    assert.equal(n.webhookId, undefined);
  }
  for (const w of [core, central]) {
    const chamada = w.nodes.find(n => n.parameters.workflowId?.value === '={{ $env.APROVA_FEEDBACK_WORKFLOW_ID }}');
    assert.ok(chamada);
    assert.equal(chamada.parameters.mode, 'each');
    assert.equal(chamada.parameters.options.waitForSubWorkflow, false);
  }
  assert.equal(core.connections['07H | aprovacao_registrada | Conectar workflow'].main[0][0].node,
    '07I | Solicitar Motivo da Aprovação');
  assert.equal(central.connections['07 | Confirmar Recebimento'].main[0][0].node, '07F | É Feedback de Aprovação');
  assert.equal(central.connections['07F | É Feedback de Aprovação'].main[1][0].node, '08 | Separar Tipo de Interação');
});

test('Feedback: sem vínculo não pergunta; somente aprovação ALTO autenticada é elegível', () => {
  const e = rodar('02 |', evento());
  assert.equal(e.configuracao_feedback.api_feedback_habilitada, false);
  assert.equal(e.configuracao_feedback.caminho_api_feedback, '');
  assert.equal(rodar('04 |', e).feedback.status, 'configuracao_pendente');
  for (const alteracao of [{ risco: { nivel_original: 'BAIXO' } },
    { aprovacao: { ...e.aprovacao, decisao: 'alteracoes_solicitadas' } }]) {
    assert.equal(rodar('04 |', { ...e, ...alteracao }).feedback.status, 'ignorado');
  }
  assert.throws(() => rodar('04 |', { ...e, meta: { ...e.meta, origem_verificada: false } }));
});

test('Feedback: os três motivos usam contexto assinado e não são persistidos com API bloqueada', () => {
  const p = pergunta();
  const botoes = p.slack_payload.blocks.find(b => b.type === 'actions').elements;
  assert.deepEqual(botoes.map(b => b.action_id), ['regra_conservadora', 'risco_aceito', 'falso_positivo'].map(m => 'aprova_feedback_' + m));
  for (const b of botoes) {
    const v = rodar('10 |', clique(b, p.configuracao_feedback));
    assert.equal(v.feedback.status, 'resposta_validada');
    assert.equal(v.pedido_feedback.feedback_reason, b.action_id.slice('aprova_feedback_'.length));
    assert.throws(() => rodar('12 |', v));
    const retorno = rodar('14 |', v, { '10 | Validar Clique e Contexto': v });
    assert.equal(retorno.feedback.status, 'pendente_api');
    assert.equal(rodar('99 |', retorno).feedback.registrado_no_nucleo, false);
    const outro = clique(b, p.configuracao_feedback); outro.slack.user.id = 'U456';
    assert.equal(rodar('10 |', outro).feedback.status, 'resposta_recusada');
    const adulterado = clique(b, p.configuracao_feedback);
    const token = JSON.parse(b.value); token.assinatura = '0'.repeat(64);
    adulterado.slack.actions[0].value = JSON.stringify(token);
    assert.equal(rodar('10 |', adulterado).feedback.status, 'resposta_recusada');
  }
});

test('Feedback: corpo assinado é exato; 202 ou aprovação diferente não confirmam persistência', () => {
  const p = pergunta();
  const b = p.slack_payload.blocks.find(b => b.type === 'actions').elements[0];
  const v = rodar('10 |', clique(b, p.configuracao_feedback));
  v.configuracao_feedback = { ...v.configuracao_feedback, api_feedback_habilitada: true,
    caminho_api_feedback: '/feedback-aprovacao' };
  const r = rodar('12 |', v).requisicao_feedback;
  assert.equal(r.corpo, JSON.stringify(v.pedido_feedback));
  assert.equal(r.headers['X-Aprova-Assinatura'], 'sha256=' + crypto.createHmac('sha256', env.N8N_SEGREDO_ACOES)
    .update(r.headers['X-Aprova-Timestamp'] + '.' + r.corpo).digest('hex'));
  const resposta = { statusCode: 200, body: { status: 'registrado', installation_id: 123,
    id_aprovacao: 7, feedback_reason: v.pedido_feedback.feedback_reason } };
  const refs = { '10 | Validar Clique e Contexto': v };
  assert.equal(rodar('14 |', resposta, refs).feedback.status, 'registrado');
  assert.equal(rodar('14 |', { ...resposta, statusCode: 202 }, refs).feedback.status, 'nao_confirmado');
  assert.equal(rodar('14 |', { ...resposta, body: { ...resposta.body, id_aprovacao: 8 } }, refs).feedback.status, 'nao_confirmado');
  v.configuracao_feedback.caminho_api_feedback = '/acoes-remotas';
  assert.throws(() => rodar('12 |', v));
});
