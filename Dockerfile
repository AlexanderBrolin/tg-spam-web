# stage 1: build React frontend
FROM node:20-alpine AS frontend
WORKDIR /build/frontend
COPY frontend/package*.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build -- --outDir ../app/webapi/frontend/dist

# stage 2: build Go binary
FROM ghcr.io/umputun/baseimage/buildgo:v1.17.0 AS build

ARG GIT_BRANCH
ARG GITHUB_SHA
ARG CI

ADD . /build
COPY --from=frontend /build/app/webapi/frontend/dist /build/app/webapi/frontend/dist
WORKDIR /build

RUN go version

RUN \
 if [ -z "$CI" ] ; then \
   echo "runs outside of CI"; \
   if git rev-parse --git-dir > /dev/null 2>&1; then \
     version=$(git rev-parse --abbrev-ref HEAD)-$(git log -1 --format=%h)-$(date +%Y%m%dT%H:%M:%S); \
   else \
     version=local-$(date +%Y%m%dT%H:%M:%S); \
   fi; \
 else version=${GIT_BRANCH}-${GITHUB_SHA:0:7}-$(date +%Y%m%dT%H:%M:%S); fi && \
 echo "version=$version" && \
 cd app && go build -o /build/tg-spam -ldflags "-X main.revision=${version} -s -w"

# stage 3: runtime
FROM alpine:3.22
ENV TGSPAM_IN_DOCKER=1
RUN apk add --no-cache tzdata
COPY --from=build /build/tg-spam /srv/tg-spam

COPY data /srv/preset
COPY data/.not_mounted /srv/data/.not_mounted
COPY entrypoint.sh /srv/entrypoint.sh

RUN \
 adduser -s /bin/sh -D -u 1000 app && chown -R app:app /home/app && \
 chown -R app:app /srv/preset /srv/data && \
 chmod -R 777 /srv/preset && \
 chmod -R 775 /srv/data && \
 chmod +x /srv/entrypoint.sh && \
 ls -la /srv/preset

USER app
WORKDIR /srv

RUN \
 for f in /srv/preset/*.txt.loaded; do [ -f "$f" ] && mv -vf "$f" "${f%.loaded}"; done && \
 /srv/tg-spam --convert=only --files.dynamic=/srv/preset --files.samples=/srv/preset && \
 sh -c 'for f in /srv/preset/*.txt.loaded; do mv -vf "$f" "${f%.loaded}"; done' && \
 echo "preset files converted" && \
 ls -la /srv/preset && \
 ls -la /srv/data

EXPOSE 8080
ENTRYPOINT ["/srv/entrypoint.sh"]
