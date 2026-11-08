const test = require('node:test');
const assert = require('node:assert/strict');
const workflow = require('../onboarding-repositorio.json');
const core = require('../entrada-eventos.json');
const exemplo = require('../../exemplos/instalacao.json');
const no = prefixo => workflow.nodes.find(n => n.name.startsWith(prefixo));
const preparar = e => new Function('$input', no('02 |').parameters.jsCode)({ all: () => [{ json: e }] });
function evento() {
  const normalizador = core.nodes.find(n => n.name === '04 | Normalizar Evento aPRova');
  return new Function('$json', '$execution', normalizador.parameters.jsCode)(
    { original: exemplo, meta: { origem_verificada: true } }, { id: 'teste-local' },
  ).json;
}
function rodar(prefixo, json, contexto) {
  return new Function('$json', '$', no(prefixo).parameters.jsCode)(json,
    () => ({ item: { json: contexto } })).json;
}
const fonteClassificador = no('08 |').parameters.jsCode.split('\nconst c=')[0];
const classificar = new Function(fonteClassificador + '\nreturn classificar;')();
const configCheck = { check_nome: 'check-apenas-do-teste', app_id: 123 };
const evidencia = () => ({ branch: 'main', status_http: 200, simulado: false,
  protecao: { required_status_checks: { checks: [{ context: configCheck.check_nome, app_id: 123 }] },
    enforce_admins: { enabled: true } } });

test('Onboarding: export portátil e integração explícita no Core', () => {
  assert.equal(workflow.active, false);
  assert.deepEqual(workflow.pinData, {});
  assert.equal(workflow.settings.callerPolicy, 'workflowsFromSameOwner');
  assert.equal(workflow.id, undefined);
  assert.equal(workflow.versionId, undefined);
  for (const n of workflow.nodes) {
    assert.equal(n.credentials, undefined);
    assert.equal(n.webhookId, undefined);
    assert.notEqual(n.type, 'n8n-nodes-base.manualTrigger');
    assert.notEqual(n.type, 'n8n-nodes-base.webhook');
  }
  const c = preparar(evento())[0].json.config;
  assert.equal(c.trello_habilitado, false);
  assert.equal(c.slack_habilitado, false);
  assert.equal(c.quadro_id, 'CONFIGURE_ID_QUADRO_TRELLO');
  assert.equal(c.slack_canal_id, 'CONFIGURE_ID_CANAL_SLACK');
  const chamada = core.nodes.find(n => n.name === '07L | Onboarding de Repositório');
  assert.equal(chamada.parameters.workflowId.value, 'CONFIGURE_WORKFLOW_ONBOARDING');
  assert.equal(chamada.parameters.mode, 'each');
  assert.equal(chamada.parameters.options.waitForSubWorkflow, true);
  assert.equal(core.connections['07K | instalacao | Conectar workflow'].main[0][0].node, chamada.name);
  assert.equal(no('03 |').parameters.batchSize, 1);
  assert.equal(workflow.connections[no('19 |').name].main[0][0].node, no('03 |').name);
});

test('Onboarding: consome normalizador real, separa repositórios e valida antes de efeitos', () => {
  const e = evento();
  assert.equal(preparar(e)[0].json.repositorio, exemplo.dados.repositorios[0]);
  e.instalacao.repositorios = ['equipe/um', 'equipe/dois', 'equipe/um'];
  assert.deepEqual(preparar(e).map(i => i.json.repositorio), ['equipe/um', 'equipe/dois']);
  e.instalacao.repositorios.push('invalido');
  assert.throws(() => preparar(e));
  assert.throws(() => preparar({ ...evento(), meta: { ...evento().meta, origem_verificada: false } }));
});

test('Onboarding: adições são aceitas; remoções e listas vazias encerram sem efeitos', () => {
  const e = evento();
  e.evento.acao = e.instalacao.acao = 'repositorios_adicionados';
  assert.equal(preparar(e).length, 1);
  e.instalacao.repositorios = [];
  assert.deepEqual(preparar(e), []);
  for (const acao of ['removido', 'repositorios_removidos']) {
    const removido = evento();
    removido.evento.acao = removido.instalacao.acao = acao;
    assert.deepEqual(preparar(removido), []);
  }
});

test('Onboarding: branch vem do GitHub, sem fallback presumido', () => {
  const c = preparar(evento())[0].json;
  assert.equal(rodar('05 |', { statusCode: 200, body: { full_name: c.repositorio, default_branch: 'release' } }, c).branch, 'release');
  assert.equal(rodar('05 |', { statusCode: 404 }, c).branch, null);
  assert.equal(rodar('05 |', { statusCode: 200, body: { full_name: 'outra/conta', default_branch: 'main' } }, c).branch, null);
});

test('Onboarding: proteção confirmada exige identidade do Check e administradores', () => {
  assert.equal(classificar(evidencia(), configCheck).pronto_fluxo_protegido, true);
  assert.equal(classificar(evidencia(), { ...configCheck, app_id: 456 }).status, 'check_obrigatorio_pendente');
  const e = evidencia();
  e.protecao.required_status_checks = null;
  assert.equal(classificar(e, configCheck).status, 'check_obrigatorio_pendente');
  const admin = evidencia();
  admin.protecao.enforce_admins.enabled = false;
  assert.equal(classificar(admin, configCheck).status, 'check_obrigatorio_pendente');
});

test('Onboarding: falhas, nome indefinido e simulação nunca confirmam proteção', () => {
  for (const status_http of [401, 403, 404, 429, 500, null]) {
    const p = classificar({ ...evidencia(), status_http }, configCheck);
    assert.equal(p.status, 'verificacao_indisponivel');
    assert.equal(p.pronto_fluxo_protegido, false);
  }
  assert.equal(classificar(evidencia(), { check_nome: '', app_id: null }).status, 'verificacao_indisponivel');
  assert.equal(classificar({ ...evidencia(), simulado: true }, configCheck).pronto_fluxo_protegido, false);
  assert.equal(classificar({ ...evidencia(), protecao: {} }, configCheck).status, 'verificacao_indisponivel');
  const c = { ...preparar(evento())[0].json, branch: 'main' };
  const real = rodar('08 |', { statusCode: 401, body: {}, protecao: { status: 'protecao_confirmada' } }, c);
  assert.equal(real.protecao.pronto_fluxo_protegido, false);
});

test('Onboarding: Trello reutiliza, impede duplicata arquivada/ambígua e recusa outro quadro', () => {
  const c = preparar(evento())[0].json;
  const quadro = { id: c.config.quadro_id, name: c.config.quadro_nome, closed: false, lists: [] };
  const buscar = body => rodar('11 |', { statusCode: 200, body }, c).trello;
  assert.equal(buscar(quadro).status, 'criar');
  const lista = { id: 'lista-do-teste', idBoard: quadro.id, name: c.repositorio, closed: false };
  assert.equal(buscar({ ...quadro, lists: [lista] }).status, 'reutilizada');
  assert.equal(buscar({ ...quadro, lists: [{ ...lista, closed: true }] }).status, 'pendente');
  assert.equal(buscar({ ...quadro, lists: [lista, { ...lista, id: 'outra' }] }).status, 'pendente');
  assert.equal(buscar({ ...quadro, id: 'outro' }).status, 'indisponivel');
  assert.equal(rodar('11 |', { statusCode: 500 }, c).trello.status, 'indisponivel');
});

test('Onboarding: retorno não confirmado não vira sucesso Trello ou Slack', () => {
  const c = { ...preparar(evento())[0].json, branch: 'main', protecao: classificar({ status_http: 401 }, configCheck) };
  const falha = rodar('14 |', { error: 'timeout' }, c);
  assert.equal(falha.trello.status, 'criacao_nao_confirmada');
  const resumo = rodar('15 |', falha);
  assert.match(resumo.resumo_slack, /Verificação indisponível/);
  assert.match(resumo.resumo_slack, /Ação necessária:/);
  assert.equal(rodar('18 |', { error: 'timeout' }, resumo).notificacao_slack.status, 'nao_confirmada');
});
