const test = require('node:test');
const assert = require('node:assert/strict');
const crypto = require('node:crypto');
const workflow = require('../entrada-eventos.json');
const code = prefix => workflow.nodes.find(n => n.name.startsWith(prefix)).parameters.jsCode;
const evento = () => ({
  meta: { tenant_id: 123, origem_verificada: true },
  evento: { tipo: 'relatorio_periodico' }, dados: { inicio: '2026-10-26' },
  original: { preservado: true },
});
const assinar = json => new Function('require', '$json', '$env', code('10 |'))(
  require, json, { N8N_SEGREDO_ACOES: 'segredo-teste', APROVA_API_URL: 'http://api:8080/' },
).json;
const normalizar = (resposta, origem = assinar(evento())) => new Function('$json', '$', code('12 |'))(
  resposta, () => ({ item: { json: origem } }),
).json;
const metricas = () => ({
  installation_id: 123, inicio: '2026-10-26T00:00:00Z', fim: '2026-11-02T00:00:00Z',
  total_execucoes: 5, concluidas: 2, rejeitadas: 1, falhas: 1, em_andamento: 1,
  taxa_sucesso: 0.5, exigiram_aprovacao_humana: 3, risco_medio: 0, revertidas: 0,
});

test('consulta assina exatamente a query transmitida e usa o tenant autenticado', () => {
  const e = evento();
  e.dados.installation_id = 999;
  for (const inicio of ['2026-10-26', undefined]) {
    e.dados.inicio = inicio;
    const r = assinar(e).requisicao_relatorio;
    const url = new URL(r.url);
    assert.equal(url.origin, 'http://api:8080');
    assert.equal(url.pathname, '/reports/weekly');
    assert.equal(url.searchParams.get('installation_id'), '123');
    assert.equal(url.searchParams.get('inicio'), inicio ?? null);
    const timestamp = r.headers['X-Aprova-Timestamp'];
    assert.equal(r.headers['X-Aprova-Assinatura'], 'sha256=' + crypto.createHmac('sha256', 'segredo-teste')
      .update(timestamp + '.' + url.search.slice(1)).digest('hex'));
  }
});

test('consulta recusa origem não autenticada, tenant inválido e datas inválidas', () => {
  for (const alterar of [
    e => e.meta.origem_verificada = false,
    e => e.meta.tenant_id = -1,
    e => e.evento.tipo = 'analise_concluida',
    ...['2026-02-30', '2026-13-01', '', 123, '2026-10-26&installation_id=999'].map(v => e => e.dados.inicio = v),
  ]) {
    const e = evento(); alterar(e);
    assert.throws(() => assinar(e));
  }
});

test('normalização preserva o evento e distingue zero execuções de falha', () => {
  const sucesso = normalizar({ statusCode: 200, body: metricas() });
  assert.deepEqual(sucesso.original, evento().original);
  assert.equal(sucesso.relatorio_semanal.status, 'disponivel');
  assert.equal(sucesso.requisicao_relatorio, undefined);
  const vazio = metricas();
  for (const k of Object.keys(vazio)) if (typeof vazio[k] === 'number' && k !== 'installation_id') vazio[k] = 0;
  vazio.taxa_sucesso = null;
  assert.equal(normalizar({ statusCode: 200, body: vazio }).relatorio_semanal.status, 'disponivel');
  for (const resposta of [
    ...[400, 401, 500, 503].map(statusCode => ({ statusCode, body: metricas() })),
    { error: 'ECONNREFUSED' }, { statusCode: 200, body: 'não JSON' },
    { statusCode: 200, body: { ...metricas(), installation_id: 999 } },
    { statusCode: 200, body: { ...metricas(), total_execucoes: -1 } },
    { statusCode: 200, body: { ...metricas(), inicio: '2026-10-25T00:00:00Z' } },
  ]) {
    const r = normalizar(resposta).relatorio_semanal;
    assert.equal(r.status, 'indisponivel');
    assert.equal(r.dados, null);
  }
});

test('rota de relatório termina na normalização sem acionar operações em PR', () => {
  let nome = '07E | relatorio_periodico | Conectar workflow';
  const caminho = [];
  while (workflow.connections[nome]) {
    const destinos = workflow.connections[nome].main[0];
    assert.equal(destinos.length, 1);
    nome = destinos[0].node; caminho.push(nome);
  }
  assert.deepEqual(caminho, ['10 | Assinar Consulta Semanal', '11 | Consultar Relatório Semanal', '12 | Normalizar Relatório Semanal']);
});
