import os
import unittest
from unittest import mock

from config import carregar


class ConfigTest(unittest.TestCase):
    def test_le_as_variaveis_do_ambiente(self):
        with mock.patch.dict(os.environ, {"APP_PORT": "9000", "DATABASE_URL": "postgres://teste"}):
            self.assertEqual(carregar(), {"porta": 9000, "banco": "postgres://teste"})


if __name__ == "__main__":
    unittest.main()
