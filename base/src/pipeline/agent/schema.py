from pydantic import BaseModel
from datetime import datetime
from typing import Literal

states = Literal[
    "AC", "AL", "AP", "AM", "BA", "CE", "DF", "ES", "GO", "MA", "MT", "MS", "MG",
    "PA", "PB", "PR", "PE", "PI", "RJ", "RN", "RS", "RO", "RR", "SC", "SP", "SE", "TO",
]

class Args(BaseModel):

    cpp: int | None
    ccs: int | None
    temperature: float | None
    date: datetime | None
    state: states | None
    city: str | None



    
class Image(BaseModel):

    valid: bool 
    type: Literal["text", "analytic"]
    content: list[Args] | str 
    

