from pydantic import BaseModel, Field
from typing import Literal

states = Literal[
    "AC", "AL", "AP", "AM", "BA", "CE", "DF", "ES", "GO", "MA", "MT", "MS", "MG",
    "PA", "PB", "PR", "PE", "PI", "RJ", "RN", "RS", "RO", "RR", "SC", "SP", "SE", "TO",
]
class SchemaResponse(BaseModel):

    temperature: float | None = None
    state: states | None = None
    data: str | None = None
    ccs: float | None = None
    cpp: float | None = None
    cidade: str | None = None
    tanque: str | None = None

