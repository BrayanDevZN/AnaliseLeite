"Interface das camadas"
from.gold import Gold
from typing import Literal
import pandas as pd
from .train import Train
class LoadData(Gold, Train):

    def __init__(self, layer:Literal["processed", "cleaned", "raw", "train"]="processed")->None:
        super().__init__()
        self.layer = layer 


    def load(self) -> pd.DataFrame | dict[str, pd.DataFrame]:

        executing = {
            "processed": self.get_gold,
            "cleaned": self.get_silver,
            "raw": self.get_raw,
            "train": self.get_train
        }

        return executing[self.layer]()

        