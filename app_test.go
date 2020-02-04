package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"testing"

	"github.com/go-openapi/strfmt"
	"github.com/gorilla/mux"
	"github.com/prometheus/alertmanager/api/v2/models"
	"github.com/stretchr/testify/mock"
)

type MockAlertManagerClient struct {
	mock.Mock
}

func (m *MockAlertManagerClient) ListAlerts() (models.GettableAlerts, error) {
	args := m.Called()
	return args.Get(0).(models.GettableAlerts), args.Error(1)
}

func (m *MockAlertManagerClient) CreateSilenceWith(start, end string, request APISilenceRequest) (string, error) {
	args := m.Called(start, end, request)
	return args.Get(0).(string), args.Error(1)
}
func (m *MockAlertManagerClient) UpdateSilenceWith(uuid, start, end string, request APISilenceRequest) (string, error) {
	args := m.Called(start, end, request)
	return args.Get(0).(string), args.Error(1)
}
func (m *MockAlertManagerClient) GetSilenceWithID(uuid string) (models.GettableSilence, error) {
	args := m.Called(uuid)
	return args.Get(0).(models.GettableSilence), args.Error(1)
}
func (m *MockAlertManagerClient) ListSilences() (models.GettableSilences, error) {
	args := m.Called()
	return args.Get(0).(models.GettableSilences), args.Error(1)
}
func (m *MockAlertManagerClient) ExpireSilenceWithID(uuid string) error {
	args := m.Called(uuid)
	return args.Error(0)
}

func Test_getAlerts(t *testing.T) {
	client := MockAlertManagerClient{}

	fingerprint := "1ece099dde7bdc1b"
	name := "web.hook"
	receivers := []*models.Receiver{&models.Receiver{Name: &name}}
	startsAt, _ := strfmt.ParseDateTime("2019-10-28T12:40:25.486-07:00")
	endsAt, _ := strfmt.ParseDateTime("2019-10-28T12:46:25.486-07:00")
	updatedAt, _ := strfmt.ParseDateTime("2019-10-28T12:43:25.492-07:00")
	state := "active"

	want := models.GettableAlerts{
		{
			Annotations: map[string]string{"description": "Resource Group: fake-rg-01", "summary": "High CPU alert for fake-vm-01"},
			EndsAt:      &endsAt,
			Fingerprint: &fingerprint,
			Receivers:   receivers,
			StartsAt:    &startsAt,
			Status: &models.AlertStatus{
				InhibitedBy: make([]string, 0),
				SilencedBy:  make([]string, 0),
				State:       &state,
			},
			UpdatedAt: &updatedAt,
			Alert: models.Alert{
				GeneratorURL: "http://MacBook-Pro.local:9090/graph?g0.expr=percentage_cpu_percent_average+%3E+0.5\u0026g0.tab=1",
				Labels: map[string]string{
					"alertname":      "HighCPU",
					"instance":       "localhost:2112",
					"job":            "FakeApp",
					"resource_group": "fake-rg-01",
					"resource_name":  "fake-vm-01",
					"severity":       "critical",
				},
			},
		},
	}
	client.On("ListAlerts").Return(want, nil)
	app := App{
		config: &Config{},
		client: &client,
	}
	// Create a request to pass to the handler
	req := httptest.NewRequest("GET", "/webhook", nil)

	// Create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(app.getAlerts)

	// Test the handler with the request and record the result
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("wrong status code: got '%d', want '%d'", status, http.StatusInternalServerError)
	}
	expected := `
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
	if ok, err := AreEqualJSON(rr.Body.String(), expected); !ok || err != nil {
		t.Errorf("handler returned unexpected body\ngot: '%s'\nwant: '%s'\nerror: %s\n",
			rr.Body.String(), expected, err.Error())
	}
}

func AreEqualJSON(s1, s2 string) (bool, error) {
	var o1 interface{}
	var o2 interface{}

	var err error
	err = json.Unmarshal([]byte(s1), &o1)
	if err != nil {
		return false, fmt.Errorf("error mashalling string 1: '%s'", err.Error())
	}
	err = json.Unmarshal([]byte(s2), &o2)
	if err != nil {
		return false, fmt.Errorf("error mashalling string 2: '%s'", err.Error())
	}
	return reflect.DeepEqual(o1, o2), nil
}

func TestApp_createSilence(t *testing.T) {
	client := MockAlertManagerClient{}
	client.On("CreateSilenceWith",
		mock.AnythingOfType("string"),
		mock.AnythingOfType("string"),
		mock.AnythingOfType("APISilenceRequest")).Return("1234", nil)
	app := App{
		config: &Config{},
		client: &client,
	}

	// initialize router so createSilence handler doesn't fail at redirect
	router = mux.NewRouter().StrictSlash(true)
	router.HandleFunc("/", indexHandler).Name("indexHandler")

	form := url.Values{}
	form.Add("Comment", "test")
	form.Add("CreatedBy", "test")
	form.Add("Matchers.0.Name", "job")
	form.Add("Matchers.0.Value", "MockApp")
	form.Add("Matchers.0.IsRegex", "false")
	form.Add("Schedule.StartTime", "2020-11-01T22:12:33.533Z")
	form.Add("Schedule.EndTime", "2020-11-01T23:11:44.603Z")
	form.Add("Schedule.Repeat.Count", "1")
	form.Add("Schedule.Repeat.Interval", "h")

	req := httptest.NewRequest("POST", "/webhook", bytes.NewBufferString(form.Encode()))
	req.Form = form
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; param=value")

	// Create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(app.createSilence)

	// Test the handler with the request and record the result
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusFound {
		t.Errorf("wrong status code: got '%d' want '%d'", status, http.StatusFound)
	}
}

func TestApp_expireSilence(t *testing.T) {
	client := MockAlertManagerClient{}

	client.On("ExpireSilenceWithID",
		mock.AnythingOfType("string")).Return(nil)
	app := App{
		config: &Config{},
		client: &client,
	}
	// Create a request to pass to the handler
	req := httptest.NewRequest("GET", "/webhook", nil)

	// Create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(app.expireSilence)

	// Test the handler with the request and record the result
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("wrong status code: got '%d' want '%d'", status, http.StatusInternalServerError)
	}

}

func TestApp_getAllSilences(t *testing.T) {
	client := MockAlertManagerClient{}

	id := "7d8eb77e-00f9-4e0e-9f20-047695569296"
	state := "pending"
	updatedAt, _ := strfmt.ParseDateTime("2019-10-29T15:16:35.232Z")
	comment := "Silence"
	createdBy := "api"
	endsAt, _ := strfmt.ParseDateTime("2019-11-01T23:11:44.603Z")
	startsAt, _ := strfmt.ParseDateTime("2019-11-01T22:12:33.533Z")
	name := "job"
	value := "FakeApp"
	isRegex := false

	want := models.GettableSilences{{
		ID:        &id,
		Status:    &models.SilenceStatus{State: &state},
		UpdatedAt: &updatedAt,
		Silence: models.Silence{
			Comment:   &comment,
			CreatedBy: &createdBy,
			EndsAt:    &endsAt,
			Matchers: models.Matchers{
				&models.Matcher{Name: &name, Value: &value, IsRegex: &isRegex},
			},
			StartsAt: &startsAt,
		},
	}}

	client.On("ListSilences").Return(want, nil)
	app := App{
		config: &Config{},
		client: &client,
	}
	// Create a request to pass to the handler
	req := httptest.NewRequest("GET", "/webhook", nil)

	// Create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(app.getAllSilences)

	// Test the handler with the request and record the result
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("wrong status code: got '%d' want '%d'", status, http.StatusInternalServerError)
	}
	expected := `
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
	if ok, err := AreEqualJSON(rr.Body.String(), expected); !ok || err != nil {
		t.Errorf("handler returned unexpected body\ngot: '%s'\nwant: '%s'\nerror: '%s'",
			rr.Body.String(), expected, err.Error())
	}
}

func TestApp_getSilenceWithID(t *testing.T) {
	client := MockAlertManagerClient{}

	id := "7d8eb77e-00f9-4e0e-9f20-047695569296"
	status := "pending"
	updatedAt, _ := strfmt.ParseDateTime("2019-10-29T15:16:35.232Z")
	endsAt, _ := strfmt.ParseDateTime("2019-11-01T23:11:44.603Z")
	startsAt, _ := strfmt.ParseDateTime("2019-11-01T22:12:33.533Z")
	comment := "silence"
	createdBy := "api"
	name := "job"
	value := "FakeApp"
	isRegex := false

	want := models.GettableSilence{
		ID:        &id,
		Status:    &models.SilenceStatus{State: &status},
		UpdatedAt: &updatedAt,
		Silence: models.Silence{
			Comment:   &comment,
			CreatedBy: &createdBy,
			EndsAt:    &endsAt,
			Matchers: models.Matchers{
				&models.Matcher{
					Name:    &name,
					Value:   &value,
					IsRegex: &isRegex,
				},
			},
			StartsAt: &startsAt,
		},
	}
	client.On("GetSilenceWithID", mock.AnythingOfType("string")).Return(want, nil)
	app := App{
		config: &Config{},
		client: &client,
	}
	// Create a request to pass to the handler
	req := httptest.NewRequest("GET", "/webhook", nil)

	// Create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(app.getSilenceWithID)

	// Test the handler with the request and record the result
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("wrong status code: got '%d' want '%d'", status, http.StatusInternalServerError)
	}
	expected := `
{
  "id": "7d8eb77e-00f9-4e0e-9f20-047695569296",
  "status": {
    "state": "pending"
  },
  "updatedAt": "2019-10-29T15:16:35.232Z",
  "comment": "silence",
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
	if ok, err := AreEqualJSON(rr.Body.String(), expected); !ok || err != nil {
		t.Errorf("handler returned unexpected body\ngot: '%s'\nwant: '%s'\nerror: '%v'",
			rr.Body.String(), expected, err)
	}
}

func TestAddDuration(t *testing.T) {
	var cases = []struct {
		timestamp string
		interval  string
		count     int
		want      string
	}{
		{"2019-10-27T20:34:28.132Z", "h", 0, "2019-10-27T20:34:28.132Z"},
		{"2019-10-27T20:34:28.132Z", "w", 0, "2019-10-27T20:34:28.132Z"},
		{"2019-10-27T20:34:28.132Z", "h", 12, "2019-10-28T08:34:28.132Z"},
		{"2019-10-27T20:34:28.132Z", "d", 3, "2019-10-30T20:34:28.132Z"},
		{"2019-10-27T20:34:28.132Z", "w", 1, "2019-11-03T20:34:28.132Z"},
	}

	for _, c := range cases {
		got, err := addDuration(c.timestamp, c.interval, c.count)
		if err != nil || got != c.want {
			t.Errorf("error adding duration\ngot: '%s'\nwant: '%s'", got, c.want)
		}
	}
}

var (
	validationSuccess = true
	validationError   = false
)

func TestAPISilenceRequest_Valid(t *testing.T) {
	name := "foo"
	value := "bar"
	isRegex := false
	var cases = []struct {
		request APISilenceRequest
		want    bool
	}{
		// happy path
		{APISilenceRequest{
			Comment:   "scheduled maintenance",
			CreatedBy: "scheduler",
			Matchers: []Matcher{
				Matcher{
					Name:    name,
					Value:   value,
					IsRegex: isRegex,
				},
			},
			Schedule: Schedule{
				StartTime: "2021-10-12T12:34:02.566Z",
				EndTime:   "2021-10-12T13:34:02.566Z",
				Repeat: Repeat{
					Enabled:  true,
					Interval: "h",
					Count:    1,
				},
			},
		}, validationSuccess},

		// missing Comment & CreatedBy
		{APISilenceRequest{
			Matchers: []Matcher{
				Matcher{
					Name:    name,
					Value:   value,
					IsRegex: isRegex,
				},
			},
			Schedule: Schedule{
				StartTime: "2021-10-12T12:34:02.566Z",
				EndTime:   "2021-10-12T13:34:02.566Z",
				Repeat: Repeat{
					Enabled:  true,
					Interval: "h",
					Count:    1,
				},
			},
		}, validationError},

		// empty Matchers
		{APISilenceRequest{
			Comment:   "scheduled maintenance",
			CreatedBy: "scheduler",
			Matchers:  []Matcher{},
			Schedule: Schedule{
				StartTime: "2021-10-12T12:34:02.566Z",
				EndTime:   "2021-10-12T13:34:02.566Z",
				Repeat: Repeat{
					Enabled:  true,
					Interval: "h",
					Count:    1,
				},
			},
		}, validationError},
	}

	for _, c := range cases {
		_, got := c.request.Valid()
		if got != c.want {
			t.Errorf("APISilenceRequest validation not returning expected result for => '%v'\n", c.request)
		}
	}
}

func TestSchedule_Valid(t *testing.T) {
	var cases = []struct {
		schedule Schedule
		want     bool
	}{
		// happy path
		{Schedule{
			StartTime: "2021-10-12T12:34:02.566Z",
			EndTime:   "2021-10-12T13:34:02.566Z",
		}, validationSuccess,
		},

		// empty Schedule
		{Schedule{}, validationError},

		// invalid Schedule StartTime
		{Schedule{
			StartTime: "2021-10-12T12:34",
			EndTime:   "2021-10-12T13:34:02.566Z",
		}, validationError,
		},

		// invalid Schedule EndTime
		{Schedule{
			StartTime: "2021-10-12T12:34:02.566Z",
			EndTime:   "2021-10-12T13:34",
		}, validationError,
		},
	}

	for _, c := range cases {
		_, got := c.schedule.Valid()
		if got != c.want {
			t.Errorf("Schedule validation not returning expected result for => '%v'\n", c.schedule)
		}
	}
}

func TestRepeat_Valid(t *testing.T) {
	var cases = []struct {
		repeat Repeat
		want   bool
	}{
		// happy path
		{Repeat{
			Enabled:  true,
			Interval: "h",
			Count:    10,
		}, validationSuccess},

		// empty Repeat
		{Repeat{}, validationError},

		// invalid Repeat interval
		{Repeat{
			Enabled:  true,
			Interval: "purple",
			Count:    1,
		}, validationError},

		// Repeat count too high
		{Repeat{
			Enabled:  true,
			Interval: "h",
			Count:    100,
		}, validationError},
	}

	for _, c := range cases {
		_, got := c.repeat.Valid()
		if got != c.want {
			t.Errorf("Repeat validation not returning expected result for => '%v'\n", c.repeat)
		}
	}
}
