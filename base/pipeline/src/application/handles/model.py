from fastapi import APIRouter, HTTPException, UploadFile, File
from base.model.src.ml.inference import ModelInference
router_model = APIRouter(prefix="/model", tags=["model"])
import pandas as pd

from base.model.src.application.schema import SchemaResponse
import io
@router_model.post("/")
async def response(data: SchemaResponse, model:str|None=None):

    try:
        instance = ModelInference(model=model)

        response = instance.response(
            temperatura=data.temperature,
            tanque=data.tanque,
            data=data.data,
            ccs=data.ccs,
            cpp=data.cpp,
            cidade=data.cidade,
            estado=data.state
        )

        return response 

    except Exception as error:

        raise HTTPException(
            status_code=501,
            detail=str(error)
        )


@router_model.post("/file")
async def response_file(file: UploadFile = File(...), model:str|None=None):

    try:
        content = await file.read()
        instance = ModelInference(model=model)

        if file.filename.endswith(".parquet"):
            
            df = pd.read_parquet(io.BytesIO(content))

        elif file.filename.endswith(".csv"):
            df = pd.read_csv(io.BytesIO(content), sep=";")

        else:
            raise HTTPException(
                status_code=429,
                detail="Expeted csv or parquet file"
            )

      

        response = instance.response_dataframe(df=df)

        return response.to_dict(orient="records")
    except Exception as error:

        raise HTTPException(
            status_code=501, detail=str(error)
        )

@router_model.get("/")
async def response_models():
    instance = ModelInference()

    return instance.get_models()



    
