const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const root = path.resolve(__dirname, '../../..');
const dir = path.join(root, 'workflows/n8n');
const workflows = fs.readdirSync(dir).filter(f => f.endsWith('.json')).map(f => JSON.parse(fs.readFileSync(path.join(dir, f))));
const example = fs.readFileSync(path.join(root, '.env.example'), 'utf8');
const compose = fs.readFileSync(path.join(root, 'docker-compose.yml'), 'utf8');
const variables = new Set([...example.matchAll(/^([A-Z_0-9]+)=/gm)].map(m => m[1]));

test('configuração: todas as leituras dos workflows estão no exemplo e no Compose', () => {
  for (const w of workflows) {
    for (const [, key] of JSON.stringify(w).matchAll(/\$env\.([A-Z_0-9]+)/g)) {
      assert.ok(variables.has(key), `${w.name}: ${key} ausente no exemplo`);
      assert.ok(compose.includes(`${key}:`), `${w.name}: ${key} ausente no Compose`);
    }
    assert.equal(w.active, false);
    assert.deepEqual(w.pinData, {});
    assert.equal(w.id, undefined);
    for (const n of w.nodes) {
      assert.equal(n.credentials, undefined);
      if (n.type.endsWith('.executeWorkflow') || n.type.endsWith('.toolWorkflow')) {
        assert.match(n.parameters.workflowId.value, /^=\{\{ \$env\.APROVA_[A-Z_]+_WORKFLOW_ID \}\}$/);
      }
    }
    assert.doesNotMatch(JSON.stringify(w), /CONFIGURE_WORKFLOW_|CONFIGURE_ID_/);
  }
  const go = fs.readFileSync(path.join(root, 'internal/config/configuracao.go'), 'utf8');
  for (const [, key] of go.matchAll(/"([A-Z][A-Z_0-9]+)"/g)) assert.ok(variables.has(key), key);
  assert.match(compose, /WEBHOOK_URL: \$\{N8N_WEBHOOK_URL_PUBLICA:/);
  assert.doesNotMatch(compose, /^\s+N8N_WEBHOOK_URL:/m);
});

test('configuração: onboarding usa flags e destinos independentes de consultas; feedback compartilha vínculos', () => {
  const w = workflows.find(w => w.name.includes('Onboarding'));
  const code = w.nodes.find(n => n.name.startsWith('02 |')).parameters.jsCode;
  const input = {evento:{tipo:'instalacao',acao:'instalado'}, meta:{origem_verificada:true,validacao:{valido:true},versao_evento:1,tenant_id:1,id_evento:'00000000-0000-4000-8000-000000000001'},instalacao:{acao:'instalado',repositorios:['equipe/repo']}};
  const run = env => new Function('$input', '$env', code)({all:()=>[{json:input}]}, env)[0].json.config;
  const config = run({APROVA_ONBOARDING_TRELLO_HABILITADO:'true', APROVA_ONBOARDING_SLACK_HABILITADO:'true',APROVA_TRELLO_QUADRO_ID:'quadro-ficticio',APROVA_SLACK_CANAL_ONBOARDING_ID:'canal-onboarding',APROVA_SLACK_CANAL_CONSULTAS_ID:'canal-consultas'});
  assert.equal(config.trello_habilitado, true);
  assert.equal(config.slack_habilitado, true);
  assert.equal(config.quadro_id, 'quadro-ficticio');
  assert.equal(config.slack_canal_id, 'canal-onboarding');
  assert.equal(run({APROVA_ONBOARDING_SLACK_HABILITADO:'false'}).slack_habilitado, false);
  const feedback = workflows.find(w => w.name.includes('Feedback'));
  const configCode = feedback.nodes.find(n => n.name.startsWith('02 |')).parameters.jsCode;
  assert.match(configCode, /\$env\.SLACK_GITHUB_VINCULOS/);
  assert.doesNotMatch(configCode, /vinculosLocais/);
});
