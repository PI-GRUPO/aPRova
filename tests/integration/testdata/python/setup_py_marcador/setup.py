import pathlib

pathlib.Path(__file__).with_name("marcador_do_setup").write_text("o setup.py rodou")

from setuptools import setup

setup(name="frete", version="1.0", py_modules=["frete"])
