# Observabilidade

O que o aPRova expõe para quem opera o núcleo: logs estruturados, métricas no
formato do Prometheus e rotas de saúde. Tudo sai do próprio binário da API e do
worker. Nenhum Prometheus, Grafana ou Sentry roda como parte do projeto.

## Logs

Os dois processos escrevem uma linha JSON por evento na saída padrão, pelo
`log/slog`. O nível vem de `LOG_LEVEL` (`debug`, `info`, `warn` ou `error`;
padrão `info`).

### Campos de identificação

| Onde | Campos em toda linha |
| --- | --- |
| Worker, durante um task_run | `task_run_id`, `tenant_id`, `repositorio`, `numero_pr` e, no motor e no pool, `trabalhador` |
| API, durante uma requisição | `request_id` |
| API, webhook do GitHub | `request_id` e `delivery_id`, inclusive na linha de acesso `requisição atendida` |

O worker coloca os campos do task_run no `context.Context` (pacote
`internal/rastro`) e no diário do motor. Qualquer pacote que registre com
`slog.*Context(ctx, ...)` herda esses campos sem repassá-los à mão. O
executor local do sandbox registra assim.

O `request_id` é aleatório, com 32 caracteres hexadecimais, e volta no
cabeçalho `X-Request-Id` da resposta. Ele não leva o nome da máquina, como o
gerador padrão do chi levava.

### O que nunca vai para o log

- segredo do webhook, assinatura do webhook, token do GitHub e chaves;
- corpo inteiro de payload, inclusive o texto do pull request;
- conteúdo do diff e dos patches;
- saída bruta do sandbox. Ela é produzida pelo código do PR. Quando as
  tentativas do sandbox se esgotam, o diagnóstico completo continua gravado no
  task_run e publicado no Check Run, mas a linha `task_run mudou de estado`
  registra só `tentativas esgotadas em <etapa>`.

Os testes `TestDiarioDoWorkerNuncaLevaDiffNemSaidaDoSandbox`,
`TestDiarioDoWebhookNuncaLevaSegredoAssinaturaNemCorpo` e
`TestAcoesRemotasNaoRegistraSegredoNemAssinatura` guardam essa regra.

### Linhas de dentro do contêiner do sandbox

O programa que roda dentro do contêiner (`cmd/sandbox`) monta o logger com o
`task_run_id` da variável `APROVA_TASK_RUN_ID` logo ao iniciar, e toda linha
dele leva o campo.

- **Executor local:** o worker guarda a saída do contêiner em memória e não a
  reescreve no próprio log.
- **Executor da Azure:** as linhas ficam no log da execução do job, já com o
  `task_run_id`.

### Limitação conhecida: o oráculo do benchmark

O oráculo do benchmark (`internal/benchmark/oraculo`) chama o executor local do
sandbox sem o `task_run_id` no contexto. Além disso, o logger do
`cmd/benchmark` não usa o handler do `rastro`. Por isso, nas rodadas do
benchmark, as linhas do executor local saem sem `task_run_id`. O benchmark é
uma ferramenta de medição, fora do caminho de uma análise real, e ficou assim
de propósito.

## Métricas

### Onde ficam

As métricas ficam num listener interno, separado da porta pública. Ele só sobe
quando `METRICAS_ENDERECO` tem um endereço no formato `host:porta`, por
exemplo `:9090`. Vazio, que é o padrão, o listener fica desligado.

| Rota do listener interno | O que responde |
| --- | --- |
| `GET /metrics` | Métricas no formato texto do Prometheus |
| `GET /health/live` | 200 enquanto o processo está de pé |
| `GET /health/ready` | 200 com as dependências prontas, 503 nomeando a que falhou |

O `/metrics` não tem autenticação. Por isso não fica na porta pública da API.
Na Azure, o Container App só expõe a porta do ingress, e `METRICAS_ENDERECO`
deve usar outra porta, que não vai para o ingress. Expor `/metrics` no
endereço público é uma decisão que não foi tomada.

As métricas do motor ficam no worker. O `/metrics` da API tem só as métricas do
runtime do Go (`go_*`) e do processo (`process_*`), que os dois processos
expõem. Os contadores são por processo e voltam a zero quando o processo
reinicia. Com mais de uma réplica, a soma entre elas é feita na consulta.

### Métricas do aPRova

Os rótulos são sempre de baixa cardinalidade. Nenhuma métrica tem
`task_run_id`, número de PR ou SHA como rótulo. O teste
`TestNenhumaMetricaTemRotuloDeAltaCardinalidade` percorre as métricas
registradas e recusa rótulo fora da lista ou valor com cara de id, SHA ou
número.

| Métrica | Tipo | Rótulos | O que significa |
| --- | --- | --- | --- |
| `aprova_analises_total` | contador | `estado`, `nivel` | Task_runs que chegaram a um estado final: `concluido`, `rejeitado` ou `falhou`. `nivel` é o nível do gate (`baixo`, `medio`, `alto`) ou `sem_nivel` quando a análise acabou antes da decisão |
| `aprova_motivos_do_gate_total` | contador | `motivo` | Cada motivo acionado pelo gate determinístico, pelo id (`area_sensivel`, `falha_sandbox` etc.). Conta uma vez por decisão |
| `aprova_calibracoes_total` | contador | `resultado` | Resultado da calibração por histórico, o mesmo valor gravado em `risk_gates.calibration_status` (`desligada`, `calibrada`, `historico_insuficiente` etc.) |
| `aprova_duracao_da_etapa_segundos` | histograma | `etapa` | Duração de cada passagem do motor por um estado (`mapeando_contexto`, `testando_sandbox`, `avaliando_risco` e os demais) e de cada chamada que publica no PR, com `etapa="publicacao"` |
| `aprova_execucoes_do_sandbox_total` | contador | `linguagem`, `tipo_falha` | Execuções do sandbox que devolveram resultado, por linguagem e tipo de falha (`nenhuma`, `teste`, `compilacao`, `infraestrutura`) |
| `aprova_novas_tentativas_total` | contador | `etapa` | Novas tentativas agendadas depois de uma falha transitória: GitHub fora do ar, limite de requisições ou infraestrutura do sandbox |
| `aprova_recibos_emitidos_total` | contador | — | Recibos assinados emitidos. Um por decisão do gate |
| `aprova_fila_pendentes` | gauge | — | Task_runs no estado `pendente`, contados no banco a cada coleta. Some da coleta quando a contagem falha, em vez de mostrar um valor falso |
| `aprova_leases_vencidos_retomados_total` | contador | — | Task_runs reivindicados depois de o prazo de outro trabalhador vencer, sinal de um worker que morreu no meio de uma etapa |
| `aprova_erros_github_total` | contador | `tipo` | Erros da API do GitHub vistos pelo motor: `transitorio` (5xx ou limite de requisições), `permissao` (403 sem limite) ou `definitivo` (os outros) |

Duas observações sobre a duração das etapas:

- **Etapas que se sobrepõem.** A publicação do bloqueio de risco alto acontece
  dentro de `avaliando_risco`, então o tempo dela aparece nas duas etapas.
- **Uma observação por passagem.** Cada passagem pela etapa é uma observação.
  Uma etapa que falhou e foi tentada de novo gera mais de uma.

### Exemplos de consulta

Em PromQL, a linguagem de consulta do Prometheus:

```promql
# Análises por estado final na última hora
sum by (estado) (increase(aprova_analises_total[1h]))

# Fração das análises que falharam no último dia
sum(increase(aprova_analises_total{estado="falhou"}[1d]))
  / sum(increase(aprova_analises_total[1d]))

# Os cinco motivos do gate mais acionados na semana
topk(5, sum by (motivo) (increase(aprova_motivos_do_gate_total[7d])))

# Percentil 95 da duração de cada etapa
histogram_quantile(0.95, sum by (le, etapa) (rate(aprova_duracao_da_etapa_segundos_bucket[1h])))

# Falhas de infraestrutura do sandbox por linguagem
sum by (linguagem) (increase(aprova_execucoes_do_sandbox_total{tipo_falha="infraestrutura"}[1h]))

# Fila crescendo: mais de 10 pendentes por 10 minutos
aprova_fila_pendentes > 10

# Worker que morreu no meio de uma etapa
increase(aprova_leases_vencidos_retomados_total[1h]) > 0

# Erros do GitHub por tipo
sum by (tipo) (rate(aprova_erros_github_total[15m]))
```

### Custo

O aPRova não chama modelo pago para decidir: o gate é determinístico. Por isso
não há métrica de custo acumulado, e inventar um custo seria uma métrica sem
dado por trás. A única chamada externa ligada à análise é o embedding da
calibração por histórico. Ela só acontece com `CALIBRACAO_LIGADA`, que vem
desligada por padrão. Se houver cobrança por ela, quem a registra é a conta do
provedor, não o aPRova.

## Saúde

| Rota | Onde | O que confere |
| --- | --- | --- |
| `/health/live` e o apelido `/health` | API pública e listener interno | Só que o processo responde |
| `/health/ready` e o apelido `/ready` | API pública e listener interno | As dependências, em paralelo, com limite de 2 segundos |

O `/health/ready` de cada processo confere:

- **API:** Postgres acessível com `schema_migrations` limpa e na versão que o
  binário espera, e Redis.
- **Worker:** Postgres do mesmo jeito e, com `SANDBOX_EXECUTOR=local`, o docker
  do sandbox (`docker version`).

A versão esperada é `repository.VersaoEsperadaDoBanco`. O teste
`TestVersaoEsperadaDoBancoEAUltimaMigrationDoRepositorio` falha se uma
migration nova chegar sem atualizar a constante.

## Quem consumiria isto em produção

O formato das métricas é o que Prometheus e Grafana leem, e os logs JSON são o
que um agregador de logs indexa. Em produção, os consumidores seriam:

- um Prometheus, ou o Prometheus gerenciado da Azure, coletando o
  `/metrics` do listener interno;
- um Grafana com painéis sobre as consultas acima;
- um Sentry, ou um agregador de logs, para os erros.

Nenhum deles faz parte deste projeto. O aPRova só expõe os dados.
