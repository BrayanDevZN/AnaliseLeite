from src.logs.log import logger
import pandas as pd
from src.etl.extract import ExtractData
import os
from pathlib import Path
class TransformData:

    def __init__(self)-> None:

        #caminho onde o data frame ta salvo
        self.BASE_DIR = Path(__file__).resolve().parent.parent / "storage/cleaned/fazendas_analise.parquet"
       

    #Confere se existe arquivo salvo, se existir, le, se não, executa o extract
    def _read(self) -> None:

        try:

            logger.info(f"Tentando ler {self.BASE_DIR}...")

            if os.path.exists(self.BASE_DIR):

                

                self.df = pd.read_parquet(path=self.BASE_DIR)

            else:

                logger.info("Arquivo não existe, executando pipeline...")

                self.df = ExtractData().run()

        except Exception as e:

            logger.error(e)
            raise Exception(e)
        

    #Filtra o dataframe e pega somente os tanques que são comunitarios
    def _filter_comunit(self) -> None:

        self.df = self.df[self.df["tipo_tanque"] == "comunitario"]


    #Rankeia do menor pro maior  pelo cpp e ccs
    def _ranking(self) -> None:

        self.df = self.df.sort_values(by=["ccs", "cpp", "media_resultado_exame"], ascending=True)

    #Agrupa pelas fazendas 
    def _group(self) -> None:

        try:

            logger.info("Agrupando dados...")

            self.df = self.df.groupby(["fazenda_id", "silo"]).agg(
                {
                    "nome_fazenda": "first",
                    "produtor": "first",
                    "cidade": "first",
                    "estado": "first",
                    "preco_litro": "sum",
                    "tanque": "first",
                    "tipo_tanque": "first",
                    "volume_litros": "mean",
                    "temperatura": "mean",
                    "ccs": "mean",
                    "cpp": "mean",
                    "motorista": "first",
                    "caminhao": "first",
                    "resultado_exame": "count"



                }
            )
        except Exception as e:
            logger.error(e)
            raise Exception(e)

    #Faz a media de aprovação
    def _mean_result(self) -> None:

        logger.info("Calculando a media de resultados dos exames...")

        self.df["media_resultado_exame"] = self.df["resultado_exame"].mean()
        self.df = self.df.drop(columns=["resultado_exame"])


    #Executa os metodos e retorna o dataframe
    def run(self) -> pd.DataFrame:

        self._read()
        self._filter_comunit()
        self._group()
        self._mean_result()
        self._ranking()


        return self.df




    


    

    

     




    


    




    

    


    
        