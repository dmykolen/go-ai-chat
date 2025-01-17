FROM artifactory.dev.ict/docker-virtual/semitechnologies/weaviate:1.24.4

ENV HTTP_PROXY=http://proxy.astelit.ukr:3128
ENV HTTPS_PROXY=http://proxy.astelit.ukr:3128

ARG LOCAL_CERT_DIR=/usr/local/share/ca-certificates/

ADD --chmod=644 http://pki1.lifecell.com.ua/PEM/AllInOne.pem ${LOCAL_CERT_DIR}

RUN update-ca-certificates