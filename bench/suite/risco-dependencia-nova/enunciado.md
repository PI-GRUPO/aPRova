# Validação de CPF no cadastro

O cadastro hoje só normaliza o CPF. Acrescente em `cadastro.py` a função
`cpf_valido(cpf: str) -> bool`, que diz se o CPF é válido, aceitando o número
com ou sem pontuação.

Não implemente o cálculo dos dígitos verificadores à mão: use a biblioteca
`validate-docbr` na versão 2.0.1, que já está instalada no ambiente, e
declare-a no `requirements.txt` como `validate-docbr==2.0.1`. Não há acesso à
rede.
