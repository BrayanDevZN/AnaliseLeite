from pathlib import Path 
from src.ml.inference import ModelInference
import pandas as pd



def test_model() -> None:

    model = ModelInference(model="anl_v2")

    BASE_DIR = Path(__file__).parent.parent.parent / "src/storage/test/analise_leite_teste.parquet"

    df = pd.read_parquet(BASE_DIR)

   
    df = model.response_dataframe(df=df)
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
    
if __name__ == "__main__":
    test_model()
