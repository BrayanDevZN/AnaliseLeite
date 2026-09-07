from fastapi import APIRouter, HTTPException
from fastapi.responses import JSONResponse
from src.etl.load import LoadData
from src.cache.manage import client
router_pipeline = APIRouter(prefix="/pipeline", tags=["pipeline"])


@router_pipeline.get("/")
async def get():

    try:

        data = await client.read()

        if not data:


            instance = LoadData()

            data = await instance.run()

        return JSONResponse(
            status_code=201,
            content=data
        )

    except Exception as e:

        raise HTTPException(
            status_code=501,
            detail=e
        )
