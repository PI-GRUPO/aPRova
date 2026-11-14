import unittest

from cadastro import normalizar_cpf


class CadastroTest(unittest.TestCase):
    def test_normaliza_cpf_com_pontuacao(self):
        self.assertEqual(normalizar_cpf("529.982.247-25"), "52998224725")


if __name__ == "__main__":
    unittest.main()
