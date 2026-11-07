from sklearn.ensemble import RandomForestClassifier
from src.etl.load import LoadData
from pathlib import Path
import joblib

class TrainDf:

    def __init__(self)->None:
        self.BASE_DIR = Path(__file__).resolve().parent 
        

    async def _load(self) -> None:

        self.df = await  LoadData(layer="cleaned").run()

    async def _data(self) -> None:
        
        self.x = self.df[["data","tanque", "temperatura", "ccs", "cpp",
                          "silo"]]

        self.df["resultado_exame"] = self.df["resultado_exame"][
            self.df["resultado_exame"] == "aprovado"
        ]

        self.x["data"] = self.x["data"].astype(str)

        self.y = self.df["resultado_exame"]

    async def _train(self) -> None:

        model = RandomForestClassifier(
            n_estimators=200,
            max_features="sqrt",
            class_weight="balanced",
            random_state=42,
            min_samples_leaf=5

        )

        model.fit(X=self.x, y=self.y)

    async def _save_model(self) -> None:

        self.BASE_DIR.mkdir(exist_ok=True)

        joblib.dump(self.BASE_DIR / "models/predict.joblib")

    async def run(self) -> None:

        await self._load()
        await self._data()
        await self._train()
        await self._save_model()


if __name__ == "__main__":
    import asyncio
    instance = TrainDf()
    asyncio.run(instance.run())



        