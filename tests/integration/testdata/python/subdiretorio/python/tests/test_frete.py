from frete import calcular


def test_frete():
    assert calcular(10, 5) == 15


def test_frete_sem_peso():
    assert calcular(0, 5) == 5
