from .silver import Silver
from src.logs.log import logger
import pandas as pd


class Gold(Silver):

    def __init__(self)-> None:
        super().__init__()

        
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


    def get_gold(self) -> None:

        self.df = self.read(layer="processed")

        if self.df is None:

            self.df = self.get_silver()

            logger.info("Executing layer processed...")

            self._result()
            self._group()
            self._rename()
            self._ranking()
            self.save(layer="processed", df=self.df)

        return self.df
            
