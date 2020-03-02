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

The exporter's Alertmanager API connection can also be configured by defining the following environment variable(s). If they are present, they will take precedence over the corresponding variables in the config file.

Environment Variable | Description
---------------------| -----------
ALERTMANAGER_URL | URL of the exported alertmanager api (eg: "http://localhost:9093/api/v2")


Use -h flag to list available options.

### Configuration & Running
The tool relies on a YAML config file to specify the Alertmanager address it is supposed to send requests to:
```yaml
---
alertmanager_api: "http://localhost:9093/"
```

```bash
./alertmanager-maintenance-scheduler --config.file=/path/to/config.yml
```

## Configuration

An example can be found in
[sample-config.yml](https://github.com/FXinnovation/alertmanager-maintenance-scheduler/blob/master/sample-config.yml).

Configuration element | Description
--------------------- | -----------
alertmanager_url | (Mandatory) URL of the exported alertmanager api (eg: "http://localhost:9093/")

## Testing

### Running unit tests

```bash
make test
```

## Contributing

Refer to [CONTRIBUTING.md](https://github.com/FXinnovation/alertmanager-maintenance-scheduler/blob/master/CONTRIBUTING.md).

## License

Apache License 2.0, see [LICENSE](https://github.com/FXinnovation/alertmanager-maintenance-scheduler/blob/master/LICENSE).
