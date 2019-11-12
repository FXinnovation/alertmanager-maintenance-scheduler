# alertmanager-maintenance-scheduler
A maintenance scheduler UI for Prometheus AlertManager

The project in its current status is meant to support a specific feature around the Alertmanager API, and that is to be able to schedule a finite number of repeating silences. 
The tool is intended to be used via a Web based UI, however development of this UI is in progress. Certain endpoints can still be leveraged to help automate scheduling & expiring silences. 