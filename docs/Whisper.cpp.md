# Whisper local run

## Add as git **submodule**

```bash
cd ~/go/src/go-ai

# Add the GitHub Repository as a Submodule
git submodule add https://github.com/ggerganov/whisper.cpp.git third_party/whisper.cpp

# Initialize the Submodule: If this is the first time you're adding a submodule to your project, you need to initialize it.
git submodule init

# Update the Submodule: This step is necessary to pull all the data from the submodule repository.
git submodule update

# Commit changes
git add . && git commit -m "Added submodule" && git push
```

## Compile `server`

```bash
cd ./whisper.cpp
make server
```

## SYSTEM REQUIREMENTS

- recommendedMaxWorkingSetSize  = 11453.25 MB

## Convert your AUDIO to WAV 16kHz

```bash
ffmpeg -i test.mp3 -ar 16000 -ac 1 -c:a pcm_s16le test.wav
```

## Examples to run whisper server

- `cd ./whisper.cpp && ./server -m models/ggml-large.bin -l uk -debug --port 5552`
- `cd ./whisper.cpp && ./server -m models/ggml-medium.bin -l uk -debug --port 5552`
- `cd ./whisper.cpp && ./server -m ~/git/models/ggml-large-v3-q5_0.bin -l uk --port 5552 -t 8 -p 2 -debug -pp -pr -pc --convert`
- `cd ./whisper.cpp && ./server -m ~/git/models/ggml-large-v3-q5_0.bin -l uk --port 5552 -t 8 -p 2 -debug -pp -pr -pc -ps --convert`

## Example call
### with response format `json`

```bash
curl 127.0.0.1:5552/inference \
-H "Content-Type: multipart/form-data" \
-F file="@example/test.wav" \
-F temperature="0.2" \
-F response-format="json"
 ```

Response:
> `{"text":"Алло, це тестовий запис. Слава Україні, героям слава, слава нації і піздець РФ."}`

### `ggml-medium.bin`

```bash
curl 127.0.0.1:5552/inference \
-H "Content-Type: multipart/form-data" \
-F file="@example/test.wav" \
-F temperature="0.2" \
-F response-format="text"
 ```

### Another example

```bash
#!/bin/bash

curl 127.0.0.1:10555/inference -H "Content-Type: multipart/form-data" -F file="@/home/dmykolen/audio/globalbilgi_pcm16_10s.wav" -F temperature="0.2" -F response-format="json"
curl -v 127.0.0.1:10555/inference -H "Content-Type: multipart/form-data" -F file=@/home/dmykolen/audio/globalbilgi_pcm16_10s.wav -F response-format="json"
```

Response:
> Алло, це тестовий запис. Слава Україні, героям слава, слава нації, пиздець Російській Федерації.

## Build from source

1. `git clone https://github.com/ggerganov/whisper.cpp`
2. `cd whisper.cpp`
3. `make server`

## More info

- <https://github.com/ggerganov/whisper.cpp/blob/master/examples/server/README.md>
- <https://github.com/ggerganov/whisper.cpp/>
