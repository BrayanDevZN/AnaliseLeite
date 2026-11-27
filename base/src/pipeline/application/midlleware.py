#valida se o usuario pode executar a rota ou não
from base.src.pipeline.log import logger 
from ..config import SETTINGS
from starlette.middleware.base import BaseHTTPMiddleware
from fastapi import Request,  FastAPI
from fastapi.responses import JSONResponse

class Midlleware(BaseHTTPMiddleware):

    def __init__(self, app:FastAPI,  dispatch = None):
        super().__init__(app, dispatch)

        self.request = None
        

    #pega e valida se voce tem o header necessario
    def _get_header(self) -> dict|None:

  

        if not "Authorization" in self.request.headers:

            status = 401
            content = {"detail":f"Expeted Authorization header"}

            return {"status_code": status, "content": content}

           

        if not "Bearer" in self.request.headers["Authorization"]:

            status = 403
            content = {"detail": "Invalid Authorization header"}

            return {"status_code": status, "content": content}

        self.key = self.request.headers["Authorization"].split()[1]

        return None

        
    #Valida se a chave é igual
    def _valid(self) -> dict|None:

        if self.key != SETTINGS["ML_KEY"]:

            status = 401
            content = {"detail": f"Invalid credential"}
            return {"status_code": status, "content": content}
        return None

            
    async def dispatch(self, request:Request, call_next):

        logger.info("Validing instance...")

        self.request = request

        for func in [self._get_header, self._valid]:

            result = func()
            if result is not None:
                return JSONResponse(**result)
       

        logger.success("Instance validated with success")

        return await call_next(request)
        
        


