from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware
from src.application.handles.pipeline import router_pipeline
from src.application.handles.model import router_model

app = FastAPI()

#Configuração de cors
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_methods=["GET"],
    allow_credentials=False
)


#Inclui a rota
app.include_router(router_pipeline)
app.include_router(router_model)




