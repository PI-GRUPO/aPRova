import pathlib


def test_setup_py_nunca_rodou():
    marcador = pathlib.Path(__file__).with_name("marcador_do_setup")
    assert not marcador.exists(), "o setup.py do PR rodou antes dos testes"
