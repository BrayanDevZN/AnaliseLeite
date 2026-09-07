
url = "http://0.0.0.0:8000/pipeline"

import requests

response = requests.get(url=url)

if response.status_code != 201:

    raise Exception(response.text)

data = response.json()
print(data)

