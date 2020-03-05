# alertmanager-maintenance-scheduler

A maintenance scheduler UI for Prometheus AlertManager

The project in its current status is meant to support a specific feature around the Alertmanager API, and that is to be able to schedule a finite number of repeating silences. The tool is intended to be used via its Web based UI.

## Getting Started

### Prerequisites

To run this project, you will need a [working Go environment](https://golang.org/doc/install).

### Installing

```bash
go get -u github.com/FXinnovation/alertmanager-maintenance-scheduler
```

### Building

To build simply run:

```bash
make build
```

## Run the binary

The tool expects a config file as one of its arguments:

```bash
./alertmanager-maintenance-scheduler --config.file=/path/to/config.yml
```

The Alertmanager API connection can also be configured by defining the following environment variable(s). If they are present, they will take precedence over the corresponding variables in the config file.

Environment Variable | Description
---------------------| -----------
ALERTMANAGER_URL | URL of Alertmanager (eg: "http://localhost:9093/")

Use -h flag to list available options.

## Configuration

An example can be found in
[sample-config.yml](https://github.com/FXinnovation/alertmanager-maintenance-scheduler/blob/master/sample-config.yml).

Configuration element | Description
--------------------- | -----------
alertmanager_url | (Mandatory) URL of Alertmanager (eg: "http://localhost:9093/")

## Docker image

You can run images published in [dockerhub](https://hub.docker.com/r/fxinnovation/alertmanager-maintenance-scheduler).

You can also build a docker image using:

```bash
make docker
```

The resulting image is named `fxinnovation/alertmanager-maintenance-scheduler:<git-branch>`.

The image exposes port 8080 and expects a config in `/opt/alertmanager-maintenance-scheduler/config.yml`.
To configure it, you can pass the environment variables, and bind-mount a config from your host:

```bash
docker run -p 8080:8080 -v /path/on/host/config/config.yml:/opt/alertmanager-maintenance-scheduler/config/config.yml -e ALERTMANAGER_URL="http://localhost:9093/" fxinnovation/alertmanager-maintenance-scheduler:<git-branch>
```

## Testing

### Running unit tests

```bash
make test
```

## Contributing

Refer to [CONTRIBUTING.md](https://github.com/FXinnovation/alertmanager-maintenance-scheduler/blob/master/CONTRIBUTING.md).

## License

Apache License 2.0, see [LICENSE](https://github.com/FXinnovation/alertmanager-maintenance-scheduler/blob/master/LICENSE).
