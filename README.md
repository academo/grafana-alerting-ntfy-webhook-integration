# grafana-alerting-ntfy-webhook-integration

# Grafana Ntify Webhook integration

Integration between https://ntfy.sh/ and grafana alerting.

## What's this?

It is a small simple http server that takes a grafana alerting json webhook and re-formats it for ntfy format.
Once https://github.com/grafana/grafana/issues/6956 is implemented this tool will be no longer necessary.

## Why?

You could use grafana alerting webhooks directly with ntfy but he notification will contain the json payload grafana sends which is not too useful in a notification.

## How to use

[Download](https://github.com/academo/grafana-alerting-ntfy-webhook-integration/releases/) the release binary and run it as a server

```bash
grafana-ntfy -ntfy-url "https://ntfy.sh/mytopic"

```

This will create an http (no https) server in port 8080 that will accept POST request, re-format them and send them to the ntfy url. You can use custom ntfy servers if you want.

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

# License

Apache License 2.0

# Disclaimer

This project is not associated with GrafanaLabs
