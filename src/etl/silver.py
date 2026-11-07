from .bronze import  Bronze
from src.logs.log import logger
import pandas as pd

class Silver(Bronze):

    def __init__(self) -> None:
        super().__init__()

    #junta os dataframes
    def _merge(self) -> None:

        logger.info("Merging dataframes...")

        
        self.df = pd.merge(
            left=self.dfs["fazendas"],
            right=self.dfs["coletas"],
            on="fazenda_id",
            how="inner"
            )    
                    

    #Muda o tipo de da coluna 
    def _type(self) -> None:

        logger.info("Alterando tipos...")
        self.df["data"] = pd.to_datetime(self.df["data"])
        self.df["temperatura"] = self.df["temperatura"].str.replace(",", ".")
        self.df["temperatura"] = self.df["temperatura"].astype(float)
        self.df["preco_litro"] = self.df["preco_litro"].str.replace(",", ".")
        self.df["preco_litro"] = self.df["preco_litro"].astype(float)
        self.df["ccs"] = self.df["ccs"].astype(int)
        self.df["cpp"] = self.df["cpp"].astype(int)


    def get_silver(self) -> pd.DataFrame:

        self.df = self.read(layer="cleaned")

        if self.df is None:

            self.dfs = self.get_raw()

            logger.info(f"Executing layer cleaned...")

            self._merge()
            self._type()
            self.save(layer="cleaned", df=self.df)

            logger.success("Layer cleaned executed with success")

        return self.df



    

