from src.logs.log import logger
import pandas as pd
from pathlib import Path

class ExtractData:
    def __init__(self)-> None:

        #caminho base
        self.BASE_DIR = Path(__file__).resolve().parent.parent / "storage"



    #carrega as duas tabelas
    def _load(self) -> None:

        try:

            logger.info("Lendo as duas tabelas na camada raw...")
            print(self.BASE_DIR)

            self.df_fazendas = pd.read_csv(self.BASE_DIR / "raw/fazendas.csv", sep=";")
            self.df_coletas = pd.read_csv(self.BASE_DIR / "raw/coletas.csv", sep=";")

        except Exception as e:
            logger.error(e)
            raise Exception(e)

            

    #Junta as duas 
    def _merge(self) -> None:

        try:
            logger.info("Juntando tabelas...")

            self.df = pd.merge(
                left=self.df_fazendas,
                right=self.df_coletas,
                how="inner",
                on="fazenda_id"
            )

        except Exception as e:
            logger.error(e)
            raise Exception(e)

    #Muda o tipo de da coluna data
    def _date(self) -> None:

        self.df["data"] = pd.to_datetime(self.df["data"])


    #Cria o caminho onde a nova tabela vai ser salva se ele não existir
    def _path(self) -> None:

        self.path = self.BASE_DIR / "cleaned/"

        self.path.mkdir(exist_ok=True, parents=True)

    #Salva o df no caminho
    def _save(self) -> None:

        try:

            logger.info("Salvando na camada cleaned...")
        
            

            self.df.to_parquet(path=self.path / "fazendas_analise.parquet")

        except Exception as e:

            logger.error(e)
            raise Exception(e)

    #Executa os metodos e retorna o novo dataframe
    def run(self) -> pd.DataFrame:

        self._load()
        self._merge()
        self._date()
        self._path()
        self._save()

        return self.df



        


    
