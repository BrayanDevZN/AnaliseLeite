"Junta todos os templates"
from base.src.pipeline.logs.log import logger
from .balance import Balance 
from .threshold import Threshold
from .base import BaseTemplate
class TemplateNotFound(Exception):
    pass 

class TrainModel:

    def __init__(self,model:str, template:str|None=None)->None:
        self.model = model

        self.template = template

        self.templates = {
            "balance": Balance,
            "threshold": Threshold
        }


    def _get_template_(self) -> None:

        if self.template is None:

            self.template = self.templates[list(self.templates.keys())[-1]]

        else:

            if not self.template in self.templates.keys():
                raise TemplateNotFound(f"Not found template {self.template}")

            self.template = self.templates[self.template]

        self.instance = self.template(model_name=self.model)

    #Decide como vai salvar
    async def _execute(self) -> None:

        result = await self.instance.run()
        
        base = (BaseTemplate(model=self.model, futures=list(result["model"].feature_names_in_), threshold=result["trheshold"])
                if isinstance(dict, result) else BaseTemplate(model=self.model, futures=list(result["model"].feature_names_in_)))

        base.save()
        

    async def run(self) -> None:

        self._get_template_()
        await self._execute()