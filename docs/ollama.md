# OLLAMA INFO

Ollama default port is **11434**.

## ENV vars

> Export before running OLLAMA or add to `~/.bashrc` or `/etc/systemd/system/ollama.service` if running as a service

### OLLAMA vars

* `OLLAMA_HOST=0.0.0.0:11444`
* `OLLAMA_DEBUG=1`

## How to run OLLAMA

* OLLAMA_HOST=0.0.0.0:11444 /usr/local/bin/ollama serve
* `systemctl restart ollama` - run as service

## Other commands

* `systemctl cat ollama.service`
* `journalctl -u ollama -r -f`

## Docker

1. `docker run -d --gpus=all -v ollama:/root/.ollama -p 11434:11434 --name ollama ollama/ollama`
2. `docker exec -it ollama /bin/bash`

### Docker help

* `docker ps -a`
* `docker logs ollama`
* `docker stop ollama`
* `docker start ollama`
* `docker rm ollama`

## API OLLAMA

### Chat completions

```bash
curl http://192.168.31.69:5030/v1/chat/completions -H "Content-Type: application/json" -d '{"model":"llama2","messages":[{"role":"system","content":"You are a helpful assistant."},{"role":"user","content":"Hello!"}]}'
```

### Download new model

```bash
curl http://localhost:11444/api/pull -d '{"name": "llama2"}'
```

### List models

```bash
curl http://192.168.31.69:11434/api/tags
```
