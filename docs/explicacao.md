# Explicação por IA no comentário

Com `EXPLICACAO_LIGADA=true`, o comentário do aPRova no pull request ganha um
parágrafo em português simples. Ele explica o risco que o gate **já decidiu**:
o nível, o que ele significa para o merge, os motivos com os arquivos das
evidências e o que a pessoa pode fazer. Um modelo Gemini escreve esse
parágrafo, mas só descreve a decisão; quem decide continua sendo o aPRova.

## Por que a decisão continua sendo das regras

A explicação é pedida **depois** de tudo o que decide:

1. O gate grava o nível e os motivos em `risk_gates`.
2. O recibo é assinado com essa decisão e gravado em `receipts`.
3. O Check Run é concluído no GitHub.
4. Só então o modelo é chamado, e o texto dele entra apenas no comentário.

O modelo não tem como mudar nível, estado, Check Run nem recibo:

- **Recibo e gate:** quando ele é chamado, os dois já estão gravados. O texto
  não volta para o gate, não é assinado e não entra no payload do recibo.
- **Check Run:** é sempre o mesmo, com ou sem explicação. Há teste para isso.
- **Proteção de branch:** só lê o Check Run, nunca o comentário.

Com a variável desligada, que é o padrão, o comentário é byte a byte o de
antes.

## O que o modelo recebe

Um JSON montado em `internal/explicacao/entrada.go`, só com o que o gate gravou:

| Campo | Conteúdo |
| --- | --- |
| `nivel` e `merge` | nível decidido e o que ele significa para o merge |
| `motivos` | id, nível, descrição e evidências de cada motivo de risco do `rules_hit` |
| `sandbox` | compilou, testes que passaram, testes que falharam, tipo de falha |
| `nivel_pelas_regras` e `calibracao` | só quando a calibração reduziu o nível: o nível pelas regras e a razão |

As evidências do gate são caminhos de arquivo (`go/frete/frete.go`) ou
`arquivo: não foi possível ler o diff`. A exceção é `falha_sandbox`: a
evidência desse motivo é uma frase fixa (`1 de 11 testes falharam`), que o
modelo copiava com erro de concordância. Por isso ela vai vazia, e o modelo
escreve a partir dos números do campo `sandbox` ("1 dos 11 testes falhou"). O
gate e o recibo continuam com a frase original.

## O que o modelo nunca recebe

- **Código e diff:** nem diff, nem conteúdo de arquivo, nem a saída bruta do sandbox.
- **Pessoas e identificadores:** nem autor, nem quem aprovou, nem SHA, nome do
  repositório, número do PR ou id do task_run.
- **Dependências novas:** o nome delas sai do conteúdo do diff, então não vai.
- **Calibração:** os links dos PRs vizinhos não vão.
- **Segredos:** nenhum.

Um teste monta um pedido com segredo, diff, saída do sandbox, dependência e
vizinhos nos dados de origem, e confere que nada disso chega ao modelo.

## Validação da resposta

Antes de publicar, `Validar` normaliza a resposta num parágrafo e a descarta
se:

1. **Tamanho:** tiver menos de 120 ou mais de 900 caracteres.
2. **Garantia:** afirmar o que o aPRova não sabe: qualquer forma de "segur…"
   (seguro, segura, segurança), "sem riscos", "sem problemas", "nenhuma ação"
   ou "pode seguir". Quando o motivo é `area_sensivel`, "de segurança" ou "da
   segurança" é aceito, porque descreve o caminho ("uma área sensível de
   segurança"), não uma garantia.
3. **Só texto:** tiver link, `www.`, link em markdown, HTML, `@` (uma menção
   notificaria alguém de verdade), referência `#número`, bloco de código, `$(`
   ou comando de terminal (`git`, `curl`, `sudo`, `npm`, `pip`, `rm -`…).
4. **Nível:** citar um nível diferente do decidido. Só conta quando "baixo",
   "médio" ou "alto" vem junto de "risco" ou "nível" ("risco alto", "o nível
   é baixo", "risco foi considerado médio"). "Um número alto de arquivos" não
   descarta. Com calibração aplicada, o nível pelas regras também é aceito.
5. **Motivo:** citar um id de motivo (`palavra_palavra`) que não está no
   `rules_hit`.
6. **Arquivo:** citar um arquivo (algo com `/` ou com extensão) que não está nas
   evidências. O nome curto de um arquivo das evidências vale.

A regra 6 também segura boa parte da injeção de instrução por nome de arquivo:
um texto que cite um caminho fora das evidências é descartado.

## Limitações conhecidas

- **Estilo pouco natural às vezes:** o texto pode sair com fórmulas como
  "conforme visto no arquivo go.mod" ou repetir a descrição do motivo palavra
  por palavra. O prompt pede o contrário, mas o `gemini-3.5-flash-lite` nem
  sempre segue. É questão de estilo, não de correção, então a validação não
  descarta por isso.
- **Frases no limite do proibido:** a validação procura palavras e trechos
  fixos, não sentido. Uma frase como "não é preciso seguir passos adicionais
  além da conferência padrão" chega perto de "nenhuma ação é necessária" e
  passa. Também passa um motivo inventado descrito só com palavras comuns, sem
  id nem arquivo.

## Falha, descarte e reuso

- Cada task_run tem **uma única** tentativa, gravada em `risk_explanations`
  (migração 26). O resultado pode ser `gerada`, `descartada` ou `falha`, e os
  dois últimos guardam o motivo.
- Uma resposta descartada, um erro, um 429 ou o fim do prazo
  (`EXPLICACAO_TIMEOUT`, padrão 5s) deixam o comentário com o texto fixo de
  sempre. A análise segue normal.
- Na retomada, ou na segunda publicação do risco alto depois da aprovação, a
  tentativa gravada é reaproveitada e o modelo não é chamado de novo. O texto
  do bloqueio continua no comentário aprovado e não cita quem aprovou; isso
  fica no cabeçalho.
- O Check Run nunca espera o modelo, porque é concluído antes. No pior caso,
  o comentário atrasa até o fim do prazo.
- Métrica `aprova_explicacoes_total{resultado}`, com os valores `gerada`,
  `descartada`, `falha` e `reusada`. O descarte e a falha também vão para o
  log, com o motivo e sem o texto.

## Modelo e custo

- `EXPLICACAO_MODELO` define o modelo, com padrão `gemini-3.5-flash-lite`. Ele
  usa a mesma `GEMINI_API_KEY` da calibração. Cotas do Gemini valem por modelo,
  então a explicação e o embedding não competem.
- O pedido vai com temperatura 0,2 e `thinkingConfig.thinkingLevel` `minimal`.
  Nesse modelo, `thinkingBudget: 0` é recusado; com `minimal`, o pedido real
  gastou 0 tokens de pensamento.
- No plano gratuito, a cota medida em outubro de 2026 era de 15 pedidos por
  minuto e 500 por dia para Flash-Lite. Ela varia por projeto e muda sem
  aviso. Com uma chamada por task_run, o volume do projeto fica bem abaixo
  disso.
- Acima da cota, o Gemini responde 429, e o comentário sai com o texto fixo.

## Aviso de privacidade

No plano gratuito do Gemini, o Google pode usar o conteúdo dos pedidos para
melhorar os próprios produtos. O pedido leva os **nomes de arquivo** das
evidências, que revelam parte da estrutura do repositório analisado (por
exemplo, `internal/auth/sessao.go`). Não leva código.

Antes de ligar `EXPLICACAO_LIGADA` num repositório privado, confira se isso é
aceitável. Se não for, use uma chave de projeto pago, em que os pedidos não são
usados para treino, ou deixe a explicação desligada.

## Teste manual com o Gemini real

Fica fora do CI. Ele chama o modelo cinco vezes em cada caso: o PR #6 do
aPRova-TESTES (`falha_sandbox`), uma dependência nova e um risco baixo sem
motivos. Mostra o texto, a validação e a latência. Entre as chamadas espera
4,5s, para ficar abaixo da cota de 15 por minuto:

```sh
APROVA_TESTE_GEMINI_REAL=1 GEMINI_API_KEY=... go test ./internal/explicacao -run GeminiReal -v
APROVA_TESTE_GEMINI_REAL=1 GEMINI_API_KEY=... go test ./internal/explicacao -run 'GeminiReal/falha_sandbox' -v
```

## Variáveis

| Variável | Padrão | Efeito |
| --- | --- | --- |
| `EXPLICACAO_LIGADA` | `false` | liga a explicação no comentário; sem `GEMINI_API_KEY`, o worker avisa e segue sem ela |
| `EXPLICACAO_MODELO` | `gemini-3.5-flash-lite` | modelo Gemini de geração de texto |
| `EXPLICACAO_TIMEOUT` | `5s` | prazo da chamada ao modelo; passou, vale o texto fixo |
