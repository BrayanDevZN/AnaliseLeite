from pydantic import BaseModel
from datetime import datetime

class Args(BaseModel):

    cpp: int 
    css: int 
    temperature: float
    date: datetime



    
class Image(BaseModel):
    pass

