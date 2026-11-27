"""Teste manual da rota POST /model/image (agente de imagem).

Fica separado do api.py de proposito: cada chamada consome o modelo de linguagem (cota/tempo),
entao ela nao entra no CI. Suba o pipeline e rode: python -m tests.functional.image
"""
import requests
from pathlib import Path
from base.src.pipeline.config import SETTINGS

url = "http://0.0.0.0:7000"
header = {"Authorization": f"Bearer {SETTINGS['ML_KEY']}"}

IMAGE = Path(__file__).resolve().parent.parent.parent / "src/pipeline/storage/test/imagem_valida.jpg"

# uma unica requisicao; o timeout evita ficar pendurado se o modelo demorar ou travar
with open(IMAGE, "rb") as f:
    response = requests.post(url + "/model/image", files={"image": (IMAGE.name, f, "image/jpeg")},
                             headers=header, timeout=300)

print("status:", response.status_code)

if response.status_code > 400:

    raise Exception(response.text)

print(response.json())
