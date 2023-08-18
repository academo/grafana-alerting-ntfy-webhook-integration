# Grafana Ntify Webhook integration

Integration between https://ntfy.sh/ and grafana alerting.

## What's this?

It is a small simple http server that takes a grafana alerting json webhook and re-formats it for ntfy format.
Once https://github.com/grafana/grafana/issues/6956 is implemented this tool will be no longer necessary.

## Why?

You could use grafana alerting webhooks directly with ntfy but the notification will contain the json payload grafana sends which is not too useful in a notification.

## How to use

### Binary

[Download](https://github.com/academo/grafana-alerting-ntfy-webhook-integration/releases/) the release binary and run it as a server

```bash
grafana-ntfy -ntfy-url "https://ntfy.sh/mytopic"

```

This will create an http (no https) server in port 8080 that will accept POST request, re-format them and send them to the ntfy url. You can use custom ntfy servers if you want.

### Docker

A docker image is provided for convenience. You can use a docker-compose file like this

```yaml
# Example of a docker-compose service
version: "3.7"
services:
  grafana-ntfy:
    image: academo/grafana-ntfy:latest
    hostname: grafana-ntfy
    command:
      - "-ntfy-url=https://ntfy.sh/mytopic"
```

#### Example of a more complete docker-compose approach

```yaml
# Example of a docker-compose service
version: "3.7"

# Creates an inner network between containers
networks:
  internal:

services:
  grafana:
    image: grafana/grafana
    depends_on:
      - grafana-ntfy
    ports:
      - 3000:3000
    networks:
      - internal
  grafana-ntfy:
    image: academo/grafana-ntfy:latest
    hostname: grafana-ntfy
    expose:
      - 8080 #only accesible to other containers
    networks:
      - internal
    command:
      - "-ntfy-url=https://ntfy.sh/mytopic"
```

### Example using Docker and a non amd64 architecture

The provided docket image is amd64. Should you wish to use a different architecture, you can [download the binary](https://github.com/academo/grafana-alerting-ntfy-webhook-integration/releases) and use the following docker-compose file as example

```
version: "3"
services:
  grafana-ntfy:
    image: arm64v8/alpine
    # make sure to put the correct architecture binary
    command: sh -c "/app/grafana-ntfy -ntfy-url 'https://ntfy.sh/mytopic'"
    # point the volume to where your downloaded your binary
    volumes:
      - ./:/app
    working_dir: /app
    expose:
      - 8080 #only accesible to other containers
```

Then in your grafana alerting webhook contact point you would configure the url as `http://grafana-ntfy:8080` (notice is the hostname value)

## Options

See grafana-ntfy -h for all options.

```
Usage of grafana-ntfy:
  -allow-insecure
        Allow insecure connections to ntfy-url
  -ntfy-url string
        The ntfy url including the topic. e.g.: https://ntfy.sh/mytopic
  -port int
        The port to listen on (default 8080)
```

# No https?

This webhook is suppose to run next to your grafana instance and only accepts local request. You should not expose this server to the internet.

# Other projects

https://github.com/kittyandrew/grafana-to-ntfy

Much similar to this project but written in Rust and not compatible with the latest Grafana. I would have pushed updates to said project if I were proficient enough in Rust.

# License

Apache License 2.0

# Disclaimer

This project is not associated with GrafanaLabs
