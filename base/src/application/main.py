from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware
from base.src.application.handles.pipeline import router_pipeline
from base.src.application.handles.model import router_model
from base.src.application.midlleware import Midlleware

app = FastAPI()

#Configuração de cors
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_methods=["GET"],
    allow_credentials=False
)


#Adiciona o midlleware
app.add_middleware(Midlleware)


#Inclui a rota
app.include_router(router_pipeline)
app.include_router(router_model)



#Adiciona rota de verificação de saude
@app.get("health")
async def valid():
    return {"status": True}




