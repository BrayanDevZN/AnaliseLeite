import joblib 
from pathlib import Path 
import pandas as pd
from base.src.pipeline.logs.log import logger
from sklearn.ensemble import RandomForestClassifier
import os
import json
class ModelInference:

    def __init__(self, model:str|None=None)-> None:

        self.BASE_DIR = Path(__file__).resolve().parent / "models"
        self.model = model
        self._get_model()
        
        self._load()
        

    #Pega o nome do modelo
    def _get_model_path(self) -> None:

        self.models = [str(model)  for model in os.listdir(self.BASE_DIR)]
        
        if self.model is None:
            self.model = max(self.BASE_DIR.glob("*/joblib"), key=lambda p: p.stat().st_mtime).parent.name

        
        if not self.model in self.models:
            logger.error(f"Not exist model {self.model}")
            raise

        self.path_model = str(self.BASE_DIR / self.model / "joblib")
        self.path_config = str(self.BASE_DIR / self.model / "config.json")

    def _get_config(self) -> None:

        with open(self.path_config, "r", encoding="utf-8") as f:

            self.config = json.load(f)


    #Gera a resposta
    def _predict(self, df:pd.DataFrame):

        if len(df) == 1:

            if self.config["threshold"] is not None:

                prob = self.model.predict_proba(df)[0, 1]
                return "aprovado" if prob >= self.config["threshold"] else "reprovado"

            else:

                return "aprovado" if self.model.predict(df).item() == 0 else "repovado"


        if self.config["threshold"] is not None:

            prob = self.model.predict_proba(df)[:, 1]
            response = (prob >=self.config["threshold"]).astype(int)

        else:

            response = self.model.predict(df)

        df["predict"] = pd.Series(response).map({1: "repovado", 0: "aprovado"})
        return df




    def _load(self) -> None:

        logger.info(f"Loading model {self.model}...")

        self.model:RandomForestClassifier = joblib.load(self.path_model)

        logger.success(f"model {self.model} loaded with success")

    def response(self, data:str, temperatura:float, ccs:float, cpp:float, 
                 estado:str, cidade:str, tanque:str) -> str:

        logger.info("Generating response...")

        schema = pd.DataFrame(
            [{
                "data": data,
                "temperatura": temperatura,
                "ccs": ccs,
                "cpp": cpp,
                
                "estado": estado,
                "cidade": cidade,
                "tanque": tanque
            }]
        )



        schema["data"] = pd.to_datetime(schema["data"])
        schema["month"] = schema["data"].dt.month
        schema["year"] = schema["data"].dt.year
        schema["day"] = schema["data"].dt.dayofweek
        schema = schema.drop(columns=["data"])
        schema = pd.get_dummies(columns=["tanque", "estado", "cidade"], dtype=int, data=schema)
        schema = schema.reindex(columns=self.model.feature_names_in_, fill_value=0)
        

        resp = self._predict(df=schema)
        logger.success("Response generate with success")

        return resp

    

    def response_dataframe(self, df:pd.DataFrame) -> pd.DataFrame:

        columns = ["data", "temperatura", "ccs", "cpp", "tanque", "estado", "cidade"]
        schema = df.copy()
        schema = pd.get_dummies(columns=["tanque", "estado", "cidade"], dtype=int, data=schema)

        for column in columns:

            if not column in df.columns:
                raise KeyError(f"Expeted column {column}")

        droped = []

        for column in df.columns:
            if not column in columns:

                droped.append(column)

        if droped:

            schema = schema.drop(columns=droped)
            logger.info(f"Droped columns {" ,".join(droped)}")

        logger.info("Generating response for dataframe...")

        schema["data"] = pd.to_datetime(schema["data"])
        schema["month"] = schema["data"].dt.month
        schema["year"] = schema["data"].dt.year
        schema["day"] = schema["data"].dt.dayofweek
        schema = schema.drop(columns=["data"])
        schema = schema.reindex(columns=self.model.feature_names_in_, fill_value=0)


        df = self._predict(df=schema)
        
        logger.success("Response generated with success")

        return df

    def get_models(self) -> list[str]:

        return self.models


               



        