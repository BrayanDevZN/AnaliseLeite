from pydantic import BaseModel
from typing import Literal

states = Literal[
    "AC", "AL", "AP", "AM", "BA", "CE", "DF", "ES", "GO", "MA", "MT", "MS", "MG",
    "PA", "PB", "PR", "PE", "PI", "RJ", "RN", "RS", "RO", "RR", "SC", "SP", "SE", "TO",
]
class SchemaResponse(BaseModel):

    temperature: float 
    state: states
    data: str 
    ccs: float 
    cpp: float 
    cidade: str 
    tanque: str

