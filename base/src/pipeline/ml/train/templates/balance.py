from sklearn.ensemble import RandomForestClassifier
from base.src.pipeline.etl.interface import LoadData
from pathlib import Path
from base.src.pipeline.log import logger
import joblib
import pandas as pd
class Balance:

    def __init__(self, model_name:str)->None:
        self.BASE_DIR = Path(__file__).resolve().parent.parent.parent
        self.name = model_name
        

    async def _load(self) -> None:

        self.df =   LoadData(layer="train").load()

    async def _data(self) -> None:

        logger.info("getting collumns...")
        
        self.x = self.df[["estado", "cidade", "data", "temperatura", "ccs", "cpp"]]
        
        self.x = pd.get_dummies(columns=["estado", "cidade"], dtype=int, data=self.x)

        self.x["year"] = self.x["data"].dt.year
        self.x["month"] = self.x["data"].dt.month
        self.x["day"] = self.x["data"].dt.dayofweek
        self.x = self.x.drop(columns=["data"])

        

        self.y = (self.df["resultado_exame"] == "reprovado").astype(int) 

    async def _train(self) -> None:

        logger.info(f"Training model {self.name}...")

        

        self.model = RandomForestClassifier(
            n_estimators=180,
            max_features=0.6,
           class_weight="balanced",
            random_state=42,
            min_samples_leaf=9,
            max_depth=8,
            criterion="gini"
            
       

        )

        self.model.fit(X=self.x, y=self.y)

        logger.success(f"model {self.name} trained with success")

    
    async def run(self) -> RandomForestClassifier:

        await self._load()
        await self._data()
        await self._train()

        return self.model 




        