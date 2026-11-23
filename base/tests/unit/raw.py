import asyncio
try:

    from src.etl.bronze import Bronze


    instance = Bronze()

    df = instance.get_raw()

    print(df["fazendas"])
    print(df["coletas"])

except Exception as e:

    raise Exception(e)