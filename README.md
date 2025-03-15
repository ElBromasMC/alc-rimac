# go-webserver-template

All Last Computing website (serves as a template)

## Development environment

### Prerequisites

* Docker

### .env file example

```shell
USER_UID="1000" # It must match your current user UID
ENV="development"
PORT="8080"
REL="1"
```

### Live reload

```shell
$ bin/live.sh
```

## Production environment

### Prerequisites

* [Traefik](https://doc.traefik.io/traefik/getting-started/quick-start/)

### Docker compose .env file example

```shell
WEBSERVER_HOSTNAME=domain.tld
ENV=production
PORT=8080
REL=1
```
