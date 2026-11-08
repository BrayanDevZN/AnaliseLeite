from fastapi import APIRouter, HTTPException, UploadFile, File
from src.ml.inference import ModelInference
router_model = APIRouter(prefix="/model", tags=["model"])
import pandas as pd
instance = ModelInference(model="anl_v2")
from src.application.schema import SchemaResponse
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

        end = [".csv", ".parquet"]

        if file.filename.endswith((".csv", ".parquet")):
            raise HTTPException(
                detail="Expeted csv or parquet file",
                status_code=429
            )

        content = await file.read()

        df = pd.DataFrame(content)

        response = instance.response_dataframe(df=df)

        return response.to_dict(orient="records")
    except Exception as error:

        raise HTTPException(
            status_code=501, detail=str(error)
        )

    
