# HELP commands

## PostgresDB

### Check postgreas last 20 lines of logs

`cd services/weaviate/docker/dev/weaviate_prom_graphana && docker compose logs -n 20 postgresql`

### Restart

`cd services/weaviate/docker/dev/weaviate_prom_graphana && docker compose restart postgresql`

## llama.cpp

- run

  ```bash
  (base) imamcs@Imams-MacBook-Pro llama.cpp % ./main -ngl 32 -m mistral-7b-instruct-v0.1.Q4_0.gguf --color -c 4096 --temp 0.7 --repeat_penalty 1.1 -n -1 -p "{create code python to count circle area}"
  ```

## APP: **go-ai**

### BUILD image prod/test

`docker compose build`

### Push IMAGE to docker

1. `docker tag go-ai-app-prod:latest artifactory.dev.ict/docker-local/go-ai:1.0.3-prod`
2. `docker push artifactory.dev.ict/docker-local/go-ai:1.0.3-prod`

### Run image + enter to it

`docker run -it --entrypoint /bin/bash artifactory.dev.ict/docker-local/go-ai:1.0.3-prod`

### docker compose BUILD+RUN+AutoClean(after stop)

`docker compose -f docker-compose.yml run --build --rm -i -p 8090:8080 app-prod`

### Docker commands

#### BUILD+RUN+RemoveContainer(after exit) specific service(`apptest`) from specific docker-compose.yml(docker-compose.dev.yml)

`docker compose -f docker-compose.dev.yml run --build --rm -i apptest`

```
(base) [dmykolen@ai go-ai]$ docker compose ps
NAME          IMAGE       COMMAND                SERVICE   CREATED       STATUS       PORTS
go-app-cont   go-ai-app   "/go/bin/go-ai -dev"   app       7 hours ago   Up 7 hours   0.0.0.0:8008->8080/tcp, :::8008->8080/tcp
```

```
(base) [dmykolen@ai go-ai]$ docker compose ls
NAME                     STATUS              CONFIG FILES
go-ai                    running(1)          /home/dmykolen/go/src/go-ai/docker-compose.yml
weaviate_prom_graphana   running(2)          /home/dmykolen/go/src/go-ai/services/weaviate/docker/dev/weaviate_prom_graphana/docker-compose.yml
```

```Bash
docker images
docker image ls -a
docker compose up --build --force-recreate
docker run -it --entrypoint /bin/bash go-ai-multi:latest
docker build -f Dockerfile.multi-stage -t go-ai-multi:latest .
docker run -it -p 8008:8080 -t go-ai-multi:latest .

### Enter into working container
docker exec -it 9e56d2f918f1 /bin/bash

### Enter into working container, which was started using docker-compose
docker compose exec app /bin/bash

docker container ls -n 5 -s
docker container ls -n 5
docker container ls -a

docker image history -H --no-trunc go-ai-multi-02

### Delete all NOT working containers
docker container prune
```
