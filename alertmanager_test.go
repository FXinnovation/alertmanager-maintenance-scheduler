package main

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestAlertmanagerClient_constructURL(t *testing.T) {
	var cases = []struct {
		url   string
		paths []string
		want  string
	}{
		{"http://localhost:9093/api/v2", []string{"silences"}, "http://localhost:9093/api/v2/silences"},
		{"http://localhost:9093/api/v2/", []string{"silences"}, "http://localhost:9093/api/v2/silences"},
		{"http://localhost:9093//api//v2/", []string{"silences"}, "http://localhost:9093/api/v2/silences"},

		{"http://localhost:9093/api/v2", []string{"silence", "1234"}, "http://localhost:9093/api/v2/silence/1234"},
		{"http://localhost:9093/api/v2/", []string{"silence", "1234"}, "http://localhost:9093/api/v2/silence/1234"},
	}

	for _, c := range cases {
		ac := AlertmanagerClient{AlertManagerAPIURL: c.url}

		got, err := ac.constructURL(c.paths...)
		if err != nil {
			t.Errorf("unable to construct Alertmanager URL: %s", err.Error())
		}

		if got != c.want {
			t.Errorf("unexpected Alertmanager URL\ngot: '%s'\nwant: '%s'", got, c.want)
		}
	}
}

func TestAlertmanagerClient_listAlerts_OK(t *testing.T) {
	oneAlert := `
[
  {
    "annotations": {
      "description": "Resource Group: fake-rg-01",
      "summary": "High CPU alert for fake-vm-01"
    },
    "endsAt": "2019-10-28T12:46:25.486-07:00",
    "fingerprint": "1ece099dde7bdc1b",
    "receivers": [
      {
        "name": "web.hook"
      }
    ],
    "startsAt": "2019-10-28T12:40:25.486-07:00",
    "status": {
      "inhibitedBy": [],
      "silencedBy": [],
      "state": "active"
    },
    "updatedAt": "2019-10-28T12:43:25.492-07:00",
    "generatorURL": "http://MacBook-Pro.local:9090/graph?g0.expr=percentage_cpu_percent_average+%3E+0.5\u0026g0.tab=1",
    "labels": {
      "alertname": "HighCPU",
      "instance": "localhost:2112",
      "job": "FakeApp",
      "resource_group": "fake-rg-01",
      "resource_name": "fake-vm-01",
      "severity": "critical"
    }
  }
]
`
	resourcesHandler := func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(oneAlert))
	}
	// create test server with handler
	ts := httptest.NewServer(http.HandlerFunc(resourcesHandler))
	defer ts.Close()
	wanted := []AlertmanagerAlert{
		{
			Annotations: map[string]string{"description": "Resource Group: fake-rg-01", "summary": "High CPU alert for fake-vm-01"},
			EndsAt:      "2019-10-28T12:46:25.486-07:00",
			FingerPrint: "1ece099dde7bdc1b",
			Receivers:   []map[string]string{{"name": "web.hook"}},
			StartsAt:    "2019-10-28T12:40:25.486-07:00",
			Status: struct {
				InhibitedBy []string `json:"inhibitedBy"`
				SilencedBy  []string `json:"silencedBy"`
				State       string   `json:"state"`
			}{InhibitedBy: make([]string, 0), SilencedBy: make([]string, 0), State: "active"},
			UpdatedAt:    "2019-10-28T12:43:25.492-07:00",
			GeneratorURL: "http://MacBook-Pro.local:9090/graph?g0.expr=percentage_cpu_percent_average+%3E+0.5\u0026g0.tab=1",
			Labels: map[string]string{
				"alertname":      "HighCPU",
				"instance":       "localhost:2112",
				"job":            "FakeApp",
				"resource_group": "fake-rg-01",
				"resource_name":  "fake-vm-01",
				"severity":       "critical"},
		},
	}

	ac := NewAlertManagerClient(ts.URL)
	got, err := ac.ListAlerts()
	if err != nil {
		t.Errorf("Got unwanted error %v", err)
	}
	if !reflect.DeepEqual(got, wanted) {
		t.Errorf("Got not what was expected, got %v, want %v", got, wanted)
	}
}

func TestAlertmanagerClient_createSilenceWith(t *testing.T) {
	silenceBody := `{"silenceID":"7d8eb77e-00f9-4e0e-9f20-047695569296"}`
	resourcesHandler := func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(silenceBody))
	}
	// create test server with handler
	ts := httptest.NewServer(http.HandlerFunc(resourcesHandler))
	defer ts.Close()

	ac := NewAlertManagerClient(ts.URL)
	matchers := []map[string]interface{}{
		{"name": "job", "value": "FakeApp", "isRegex": false},
	}
	wanted := "7d8eb77e-00f9-4e0e-9f20-047695569296"
	got, err := ac.CreateSilenceWith("2019-11-01T22:12:33.533330795Z", "2019-11-01T23:11:44.603Z", matchers)
	if err != nil {
		t.Errorf("Got unwanted error %v", err)
	}
	if !reflect.DeepEqual(got, wanted) {
		t.Errorf("Got not what was expected, got %v, want %v", got, wanted)
	}
}

func TestAlertmanagerClient_getSilenceWithID(t *testing.T) {
	oneSilence := `
{
  "id": "7d8eb77e-00f9-4e0e-9f20-047695569296",
  "status": {
    "state": "pending"
  },
  "updatedAt": "2019-10-29T15:16:35.232Z",
  "comment": "Silence",
  "createdBy": "api",
  "endsAt": "2019-11-01T23:11:44.603Z",
  "matchers": [
    {
      "isRegex": false,
      "name": "job",
      "value": "FakeApp"
    }
  ],
  "startsAt": "2019-11-01T22:12:33.533Z"
}
`
	resourcesHandler := func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(oneSilence))
	}
	// create test server with handler
	ts := httptest.NewServer(http.HandlerFunc(resourcesHandler))
	defer ts.Close()
	wanted := AlertmanagerSilence{
		ID: "7d8eb77e-00f9-4e0e-9f20-047695569296",
		Status: struct {
			State string `json:"state"`
		}{"pending"},
		UpdatedAt: "2019-10-29T15:16:35.232Z",
		Comment:   "Silence",
		CreatedBy: "api",
		EndsAt:    "2019-11-01T23:11:44.603Z",
		Matchers: []map[string]interface{}{
			{"name": "job", "value": "FakeApp", "isRegex": false},
		},
		StartsAt: "2019-11-01T22:12:33.533Z",
	}
	ac := NewAlertManagerClient(ts.URL)
	got, err := ac.GetSilenceWithID("7d8eb77e-00f9-4e0e-9f20-047695569296")
	if err != nil {
		t.Errorf("Got unwanted error %v", err)
	}
	if !reflect.DeepEqual(got, wanted) {
		t.Errorf("Got not what was expected, got %v, want %v", got, wanted)
	}
}

func TestAlertmanagerClient_listSilences(t *testing.T) {
	oneSilence := `
[{
  "id": "7d8eb77e-00f9-4e0e-9f20-047695569296",
  "status": {
    "state": "pending"
  },
  "updatedAt": "2019-10-29T15:16:35.232Z",
  "comment": "Silence",
  "createdBy": "api",
  "endsAt": "2019-11-01T23:11:44.603Z",
  "matchers": [
    {
      "isRegex": false,
      "name": "job",
      "value": "FakeApp"
    }
  ],
  "startsAt": "2019-11-01T22:12:33.533Z"
}]
`
	resourcesHandler := func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(oneSilence))
	}
	// create test server with handler
	ts := httptest.NewServer(http.HandlerFunc(resourcesHandler))
	defer ts.Close()
	wanted := AlertmanagerSilenceList{{
		ID: "7d8eb77e-00f9-4e0e-9f20-047695569296",
		Status: struct {
			State string `json:"state"`
		}{"pending"},
		UpdatedAt: "2019-10-29T15:16:35.232Z",
		Comment:   "Silence",
		CreatedBy: "api",
		EndsAt:    "2019-11-01T23:11:44.603Z",
		Matchers: []map[string]interface{}{
			{"name": "job", "value": "FakeApp", "isRegex": false},
		},
		StartsAt: "2019-11-01T22:12:33.533Z",
	}}
	ac := NewAlertManagerClient(ts.URL)
	got, err := ac.ListSilences()
	if err != nil {
		t.Errorf("Got unwanted error %v", err)
	}
	if !reflect.DeepEqual(got, wanted) {
		t.Errorf("Got not what was expected, got %v, want %v", got, wanted)
	}
}

func TestAlertmanagerClient_expireSilenceWithID(t *testing.T) {
	resourcesHandler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}
	// create test server with handler
	ts := httptest.NewServer(http.HandlerFunc(resourcesHandler))
	defer ts.Close()

	ac := NewAlertManagerClient(ts.URL)
	err := ac.ExpireSilenceWithID("7d8eb77e-00f9-4e0e-9f20-047695569296")
	if err != nil {
		t.Errorf("Got unwanted error %v", err)
	}
}

func TestAlertmanagerClient_expireSilenceWithID_AlreadyExpired(t *testing.T) {
	returnString := "silence 7d8eb77e-00f9-4e0e-9f20-047695569296 already expired"
	resourcesHandler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		w.Write([]byte(returnString))
	}
	// create test server with handler
	ts := httptest.NewServer(http.HandlerFunc(resourcesHandler))
	defer ts.Close()

	ac := NewAlertManagerClient(ts.URL)
	err := ac.ExpireSilenceWithID("7d8eb77e-00f9-4e0e-9f20-047695569296")
	if err == nil {
		t.Errorf("Got no error while wanted one")
	}
}

// helps testing promoted fields
type Status struct {
	State string `json:"state"`
}

func TestFilterExpired(t *testing.T) {
	var cases = []struct {
		list AlertmanagerSilenceList
		want AlertmanagerSilenceList
	}{
		{
			AlertmanagerSilenceList{
				AlertmanagerSilence{Status: Status{State: "pending"}},
				AlertmanagerSilence{Status: Status{State: "expired"}},
				AlertmanagerSilence{Status: Status{State: "running"}},
				AlertmanagerSilence{Status: Status{State: "expired"}},
				AlertmanagerSilence{Status: Status{State: "expired"}},
			},
			AlertmanagerSilenceList{
				AlertmanagerSilence{Status: Status{State: "pending"}},
				AlertmanagerSilence{Status: Status{State: "running"}},
			},
		},
	}

	for _, c := range cases {
		got := FilterExpired(c.list)

		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("FilterExpired doesn't filter as expected.\ngot: '%v'\nwant: '%v'\n", got, c.want)
		}
	}
}
