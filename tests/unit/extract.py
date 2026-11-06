import asyncio
try:

    from src.etl.extract import ExtractData


    instance = ExtractData()

    df = asyncio.run(instance.run())

    print(df)

except Exception as e:

    raise Exception(e)