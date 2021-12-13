FROM docker.io/alpine:3.14 AS build

RUN apk add go pacman runuser
RUN adduser -D builder

WORKDIR /build
ADD go.mod go.sum *.go /build
RUN go build .

FROM docker.io/alpine:3.14
RUN apk add --no-cache pacman
COPY --from=build /build/repo-add-http /
VOLUME ["/data"]
EXPOSE 8545/tcp
ENV REPO=repo
CMD exec /repo-add-http --listen :8545 --db /data/$REPO.db.tar.zst
