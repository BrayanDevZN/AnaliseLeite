FROM python:3.14.7

WORKDIR /app

COPY base /app/base

RUN pip install -r base/requirements.txt

CMD ["python", "-m", "uvicorn", "base.src.application.main:app", "--port", "8000", "--host", "0.0.0.0", "--reload"]
