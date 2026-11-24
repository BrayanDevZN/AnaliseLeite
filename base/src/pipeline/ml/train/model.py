import sys 
from .templates.template import TrainModel
import os
from pathlib import Path
from base.src.pipeline.logs.log import logger
import asyncio
USAGE = "Uso: model <train [nome] | list | drop <nome>>"


def main() -> None:
    args = sys.argv[1:]
    BASE_DIR = Path(__file__).resolve().parent.parent / "models"
    BASE_DIR.mkdir(exist_ok=True)
    models = [str(model).replace(".joblib", "") for model in os.listdir(BASE_DIR)]
    instance = TrainModel()

    if not args:
        print(USAGE)
        sys.exit(1)

    command = args[0]

    if command == "train":

        if len(args) >= 2:

            name = args[1]
            if name in models:
                logger.warning(f"model will be {name} recreated")

        else:
            name = f"anl_v{len(models) + 1}"

        instance.name = name

        if len(args) == 3:

            template = args[2]
            instance.template = template

        
        asyncio.run(instance.run())

    elif command == "list":

        if len(args) != 2:
            raise ValueError("Expeted arg model or templates")

        arg = args[1]

        if arg == "model":
            logger.info(f"Modelos disponiveis: {', '.join(models)}")

        elif arg == "templates":
            logger.info(f"templates disponiveis: {", ".join(instance.templates.keys())}")

        else:
            raise ValueError(f"Not expeted arg {arg}")

        

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
