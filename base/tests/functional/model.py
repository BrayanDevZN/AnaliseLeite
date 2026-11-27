from pathlib import Path 
from base.src.pipeline.ml.inference import ModelInference
import pandas as pd
from sklearn.metrics import recall_score, precision_score, f1_score, confusion_matrix

class RunModel:

    @staticmethod
    def df(df:pd.DataFrame, model:str|None=None) -> None:

        instance = ModelInference(model=model)

        df = instance.response_dataframe(df=df)
        df["predict_acert"] = (df["predict"] == df["resultado_exame"])
        acerts = df["predict_acert"].sum()
        total = len(df["predict_acert"])
        error = total - acerts
        print(df)
        results = df[["resultado_exame"]]
        results = (results["resultado_exame"] == "aprovado")
        mean_result = round(results.mean(), 2)
        print(f"Media de exames aprovados: {mean_result*100}%")
        percent_acert = acerts / total 
        percent_error = error / total 
        print(f"percentual de acertos: {round(percent_acert * 100, 1)}% \npercentual de erros: {round(percent_error * 100, 1)}%")

        y_real = df["resultado_exame"]
        y_pred = df["predict"]
        recall = recall_score(y_real, y_pred, pos_label="reprovado", zero_division=0)
        precision = precision_score(y_real, y_pred, pos_label="reprovado", zero_division=0)
        f1 = f1_score(y_real, y_pred, pos_label="reprovado", zero_division=0)
        print(f"recall: {recall:.2%} | precision: {precision:.2%} | f1: {f1:.2%}")
        print(confusion_matrix(y_real, y_pred, labels=["aprovado", "reprovado"]))
        
            
    @staticmethod
    def response(data:str, temperatura:float, ccs:float, cpp:float, 
                 estado:str, cidade:str, model:str|None=None) -> None:

        instance = ModelInference(model=model)
        print(instance.response(data=data, temperatura=temperatura, ccs=ccs, cpp=cpp,
                                estado=estado, cidade=cidade))


def test_model() -> None:

    
    BASE_DIR = Path(__file__).parent.parent.parent / "src/pipeline/storage/test/analise_leite_teste.parquet"

    df = pd.read_parquet(BASE_DIR)
    models = ModelInference().get_models()
    instance = RunModel()
    models.append(None)
    print(models)
    for model in models:
        instance.df(df=df, model=model)
        row = df.iloc[0]
        instance.response(
            data=str(row["data"]), temperatura=row["temperatura"], ccs=row["ccs"], cpp=row["cpp"],
            estado=row["estado"], cidade=row["cidade"], model=model
        )
        



   
if __name__ == "__main__":
    test_model()
