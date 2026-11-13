from frete import calcular


def test_frete_leve():
    assert calcular(1, 2) == 3


def test_frete_com_taxa():
    assert calcular(10, 5) == 16, f"frete com taxa: esperava 16, obtive {calcular(10, 5)}"


def test_frete_zero():
    assert calcular(0, 0) == 0