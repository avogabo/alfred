FROM node:22-bookworm AS webbuild
WORKDIR /src/web
COPY web/package.json web/package-lock.json* ./
RUN npm install
COPY web/ ./
RUN npm run build

FROM golang:1.24 AS build
WORKDIR /src
COPY . .
COPY --from=webbuild /src/web/dist /src/web/dist
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/alfred .

FROM eclipse-temurin:21-jre-jammy
WORKDIR /app
RUN apt-get update \
 && apt-get install -y curl ca-certificates unzip xz-utils \
 && rm -rf /var/lib/apt/lists/* \
 && curl -fsSL --retry 5 --retry-delay 2 --max-time 300 https://get.filebot.net/filebot/FileBot_5.1.6/FileBot_5.1.6-portable.tar.xz -o /tmp/filebot.tar.xz \
 && mkdir -p /opt/filebot /tmp/filebot-extract \
 && tar -xJf /tmp/filebot.tar.xz -C /tmp/filebot-extract \
 && cp -a /tmp/filebot-extract/. /opt/filebot/ \
 && ln -sf /opt/filebot/filebot.sh /usr/local/bin/filebot \
 && chmod +x /opt/filebot/filebot.sh /usr/local/bin/filebot \
 && rm -rf /tmp/filebot.tar.xz /tmp/filebot-extract

ENV FILEBOT_HOME=/config/filebot
COPY --from=build /out/alfred /app/alfred
COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

EXPOSE 8091
ENTRYPOINT ["/entrypoint.sh"]
