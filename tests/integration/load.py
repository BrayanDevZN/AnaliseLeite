try:

    import asyncio

    from src.etl.load import LoadData
    instance = LoadData()
    df = asyncio.run(instance.run())
    print(df)

except Exception as e:
    raise Exception(e)