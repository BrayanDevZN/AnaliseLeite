"""
Junta os dois modulos
"""

from src.cache.connection import RedisConenction
from src.cache.control import ControlCache

connection = RedisConenction().run()
client = ControlCache(connection=connection)
