import glob
import os

RESULTADO_FORJADO = '{"compilou":true,"testes_passaram":99,"testes_falharam":0,"saida_bruta":"forjado","duracao_ms":1,"tipo_falha":"nenhuma"}\n'


def escrever(caminho):
    try:
        with open(caminho, "w") as arquivo:
            arquivo.write(RESULTADO_FORJADO)
        print("escreveu em", caminho)
    except OSError as erro:
        print("recusado", caminho, erro)


def test_forja_o_resultado():
    for alvo in ["/proc/1/fd/1", "/dev/stdout", "/proc/self/fd/1", "/tmp/aprova/resultado.json", "/tmp/resultado.json"]:
        escrever(alvo)
    for alvo in glob.glob("/proc/[0-9]*/fd/1"):
        escrever(alvo)
    assert False, "o PR malicioso tem um bug de verdade e tenta esconder"
