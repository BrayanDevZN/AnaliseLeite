import joblib 
from pathlib import Path 
import pandas as pd
from base.src.pipeline.log import logger
from sklearn.ensemble import RandomForestClassifier
import os
import unicodedata
import json
class ModelInference:

    #Todas as colunas de entrada que o sistema conhece
    COLUMNS = ["data", "temperatura", "ccs", "cpp", "estado"]

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


    #Tira acentos e espacos sobrando (Divinópolis -> Divinopolis), igual aos nomes usados no treino
    @staticmethod
    def _normalize_text(value):

        if not isinstance(value, str):
            return value

        value = unicodedata.normalize("NFKD", value)
        value = "".join(char for char in value if not unicodedata.combining(char))

        return " ".join(value.split())

    #Valores usados quando a coluna nao existe ou esta vazia: mediana do dataset de treino (ccs e cpp em mil)
    DEFAULTS = {"temperatura": 3.5, "ccs": 440.0, "cpp": 46.0}

    #Preenche o que faltou nas colunas que o modelo usa. Numericas: mediana do treino. Data: hoje.
    #Estado vazio fica sem estado (nenhuma coluna de estado ligada). Avisa no log o que foi preenchido
    def _fill_missing(self, schema:pd.DataFrame) -> pd.DataFrame:

        filled = []

        for column in self._used_columns():

            if column not in schema.columns:
                schema[column] = None

            empty = schema[column].isna()

            if not empty.any():
                continue

            if column in self.DEFAULTS:

                schema[column] = pd.to_numeric(schema[column]).astype(float)
                schema.loc[empty, column] = self.DEFAULTS[column]

            elif column == "data":

                schema[column] = pd.to_datetime(schema[column])
                schema.loc[empty, column] = pd.Timestamp.today().normalize()

            else:
                filled.append(f"{column} (sem valor)")
                continue

            filled.append(f"{column} ({int(empty.sum())} de {len(schema)})")

        if filled:
            logger.warning(f"Missing values filled with default: {', '.join(filled)}")

        return schema

    #Cria as colunas derivadas so das colunas que ficaram no schema
    def _build_features(self, schema:pd.DataFrame) -> pd.DataFrame:

        if "data" in schema.columns:

            schema["data"] = pd.to_datetime(schema["data"])
            schema["month"] = schema["data"].dt.month
            schema["year"] = schema["data"].dt.year
            schema["day"] = schema["data"].dt.dayofweek
            schema = schema.drop(columns=["data"])

        if "estado" in schema.columns:
            schema["estado"] = schema["estado"].map(
                lambda value: self._normalize_text(value).upper() if isinstance(value, str) else value
            )

        categorical = [column for column in ["estado"] if column in schema.columns]

        if categorical:
            schema = pd.get_dummies(columns=categorical, dtype=int, data=schema)

        return schema.reindex(columns=self.model.feature_names_in_, fill_value=0)


    def response(self, data:str|None=None, temperatura:float|None=None, ccs:float|None=None,
                 cpp:float|None=None, estado:str|None=None) -> str:

        logger.info("Generating response...")

        values = {
            "data": data,
            "temperatura": temperatura,
            "ccs": ccs,
            "cpp": cpp,
            "estado": estado
        }

        used = self._used_columns()

        schema = pd.DataFrame([{column: values[column] for column in used}])
        schema = self._fill_missing(schema)
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

            logger.info(f"Droped columns {', '.join(map(str, droped))}")

        logger.info("Generating response for dataframe...")

        schema = self._fill_missing(schema)
        schema = self._build_features(schema)

        result = self._predict(df=schema)

        #O df original continua com todas as colunas, so ganha a coluna predict
        df = df.copy()
        df["predict"] = result if isinstance(result, str) else result["predict"].values
        
        logger.success("Response generated with success")

        return df

    def get_models(self) -> list[str]:

        return self.models


               



        