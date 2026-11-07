"Interface das camadas"
from.gold import Gold
from typing import Literal
import pandas as pd
class LoadData(Gold):

    def __init__(self, layer:Literal["processed", "cleaned", "raw"]="processed")->None:
        super().__init__()
        self.layer = layer 


    def load(self) -> pd.DataFrame | dict[str, pd.DataFrame]:

        executing = {
            "processed": self.get_gold,
            "cleaned": self.get_silver,
            "raw": self.get_raw
        }

        return executing[self.layer]()

        