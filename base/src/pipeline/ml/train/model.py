import argparse
from .templates.template import TrainModel
import os
from pathlib import Path
from base.src.pipeline.logs.log import logger
import asyncio


#Cria o parser com os comandos train, list e drop
def _parser() -> argparse.ArgumentParser:

    parser = argparse.ArgumentParser(prog="model", description="Gerencia os modelos treinados")
    commands = parser.add_subparsers(dest="command", required=True)

    train = commands.add_parser("train", help="treina um modelo")
    train.add_argument("--name", default=None, help="nome do modelo (padrao: anl_v<numero>)")
    train.add_argument("--template", default=None, help="template de treino (padrao: o ultimo registrado)")

    list_ = commands.add_parser("list", help="lista modelos ou templates")
    list_.add_argument("what", choices=["model", "templates"], help="o que listar")

    drop = commands.add_parser("drop", help="apaga um modelo")
    drop.add_argument("name", help="nome do modelo")

    return parser


def main() -> None:

    args = _parser().parse_args()

    BASE_DIR = Path(__file__).resolve().parent.parent / "models"
    BASE_DIR.mkdir(exist_ok=True)
    models = [str(model).replace(".joblib", "") for model in os.listdir(BASE_DIR)]

    if args.command == "train":

        if args.name is None:
            name = f"anl_v{len(models) + 1}"

        else:
            name = args.name

            if name in models:
                logger.warning(f"model will be {name} recreated")

        instance = TrainModel(model=name, template=args.template)
        asyncio.run(instance.run())

    elif args.command == "list":

        if args.what == "model":
            logger.info(f"Modelos disponiveis: {', '.join(models)}")

        else:
            templates = TrainModel(model="").templates
            logger.info(f"templates disponiveis: {', '.join(templates.keys())}")

    elif args.command == "drop":

        name = args.name

        if name not in models:
            logger.warning(f"not exists model {name}")

        else:

            os.remove(BASE_DIR / f"{name}.joblib")
            logger.info(f"Droped model {name}")


if __name__ == "__main__":
    main()
