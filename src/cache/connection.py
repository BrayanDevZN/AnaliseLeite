from src.logs.log import logger


"""
Cria conexão com redis e testa
"""

from redis import Redis
import os
class RedisConenction:

    def __init__(self)-> None:

        self.port = 6379

        redis_host = os.getenv("redis_host")

        self.host = "redis" if redis_host is None else redis_host


    #Cria a conexão
    def _connection(self) -> None:

        try:

            logger.info("Criando conexão com redis...")

            self.con = Redis(
                host=self.host, port=self.port, decode_responses=True
            )

        except Exception as e:
            logger.error(e)
            raise Exception(e)


    #Testa a conexão
    def _test(self) -> None:

        try:

            logger.info("Testando conexão com redis...")

            self.con.ping()

        except Exception as e:

            logger.error(e)
            raise Exception(e)

    #Executa os metodos e retorna conexão
    def run(self) -> Redis:
        self._connection()
        self._test()
        return self.con

    


        