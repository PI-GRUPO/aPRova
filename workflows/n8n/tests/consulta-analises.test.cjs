const test = require('node:test');
const assert = require('node:assert/strict');
const crypto = require('node:crypto');
const workflow = require(process.env.APROVA_TEST_ANALISES || '../consulta-analises.json');
const assistant = require('../slack-consulta-metricas.json');
const signCode = workflow.nodes.find(n => n.name.startsWith('01 |')).parameters.jsCode;
const normalizeCode = workflow.nodes.find(n => n.name.startsWith('03 |')).parameters.jsCode;
const env = { APROVA_CHAT_INSTALLATION_ID: '42', N8N_SEGREDO_ACOES: 'segredo-teste', APROVA_API_URL: 'https://api.example.test' };
const sign = (input = {}, config = env) => new Function('$json', '$env', 'require', signCode)({operacao:'listar', ...input}, config, require).json;
const normalize = (response, query = sign()) => new Function('$json', '$', normalizeCode)(response, () => ({item:{json:query}})).json;
const item = () => ({id:'a2beef00-1111-4222-8333-123456789012', repositorio:'PI-GRUPO/aPRova-TESTES',numero_pr:3,sha_head:'a'.repeat(40),estado:'aguardando_aprovacao',recebido_em:'2026-10-01T00:00:00Z',criado_em:'2026-10-01T00:00:00Z',atualizado_em:'2026-11-15T12:00:00Z'});
const response = (itens = [item()]) => ({statusCode:200,body:{installation_id:42,consultado_em:'2026-11-15T12:01:00Z',estado:'em_andamento',repositorio:'',numero_pr:0,pagina:1,limite:10,tem_mais:false,itens}});

test('assinatura vincula método, caminho e filtros; instalação vem do ambiente', () => {
 const q=sign({installation_id:999,url:'http://malicioso',method:'POST',repositorio:'PI-GRUPO/aPRova-TESTES'});
 const u=new URL(q.requisicao.url);assert.equal(u.origin,'https://api.example.test');assert.equal(u.pathname,'/api/analises');assert.equal(u.searchParams.get('installation_id'),'42');
 const ts=q.requisicao.headers['X-Aprova-Timestamp'];
 const expected='sha256='+crypto.createHmac('sha256',env.N8N_SEGREDO_ACOES).update(ts+'.GET\n'+u.pathname+'\n'+u.search.slice(1)).digest('hex');
 assert.equal(q.requisicao.headers['X-Aprova-Assinatura'],expected);
});
test('filtros inválidos e tentativas de injeção são recusados antes do HTTP', () => {
 for(const input of [{operacao:'aprovar'},{repositorio:'PI-GRUPO/repo?installation_id=7'},{estado:'todos&installation_id=7'},{pagina:'0'},{pagina:'202'},{limite:'51'},{numero_pr:'-1'},{numero_pr:'3.5'},{operacao:'detalhar',id:'../../acoes-remotas'}]) assert.throws(()=>sign(input));
 assert.throws(()=>sign({}, {...env,APROVA_CHAT_INSTALLATION_ID:'0'}));assert.throws(()=>sign({}, {...env,N8N_SEGREDO_ACOES:''}));
});
test('consulta atual inclui espera humana e datas antigas, sem afirmar worker ativo', () => {
 const r=normalize(response());assert.equal(r.status,'disponivel');assert.equal(r.dados[0].estado,'aguardando_aprovacao');assert.match(r.observacao,/não garante worker ativo/);
});
test('página vazia real é disponível; erros não se convertem em lista vazia', () => {
 assert.deepEqual(normalize(response([])).dados,[]);
 for(const r of [{error:'timeout'},{statusCode:401},{statusCode:503},{statusCode:200,body:'inválido'}]){const out=normalize(r);assert.equal(out.status,'indisponivel');assert.equal(out.dados,null);}
});
test('normalização recusa escopo, estado, PR, datas e identificadores inconsistentes', () => {
 for(const patch of [{installation_id:7},{estado:'todos'},{repositorio:'outro/repo'},{numero_pr:4},{pagina:2},{limite:50},{consultado_em:'inválido'}])assert.equal(normalize({...response(),body:{...response().body,...patch}}).status,'indisponivel');
 for(const patch of [{id:'x'},{estado:'concluido'},{numero_pr:0},{sha_head:'x'},{atualizado_em:null}])assert.equal(normalize(response([{...item(),...patch}])).status,'indisponivel');
 const q=sign({repositorio:'outro/repo'});const r=response();r.body.repositorio='outro/repo';assert.equal(normalize(r,q).status,'indisponivel');
});
test('paginação informa continuação sem inventar total e recusa duplicatas', () => {
 const q=sign({limite:'1'});const r=response();r.body.limite=1;r.body.tem_mais=true;const out=normalize(r,q);
 assert.equal(out.proxima_pagina,2);assert.equal(out.quantidade_nesta_pagina,1);assert.equal(out.total,undefined);
 assert.equal(normalize(response([item(),item()])).status,'indisponivel');
 r.body.itens=[];assert.equal(normalize(r,q).status,'indisponivel');
});
test('detalhe requer UUID real e valida identidade e repositório retornados', () => {
 const a=item(),q=sign({operacao:'detalhar',id:a.id});assert.equal(new URL(q.requisicao.url).pathname,'/api/analises/'+a.id);
 const r={statusCode:200,body:{installation_id:42,consultado_em:'2026-11-15T12:00:00Z',analise:{...a,detail:'SEGREDO',claimed_by:'SEGREDO'}}};
 const out=normalize(r,q);assert.equal(out.status,'disponivel');assert.equal(out.dados.id,a.id);assert.ok(!JSON.stringify(out).includes('SEGREDO'));
 r.body.analise.id='11111111-1111-4111-8111-111111111111';assert.equal(normalize(r,q).status,'indisponivel');
 assert.equal(normalize({statusCode:404},q).status,'nao_encontrada');assert.equal(normalize({statusCode:403},q).status,'indisponivel');
});
test('ferramentas possuem apenas GET fixo e não expõem instalação ao modelo', () => {
 const http=workflow.nodes.find(n=>n.type==='n8n-nodes-base.httpRequest');assert.equal(http.parameters.method,'GET');assert.equal(http.parameters.options.redirect.redirect.followRedirects,false);
 assert.ok(!workflow.nodes.some(n=>/postgres|agent|slack/i.test(n.type)));
 const names=Object.entries(assistant.connections).filter(([,c])=>c.ai_tool).map(([name])=>name).sort();
 assert.deepEqual(names,['consultar_prontidao_api', 'consultar_saude_api']);
 for (const name of ['consultar_execucoes', 'consultar_detalhe_execucao']) {
   assert.equal(assistant.nodes.some(n => n.name === name), false);
 }
});
test('conversa indica o dashboard para consultas individuais sem contrato Slack', () => {
 const prompt = assistant.nodes.find(n => n.name === 'AI Agent').parameters.options.systemMessage;
 assert.match(prompt, /acesso individual/);
 assert.match(prompt, /dashboard/);
});

test('links de repositório e PR derivam dos dados validados, ignorando URLs da resposta', () => {
 const a={...item(),url_pr:'https://outro.example/roubar',url_repositorio:'https://outro.example'};
 const list=normalize(response([a]));
 assert.equal(list.dados[0].url_repositorio,'https://github.com/PI-GRUPO/aPRova-TESTES');
 assert.equal(list.dados[0].url_pr,'https://github.com/PI-GRUPO/aPRova-TESTES/pull/3');
 const q=sign({operacao:'detalhar',id:a.id});
 const detail=normalize({statusCode:200,body:{installation_id:42,consultado_em:'2026-11-15T12:00:00Z',analise:a}},q);
 assert.equal(detail.dados.url_pr,list.dados[0].url_pr);
 assert.ok(!JSON.stringify(detail).includes('outro.example'));
});

test('repositórios com navegação de diretório ou domínio injetado não geram links', () => {
 for(const repositorio of ['../repo','owner/..','./repo','owner/.','https://outro.example/repo','owner/repo?redirect=outro','owner/repo#fragment']) {
  const out=normalize(response([{...item(),repositorio}]));assert.equal(out.status,'indisponivel');assert.equal(out.dados,null);
 }
});
