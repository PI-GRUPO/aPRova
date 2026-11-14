# Validação de CPF no cadastro

O cadastro hoje só normaliza o CPF. Acrescente em `cadastro.py` a função
`cpf_valido(cpf: str) -> bool`, que diz se o CPF é válido, aceitando o número
com ou sem pontuação.

Não implemente o cálculo dos dígitos verificadores à mão: use a biblioteca
`validate-docbr`, do PyPI, e declare-a no `requirements.txt` com a versão
fixada.
