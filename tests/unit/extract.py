
try:

    from src.etl.extract import ExtractData


    instance = ExtractData()

    df = instance.run()

    print(df)

except Exception as e:

    raise Exception(e)