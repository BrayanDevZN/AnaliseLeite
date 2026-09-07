from src.cache.manage import client



async def main():
    data = {"type": "teste"}


    await client.save(data=data)

    data = await client.read()
    print(data)



if __name__ == "__main__":
    import asyncio
    asyncio.run(main())