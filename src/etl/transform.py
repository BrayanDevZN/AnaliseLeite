from src.logs.log import logger
import pandas as pd
from src.etl.extract import ExtractData
import os
from pathlib import Path
class TransformData:

    def __init__(self)-> None:

        #caminho onde o data frame ta salvo
        self.BASE_DIR = Path(__file__).resolve().parent.parent / "storage"
       

    #Confere se existe arquivo salvo, se existir, le, se não, executa o extract
    def _read(self) -> None:

        try:

            path = self.BASE_DIR / "cleaned/fazendas_analise.parquet"

            logger.info(f"Tentando ler {path}...")

            if os.path.exists(path):

                

                self.df = pd.read_parquet(path=path)

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

        self.df = self.df.sort_values(by=["ccs", "cpp", "media_aprovacao"], ascending=True)

    #Transforma resultado exame em 0 ou 1
    def _result(self) -> None:

        self.df["resultado_exame"] = (self.df["resultado_exame"] == "aprovado").astype(int)
       

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
                    "resultado_exame": "mean"



                }
            )

            
        except Exception as e:
            logger.error(e)
            raise Exception(e)


    #Renomeia a coluna resultado exame e arredonda
    def _rename(self) -> None:

        self.df = self.df.rename(columns={"resultado_exame": "media_aprovacao"})
        self.df["media_aprovacao"] = self.df["media_aprovacao"].round(2)

    

    #Cria o caminho se não existir
    def _path(self) -> None:

        self.path = self.BASE_DIR / "processed"
        self.path.mkdir(exist_ok=True, parents=True)


    #Salva o dataframe
    def _save(self) -> None:

        self.df.to_parquet(path=self.path / "fazenda_analise_resultado.parquet")


    #Executa os metodos e retorna o dataframe
    def run(self) -> pd.DataFrame:

        self._read()
        self._filter_comunit()
        self._result()
        self._group()
        self._rename()
        self._ranking()
        self._path()
        self._save()


        return self.df




    


    

    

     




    


    




    

    


    
        