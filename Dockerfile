FROM golang:1.12 as builder
WORKDIR /alertmanager-maintenance-scheduler/
COPY . .
RUN make test build

FROM ubuntu:18.04
COPY --from=builder /alertmanager-maintenance-scheduler/alertmanager-maintenance-scheduler /alertmanager-maintenance-scheduler
ADD ./resources /resources
RUN /resources/build && rm -rf /resources
USER ams
EXPOSE 8080
WORKDIR /opt/alertmanager-maintenance-scheduler
ENTRYPOINT  [ "/opt/alertmanager-maintenance-scheduler/alertmanager-maintenance-scheduler" ]

LABEL maintainer="FXinnovation CloudToolDevelopment <CloudToolDevelopment@fxinnovation.com>" \
      "org.label-schema.name"="alertmanager-maintenance-scheduler" \
      "org.label-schema.base-image.name"="docker.io/library/ubuntu" \
      "org.label-schema.base-image.version"="18.04" \
      "org.label-schema.description"="alertmanager-maintenance-scheduler in a container" \
      "org.label-schema.url"="https://github.com/FXinnovation/alertmanager-maintenance-scheduler" \
      "org.label-schema.vcs-url"="https://github.com/FXinnovation/alertmanager-maintenance-scheduler" \
      "org.label-schema.vendor"="FXinnovation" \
      "org.label-schema.schema-version"="1.0.0-rc.1" \
      "org.label-schema.usage"="Please see README.md"