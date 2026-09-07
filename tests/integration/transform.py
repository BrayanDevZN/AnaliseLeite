from src.etl.transform import TransformData


try:

    instance = TransformData()
    df = instance.run()
    print(df)


except Exception as e:

    raise Exception(e)