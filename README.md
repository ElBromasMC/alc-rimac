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
DB_NAME=alc-rimac
DB_PASSWORD=qwerty$321
SESSION_KEY=mysecretkey
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
ENV="production"
PORT="8080"
REL="1"
DB_NAME=alc-rimac
DB_PASSWORD=qwerty$321
SESSION_KEY=mysecretkey
WEBSERVER_HOSTNAME=www.domain.tld
```

