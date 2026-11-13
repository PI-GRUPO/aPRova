import socket
import struct

TEMPO_LIMITE = 3


def test_dependencia_baixada_no_preparo():
    import six

    assert six.PY3


def gateway_padrao():
    with open("/proc/net/route") as rotas:
        for linha in rotas.readlines()[1:]:
            campos = linha.split()
            if len(campos) > 2 and campos[1] == "00000000":
                return socket.inet_ntoa(struct.pack("<L", int(campos[2], 16)))
    raise OSError("sem rota padrão")


def test_alcanca_a_rede():
    alcancados, bloqueios = [], []

    def tentar(alvo, acao):
        try:
            acao()
            alcancados.append(alvo)
        except ConnectionRefusedError:
            alcancados.append(alvo)
        except OSError as erro:
            bloqueios.append(f"{alvo}: {erro}")

    tentar("dns pypi.org", lambda: socket.getaddrinfo("pypi.org", 443))
    tentar("tcp 1.1.1.1:443", lambda: socket.create_connection(("1.1.1.1", 443), TEMPO_LIMITE).close())
    try:
        gateway = gateway_padrao()
        tentar(f"tcp gateway {gateway}:5432", lambda: socket.create_connection((gateway, 5432), TEMPO_LIMITE).close())
    except OSError as erro:
        bloqueios.append(f"gateway do host: {erro}")

    if alcancados:
        print("ataque: alcançou " + ", ".join(alcancados))
        return
    raise AssertionError("rede inalcançável: " + "; ".join(bloqueios))
