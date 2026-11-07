<div align="center">
  <h1>🥛 Análise Leite</h1>
  <h3>Milk Traceability &amp; Data Intelligence</h3>
  <p>
    <em>From farm collection to smarter, safer dairy decisions.</em>
  </p>

  <p>
    <a href="https://github.com/BrayanDevZN/AnaliseLeite/actions/workflows/unit.yaml"><img src="https://img.shields.io/github/actions/workflow/status/BrayanDevZN/AnaliseLeite/unit.yaml?branch=develop&style=for-the-badge&logo=githubactions&logoColor=white&label=Unit%20Tests" alt="Unit tests"></a>
    <a href="https://github.com/BrayanDevZN/AnaliseLeite/actions/workflows/integration.yaml"><img src="https://img.shields.io/github/actions/workflow/status/BrayanDevZN/AnaliseLeite/integration.yaml?branch=develop&style=for-the-badge&logo=githubactions&logoColor=white&label=Integration%20Tests" alt="Integration tests"></a>
    <a href="https://github.com/BrayanDevZN/AnaliseLeite/actions/workflows/functional.yaml"><img src="https://img.shields.io/github/actions/workflow/status/BrayanDevZN/AnaliseLeite/functional.yaml?branch=develop&style=for-the-badge&logo=githubactions&logoColor=white&label=Functional%20Tests" alt="Functional tests"></a>
    <img src="https://img.shields.io/badge/CI%2FCD-GitHub_Actions-2088FF?style=for-the-badge&logo=githubactions&logoColor=white" alt="CI/CD with GitHub Actions">
  </p>

  <p>
    <img src="https://img.shields.io/badge/Python-3.14-3776AB?style=for-the-badge&logo=python&logoColor=yellow" alt="Python 3.14">
    <img src="https://img.shields.io/badge/FastAPI-0.143-009688?style=for-the-badge&logo=fastapi&logoColor=cian" alt="FastAPI 0.143">
    <img src="https://img.shields.io/badge/Pandas-3.0-150458?style=for-the-badge&logo=pandas&logoColor=green" alt="Pandas 3.0">
    <img src="https://img.shields.io/badge/scikit--learn-1.9-F7931E?style=for-the-badge&logo=scikitlearn&logoColor=white" alt="scikit-learn 1.9">
    <img src="https://img.shields.io/badge/Redis-8-DC382D?style=for-the-badge&logo=redis&logoColor=red" alt="Redis 8">
    <img src="https://img.shields.io/badge/Docker-ready-2496ED?style=for-the-badge&logo=docker&logoColor=blue" alt="Docker ready">
  </p>
</div>

<p align="center">
  <img src="assets/analise-leite-flow.gif" alt="Animated milk traceability flow from farm to dairy plant" width="100%">
</p>

## Sobre o projeto

O **Análise Leite** é um projeto de engenharia de dados voltado ao rastreamento da cadeia de coleta de uma cooperativa leiteira. Ele relaciona fazendas e coletas, trata os dados operacionais, calcula indicadores de qualidade, disponibiliza o resultado por uma API REST e prepara a base para um modelo de machine learning que prevê a aprovação dos exames.

O cenário parte de um problema real do setor: quando leite com resíduos ou fora dos padrões entra em um tanque comunitário, segue em um caminhão e chega a um silo, vários lotes podem ser afetados. Sem dados relacionados, é difícil reconstruir a origem do problema e distribuir responsabilidades de forma justa.

O pipeline ajuda a responder perguntas como:

- quais fazendas e silos apresentam os melhores indicadores de qualidade;
- qual é a média de aprovação dos exames;
- quais volumes, temperaturas, níveis de CCS e CPP foram registrados;
- quais tanques, motoristas e caminhões participaram da coleta;
- como os dados de cadastro das fazendas se relacionam aos eventos de coleta;
- quais características de uma coleta (temperatura, CCS, CPP, tanque, silo) estão associadas à reprovação do exame.

> Os dados presentes no repositório são fictícios e foram preparados para fins acadêmicos.

## Visão geral

| Componente | Responsabilidade |
|---|---|
| **FastAPI** | Expõe o resultado do pipeline no endpoint `GET /pipeline/` |
| **Pandas** | Lê, relaciona, limpa, agrupa e ordena os dados |
| **Parquet / PyArrow** | Armazena as camadas tratada e processada de forma eficiente |
| **scikit-learn / joblib** | Treina um `RandomForestClassifier` e persiste o modelo |
| **Redis** | Disponibiliza uma camada de cache com expiração de 120 segundos |
| **Docker Compose** | Sobe a API e o Redis em serviços integrados |
| **GitHub Actions** | Executa verificações unitárias, de integração e funcionais |

Atualmente, a base contém:

- **40 fazendas** cadastradas;
- **1.092 coletas** na camada bruta;
- **120 registros agregados** por fazenda e silo na camada processada.

## Arquitetura

O ETL segue a arquitetura em camadas (medalhão): **bronze** (`raw`), **silver** (`cleaned`) e **gold** (`processed`). Cada classe herda da anterior, e `LoadData` escolhe qual camada entregar.

```mermaid
flowchart LR
    U[Cliente HTTP] -->|GET /pipeline/?layer=...| API[FastAPI]
    API --> CACHE{Redis possui dados?}
    CACHE -->|Sim| RESP[Resposta JSON]
    CACHE -->|Não| LOAD[LoadData]

    LOAD -->|raw| BRONZE[Bronze.get_raw]
    LOAD -->|cleaned| SILVER[Silver.get_silver]
    LOAD -->|processed| GOLD[Gold.get_gold]

    RAW[(CSV raw)] --> BRONZE
    BRONZE --> SILVER
    SILVER --> CLEAN[(Parquet cleaned)]
    CLEAN --> GOLD
    GOLD --> PROCESSED[(Parquet processed)]

    BRONZE --> RESP
    SILVER --> RESP
    GOLD --> RESP

    CLEAN -.-> TRAIN[TrainDf]
    TRAIN -.-> MODEL[(predict.joblib)]
```

### Fluxo de uma requisição

1. O cliente chama `GET /pipeline/`, opcionalmente com `?layer=raw|cleaned|processed` (padrão: `processed`).
2. A API consulta a chave `cache` no Redis.
3. Em caso de cache vazio, `LoadData(layer)` executa o método da camada pedida.
4. Cada camada reaproveita o arquivo já salvo quando ele existe e, se não existir, gera a partir da camada anterior.
5. O resultado é convertido em JSON: uma lista de registros, ou um objeto com as tabelas `fazendas` e `coletas` na camada `raw`.
6. A API responde com HTTP `201`.

## Pipeline ETL

### 1. Bronze — camada `raw`

O módulo `src/etl/bronze.py` (`Bronze.get_raw`) lê `fazendas.csv` e `coletas.csv`, ambos separados por `;`, e devolve as duas tabelas **sem alterações**, em um dicionário `{"fazendas": df, "coletas": df}`.

### 2. Silver — camada `cleaned`

O módulo `src/etl/silver.py` (`Silver.get_silver`):

- relaciona as tabelas por `fazenda_id` usando um *inner join*;
- converte `data` para data/hora;
- troca vírgula por ponto nos campos decimais;
- converte preço, temperatura, CCS e CPP para tipos numéricos;
- persiste o resultado em Parquet.

### 3. Gold — camada `processed`

O módulo `src/etl/gold.py` (`Gold.get_gold`):

- converte `resultado_exame` em `1` para aprovado e `0` para não aprovado;
- agrupa os dados por fazenda e silo;
- calcula médias de volume, temperatura, CCS, CPP e aprovação;
- aplica soma ao campo `preco_litro` dentro de cada agrupamento;
- preserva dados descritivos, como fazenda, produtor, tanque, motorista e caminhão;
- renomeia o indicador calculado para `media_aprovacao`;
- ordena os registros por CCS, CPP e média de aprovação;
- grava o resultado final em Parquet.

### 4. Interface — `LoadData`

O módulo `src/etl/interface.py` mapeia `raw`, `cleaned` e `processed` para `get_raw`, `get_silver` e `get_gold`. O formato devolvido depende **apenas da camada pedida**, e não do que já existe em disco. A leitura e a escrita dos arquivos ficam em `src/etl/file.py` (`DfFile`).

## Machine learning

O módulo `src/ml/train.py` (`TrainDf`) treina um classificador para prever se o exame de uma coleta é aprovado, a partir das informações registradas na coleta.

| Item | Detalhe |
|---|---|
| **Algoritmo** | `RandomForestClassifier` (scikit-learn) |
| **Fonte dos dados** | Camada `cleaned`, com uma linha por coleta |
| **Alvo** | `resultado_exame` (aprovado / reprovado) |
| **Features** | `data`, `tanque`, `temperatura`, `ccs`, `cpp`, `silo` |
| **Saída** | Modelo serializado com `joblib` em `src/ml/models/predict.joblib` |

Parâmetros: 200 árvores, `max_features="sqrt"`, `min_samples_leaf=5`, `class_weight="balanced"` (compensa o desbalanceamento entre aprovados e reprovados) e `random_state=42`.

Para treinar, na raiz do projeto:

```bash
python -m src.ml.train
```

> **Status:** o módulo de treino está em desenvolvimento. Ainda faltam a codificação das variáveis categóricas, a separação em treino e teste com métricas, e a integração com a API.

## Camadas de dados

```text
src/storage/
├── raw/        # arquivos CSV recebidos
├── cleaned/    # dados relacionados, tipados e salvos em Parquet
└── processed/  # indicadores agregados, prontos para consumo
```

| Camada | Arquivo | Conteúdo |
|---|---|---|
| `raw` | `fazendas.csv` | Cadastro de fazendas, produtores, localização e preço por litro |
| `raw` | `coletas.csv` | Eventos de coleta, qualidade, logística, silo e resultado de exame |
| `cleaned` | `fazendas_analise.parquet` | Junção das fontes com tipos normalizados |
| `processed` | `fazenda_analise_resultado.parquet` | Agregação analítica por fazenda e silo |

Os principais indicadores de qualidade são:

- **CCS:** contagem de células somáticas; quanto menor, melhor;
- **CPP:** contagem padrão em placas, associada à carga bacteriana; quanto menor, melhor;
- **temperatura:** temperatura do leite no momento do registro;
- **média de aprovação:** proporção dos exames aprovados no agrupamento, de `0` a `1`.

## Estrutura do repositório

```text
AnaliseLeite/
├── assets/                    # imagens usadas na documentação
├── documents/                 # enunciado e contexto do problema
├── src/
│   ├── application/           # aplicação FastAPI e rotas
│   ├── cache/                 # conexão e operações no Redis
│   ├── controller/            # Dockerfile, Compose e dependências
│   ├── etl/                   # camadas bronze, silver, gold e interface de carga
│   ├── logs/                  # configuração centralizada de logs
│   ├── ml/                    # treino do modelo de classificação
│   └── storage/               # camadas raw, cleaned e processed
├── tests/
│   ├── unit/                  # verificações isoladas
│   ├── integration/           # integração entre camadas do ETL e cache
│   └── functional/            # chamada HTTP à API em execução
└── .github/workflows/         # integração contínua
```

## Como executar

### Opção recomendada: Docker Compose

Pré-requisitos:

- [Docker](https://docs.docker.com/get-docker/);
- Docker Compose v2.

Clone o projeto e entre na pasta:

```bash
git clone https://github.com/BrayanDevZN/AnaliseLeite.git
cd AnaliseLeite
```

Suba a API e o Redis:

```bash
docker compose -f src/controller/compose.yml up --build
```

Quando os serviços estiverem prontos, acesse:

- API: <http://localhost:8000/pipeline/>
- Swagger UI: <http://localhost:8000/docs>
- OpenAPI: <http://localhost:8000/openapi.json>

Para encerrar os serviços:

```bash
docker compose -f src/controller/compose.yml down
```

### Execução local

Use Python 3.14 e mantenha uma instância do Redis disponível em `localhost:6379`.

```bash
python -m venv .venv
source .venv/bin/activate
python -m pip install --upgrade pip
pip install -r src/controller/requirements.txt
```

Em outro terminal, uma forma simples de iniciar apenas o Redis é:

```bash
docker run --rm -p 6379:6379 redis:alpine
```

Depois, na raiz do projeto, inicie a API:

```bash
redis_host=localhost python -m uvicorn src.application.main:app --host 0.0.0.0 --port 8000 --reload
```

## Uso da API

### `GET /pipeline/`

Retorna os dados da camada escolhida em JSON.

| Parâmetro | Valores | Padrão |
|---|---|---|
| `layer` | `raw`, `cleaned`, `processed` | `processed` |

```bash
curl http://localhost:8000/pipeline/
curl "http://localhost:8000/pipeline/?layer=raw"
```

Na camada `raw` a resposta é um objeto com as duas tabelas: `{"fazendas": [...], "coletas": [...]}`.

Exemplo resumido de resposta:

```json
[
  {
    "nome_fazenda": "Sitio Agua Limpa I",
    "produtor": "Ivan Siqueira",
    "cidade": "Pitangui",
    "estado": "MG",
    "preco_litro": 26.3,
    "tanque": "TC03",
    "tipo_tanque": "comunitario",
    "volume_litros": 1617.3,
    "temperatura": 4.55,
    "ccs": 346.7,
    "cpp": 255.6,
    "motorista": "Everaldo Pires",
    "caminhao": "CAM-101",
    "media_aprovacao": 1.0
  }
]
```

## Testes e integração contínua

Os testes deste projeto são scripts executáveis como módulos Python.

```bash
# Unitários
python -m tests.unit.raw
python -m tests.unit.connection

# Integração
python -m tests.integration.silver
python -m tests.integration.gold
python -m tests.integration.cache

# Funcional — requer a API em execução e o pacote requests
pip install requests
python -m tests.functional.api
```

Os testes de conexão e cache exigem Redis em execução. O teste funcional exige os dois serviços do Compose. No GitHub Actions, os fluxos ficam separados em:

- `unit.yaml` — leitura da camada raw e conexão com Redis;
- `integration.yaml` — camadas silver e gold e cache;
- `functional.yaml` — ambiente completo com Docker Compose e chamada à API.

## Configuração

| Variável | Padrão | Descrição |
|---|---|---|
| `redis_host` | `redis` | Host usado pela conexão com o Redis; em execução local, use `localhost` |

Portas padrão:

| Serviço | Porta |
|---|---:|
| API FastAPI | `8000` |
| Redis | `6379` |

## Logs

A aplicação registra eventos no terminal e no arquivo `src/logs/app.log`, incluindo leitura de camadas, transformações, conexão com o Redis e erros do pipeline.

## Próximos passos

- concluir o treino do modelo (encoding, divisão treino/teste, métricas) e expô-lo na API;
- adicionar testes e uma etapa de CI para o módulo de ML;
- adicionar filtros por fazenda, silo, período, motorista e resultado;
- preservar `fazenda_id` e `silo` como campos explícitos na resposta da camada `processed`;
- adotar modelos de resposta do Pydantic e documentação detalhada do contrato;
- transformar os scripts de teste em uma suíte com `pytest` e métricas de cobertura;
- incluir autenticação, paginação e observabilidade para um cenário de produção.

## Documentação do domínio

- [Cenário geral e orientações do trabalho](documents/leiame.md)
- [Problema de rastreamento do leite](documents/rastreamento.md)

---

<p align="center">
  Desenvolvido para transformar a cadeia do leite em uma operação mais rastreável, mensurável e transparente.
</p>
