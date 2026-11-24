
url = "http://0.0.0.0:8000"

import requests
from pathlib import Path
response = requests.get(url=url + "/pipeline")
import pandas as pd
if response.status_code != 201:

    raise Exception(response.text)

response = response.json()
print(response)

response = requests.get(url=url + "/model/")

if response.status_code != 200:

    raise Exception(response.text)

models = response.json()
print(models)

BASE_DIR = Path(__file__).resolve().parent.parent.parent / "src/storage/test/analise_leite_teste.parquet"

for model in models:

    with open(BASE_DIR, "rb") as f:
        response = requests.post(url + "/model/file", params={"model": model}, files={"file": f})
        if response.status_code >400:

            raise Exception(response.text)

    data = response.json()
    df = pd.DataFrame(data)
    print(model)
    print(df)
