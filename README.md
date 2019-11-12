# alertmanager-maintenance-scheduler
A maintenance scheduler UI for Prometheus AlertManager

The project in its current status is meant to support a specific feature around the Alertmanager API, and that is to be able to schedule a finite number of repeating silences. The tool is meant to be used via its Web based UI, however certain endpoints can be leveraged to help automate scheduling & expiring silences. 