import joblib 
from pathlib import Path 
import pandas as pd
from src.logs.log import logger
from sklearn.ensemble import RandomForestClassifier
import os
class ModelInference:

    def __init__(self, model:str)-> None:

        self.BASE_DIR = Path(__file__).resolve().parent / "models"
        self.model = model
        self._get_model()
        
        self._load()
        

    #Pega o nome do modelo
    def _get_model(self) -> None:

        models = [str(model).replace(".joblib", "")  for model in os.listdir(self.BASE_DIR)]
        print(models)
        if not self.model in models:
            logger.error(f"Not exist model {self.model}")
            raise

        self.path = self.BASE_DIR / f"{self.model}.joblib"


    def _load(self) -> None:

        logger.info(f"Loading model {self.model}...")

        self.model:RandomForestClassifier = joblib.load(self.path)

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
        schema["day"] = schema["data"].dt.dayofyear
        schema = schema.drop(columns=["data"])
        schema = pd.get_dummies(columns=["tanque", "estado", "cidade"], dtype=int, data=schema)
        schema = schema[self.model.feature_names_in_]
        

        resp = self.model.predict(schema).item()
        logger.success("Response generate with success")

        return "aprovado" if resp == 0 else "reprovado"

    

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
        schema["day"] = schema["data"].dt.dayofyear
        schema = schema.drop(columns=["data"])
        schema = schema[self.model.feature_names_in_]
        

        response = self.model.predict(schema)

        df["predict"] = pd.Series(response).map({1: "reprovado", 0: "aprovado"})

        logger.success("Response generated with success")

        return df


               



        