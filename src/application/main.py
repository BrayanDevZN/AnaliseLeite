from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware
from src.application.handle import router_pipeline


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




