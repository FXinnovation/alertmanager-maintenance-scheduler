#!/bin/bash
alertmanager="localhost:9093/api/v2/silences"
api="localhost:8080/api/v1/silence"

function send_request(){
    echo 'creating silence'
    curl -vv -XPOST -H "Accept: application/json" -H "Content-Type: application/json" -d "$1" $api
}

function create_body_alertmanager(){
    template='
{
  "id": "", 
  "createdBy": "scheduler",
  "comment": "scheduled maintenance",
  "startsAt": "%s",
  "endsAt": "%s",
  "matchers": [
    {
      "name": "%s",
      "value": "%s",
      "isRegex": false
    }
  ]
}'
    body=$(printf "$template" "$1" "$2" "$3" "$4")
    send_request "$body"
}

function create_body_api(){
    template='
{
  "id": "", 
  "createdBy": "scheduler",
  "comment": "scheduled maintenance",
  "matchers": [
    {
      "name": "%s",
      "value": "%s",
      "isRegex": false
    }
  ],
  "schedule": {
    "start_time": "%s",
    "end_time": "%s",
    "repeat": {
      "enabled": true,
      "interval": "h",
      "count": 5
    }
  }
}'
    body=$(printf "$template" "$1" "$2" "$3" "$4")
    send_request "$body"
}


# create_body_alertmanager '2020-10-12T12:34:02.566Z' '2020-10-12T12:34:03.566Z' 'foo' 'bar'
create_body_api 'foo' 'bar' '2021-10-12T12:34:02.566Z' '2021-10-12T12:34:03.566Z'
