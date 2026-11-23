from base.src.logs.log import logger 
from .silver import Silver
import os 
import pandas as pd
class Train(Silver):

    def __init__(self):
        super().__init__()
       
        self.path = self.BASE_DIR / "train"
        self.path.mkdir(exist_ok=True)
        self.name = "train_v0.parquet"


   

    #Pega os arquivos de train e junta com o dataframe
    def _concat(self) -> None:

            files = os.listdir(self.path)



            if files and len(files) >1:

                logger.info(f"Merging files in {self.path}...")

                
           
                count = 0
                
            
                for file in self.files:
                   
                    df = self.read(layer="train", name=file)

                    self.df = pd.concat([self.df, df], ignore_index=True)

                    if "train_v" in file:

                        name = self.name.replace(".parquet", "") if ".parquet" in self.name else self.name 
                        count += int(name[-1])

                    self.name = f"train_v{count}.parquet"

                    del_path = self.path / file
                    os.remove(del_path)
                    logger.info(f"Removed path {del_path}!!")

            

                

                logger.success("Files merge with success")

            if not os.path.exists(self.path / self.name):

                self.save(name=self.name, layer="train", df=self.df)


    #Chama get silver e depois _concat
    def get_train(self) ->pd.DataFrame:
         self.get_silver()
         self._concat()
         return self.df



         
     


    


            
