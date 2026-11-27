import asyncio
from pathlib import Path

from base.src.pipeline.agent.response import Agent

IMAGES_DIR = Path(__file__).resolve().parent.parent.parent / "src/pipeline/storage/test"

# imagem_valida: laudo com cpp, ccs, temperatura, data, e estado (deve virar analytic)
# imagem_invalida: foto de um gato, nada a ver com leite (deve voltar None)
IMAGES = ["imagem_valida.jpg", "imagem_invalida.jpg"]


async def run() -> None:

    agent = Agent()

    for name in IMAGES:

        image = (IMAGES_DIR / name).read_bytes()

        print(f"\n===== {name} =====")
        response = await agent.response_image(image=image)
        print(type(response).__name__)
        print(response)


def test_agent() -> None:

    asyncio.run(run())


if __name__ == "__main__":

    test_agent()
