#Le a variavel de ambiente
from pathlib import Path 
from dotenv import load_dotenv
import os 
BASE_DIR = Path(__file__).resolve().parent.parent / ".env"
load_dotenv(BASE_DIR if os.path.exists(BASE_DIR) else None)

class EnvNotFound(Exception):

    pass 


API_KEY = os.getenv("ML_KEY")

if API_KEY is None:
    raise EnvNotFound(f"Expeted env ML_KEY")


