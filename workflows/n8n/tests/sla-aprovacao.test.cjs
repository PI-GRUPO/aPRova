const vm=require('node:vm'),assert=require('node:assert/strict');
const workflow=require('../sla-aprovacao.json');
const logic=workflow.nodes.find(n=>n.name==='E02 | Validar e registrar fato').parameters.jsCode.split('\nconst entradas=')[0];
const funcs=vm.runInNewContext(logic+';({validarEvento,politicaPara,planejar,revalidar,confirmarEnvio})',{Date,Map,Set,JSON,Number,Error});
const {validarEvento,politicaPara,planejar,revalidar,confirmarEnvio}=funcs;
const test=require('node:test');function t(nome,f){test(nome,f);}
t('export portátil, tabelas explícitas e rotas anteriores do Core preservadas',()=>{
 const core=require('../entrada-eventos.json');
 assert.equal(workflow.active,false);assert.deepEqual(workflow.pinData,{});
 assert.equal(workflow.id,undefined);assert.equal(workflow.versionId,undefined);
 assert.equal(workflow.settings.callerPolicy,'workflowsFromSameOwner');
 for(const n of workflow.nodes){
  assert.equal(n.credentials,undefined);assert.equal(n.webhookId,undefined);
  assert.notEqual(n.type,'n8n-nodes-base.manualTrigger');
  if(n.type==='n8n-nodes-base.dataTable'){
   assert.equal(n.parameters.dataTableId.mode,'id');
   assert.match(n.parameters.dataTableId.value,/^=\{\{ \$env\.APROVA_SLA_(EVENTOS|CONTROLE)_TABELA_ID \}\}$/);
  }
 }
 assert.equal(workflow.nodes.find(n=>n.name==='T13 | Publicar lembrete ou menção').type,'n8n-nodes-base.slack');
 const chamada=core.nodes.find(n=>n.name==='07M | Acompanhar SLA de Aprovação');
 assert.equal(chamada.parameters.workflowId.value,'={{ $env.APROVA_SLA_WORKFLOW_ID }}');
 assert.equal(chamada.parameters.mode,'each');assert.equal(chamada.parameters.options.waitForSubWorkflow,false);
 for(const [source,anterior] of [['07A | aguardando_aprovacao | Conectar workflow','08 | Executar Central do Slack'],['07H | aprovacao_registrada | Conectar workflow','07I | Solicitar Motivo da Aprovação']]){
  const targets=core.connections[source].main[0].map(e=>e.node);
  assert.ok(targets.includes(anterior));assert.ok(targets.includes(chamada.name));
 }
 const nomes=new Set(workflow.nodes.map(n=>n.name));
 for(const [source,tipos] of Object.entries(workflow.connections)){
  assert.ok(nomes.has(source));for(const outputs of Object.values(tipos))for(const output of outputs)for(const edge of output)assert.ok(nomes.has(edge.node));
 }
});
t('recursos de tabela não contêm dados reais e inicializam uma única reserva de ciclo',()=>{
 const eventos=require('../recursos/tabela-sla-eventos.json');
 const controle=require('../recursos/tabela-sla-controle.json');
 assert.equal(eventos.name,'aprova_sla_eventos');assert.equal(eventos.initialRows,undefined);
 assert.deepEqual(eventos.columns.map(c=>[c.name,c.type]),[['chave','string'],['evento_id','string'],['tipo','string'],['ocorrido_em','date'],['contexto','string']]);
 assert.equal(controle.name,'aprova_sla_controle');
 assert.deepEqual(controle.initialRows,[{chave:'temporizador',bloqueado_ate:0,executor:''}]);
});
const now=Date.parse('2026-11-14T12:00:00Z');
function evento(tipo='aguardando_aprovacao') {return {meta:{origem_verificada:true,validacao:{valido:true},versao_evento:1,tenant_id:123,id_evento:tipo==='aguardando_aprovacao'?'11111111-1111-4111-8111-111111111111':'22222222-2222-4222-8222-222222222222',execution_id:'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa',ocorrido_em:'2026-11-14T10:00:00Z'},evento:{tipo},repositorio:{full_name:'equipe/repo'},pr:{numero:7},risco:{nivel:'ALTO'},aprovacao:{decisao:'aprovada'}};}
const politica={installation_id:123,repositorio:'equipe/repo',habilitado:true,canal_slack:'CTEST123456',primeiro_limite_minutos:60,segundo_limite_minutos:120,responsavel_slack:'UTEST123456'};
const env={APROVA_SLA_HABILITADO:'true',APROVA_SLA_POLITICAS_JSON:JSON.stringify([politica])};
const inicio=validarEvento(evento()),fim=validarEvento(evento('aprovacao_registrada'));
const planeja=(rows=[inicio],e=env,agora=now)=>planejar(rows,e,agora);
function registro(tipo){return {...inicio,evento_id:'teste-'+tipo,tipo};}
t('evento normalizado válido',()=>assert.equal(inicio.tipo,'inicio'));
t('origem sem autenticação recusada',()=>assert.throws(()=>validarEvento({...evento(),meta:{...evento().meta,origem_verificada:false}})));
t('contrato incompleto recusado',()=>assert.throws(()=>validarEvento({...evento(),meta:{...evento().meta,execution_id:''}})));
t('outros eventos ignorados',()=>assert.equal(validarEvento({...evento(),evento:{tipo:'analise_concluida'}}),null));
t('primeiro prazo vencido',()=>assert.equal(planeja().acoes[0].etapa,1));
t('antes do primeiro prazo não envia',()=>assert.equal(planeja([inicio],env,now-61*60000).acoes.length,0));
t('contagem inicia no horário do evento',()=>assert.equal(planeja().acoes[0].prazo,'2026-11-14T11:00:00.000Z'));
t('duplicata não cria outro aviso',()=>assert.equal(planeja([inicio,inicio]).acoes.length,1));
t('repetição não reinicia contagem',()=>assert.equal(planeja([inicio,{...inicio,ocorrido_em:'2026-11-14T11:59:00Z'}]).acoes[0].prazo,'2026-11-14T11:00:00.000Z'));
t('decisão encerra a espera',()=>assert.equal(planeja([inicio,fim]).acoes.length,0));
t('decisão antes do início não reabre',()=>assert.equal(planeja([fim,inicio]).acoes.length,0));
t('pedido de alterações também encerra',()=>assert.equal(validarEvento({...evento('aprovacao_registrada'),aprovacao:{decisao:'alteracoes_solicitadas'}}).tipo,'encerrado'));
t('segundo prazo é absoluto desde o início',()=>assert.equal(planeja([inicio,registro('enviado_1')]).acoes[0].etapa,2));
t('reserva impede duplicação no primeiro aviso',()=>assert.equal(planeja([inicio,registro('reservado_1')]).acoes.length,0));
t('envio incerto não é repetido',()=>assert.equal(planeja([inicio,registro('incerto_1')]).acoes.length,0));
t('segundo aviso não é repetido',()=>assert.equal(planeja([inicio,registro('enviado_1'),registro('reservado_2')]).acoes.length,0));
t('sem responsável o segundo aviso fica pendente',()=>{const e={...env,APROVA_SLA_POLITICAS_JSON:JSON.stringify([{...politica,responsavel_slack:null}])};assert.equal(planeja([inicio,registro('enviado_1')],e).acoes.length,0);assert.match(planeja([inicio,registro('enviado_1')],e).pendencias[0].motivo,/responsável/)});
t('sem política mantém pendência',()=>assert.equal(planeja([inicio],{}).pendencias_total,1));
t('envios desabilitados não notificam',()=>assert.equal(planeja([inicio],{...env,APROVA_SLA_HABILITADO:'false'}).acoes.length,0));
t('política ambígua não autoriza',()=>assert.equal(planeja([inicio],{...env,APROVA_SLA_POLITICAS_JSON:JSON.stringify([politica,politica])}).acoes.length,0));
t('limites inválidos não autorizam',()=>assert.equal(planeja([inicio],{...env,APROVA_SLA_POLITICAS_JSON:JSON.stringify([{...politica,segundo_limite_minutos:1}])}).acoes.length,0));
t('decisão de outra instalação não encerra a espera',()=>{const e=evento('aprovacao_registrada');e.meta.tenant_id=456;assert.equal(planeja([inicio,validarEvento(e)]).acoes.length,1)});
t('nova análise substitui acompanhamento antigo da mesma PR',()=>{const e=evento();e.meta.execution_id='bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb';e.meta.ocorrido_em='2026-11-14T11:50:00Z';assert.equal(planeja([inicio,validarEvento(e)]).acoes.length,0)});
t('histórico inválido interrompe avisos',()=>assert.throws(()=>planeja([{...inicio,contexto:'invalido'}])));
const candidato=planeja().acoes[0];
t('releitura cancela decisão chegada após planejamento',()=>assert.equal(revalidar(candidato,[inicio,fim],env,now,now+900000).enviar,false));
t('releitura aplica configuração atual',()=>assert.equal(revalidar(candidato,[inicio],{...env,APROVA_SLA_HABILITADO:'false'},now,now+900000).enviar,false));
t('controle expirando não envia',()=>assert.equal(revalidar(candidato,[inicio],env,now,now+30000).enviar,false));
const pronto=revalidar(candidato,[inicio],env,now,now+900000);
t('primeiro lembrete sem menção',()=>assert.ok(pronto.enviar&&!pronto.texto.includes('<@')));
t('segundo aviso menciona somente responsável configurado',()=>{const a=planeja([inicio,registro('enviado_1')]).acoes[0];assert.match(revalidar(a,[inicio,registro('enviado_1')],env,now,now+900000).texto,/<@UTEST123456>/)});
t('não inventa revalidação do núcleo',()=>assert.equal(pronto.revalidacao_nucleo.status,'indisponivel'));
t('Slack deve confirmar canal e timestamp',()=>{assert.equal(confirmarEnvio(pronto,{channel:politica.canal_slack,ts:'123.456'}).tipo,'enviado_1');assert.equal(confirmarEnvio(pronto,{error:'timeout'}).tipo,'incerto_1');assert.equal(confirmarEnvio(pronto,{channel:'OUTRO',ts:'123.456'}).tipo,'incerto_1')});
