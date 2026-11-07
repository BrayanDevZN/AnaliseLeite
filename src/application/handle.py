from fastapi import APIRouter, HTTPException
from fastapi.responses import JSONResponse
from src.etl.interface import LoadData
from src.cache.manage import client
from typing import Literal
router_pipeline = APIRouter(prefix="/pipeline", tags=["pipeline"])


@router_pipeline.get("/")
async def get(layer:Literal["processed", "cleaned", "raw"]="processed"):

    try:

        data = await client.read()

        if not data:


            instance = LoadData(layer=layer)

            data =  instance.load()

            if isinstance(data, dict):
                for key, value in data.items():

                    data[key] = value.to_dict(orient="records")

            else:
                data = data.to_dict(orient="records")

            client.save(data=data)

        return JSONResponse(
            status_code=201,
            content=data
        )

    except Exception as e:

        raise HTTPException(
            status_code=501,
            detail=str(e)
        )
