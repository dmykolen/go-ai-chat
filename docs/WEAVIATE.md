# Weaviate notes

## Weaviate installation

### Docker

```bash
docker run -it -p 8080:8080 semitechnologies/weaviate:latest
```

## Prometheus

```bash
/bin/prometheus --config.file=/etc/prometheus/prometheus.yml --storage.tsdb.path=/prometheus --web.console.libraries=/usr/share/prometheus/console_libraries --web.console.templates=/usr/share/prometheus/consoles
```

## Weaviate configuration

### Available models (OpenAI) ([doc](https://weaviate.io/developers/weaviate/modules/retriever-vectorizer-modules/text2vec-openai#class-configuration))

* text-embedding-3

    Available dimensions:
  * text-embedding-3-large: 256, 1024, 3072 (default)
  * text-embedding-3-small: 512, 1536 (default)
* ada

### Schema

```bash
curl -X POST "http://localhost:8080/v1/schema" -H "accept: application/json" -H "Content-Type: application/json" -d "{ \"classes\": [ { \"class\": \"Person\", \"description\": \"A person\", \"properties\": [ { \"dataType\": [ \"string\" ], \"description\": \"The name of the person\", \"name\": \"name\" }, { \"dataType\": [ \"int\" ], \"description\": \"The age of the person\", \"name\": \"age\" } ] } ]}"
```

### Data

```bash
curl -X POST "http://localhost:8080/v1/batch" -H "accept: application/json" -H "Content-Type: application/json" -d "{ \"objects\": [ { \"class\": \"Person\", \"properties\": { \"name\": [ \"John Doe\" ], \"age\": [ 42 ] } }, { \"class\": \"Person\", \"properties\": { \"name\": [ \"Jane Doe\" ], \"age\": [ 42 ] } } ]}"
```

### Query

```bash
curl -X POST "http://localhost:8080/v1/graphql" -H "accept: application/json" -H "Content-Type: application/json" -d "{ \"query\": \"{ Person { name age } }\"}"
```

## Weaviate notes

* Weaviate is a knowledge graph
* Weaviate is a GraphQL API
* Weaviate is a vector search engine
* Weaviate is a schema-first database
