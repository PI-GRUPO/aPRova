import six


def test_a_dependencia_do_requirements_esta_instalada():
    assert six.__version__ == "1.16.0"
