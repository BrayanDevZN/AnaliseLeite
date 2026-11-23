from fastapi import APIRouter, HTTPException, UploadFile, File
from src.ml.inference import ModelInference
router_model = APIRouter(prefix="/model", tags=["model"])
import pandas as pd
instance = ModelInference(model="anl_v2")
from src.application.schema import SchemaResponse
import io
@router_model.post("/")
async def response(data: SchemaResponse):

    try:

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

        HTTPException(
            status_code=501,
            detail=str(error)
        )


@router_model.post("/file")
async def response_file(file: UploadFile = File(...)):

    try:
        content = await file.read()

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

    
