from base.model.src.logs.log import logger 
from .silver import Silver
import os 
import pandas as pd
class Train(Silver):

    def __init__(self):
        super().__init__()
       
        self.path = self.BASE_DIR / "train"
        self.path.mkdir(exist_ok=True)
       

    #Atualiza o nome dos arquivos
    def _get_names(self) -> None:

         _created = lambda path: getattr(os.stat(path), "st_birthtime",os.stat(path))
         files = sorted(os.listdir(self.path), key=lambda f: _created(self.path / f))
        

         
         self.files = []

         if files:

            for number, file in enumerate(files):

                
                new_file = f"train_v{number}.parquet"
                os.rename(self.path / file, self.path / new_file)
                self.files.append(new_file)


              
         
    #Pega os arquivos de train e junta com o dataframe
    def _concat(self) -> None:

         if self.files:

            logger.info("Merging datasets...")

            for file in self.files:

                df = self.read(layer="train", name=file)
                if not self.df.equals(df):

                    self.df = pd.concat([self.df, df], ignore_index=True)

            logger.success("Datasets mergeded with success")


        

    #Chama get silver e depois _concat
    def get_train(self) ->pd.DataFrame:
         self.get_silver()
         self._get_names()
         self._concat()
         return self.df



         
     


    


            
