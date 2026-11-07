from src.logs.log import logger
import pandas as pd
from pathlib import Path
from .file import DfFile

class Bronze(DfFile):

    def __init__(self) -> None:
        super().__init__()


      
    #le oos dois dataframes
    def get_raw(self) -> dict[str, pd.DataFrame]:

        logger.info("Executing layer raw...")


        data = {
            "fazendas": self.read(layer="raw", name="fazendas.csv", sep=";"),
            "coletas": self.read(layer="raw", name="coletas.csv", sep=";")
        }

        logger.success("layer raw executed with success")

        return data

    





    
        