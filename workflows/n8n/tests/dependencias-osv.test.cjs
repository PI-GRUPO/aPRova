const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');

const raiz = path.resolve(__dirname, '..');
const workflow = JSON.parse(fs.readFileSync(path.join(raiz, 'dependencias-osv.json')));
const core = JSON.parse(fs.readFileSync(path.join(raiz, 'entrada-eventos.json')));
const no = prefixo => workflow.nodes.find(n => n.name.startsWith(prefixo));
const consolidar = new Function('$input', '$', no('05 |').parameters.jsCode);
const alertar = new Function('$json', no('06 |').parameters.jsCode);
const avaliar = (expressao, json, response, pageCount = 0) => new Function('$json', '$response', '$pageCount',
  'return (' + expressao.slice(3, -2).trim() + ');')(json, response, pageCount);

function congelar(valor) {
  if (valor && typeof valor === 'object' && !Object.isFrozen(valor)) {
    Object.freeze(valor);
    Object.values(valor).forEach(congelar);
  }
  return valor;
}

function contexto(nome = 'pacote', evento = 'evento-1', tenant = 123) {
  return congelar({
    meta: { tenant_id: tenant, id_evento: evento, execution_id: 'analise-1' },
    evento: { tipo: 'dependencia_nova' },
    repositorio: { full_name: 'equipe/projeto' },
    pr: { numero: 42, url: 'https://github.com/equipe/projeto/pull/42' },
    risco: { nivel: 'BAIXO', score: 1, motivos: [{ descricao: 'Definido no núcleo' }] },
    dependencia: { nome, ecossistema: 'PyPI', versao: '1.0.0' },
    original: { nivel_risco: 'baixo' },
  });
}

function executar(respostas, contextos = respostas.map(() => contexto())) {
  const resultado = consolidar({ all: () => respostas.map(json => ({ json })) }, () => ({
    itemMatching: indice => ({ json: contextos[indice] }),
  }));
  return resultado.map(item => {
    const json = alertar(item.json).json;
    assert.equal(json.risco, contextos[item.pairedItem[0].item].risco);
    assert.equal(json.original, contextos[item.pairedItem[0].item].original);
    return { ...item, json };
  });
}

const resposta = body => ({ statusCode: 200, body });
const aviso = (id, aliases = [], severidade) => ({
  id, aliases, ...(severidade ? { database_specific: { severity: severidade } } : {}),
});

test('consulta OSV usa nome, versão e ecossistema exatos, sem autenticação', () => {
  const http = no('04 |');
  assert.equal(http.parameters.method, 'POST');
  assert.equal(http.parameters.url, 'https://api.osv.dev/v1/query');
  assert.ok(!http.credentials);
  assert.ok(!http.parameters.authentication || http.parameters.authentication === 'none');
  for (const dependencia of [
    { nome: 'requests', ecossistema: 'PyPI', versao: '2.19.1' },
    { nome: 'github.com/shopspring/decimal', ecossistema: 'Go', versao: 'v1.4.0' },
    { nome: 'org.javamoney:moneta', ecossistema: 'Maven', versao: '1.4.4' },
  ]) {
    assert.deepEqual(JSON.parse(avaliar(http.parameters.jsonBody, { dependencia })), {
      package: { name: dependencia.nome, ecosystem: dependencia.ecossistema }, version: dependencia.versao,
    });
  }
});

test('regra crítica: únicos destinos HTTP são OSV e Incoming Webhook do Slack', () => {
  const http = workflow.nodes.filter(n => n.type === 'n8n-nodes-base.httpRequest');
  assert.equal(http.length, 2);
  assert.deepEqual(http.map(n => n.parameters.url), [
    'https://api.osv.dev/v1/query', '={{ $env.SLACK_INCOMING_WEBHOOK_URL }}',
  ]);
  assert.ok(workflow.nodes.every(n => ['executeWorkflowTrigger', 'if', 'splitOut', 'httpRequest', 'code', 'dataTable']
    .some(tipo => n.type === 'n8n-nodes-base.' + tipo)));
  for (const n of workflow.nodes.filter(n => n.type === 'n8n-nodes-base.dataTable')) {
    assert.equal(n.parameters.dataTableId.value, 'aprova_osv_por_analise');
  }
});

test('alerta com severidade crítica preserva integralmente o risco baixo do núcleo', () => {
  const [resultado] = executar([resposta({ vulns: [aviso('GHSA-teste', ['CVE-2026-1000'], 'CRITICAL')] })]);
  assert.equal(resultado.json.risco.nivel, 'BAIXO');
  assert.equal(resultado.json.enriquecimento_osv.total_cves, 1);
  assert.match(resultado.json.texto_alerta, /crítica/);
  assert.match(resultado.json.texto_alerta, /Risco definido pelo núcleo: BAIXO/);
});

test('avisos de fontes distintas não duplicam a contagem de CVEs', () => {
  const [resultado] = executar([resposta({ vulns: [
    aviso('GHSA-um', ['CVE-2026-1000'], 'HIGH'),
    aviso('PYSEC-um', ['CVE-2026-1000', 'cve-2026-1000']),
    aviso('OSV-sem-cve'),
  ] })]);
  const consulta = resultado.json.enriquecimento_osv;
  assert.equal(consulta.total_avisos, 3);
  assert.equal(consulta.total_cves, 1);
  assert.match(resultado.json.texto_alerta, /3 avisos OSV encontrados, com 1 CVEs distintas/);
  assert.match(resultado.json.texto_alerta, /alta/);
});

test('paginação reúne avisos por dependência e preserva o vínculo de cada item', () => {
  const a = contexto('primeiro');
  const b = contexto('segundo');
  const resultados = executar([
    resposta({ vulns: [aviso('OSV-1')], next_page_token: 'pagina-2' }),
    resposta({ vulns: [aviso('OSV-1'), aviso('OSV-2')] }),
    resposta({}),
  ], [a, a, b]);
  assert.equal(resultados.length, 2);
  assert.equal(resultados[0].json.enriquecimento_osv.total_avisos, 2);
  assert.equal(resultados[0].json.enriquecimento_osv.paginas_validas, 2);
  assert.deepEqual(resultados[0].pairedItem, [{ item: 0 }, { item: 1 }]);
  assert.equal(resultados[1].json.dependencia.nome, 'segundo');
  assert.equal(resultados[1].json.enriquecimento_osv.total_avisos, 0);
});

test('paginação limitada ou falha posterior não anuncia total definitivo', () => {
  for (const respostas of [
    [resposta({ vulns: [aviso('OSV-1')], next_page_token: 'restante' })],
    [resposta({ vulns: [aviso('OSV-1')], next_page_token: 'restante' }), { statusCode: 503 }],
  ]) {
    const [resultado] = executar(respostas);
    assert.equal(resultado.json.enriquecimento_osv.status, 'incompleta');
    assert.equal(resultado.json.enriquecimento_osv.total_avisos, null);
    assert.equal(resultado.json.enriquecimento_osv.total_cves, null);
    assert.match(resultado.json.texto_alerta, /resultados são parciais/);
  }
});

test('falha HTTP, timeout e resposta inválida não viram ausência de vulnerabilidades', () => {
  for (const raw of [{ statusCode: 429 }, { statusCode: 500 }, { error: 'timeout' },
    resposta('invalid json'), resposta({ vulns: null }), resposta({ next_page_token: 123 })]) {
    const [resultado] = executar([raw]);
    assert.equal(resultado.json.enriquecimento_osv.status, 'falha');
    assert.equal(resultado.json.enriquecimento_osv.total_cves, null);
    assert.match(resultado.json.texto_alerta, /não indica ausência de vulnerabilidades/);
    assert.doesNotMatch(resultado.json.texto_alerta, /Nenhum aviso conhecido/);
  }
});

test('consulta concluída vazia é distinguida de falha', () => {
  for (const body of [{}, { vulns: [] }, '{}']) {
    const [resultado] = executar([resposta(body)]);
    assert.equal(resultado.json.enriquecimento_osv.status, 'concluida');
    assert.equal(resultado.json.enriquecimento_osv.total_cves, 0);
    assert.match(resultado.json.texto_alerta, /Nenhum aviso conhecido/);
  }
});

test('avisos retirados são ignorados e registros inválidos impedem total definitivo', () => {
  const [retirado] = executar([resposta({ vulns: [{ ...aviso('OSV-retirado'), withdrawn: '2026-11-01' }] })]);
  assert.equal(retirado.json.enriquecimento_osv.total_avisos, 0);
  const [invalido] = executar([resposta({ vulns: [null, {}] })]);
  assert.equal(invalido.json.enriquecimento_osv.status, 'incompleta');
  assert.equal(invalido.json.enriquecimento_osv.total_avisos, null);
});

test('metadados do pacote correto informam severidade; vetor CVSS não vira classificação inventada', () => {
  const registro = { ...aviso('OSV-1'),
    affected: [
      { package: { name: 'outro', ecosystem: 'PyPI' }, ecosystem_specific: { severity: 'CRITICAL' } },
      { package: { name: 'pacote', ecosystem: 'PyPI' }, ecosystem_specific: { severity: 'LOW' } },
    ],
  };
  const [resultado] = executar([resposta({ vulns: [registro] })]);
  assert.match(resultado.json.texto_alerta, /baixa/);
  assert.doesNotMatch(resultado.json.texto_alerta, /crítica/);
  const [cvss] = executar([resposta({ vulns: [{ ...aviso('OSV-2'), severity: [{ type: 'CVSS_V3', score: 'CVSS:3.1/AV:N' }] }] })]);
  assert.match(cvss.json.texto_alerta, /severidade não informada/);
  assert.equal(cvss.json.enriquecimento_osv.avisos[0].cvss.length, 1);
});

test('pacotes iguais em eventos ou tenants diferentes não misturam resultados', () => {
  const resultados = executar([resposta({ vulns: [aviso('OSV-1')] }), resposta({}), resposta({})], [
    contexto('pacote', 'evento-1', 123), contexto('pacote', 'evento-2', 123), contexto('pacote', 'evento-1', 456),
  ]);
  assert.equal(resultados.length, 3);
  assert.deepEqual(resultados.map(r => r.json.enriquecimento_osv.total_avisos), [1, 0, 0]);
});

test('alerta usa texto simples, quebras de linha reais e blocos dentro do limite Slack', () => {
  const [resultado] = executar([resposta({ vulns: [aviso('OSV-1', ['CVE-2026-1000'], 'HIGH')] })]);
  assert.ok(resultado.json.slack_payload.text.includes('\n'));
  assert.ok(resultado.json.slack_payload.blocks.every(b => b.text.type === 'plain_text' && b.text.text.length <= 3000));
  assert.match(resultado.json.slack_payload.text, /equipe\/projeto #42/);
});

test('export do Core chama OSV somente pela rota dependencia_nova e configura ID por ambiente', () => {
  const nome = "Call 'aPRova | Dependências | Consulta OSV'";
  const chamada = core.nodes.find(n => n.name === nome);
  assert.equal(chamada.parameters.workflowId.value, '={{ $env.APROVA_OSV_WORKFLOW_ID }}');
  assert.equal(chamada.parameters.options.waitForSubWorkflow, false);
  assert.ok(core.connections['07I | dependencia_nova | Conectar workflow'].main[0].some(c => c.node === nome));
  assert.equal(workflow.nodes[0].parameters.inputSource, 'passthrough');
  assert.deepEqual(workflow.pinData, {});
});
