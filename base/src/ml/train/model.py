import sys 
from .template import TrainModel
import os
from pathlib import Path
from base.src.logs.log import logger
import asyncio
def main():
    args = sys.argv
    args.pop(0)
    BASE_DIR = Path(__file__).resolve().parent.parent / "models"
    models = [str(model).replace(".joblib", "") for model in os.listdir(BASE_DIR)]

    if args[0] == "train":

        if len(args) == 2:
            
            name = args[1]
            if name in models:
                logger.warning(f"model will be {name} recreated")

        else:
            name = f"anl_{len(models) + 1}"

        

        instance = TrainModel(model_name=name)
        asyncio.run(instance.run())

    if args[0] == "list":

        logger.info(f"Modelos disponiveis: {" ,".join(models)}")

    if args[0] == "drop":


        if len(args) != 2:
            raise ValueError("Expeted name model")

        name = args[1]

        if not name in models:
            logger.warning(f"not exists model {name}")

        else:

            os.remove(BASE_DIR / f"{name}.joblib")
            logger.info(f"Droped model {name}")


if __name__ == "__main__":
    main()

        

        

        
