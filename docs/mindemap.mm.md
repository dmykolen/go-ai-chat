---
markmap:
  initialExpandLevel: 5
  colorFreezeLevel: 4
  embedAssets: true
---

# OTHER SOFTWARE

## Docker

- ### VectorDB - [Weaviate](https://weaviate.io/)
- ### PostgresDB (store: users, chats)
- ### Prometheus
- ### Grafana
- ### Swagger

## Local LLM (for async summerization)

# App

Description
: Lifecell AI RAG System

## FE Technologies

- SSE for AI stream chat communication
- Javascript + JQuery + HTML + CSS
- TailwindCSS
- DaisyUI
- HTMX

## Server
### Middleware

- logger
- requestId generator
- errors recover

### Routes

- GET    /
- GET    /metrics
- GET    /swagger/*
- GET    /web/files
- GET    /ai
- GET    /voip
- POST   /login
- GET    /logout
- GET    /login_form
- POST   /chatgpt
- GET    /sse2
- POST   /api/v1/ask-ai-voip
- POST   /api/v1/stt
- GET    /api/v1/users/:username
- GET    /api/v1/users/chats/:username
- GET    /api/v1/users/:username/chats/:type
- GET    /fe/users/:username/chats/:type
- GET    /api/vdb/v1/objects
- POST   /api/vdb/v1/objects
- DELETE /api/vdb/v1/objects/:id
- GET    /api/vdb/v1/suggest
- POST   /search
- GET    /api/vdb/v1/objects/:id
- PUT    /api/vdb/v1/objects/:id
- GET    /hw
- GET    /vectordb-admin
- GET    /wdocs
- POST   /upload
- DELETE /object/:id

## Services

- ### API

  - #### External
    - OpenAI
  - #### Internal
    - CIM-WS (rest)
    - OM-WS  (rest)

- ### RAG
  - search in VectorDB
  - insert to VectorDB
  - embed vectors creation
- ### Processors
  (process documentation and convert to vectors)
  - Docx
  - Confluence
  - WEB pages
- ### Vectors
  - convert to vectors
  - search in VectorDB
  - insert to VDB
  - update
  - delete
  - get all
  - get by id
  - get by name
- ### Postgres
  - `insert/update/get` **users**
  - `insert/update/get` **chats**

