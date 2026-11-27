import joblib 
from pathlib import Path 
import pandas as pd
from base.src.pipeline.log import logger
from sklearn.ensemble import RandomForestClassifier
import os
import json
class ModelInference:

    #Todas as colunas de entrada que o sistema conhece
    COLUMNS = ["data", "temperatura", "ccs", "cpp", "estado", "cidade"]

    def __init__(self, model:str|None=None)-> None:

        self.BASE_DIR = Path(__file__).resolve().parent / "models"
        self.model = model
        self._get_model_path()
        
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
                return "reprovado" if prob >= self.config["threshold"] else "aprovado"

            else:

                return "aprovado" if self.model.predict(df).item() == 0 else "reprovado"


        if self.config["threshold"] is not None:

            prob = self.model.predict_proba(df)[:, 1]
            response = (prob >=self.config["threshold"]).astype(int)

        else:

            response = self.model.predict(df)

        df["predict"] = pd.Series(response).map({1: "reprovado", 0: "aprovado"})
        return df




    def _load(self) -> None:

        logger.info(f"Loading model {self.model}...")

        self._get_config()
        self.model:RandomForestClassifier = joblib.load(self.path_model)

        logger.success(f"model {self.model} loaded with success")


    #Descobre quais colunas de entrada o modelo usa, a partir das features do config
    def _used_columns(self) -> list[str]:

        features = list(self.config.get("features") or self.model.feature_names_in_)
        used = []

        for column in self.COLUMNS:

            if column in features or any(feature.startswith(f"{column}_") for feature in features):
                used.append(column)

        if "data" not in used and any(feature in features for feature in ("year", "month", "day")):
            used.insert(0, "data")

        return used


    #Cria as colunas derivadas so das colunas que ficaram no schema
    def _build_features(self, schema:pd.DataFrame) -> pd.DataFrame:

        if "data" in schema.columns:

            schema["data"] = pd.to_datetime(schema["data"])
            schema["month"] = schema["data"].dt.month
            schema["year"] = schema["data"].dt.year
            schema["day"] = schema["data"].dt.dayofweek
            schema = schema.drop(columns=["data"])

        categorical = [column for column in ["estado", "cidade"] if column in schema.columns]

        if categorical:
            schema = pd.get_dummies(columns=categorical, dtype=int, data=schema)

        return schema.reindex(columns=self.model.feature_names_in_, fill_value=0)


    def response(self, data:str|None=None, temperatura:float|None=None, ccs:float|None=None,
                 cpp:float|None=None, estado:str|None=None, cidade:str|None=None) -> str:

        logger.info("Generating response...")

        values = {
            "data": data,
            "temperatura": temperatura,
            "ccs": ccs,
            "cpp": cpp,
            "estado": estado,
            "cidade": cidade
        }

        used = self._used_columns()

        missing = [column for column in used if values[column] is None]

        if missing:
            raise ValueError(f"Expeted columns {', '.join(missing)}")

        schema = pd.DataFrame([{column: values[column] for column in used}])
        schema = self._build_features(schema)

        resp = self._predict(df=schema)
        logger.success("Response generate with success")

        return resp

    

    def response_dataframe(self, df:pd.DataFrame) -> pd.DataFrame:

        used = self._used_columns()
        schema = df.copy().reset_index(drop=True)
        droped = []

        for column in list(schema.columns):

            if not column in used:

                schema = schema.drop(columns=column)
                droped.append(column)

        if droped:

            logger.info(f"Droped columns {', '.join(droped)}")

        missing = [column for column in used if column not in schema.columns]

        if missing:
            raise KeyError(f"Expeted columns {', '.join(missing)}")

        logger.info("Generating response for dataframe...")

        schema = self._build_features(schema)

        result = self._predict(df=schema)

        #O df original continua com todas as colunas, so ganha a coluna predict
        df = df.copy()
        df["predict"] = result if isinstance(result, str) else result["predict"].values
        
        logger.success("Response generated with success")

        return df

    def get_models(self) -> list[str]:

        return self.models


               



        