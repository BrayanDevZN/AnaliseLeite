import asyncio
from pathlib import Path

import pandas as pd

from base.src.pipeline.agent.response import Agent

IMAGES_DIR = Path(__file__).resolve().parent.parent.parent / "src/pipeline/storage/test"


# Cada imagem testa um caminho do agente:
#   imagem_valida:           tabela com 4 amostras (2 dentro do limite, 2 acima) -> vira DataFrame com a previsao
#   imagem_sem_parametros:   leite, mas sem ccs, cpp, temperatura, data, estado ou cidade -> texto com a analise
#   imagem_invalida:         foto de um gato, nada a ver com leite -> None
def _is_prediction(response) -> bool:
    return isinstance(response, pd.DataFrame) and len(response) == 4 and "predict" in response.columns


def _is_text(response) -> bool:
    #Tem que ser uma analise (varias palavras), e nao uma previsao solta como "reprovado"
    return (isinstance(response, str) and response.strip().lower() not in ("aprovado", "reprovado")
            and len(response.split()) >= 8)


def _is_none(response) -> bool:
    return response is None


CASES = [
    ("imagem_valida.jpg", "DataFrame de 4 linhas com a coluna predict", _is_prediction),
    ("imagem_sem_parametros.jpg", "texto com a analise (nenhum campo achado, mas ha o que concluir)", _is_text),
    ("imagem_invalida.jpg", "None (imagem sem relacao com leite)", _is_none),
]


async def run() -> list[str]:

    agent = Agent()
    failed = []

    for name, expected, check in CASES:

        image = (IMAGES_DIR / name).read_bytes()

        print(f"\n===== {name} =====")
        response = await agent.response_image(image=image)
        print(type(response).__name__)
        print(response)

        ok = check(response)
        print(f"esperado: {expected} -> {'OK' if ok else 'FALHOU'}")

        if not ok:
            failed.append(name)

    return failed


def test_agent() -> None:

    failed = asyncio.run(run())

    assert not failed, f"imagens com resultado diferente do esperado: {', '.join(failed)}"


if __name__ == "__main__":

    test_agent()
