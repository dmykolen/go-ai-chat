# App in Docker

## Start

```bash
(base) [dmykolen@ai go-ai]$ go build .
(base) [dmykolen@ai go-ai]$ docker compose up --build --force-recreate
```

## Push to docker

```bash
docker tag go-ai:latest artifactory.dev.ict/docker-local/go-ai:1.0.0-prod
docker push artifactory.dev.ict/docker-local/go-ai:1.0.0-prod
```

## Other

```Bash
docker compose -f docker-compose.dev.yml run --build --rm -i apptest
docker build -f Dockerfile.multi-stage -t go-ai-multi:latest .

docker run -it --entrypoint /bin/bash go-ai-multi-02:latest
docker run -it --entrypoint /bin/bash go-ai-multi:latest
docker run -it --entrypoint ./go-ai -h go-ai-multi:latest


docker run -p 8080:8080 -ti <image_name> /bin/sh
docker run -it -p 8080:8080 -t <container_name> .

docker run -it -p 8008:8080 -t go-ai-multi:latest .

### Enter into working container
docker exec -it 9e56d2f918f1 /bin/bash

docker container ls -n 5 -s
docker container ls -n 5
docker container ls -a

docker image history -H --no-trunc go-ai-multi-02

### Delete all NOT working containers
docker container prune

```
