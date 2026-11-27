import asyncio
import base64
import inspect
from functools import wraps
from langchain_core.prompts import ChatPromptTemplate
from langchain_openai import ChatOpenAI
from src.pipeline.config import SETTINGS
from src.pipeline.log import logger
from .prompts.file import PROMPTS
from .schema import Image
from src.pipeline.ml.inference import ModelInference
import pandas as pd
class AgentError(Exception):
    pass


#Leva o valor para milhares, que e a escala do modelo. Confere os digitos do numero:
#4 ou mais digitos (320000) e numero cheio -> divide por 1000; 1 a 3 digitos (320) ja esta em milhares -> mantem.
#Vazio (None/NaN) continua vazio
def _to_thousands(value:int|float|None) -> float|None:

    if value is None or pd.isna(value):
        return None

    digits = str(int(abs(value)))

    return float(value) / 1000 if len(digits) >= 4 else float(value)



def log_model(func):
    @wraps(func)
   
    async def wrapper(*args, **kwargs):
        countdown = 0
        max_retries = 3

        while countdown < max_retries:
            try:
                logger.info(
                    f"Generating response model {SETTINGS['MODEL']}..."
                )

                # Repassa self e os demais argumentos exatamente como vieram
                response = func(*args, **kwargs)

                # So espera (await) quando o resultado for esperavel; funcao
                # sincrona ou valor comum (str, DataFrame...) passa direto
                if inspect.isawaitable(response):
                    response = await response

                logger.success("Response generated successfully")
                return response

            except Exception as error:
                countdown += 1  

                if countdown < max_retries:
                    logger.warning(
                        f"Error ({countdown}/{max_retries}) while generating"
                        " response, trying again..."
                    )
                    await asyncio.sleep(
                        40
                    )  
                    continue

                logger.error(f"Error after {max_retries} attempts: {error}")
                raise AgentError(error) from error

    return wrapper


class Agent:

    def __init__(self, model:str|None=None)->None:

        self.llm = ChatOpenAI(
            base_url=SETTINGS["BASE_URL"],
            api_key=SETTINGS["API_KEY"],
            model=SETTINGS["MODEL"], 
            max_completion_tokens=800
        )
        self.model = ModelInference(model=model)


    #Segunda chamada, sem saida estruturada: analise em texto do que a imagem mostra
    async def _describe(self, img:str) -> str:

        prompt = ChatPromptTemplate.from_messages(
            [
                ("system", PROMPTS["text"]),
                ("human", [{"type": "image_url", "image_url": {"url": "data:image/jpeg;base64,{img}"}}])
            ]
        )

        answer = await (prompt | self.llm).ainvoke({"img": img})

        return str(answer.content).strip()


    @log_model
    async def response_image(self, image:bytes) -> pd.DataFrame|str|None:

        img = base64.b64encode(image).decode("utf-8")

        prompt = ChatPromptTemplate.from_messages(
            [
                ("system", PROMPTS["image"]),
                (
                    "human",
                    [
                        {
                            "type": "image_url",
                            "image_url": {"url": "data:image/jpeg;base64,{img}"}
                        }
                    ]
                )
            
            ]

        )

        chain = prompt | self.llm.with_structured_output(Image)
        response =  await chain.ainvoke({"img": img})

        if response.valid:
            content = response.content
           
            if response.type == "analytic" and not isinstance(content, str):

                #Modelos pequenos as vezes devolvem amostras sem nenhum numero util; essas nao entram na previsao
                content = [item for item in content if any(value is not None
                                                           for value in (item.ccs, item.cpp, item.temperature))]

                #Sem nenhum numero util nao ha o que prever: a resposta passa a ser uma analise em texto
                if not content:
                    logger.warning("No usable parameters found, generating a text analysis instead")
                    return await self._describe(img)

                if len(content) >1:
                    content = pd.DataFrame([item.model_dump() for item in content])

                    #Nomes do Args -> nomes que o modelo usa
                    content = content.rename(columns={"temperature": "temperatura",
                                                      "date": "data", "state": "estado"})

                    #O modelo foi treinado em milhares; cada valor e convertido conforme os digitos
                    for column in ("ccs", "cpp"):
                        content[column] = pd.to_numeric(content[column]).map(_to_thousands)

                    content = self.model.response_dataframe(df=content)

                else:
                    content = content[0]

                    content = self.model.response(ccs=_to_thousands(content.ccs), cpp=_to_thousands(content.cpp),
                                                  estado=content.state,
                                                  data=content.date, temperatura=content.temperature)

            return content

        return None


        




                










        


