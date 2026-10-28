from frete import calcular


def test_frete_leve():
    assert calcular(1, 2) == 3


def test_frete_pesado():
    assert calcular(10, 5) == 15