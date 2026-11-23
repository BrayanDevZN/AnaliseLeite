import sys 
from .template import TrainModel
import os
from pathlib import Path
from base.src.logs.log import logger
import asyncio
USAGE = "Uso: model <train [nome] | list | drop <nome>>"


def main() -> None:
    args = sys.argv[1:]
    BASE_DIR = Path(__file__).resolve().parent.parent / "models"
    BASE_DIR.mkdir(exist_ok=True)
    models = [str(model).replace(".joblib", "") for model in os.listdir(BASE_DIR)]

    if not args:
        print(USAGE)
        sys.exit(1)

    command = args[0]

    if command == "train":

        if len(args) == 2:

            name = args[1]
            if name in models:
                logger.warning(f"model will be {name} recreated")

        else:
            name = f"anl_{len(models) + 1}"

        instance = TrainModel(model_name=name)
        asyncio.run(instance.run())

    elif command == "list":

        logger.info(f"Modelos disponiveis: {', '.join(models)}")

    elif command == "drop":

        if len(args) != 2:
            raise ValueError("Expeted name model")

        name = args[1]

        if name not in models:
            logger.warning(f"not exists model {name}")

        else:

            os.remove(BASE_DIR / f"{name}.joblib")
            logger.info(f"Droped model {name}")

    else:
        print(USAGE)
        sys.exit(1)


if __name__ == "__main__":
    main()
