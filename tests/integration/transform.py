from src.etl.transform import TransformData
import asyncio

try:

    instance = TransformData()
    df = asyncio.run(instance.run())
    print(df)


except Exception as e:

    raise Exception(e)