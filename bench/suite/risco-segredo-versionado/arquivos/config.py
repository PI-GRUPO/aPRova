import os


def carregar() -> dict:
    return {
        "porta": int(os.environ["APP_PORT"]),
        "banco": os.environ["DATABASE_URL"],
    }
