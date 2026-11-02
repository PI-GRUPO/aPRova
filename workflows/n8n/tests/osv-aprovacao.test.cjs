const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');

const raiz = path.resolve(__dirname, '..');
const central = JSON.parse(fs.readFileSync(path.join(raiz, 'central-slack.json')));
const osv = JSON.parse(fs.readFileSync(path.join(raiz, 'dependencias-osv.json')));
const codigo = (w, prefixo) => w.nodes.find(n => n.name.startsWith(prefixo)).parameters.jsCode;
const executar = (w, prefixo, json, origem = json) => new Function('$json', '$', codigo(w, prefixo))(
  json, () => ({ item: { json: origem } }),
).json;
const expressao = (texto, json) => new Function('$json', 'return (' + texto.slice(3, -2).trim() + ');')(json);

function evento(overrides = {}) {
  return {
    meta: { tenant_id: 123, id_evento: 'evento-aprovacao', execution_id: 'c41d7e9a-2f3b-4c8d-a5e6-0b1c2d3e4f5a' },
    evento: { tipo: 'aguardando_aprovacao' },
    repositorio: { full_name: 'equipe/projeto' },
    pr: { numero: 42, sha: 'a'.repeat(40), autor: { login: 'autora' } },
    risco: { nivel: 'ALTO', motivo: 'Definido exclusivamente no núcleo.' },
    ...overrides,
  };
}
const dependencias = [{ nome: 'pacote', ecossistema: 'PyPI', versao: '1.0.0' }];

function snapshot(base = evento(), overrides = {}) {
  return {
    versao: 1,
    identidade: [base.meta.tenant_id, base.meta.execution_id, base.repositorio.full_name, base.pr.numero],
    sha: base.pr.sha, status: 'concluida', consultado_em: new Date().toISOString(),
    total_dependencias: 1,
    dependencias: [{ ...dependencias[0], status: 'concluida', resumo: '2 avisos OSV, 1 CVE distinta. Severidade alta.' }],
    ...overrides,
  };
}
const preparar = base => executar(central, '02A |', base);
const selecionar = (base, snap) => executar(central, '02C |', snap ? { contexto: JSON.stringify(snap) } : {}, preparar(base));

test('contexto da análise correspondente enriquece o alerta antes dos dois botões', () => {
  const original = evento();
  const entrada = selecionar(original, snapshot(original));
  assert.equal(entrada.consultar_osv, false);
  assert.equal(entrada.risco, original.risco);
  const saida = executar(central, '03A |', entrada);
  const blocos = saida.slack_payload.blocks;
  const indiceOsv = blocos.findIndex(b => b.text?.text.includes('Contexto OSV para a decisão'));
  assert.ok(indiceOsv >= 0 && indiceOsv < blocos.findIndex(b => b.type === 'actions'));
  assert.ok(saida.slack_payload.text.includes('Severidade alta'));
  const botoes = blocos.find(b => b.type === 'actions').elements;
  assert.deepEqual(botoes.map(b => b.action_id), ['aprovar', 'solicitar_alteracao']);
  assert.ok(botoes.every(b => b.value.length <= 2000));
  assert.equal(JSON.parse(botoes[0].value).task_run_id, original.meta.execution_id);
  assert.equal(saida.risco, original.risco);
});

test('contexto de outro tenant, análise, PR, repositório ou commit nunca é reutilizado', () => {
  const base = evento();
  for (const indice of [0, 1, 2, 3]) {
    const outro = snapshot(base);
    outro.identidade[indice] = 'outro';
    const resultado = selecionar(base, outro);
    assert.equal(resultado.contexto_osv.status, 'indisponivel');
    assert.doesNotMatch(resultado.contexto_osv.resumo, /Severidade alta/);
  }
  assert.equal(selecionar(base, snapshot(base, { sha: 'b'.repeat(40) })).contexto_osv.status, 'indisponivel');
});

test('sem resultado e sem dependências, alerta indica indisponibilidade e mantém as ações', () => {
  const entrada = selecionar(evento());
  assert.equal(entrada.consultar_osv, false);
  assert.match(entrada.contexto_osv.resumo, /não indica ausência de vulnerabilidades/);
  const resultado = executar(central, '03A |', entrada);
  assert.equal(resultado.slack_payload.blocks.find(b => b.type === 'actions').elements.length, 2);
  assert.equal(resultado.risco.nivel, 'ALTO');
});

test('dependências no próprio evento permitem consulta antes do alerta sem depender da ordem de outro evento', () => {
  const entrada = selecionar(evento({ dependencias }));
  assert.equal(entrada.consultar_osv, true);
  assert.equal(entrada.modo_osv, 'contexto_aprovacao');
  const registro = snapshot(entrada);
  const resultado = executar(central, '02F |', { contexto_osv_snapshot: registro }, entrada);
  assert.match(resultado.contexto_osv.resumo, /1 CVE distinta/);
  assert.equal(resultado.risco, entrada.risco);
});

test('consulta vencida é refeita com os pacotes da mesma análise', () => {
  const antigo = snapshot(evento(), { consultado_em: new Date(Date.now() - 25 * 3600000).toISOString() });
  const resultado = selecionar(evento(), antigo);
  assert.equal(resultado.consultar_osv, true);
  assert.deepEqual(resultado.dependencias, dependencias);
  assert.equal(resultado.contexto_osv.status, 'indisponivel');
});

test('falha de armazenamento, retorno de análise diferente e falha do subworkflow mantêm o alerta honesto', () => {
  const base = preparar(evento());
  const consulta = executar(central, '02C |', { error: 'table unavailable' }, base);
  assert.equal(consulta.contexto_osv.status, 'indisponivel');
  const falha = executar(central, '02F |', { error: 'subworkflow unavailable' }, consulta);
  assert.equal(falha.contexto_osv.status, 'indisponivel');
  const trocado = snapshot(evento()); trocado.identidade[0] = 456;
  assert.equal(executar(central, '02F |', { contexto_osv_snapshot: trocado }, consulta).contexto_osv.status, 'indisponivel');
});

test('o resumo é preservado nos dois formulários e não é anexado ao contrato da API', () => {
  const alerta = executar(central, '03A |', selecionar(evento(), snapshot()));
  const botao = alerta.slack_payload.blocks.find(b => b.type === 'actions').elements[0];
  for (const [prefixo, callback, tipo] of [
    ['10 |', 'aprova_aprovacao', 'aprovar'], ['10B |', 'aprova_revisao', 'solicitar_alteracao'],
  ]) {
    const no = central.nodes.find(n => n.name.startsWith(prefixo));
    let modal = expressao(no.parameters.jsonBody, { slack: {
      trigger_id: 'trigger', channel: { id: 'C123' }, user: { id: 'U123' }, actions: [{ value: botao.value }],
    } });
    if (typeof modal === 'string') modal = JSON.parse(modal);
    assert.ok(modal.view.blocks.some(b => b.text?.text.includes('Severidade alta')));
    assert.ok(modal.view.private_metadata.length <= 3000);
    const decisao = executar(central, '11 |', {
      meta: { assinatura_verificada: true }, slack: {
        type: 'view_submission', team: { id: 'T123' }, user: { id: 'U123' },
        view: { id: 'V123', callback_id: callback, private_metadata: modal.view.private_metadata,
          state: { values: { justificativa: { texto: { value: 'Minha justificativa humana.' } } } } },
      },
    });
    assert.equal(decisao.pedido_acao.tipo, tipo);
    assert.equal(decisao.pedido_acao.justificativa, 'Minha justificativa humana.');
    assert.equal(decisao.pedido_acao.osv_resumo, undefined);
    assert.equal(decisao.pedido_acao.contexto_osv, undefined);
  }
});

test('consulta para contextualizar aprovação não publica um alerta OSV separado', () => {
  const condicao = osv.nodes.find(n => n.name.startsWith('06D |')).parameters.conditions.conditions[0].leftValue;
  assert.equal(expressao(condicao, { modo_osv: 'contexto_aprovacao' }), false);
  assert.equal(expressao(condicao, { evento: { tipo: 'dependencia_nova' } }), true);
  const consulta = central.nodes.find(n => n.name.startsWith('02E |'));
  assert.equal(consulta.parameters.options.waitForSubWorkflow, true);
  assert.equal(consulta.parameters.mode, 'each');
});

test('snapshot guarda contexto informativo sem registrar risco nem uma decisão', () => {
  const base = evento({ dependencia: dependencias[0], enriquecimento_osv: {
    status: 'concluida', consultado_em: new Date().toISOString(),
  }, texto_alerta: '1 CVE conhecida.\n\nRisco definido pelo núcleo: ALTO.' });
  const resultado = new Function('$input', codigo(osv, '06A |'))({ all: () => [{ json: base }] });
  const snap = resultado[0].json.contexto_osv_snapshot;
  assert.equal(snap.risco, undefined);
  assert.equal(snap.pedido_acao, undefined);
  assert.doesNotMatch(snap.dependencias[0].resumo, /Risco definido/);
  assert.equal(resultado[0].json.risco, base.risco);
});

test('consulta complementar limita pacotes e tempo por requisição e mantém resultado parcial explícito', () => {
  const lista = Array.from({ length: 12 }, (_, i) => ({ ...dependencias[0], nome: 'pacote-' + i }));
  const selecionado = selecionar(evento({ dependencias: lista }));
  assert.equal(selecionado.dependencias.length, 10);
  assert.equal(selecionado.osv_pacotes_total, 12);
  const no = osv.nodes.find(n => n.name.startsWith('04 |'));
  assert.equal(expressao(no.parameters.options.timeout, selecionado), 5000);
  assert.equal(expressao(no.parameters.options.pagination.pagination.maxRequests, selecionado), 2);
  assert.equal(expressao(no.parameters.options.timeout, {}), 30000);
  const entradas = selecionado.dependencias.map(dep => ({ json: {
    ...selecionado, dependencia: dep,
    enriquecimento_osv: { status: 'concluida', consultado_em: new Date().toISOString() },
    texto_alerta: 'Nenhum aviso conhecido nesta consulta.',
  } }));
  const agregado = new Function('$input', codigo(osv, '06A |'))({ all: () => entradas });
  assert.equal(agregado.length, 1);
  assert.equal(agregado[0].json.contexto_osv_snapshot.total_dependencias, 12);
  assert.equal(agregado[0].json.contexto_osv_snapshot.status, 'incompleta');
  assert.equal(agregado[0].json.contexto_osv_snapshot.dependencias.length, 10);
});
