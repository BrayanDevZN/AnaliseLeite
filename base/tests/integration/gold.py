try:

    import asyncio

    from base.src.pipeline.etl.gold import Gold
    instance = Gold()
    df = instance.get_gold()
    print(df)

except Exception as e:
    raise Exception(e)