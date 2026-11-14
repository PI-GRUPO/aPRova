def normalizar_cpf(cpf: str) -> str:
    return "".join(caractere for caractere in cpf if caractere.isdigit())
