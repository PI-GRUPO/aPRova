const test = require('node:test');
const assert = require('node:assert/strict');
const crypto = require('node:crypto');
const w = require('../comparacao-relatorios-semanais.json');
const code = prefix => w.nodes.find(n => n.name.startsWith(prefix)).parameters.jsCode;
const preparar = (entrada,env={APROVA_CHAT_INSTALLATION_ID:'42',N8N_SEGREDO_ACOES:'segredo'}) => new Function('$input','$env','require',code('01 |'))({first:()=>({json:entrada})},env,require).map(i=>i.json);
const juntar = rs => new Function('$input',code('04 |'))({all:()=>rs.map(json=>({json}))})[0].json;
const relatorio = (inicio,concluidas=0,falhas=0) => ({status:'disponivel',installation_id:42,periodo:{inicio},dados:{total_execucoes:concluidas+falhas,concluidas,rejeitadas:0,falhas,em_andamento:0,exigiram_aprovacao_humana:0,risco_medio:0,revertidas:0,taxa_sucesso:concluidas+falhas?concluidas/(concluidas+falhas):null}});
test('comparação assina exatamente duas consultas ao mesmo agregado do Digest',()=>{
 const qs=preparar({inicio_a:'2026-11-09',inicio_b:'2026-11-02',installation_id:7,url:'http://outro'});assert.equal(qs.length,2);
 for(const q of qs){const u=new URL(q.requisicao.url);assert.equal(u.pathname,'/reports/weekly');assert.equal(u.searchParams.get('installation_id'),'42');assert.equal(q.fim,q.inicio==='2026-11-09'?'2026-11-16':'2026-11-09');const ts=q.requisicao.headers['X-Aprova-Timestamp'];assert.equal(q.requisicao.headers['X-Aprova-Assinatura'],'sha256='+crypto.createHmac('sha256','segredo').update(ts+'.'+u.search.slice(1)).digest('hex'));}
});
test('datas padrão são duas semanas completas consecutivas UTC',()=>{
 const [a,b]=preparar({inicio_a:'',inicio_b:''});assert.equal(new Date(a.inicio).getUTCDay(),1);assert.equal(new Date(a.inicio)-new Date(b.inicio),7*86400000);assert.ok(new Date(a.fim)<=new Date());
});
test('datas inválidas, iguais e instalação ausente interrompem antes do HTTP',()=>{
 for(const p of [{inicio_a:'2026-02-30'},{inicio_a:'2026-11-09&installation_id=7'},{inicio_a:7},{inicio_a:'2026-11-02',inicio_b:'2026-11-02'}])assert.throws(()=>preparar(p));
 assert.throws(()=>preparar({},{}));
});
test('comparação calcula diferenças assinadas e pontos percentuais',()=>{
 const out=juntar([relatorio('2026-11-09',3,1),relatorio('2026-11-02',1,1)]);assert.equal(out.status,'disponivel');assert.equal(out.diferenca_a_menos_b.concluidas,2);assert.equal(out.diferenca_taxa_pontos_percentuais,25);
 const inversa=juntar([relatorio('2026-11-02',1,1),relatorio('2026-11-09',3,1)]);assert.equal(inversa.diferenca_a_menos_b.concluidas,-2);assert.equal(inversa.diferenca_taxa_pontos_percentuais,-25);
});
test('taxa nula permanece nula e uma semana indisponível não gera comparação',()=>{
 assert.equal(juntar([relatorio('2026-11-09',2,1),relatorio('2026-11-02')]).diferenca_taxa_pontos_percentuais,null);
 for(const rs of [[relatorio('a'),{status:'indisponivel',dados:null}],[],[relatorio('a')],[relatorio('a'),{...relatorio('b'),installation_id:7}],[relatorio('a'),relatorio('a')]])assert.equal(juntar(rs).status,'indisponivel');
});
test('cada resposta semanal é validada antes da comparação e rejeita tenant incorreto',()=>{
 const q=preparar({inicio_a:'2026-11-09',inicio_b:'2026-11-02'})[0];
 const validate=r=>new Function('$json','$',code('03 |'))(r,()=>({item:{json:q}})).json;
 const body={installation_id:42,inicio:'2026-11-09T00:00:00Z',fim:'2026-11-16T00:00:00Z',total_execucoes:0,concluidas:0,rejeitadas:0,falhas:0,em_andamento:0,exigiram_aprovacao_humana:0,risco_medio:0,revertidas:0,taxa_sucesso:null};
 assert.equal(validate({statusCode:200,body}).status,'disponivel');assert.equal(validate({statusCode:200,body:{...body,installation_id:7}}).status,'indisponivel');assert.equal(validate({error:'timeout'}).status,'indisponivel');
});
