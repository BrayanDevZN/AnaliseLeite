from sklearn.ensemble import RandomForestClassifier
from src.etl.interface import LoadData
from pathlib import Path
import joblib

class TrainDf:

    def __init__(self)->None:
        self.BASE_DIR = Path(__file__).resolve().parent 
        

    async def _load(self) -> None:

        self.df =   LoadData(layer="cleaned").load()

    async def _data(self) -> None:
        
        self.x = self.df[["data", "temperatura", "ccs", "cpp"]]

        self.x["year"] = self.x["data"].dt.year
        self.x["month"] = self.x["data"].dt.month
        self.x["day"] = self.x["data"].dt.dayofweek
        self.x = self.x.drop(columns=["data"])

        

        self.y = (self.df["resultado_exame"] == "reprovado").astype(int) 

    async def _train(self) -> None:

        self.model = RandomForestClassifier(
            n_estimators=200,
            max_features="sqrt",
            class_weight="balanced",
            random_state=42,
            min_samples_leaf=5

        )

        self.model.fit(X=self.x, y=self.y)

    async def _save_model(self) -> None:

        path = self.BASE_DIR / "models"

        path.mkdir(exist_ok=True)

        joblib.dump(self.model, path / "joblib")

    async def run(self) -> None:

        await self._load()
        await self._data()
        await self._train()
        await self._save_model()


if __name__ == "__main__":
    import asyncio
    instance = TrainDf()
    asyncio.run(instance.run())



        