from base.src.etl.train import Train
import asyncio

try:

    instance = Train()
    df = instance.get_train()
    print(df)


except Exception as e:

    raise Exception(e)