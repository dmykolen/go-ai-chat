#!/bin/sh

GREEN_BOLD="\e[1;32m"
YELLOW_BOLD="\e[1;33m"
RESET="\e[0m"


echo -e "\e[1;32m[HINT]\e[0m"


api_key=${OPENAI_API_KEY:-$OPENAI_APIKEY}
echo " * CURL_CA_BUNDLE(CAfile) => $CURL_CA_BUNDLE"
echo " * SSL_CERT_FILE(CAfile) => $SSL_CERT_FILE"
echo " * CURL_CA_PATH(CApath) => $CURL_CA_PATH"
echo " * SSL_CERT_DIR(CApath) => $SSL_CERT_DIR"
echo "---"

if [ -z "$api_key" ]; then echo " -> [ERROR] openai api key - not exists! [ERROR]"; else echo " -> [OK] openai api key - exists"; fi
[ -f "/etc/curlrc" ] && echo "/etc/curlrc - exists" || echo " -> [OK] /etc/curlrc - does not exist"
[ -f "/etc/ssl/openssl.cnf" ] && echo "/etc/ssl/openssl.cnf - exists" || echo " -> [OK] /etc/ssl/openssl.cnf - does not exist"
[ -f "$HOME/.curlrc" ] && echo "$HOME/.curlrc - exists" || echo " -> [OK] $HOME/.curlrc - does not exist"
echo "---"
echo -e "\n${GREEN_BOLD}[HINT]${RESET} if you have problem with certificates check all necessary certificates already exist in /etc/ssl/certs/ and set env var ${YELLOW_BOLD}SSL_CERT_DIR=/etc/ssl/certs${RESET}\n"

echo "---"
echo " -> Call curl openai API..."
echo "---"
curl https://api.openai.com/v1/chat/completions -H "Content-Type: application/json" -H "Authorization: Bearer $api_key" \
-d '{"model":"gpt-4o","messages":[{"role":"system","content":[{"type":"text","text":"Act as expert of Weaviate"}]},{"role":"user","content":[{"type":"text","text":"I could not connect to open api from weaviate docker container"}]}],"temperature":1,"max_tokens":2048,"top_p":1,"frequency_penalty":0,"presence_penalty":0,"response_format":{"type":"text"}}' \
-vvv -x http://ict-proxy.vas.sn:3128 || exit 1

