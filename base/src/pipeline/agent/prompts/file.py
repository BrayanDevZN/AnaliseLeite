"Le os prompts"
from pathlib import Path 
import os 

BASE_DIR = Path(__file__).resolve().parent 
paths = [file for file in os.listdir(BASE_DIR) if BASE_DIR / file != Path(__file__).resolve()
         and ".md" in file]

#Le o prompt
def read(path:str) -> str:

    with open(path, "r", encoding="utf-8") as f:

        file = f.read()

    return file 



PROMPTS = {str(prompt).replace(".md", ""): read(BASE_DIR / prompt) for prompt in paths}

