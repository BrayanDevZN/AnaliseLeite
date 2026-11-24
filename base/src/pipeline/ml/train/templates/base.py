"classe pra salvar os arquivos"
import joblib
import json 
from pathlib import Path 
import pandas as pd
from datetime import timezone, datetime
from base.src.pipeline.logs.log import logger
from typing import Any
class BaseTemplate:

    def __init__(self,name:str,model:Any, futures:list, threshold:float|None=None)->None:
        self.BASE_DIR = Path(__file__).resolve().parent.parent.parent
        self.futures = futures
        self.threshold = threshold
        self.name = name
        self.model = model
        self.path =  self.BASE_DIR / self.name
        self.path.mkdir(exist_ok=True)
        
       

    #Salva em json
    def _json_save_(self) -> None:

        logger.info(f"Saving config from model {self.name}...")

        now = str(datetime.now(timezone.utc))
        content = {
            "name": self.name,
            "created_at": now,
            "features": self.futures,
            "threshold": self.threshold
        }

        with open(self.path / "config.json", "w", encoding="utf-8") as f:

            json.dump(content, f, ensure_ascii=False, indent=4)

        logger.success("Config saved with success")

    #Salva o modelo
    def _model_save_(self) -> None:

        logger.info(f"Saving model {self.name}...")

        joblib.dump(self.model, self.path / "joblib")

        logger.success("Model saved with sucess")


    #Chama os dois metodos de save em ordem
    def save(self) -> None:

        self._json_save_()
        self._model_save_()



       





    

    
        