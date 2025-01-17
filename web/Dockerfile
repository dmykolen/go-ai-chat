FROM artifactory.dev.ict/docker-virtual/node:20

ARG LOCAL_CERT_DIR=/usr/local/share/ca-certificates/

ADD --chmod=644 http://pki1.lifecell.com.ua/PEM/AllInOne.pem ${LOCAL_CERT_DIR}

RUN npm config set registry http://registry.npmjs.org/ \
    && npm config set cafile ${LOCAL_CERT_DIR}AllInOne.pem \
    && npm config set proxy ${HTTP_PROXY} \
    && npm config set https-proxy ${HTTPS_PROXY}

RUN npm install webpack -g

WORKDIR /usr/src/app

COPY package.json .
RUN --mount=type=cache,target=/root/.npm npm install

COPY . .

ENV NODE_ENV=production \
    NPM_CONFIG_LOGLEVEL=verbose
# RUN npm run build
# CMD [ "/bin/bash" ]
CMD ["npm", "run", "build"]