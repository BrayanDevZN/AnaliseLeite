"Junta todos os templates"
from base.src.pipeline.logs.log import logger
from .balance import Balance 


class TemplateNotFound(Exception):
    pass 

class TrainModel:

    def __init__(self,model:str, template:str|None=None)->None:
        self.model = model

        self.template = template

        self.templates = {
            "balance": Balance
        }


    def _get_template_(self) -> None:

        if self.template is None:

            self.template = self.templates[list(self.templates.keys())[-1]]

        else:

            if not self.template in self.templates.keys():
                raise TemplateNotFound(f"Not found template {self.template}")

            self.template = self.templates[self.template]

        self.instance = self.template(model_name=self.model)

    async def run(self) -> None:

        self._get_template_()
        await self.instance.run()