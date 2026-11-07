from src.etl.silver import Silver
import asyncio

try:

    instance = Silver()
    df = instance.get_silver()
    print(df)


except Exception as e:

    raise Exception(e)