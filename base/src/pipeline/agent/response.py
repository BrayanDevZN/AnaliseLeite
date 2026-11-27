import asyncio
import base64
from functools import wraps
from langchain_core.prompts import ChatPromptTemplate
from langchain_openai import ChatOpenAI
from src.pipeline.config import SETTINGS
from src.pipeline.log import logger
from .prompts.file import PROMPTS


class AgentError(Exception):
    pass



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
                response = await func(*args, **kwargs)

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

    def __init__(self):

        self.llm = ChatOpenAI(
            base_url=SETTINGS["BASE_URL"],
            api_key=SETTINGS["API_KEY"],
            model=SETTINGS["MODEL"]
        )


    @log_model
    async def response_image(self, image:bytes) -> str:

        img = base64.b64encode(bytes).decode("utf-8")

        prompt = ChatPromptTemplate.from_messages(
            [
                ("system", PROMPTS["IMAGE"]),
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

        chain = prompt | self.llm | 

        


