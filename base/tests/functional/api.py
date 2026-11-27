
url = "http://0.0.0.0:7000"

import requests
from pathlib import Path
from base.src.pipeline.config import SETTINGS
API_KEY = SETTINGS["ML_KEY"]
header = {"Authorization": f"Bearer {API_KEY}"}
response = requests.get(url=url + "/pipeline", headers=header)
import pandas as pd
if response.status_code != 201:

    raise Exception(response.text)

response = response.json()
print(response)

response = requests.get(url=url + "/model/", headers=header)

if response.status_code != 200:

    raise Exception(response.text)

models = response.json()
print(models)

BASE_DIR = Path(__file__).resolve().parent.parent.parent / "src/pipeline/storage/test/analise_leite_teste.parquet"

row = pd.read_parquet(BASE_DIR).iloc[0]

payload = {
    "temperature": float(row["temperatura"]),
    "state": row["estado"],
    "data": str(row["data"]),
    "ccs": float(row["ccs"]),
    "cpp": float(row["cpp"])
}

for model in models:

    with open(BASE_DIR, "rb") as f:
        response = requests.post(url + "/model/file", params={"model": model}, files={"file": f}, headers=header)
        if response.status_code >400:

            raise Exception(response.text)

    data = response.json()
    df = pd.DataFrame(data)
    print(model)
    print(df)

    response = requests.post(url + "/model/", params={"model": model}, json=payload, headers=header)

    if response.status_code >400:

        raise Exception(response.text)

    print(response.json())


#Agente de imagem: uma unica requisicao (fora do loop dos modelos), com a imagem valida
IMAGE = Path(__file__).resolve().parent.parent.parent / "src/pipeline/storage/test/imagem_valida.jpg"

with open(IMAGE, "rb") as f:
    response = requests.post(url + "/model/image", files={"image": (IMAGE.name, f, "image/jpeg")}, headers=header)

if response.status_code > 400:

    raise Exception(response.text)

print("imagem:")
print(response.json())
