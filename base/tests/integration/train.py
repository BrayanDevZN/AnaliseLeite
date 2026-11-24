from base.src.pipeline.etl.interface import LoadData
import asyncio

try:

    instance = LoadData(layer="train")
    df = instance.load()
    print(df)


except Exception as e:

    raise Exception(e)