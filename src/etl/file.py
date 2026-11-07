"Classe que le e salva os dataframes"
import pandas as pd 
from src.logs.log import logger
import os
from pathlib import Path
from typing import Literal
class DfFile:

    def __init__(self)-> None:

        self.BASE_DIR = Path(__file__).resolve().parent.parent / "storage"
        

    def read(self, layer:Literal["raw", "cleaned", "processed"], name:str="analise_leite.parquet") -> pd.DataFrame|None:

        try:

            path = self.BASE_DIR / layer / name


            logger.info(f"Reading {path}...")

            if not os.path.exists(path):

                logger.info(f"Not exists {path}")

                return None 

            return pd.read_csv(path, sep=";") if layer == "raw" else pd.read_parquet(path)

        except Exception as error:

            logger.error(f"error: {error}")
            raise

    def save(self, df:pd.DataFrame, layer:Literal["cleaned", "processed"]) -> None:

        try:

            path = self.BASE_DIR / layer
            path.mkdir(exist_ok=True)

            path = path / "analise_leite.parquet"

            logger.info(f"Saving dataframe in {path}...")

            df.to_parquet(path, index=False)

            logger.success(f"{path} saved with success")

        except Exception as error:

            logger.error(f"Error: {error}")
            raise







