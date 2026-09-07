import logging

# Configuração global do sistema de logs
logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s [%(levelname)s] %(name)s: %(message)s",
    datefmt="%Y-%m-%d %H:%M:%S",
    handlers=[
        logging.FileHandler("src/logs/app.log", encoding="utf-8"),  # Salva em arquivo
        logging.StreamHandler()                            # Exibe no console
    ]
)

logger = logging.getLogger(__name__)
