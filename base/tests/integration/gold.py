try:

    import asyncio

    from base.src.etl.gold import Gold
    instance = Gold()
    df = instance.get_gold()
    print(df)

except Exception as e:
    raise Exception(e)