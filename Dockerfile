####G#########################################################################

FROM artifactory.dev.ict/docker-virtual/golang:1.22.3 AS builder

ARG APP_NAME=go-ai
ARG GOPROXY=https://artifactory.dev.ict/artifactory/api/go/go-virtual
ARG SSL_CERT_DIR=/etc/ssl/certs
ARG LOCAL_CERT_DIR=/usr/local/share/ca-certificates
ARG PROXY=http://ict-proxy.vas.sn:3128
ARG http_proxy=$PROXY
ARG https_proxy=$PROXY
ARG GITLAB_USER
ARG GITLAB_ACCESS_TOKEN

ENV GOINSECURE='gitlab.dev.ict' \
    GONOPROXY='gitlab.dev.ict' \
    GONOSUMDB='gitlab.dev.ict' \
    GOPRIVATE='gitlab.dev.ict' \
    GOPROXY=${GOPROXY} \
    GITLAB_USER=${GITLAB_USER} \
    GITLAB_ACCESS_TOKEN=${GITLAB_ACCESS_TOKEN} \
    LD_LIBRARY_PATH=/opt/oracle/instantclient_21_3 \
    SSL_CERT_DIR=/etc/ssl/certs

RUN cd ${LOCAL_CERT_DIR} && \
    for i in RootCA.pem IssuerCA01.pem IssuerCA02.pem ; do \
        wget http://pki1.lifecell.com.ua/PEM/$i && cat $i >> ${LOCAL_CERT_DIR}/AllInOne.crt; done \
    && update-ca-certificates

RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates unzip curl libaio1 tree \
    && apt-get clean && apt-get autoremove \
    && rm -rf /var/lib/apt/lists/* \
    && mkdir -p /opt/oracle && cd /opt/oracle \
    && for i in instantclient-basic-linux.x64-21.3.0.0.0.zip instantclient-sqlplus-linux.x64-21.3.0.0.0.zip instantclient-sdk-linux.x64-21.3.0.0.0.zip ; do \
       curl -k -L -O https://download.oracle.com/otn_software/linux/instantclient/213000/$i && unzip $i && rm $i; done \
    && ln -s /opt/oracle/instantclient_21_3 /opt/oracle/instantclient

RUN git config --global http.sslVerify false \
    && git config --global http.sslCAInfo ${LOCAL_CERT_DIR}/AllInOne.crt \
    && git config --global http.proxy ${http_proxy}

RUN echo "machine gitlab.dev.ict login ${GITLAB_USER} password ${GITLAB_ACCESS_TOKEN}" > ~/.netrc \
    && chmod 600 ~/.netrc

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o /ai .

##################################################################################

FROM artifactory.dev.ict/docker-virtual/node:20 AS febuilder

COPY ./web/package.json ./web/package-lock.json ./web/.npmrc /usr/src/app/web/

WORKDIR /usr/src/app/web

RUN npm install webpack

RUN npm install

COPY ./web /usr/src/app/web/

ENV NODE_ENV=production \
    NPM_CONFIG_LOGLEVEL=verbose

RUN npm run build && rm -rf /app/web/node_modules

###################################################################################

FROM artifactory.dev.ict/docker-virtual/debian:bookworm-slim

ARG LOCAL_CERT_DIR=/usr/local/share/ca-certificates
ARG PROXY=http://ict-proxy.vas.sn:3128
ARG http_proxy=$PROXY
ARG https_proxy=$PROXY

ENV SSL_CERT_DIR=/etc/ssl/certs \
    LD_LIBRARY_PATH=/opt/oracle/instantclient \
    http_proxy=$PROXY https_proxy=$PROXY no_proxy=.dev.ict,.vas.sn,.lifecell.ua,lifecell.ua,.lifecell.com.ua,lifecell.com.ua,.omnicell.ua,10.0.0.0/8,192.168.0.0/16

COPY --from=builder ${LOCAL_CERT_DIR} ${LOCAL_CERT_DIR}
COPY --from=builder ${LD_LIBRARY_PATH} ${LD_LIBRARY_PATH}

RUN apt-get update && apt-get install -y --no-install-recommends libaio1 ca-certificates && apt-get autoremove && apt-get clean && rm -rf /var/lib/apt/lists/* \
    && update-ca-certificates 

WORKDIR /app

COPY --from=builder /ai /go/bin/go-ai
COPY --from=febuilder /usr/src/app/web /app/web/
COPY --from=builder /app/assets/voip_ritm_docs /app/assets/voip_ritm_docs/

ENTRYPOINT ["/go/bin/go-ai", "-l_oc", "-l_color", "-l_of", "-l_file", "logs/go-ai.log"]
