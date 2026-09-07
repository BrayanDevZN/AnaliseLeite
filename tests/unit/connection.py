


from src.cache.connection import RedisConenction
from redis import Redis
client = RedisConenction().run()

if not isinstance(client, Redis):
    raise ValueError("Connection Error")