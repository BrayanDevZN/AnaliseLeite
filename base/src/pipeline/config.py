#Le a variavel de ambiente
from pathlib import Path 
from dotenv import load_dotenv
import os 
BASE_DIR = Path(__file__).resolve().parents[3] / ".env"
load_dotenv(BASE_DIR if os.path.exists(BASE_DIR) else None)

class EnvNotFound(Exception):

    pass 


NAMES = ["ML_KEY", "BASE_URL", "API_KEY", "MODEL"]
SETTINGS = {}

for name in NAMES:

    env = os.getenv(name)

    if env is None:
        raise EnvNotFound(f"Expeted env {name}")


    SETTINGS[name] = env