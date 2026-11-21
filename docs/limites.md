# Limites de uso

O aPRova tem dois limites. O primeiro segura análises novas por instalação. O
segundo segura leituras públicas de recibo por IP. Nenhum dos dois libera um
PR sem análise.

## Teto de análises por instalação

Cada instalação do GitHub App inicia no máximo `APROVA_TETO_ANALISES_POR_HORA`
análises novas numa janela móvel de uma hora. O padrão é `60`.

### O que acontece acima do teto

- O `task_run` é gravado como sempre, em `pendente`. Nada é descartado.
- O webhook vê que a instalação está no teto e cria o Check Run `aPRova` em
  `queued`, com a explicação. `queued` não é conclusão: a proteção de branch
  continua esperando o check, e o merge segue bloqueado.
- Quando a janela abre, o worker pega o `task_run` como qualquer outro. O
  Check Run da fila é o mesmo que passa a `in_progress` e depois a
  `completed`. Não nasce um segundo check.
- Se o aviso falhar (GitHub fora, app sem credenciais), o `task_run` segue na
  fila do mesmo jeito. Sem check, a proteção de branch também não libera.

### Por que `queued` e não `neutral`

`neutral` é uma conclusão, e a proteção de branch trata `neutral` como check
aprovado. Usar `neutral` liberaria o merge de um PR que ninguém analisou.

### Onde a contagem vive

No Postgres, na coluna `task_runs.started_at`. Ela é gravada na primeira vez
que um worker reivindica o `task_run` e não muda nas retomadas.

A fila já é o Postgres (`FOR UPDATE SKIP LOCKED`). O teto entra na mesma query
de reivindicação, então contar e pegar acontecem juntos. A contagem sobrevive
a reinícios. Com Redis, a contagem e a fila ficariam em lugares diferentes e
poderiam discordar.

### Limite conhecido: ultrapassagem pelo tamanho do pool

Dois workers que reivindicam ao mesmo tempo leem a mesma contagem antes de
qualquer um gravar `started_at`. Por isso a instalação pode passar do teto em
até `WORKER_CONCORRENCIA` análises por janela (somado entre processos, se
houver mais de um). É aceito: um lock por instalação serializaria a fila para
evitar poucas análises a mais.

### O que não conta

- Retomada de um `task_run` que já começou (prazo vencido, review novo em
  `aguardando_aprovacao`): ele já foi contado e nunca é segurado pelo teto.
- Reentrega do mesmo webhook: não cria `task_run` novo.
