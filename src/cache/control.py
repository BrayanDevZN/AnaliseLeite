from src.logs.log import logger
from redis import Redis


#Salva e le o cache
class ControlCache:

    def __init__(self, connection:Redis)-> None:

        self.con = connection

    #salva o cache
    async def save(self, data:dict) -> None:

        try:
            logger.info("Salvando cache...")

            with self.con.pipeline() as session:

                session.hset(name="cache", mapping=data)
                session.expire(time=120, name="cache")
                session.execute()

        except Exception as e:

            logger.error(e)
            raise Exception(e)

    #Le o cache se existir
    async def read(self) -> dict|None:

        try:

            logger.info("Lendo cache...")

            cache = self.con.hgetall(name="cache")

            if cache is None:

                logger.info("Não existe cache!!")

            return cache

        except Exception as e:

            logger.error(e)
            raise Exception(e)

        
        