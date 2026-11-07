from src.logs.log import logger
from pathlib import Path
import pandas as pd
import os
from src.etl.transform import TransformData
from src.etl.extract import ExtractData
from typing import Literal
class LoadData:

    def __init__(self, layer:Literal["raw", "cleaned", "processed"]|None=None)-> None:
        self.layer = layer

        self.BASE_DIR = Path(__file__).resolve().parent.parent / "storage"


    #acha qual camada ler
    async def _layer(self) -> None:

        try:



            

            if self.layer is not None:
                match self.layer:
                    case "processed":
                        layers = ["processed", "cleaned", "raw"]
                    case "cleaned":
                        layers = ["cleaned", "raw"]

                    case "raw":
                        layers = ["raw"]

            else:
                layers = ["processed", "cleaned", "raw"]


            for layer in layers:

                path = self.BASE_DIR / layer

                logger.info(f"Buscando dados na camada {layer}...")

                if not os.path.exists(path) or not os.listdir(path):

                    logger.info(f"Ainda não ha dados na camada {layer}!!")
                    continue

                self.layer = layer
                
                break

        except Exception as e:
            logger.error(e)
            raise Exception(e)

    #Executa a camada escolhida
    async def _execute(self) -> None:

        execute = {
            "raw": ExtractData().run,
            "cleaned": TransformData().run,
            "processed": self._read
        }

        self.df = await execute[self.layer]()

    #Le na camada processed caso prescise
    async def _read(self) -> pd.DataFrame:

        try:

            logger.info("Lendo na camada processed...")

            return pd.read_parquet(path=self.BASE_DIR / "processed/fazenda_analise_resultado.parquet")

        except Exception as e:
            logger.error(e)
            raise Exception(e)


    #Executa os metodos e retorna os dados
    async def run(self) -> dict:

        await self._layer()
        await self._execute()

        return self.df.to_dict(orient="records")



    
    







    


                





                

        