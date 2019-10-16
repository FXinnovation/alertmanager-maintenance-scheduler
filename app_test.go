package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/mock"
)

type MockAlertManagerClient struct {
	mock.Mock
}

func (m *MockAlertManagerClient) ListAlerts() ([]AlertmanagerAlert, error) {
	args := m.Called()
	return args.Get(0).([]AlertmanagerAlert), args.Error(1)
}

func (m *MockAlertManagerClient) CreateSilenceWith(start, end string, matchers []map[string]interface{}) (string, error) {
	args := m.Called(start, end, matchers)
	return args.Get(0).(string), args.Error(1)
}
func (m *MockAlertManagerClient) UpdateSilenceWith(uuid, start, end string, matchers []map[string]interface{}) (string, error) {
	args := m.Called(start, end, matchers)
	return args.Get(0).(string), args.Error(1)
}
func (m *MockAlertManagerClient) GetSilenceWithID(uuid string) (AlertmanagerSilence, error) {
	args := m.Called(uuid)
	return args.Get(0).(AlertmanagerSilence), args.Error(1)
}
func (m *MockAlertManagerClient) ListSilences() (AlertmanagerSilenceList, error) {
	args := m.Called()
	return args.Get(0).(AlertmanagerSilenceList), args.Error(1)
}
func (m *MockAlertManagerClient) ExpireSilenceWithID(uuid string) error {
	args := m.Called(uuid)
	return args.Error(0)
}

func Test_getAlerts(t *testing.T) {
	client := MockAlertManagerClient{}
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
	client.On("ListAlerts").Return(wanted, nil)
	appli := App{
		config: &Config{},
		client: &client,
	}
	// Create a request to pass to the handler
	req := httptest.NewRequest("GET", "/webhook", nil)

	// Create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(appli.getAlerts)

	// Test the handler with the request and record the result
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Wrong status code: got %v, want %v", status, http.StatusInternalServerError)
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
		t.Errorf("handler returned unexpected body: got %v want %v, err: %v",
			rr.Body.String(), expected, err)
	}

}

func AreEqualJSON(s1, s2 string) (bool, error) {
	var o1 interface{}
	var o2 interface{}

	var err error
	err = json.Unmarshal([]byte(s1), &o1)
	if err != nil {
		return false, fmt.Errorf("Error mashalling string 1 :: %s", err.Error())
	}
	err = json.Unmarshal([]byte(s2), &o2)
	if err != nil {
		return false, fmt.Errorf("Error mashalling string 2 :: %s", err.Error())
	}

	return reflect.DeepEqual(o1, o2), nil
}

func TestApp_createSilence(t *testing.T) {
	client := MockAlertManagerClient{}
	client.On("CreateSilenceWith",
		mock.AnythingOfType("string"),
		mock.AnythingOfType("string"),
		mock.AnythingOfType("[]map[string]interface {}")).Return("1234", nil)
	appli := App{
		config: &Config{},
		client: &client,
	}
	// Create a request to pass to the handler
	stringBody := `
{
  "id": "id12343",
  "comment": "a test",
  "createdBy": "test",
  "matchers": [
    {
      "isRegex": false,
      "name": "job",
      "value": "FakeApp"
    }
  ],
  "schedule": {
    "start_time": "2019-11-01T22:12:33.533Z",
    "end_time": "2019-11-01T23:11:44.603Z",
    "repeat": {
      "enabled": false,
      "interval": "d",
      "count": 10
    }
  }
}
`
	bodyReader := strings.NewReader(stringBody)
	req := httptest.NewRequest("GET", "/webhook", bodyReader)

	// Create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(appli.createSilence)

	// Test the handler with the request and record the result
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Wrong status code: got %v, want %v", status, http.StatusOK)
	}
	expected := `{"status":"success","message":"10/10 new silences created"}`
	if ok, err := AreEqualJSON(rr.Body.String(), expected); !ok || err != nil {
		t.Errorf("handler returned unexpected body: got %v want %v, err: %v",
			rr.Body.String(), expected, err)
	}
}

func TestApp_expireSilence(t *testing.T) {
	client := MockAlertManagerClient{}

	client.On("ExpireSilenceWithID",
		mock.AnythingOfType("string")).Return(nil)
	appli := App{
		config: &Config{},
		client: &client,
	}
	// Create a request to pass to the handler
	req := httptest.NewRequest("GET", "/webhook", nil)

	// Create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(appli.expireSilence)

	// Test the handler with the request and record the result
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Wrong status code: got %v, want %v", status, http.StatusInternalServerError)
	}

}

func TestApp_getAllSilences(t *testing.T) {
	client := MockAlertManagerClient{}
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
	client.On("ListSilences").Return(wanted, nil)
	appli := App{
		config: &Config{},
		client: &client,
	}
	// Create a request to pass to the handler
	req := httptest.NewRequest("GET", "/webhook", nil)

	// Create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(appli.getAllSilences)

	// Test the handler with the request and record the result
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Wrong status code: got %v, want %v", status, http.StatusInternalServerError)
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
		t.Errorf("handler returned unexpected body: got %v want %v, err: %v",
			rr.Body.String(), expected, err)
	}
}

func TestApp_getSilenceWithID(t *testing.T) {
	client := MockAlertManagerClient{}
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
	client.On("GetSilenceWithID", mock.AnythingOfType("string")).Return(wanted, nil)
	appli := App{
		config: &Config{},
		client: &client,
	}
	// Create a request to pass to the handler
	req := httptest.NewRequest("GET", "/webhook", nil)

	// Create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(appli.getSilenceWithID)

	// Test the handler with the request and record the result
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Wrong status code: got %v, want %v", status, http.StatusInternalServerError)
	}
	expected := `
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
	if ok, err := AreEqualJSON(rr.Body.String(), expected); !ok || err != nil {
		t.Errorf("handler returned unexpected body: got %v want %v, err: %v",
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
	var cases = []struct {
		request APISilenceRequest
		want    bool
	}{
		// happy path
		{APISilenceRequest{
			Comment:   "scheduled maintenance",
			CreatedBy: "scheduler",
			Matchers: []map[string]interface{}{
				{
					"name":    "foo",
					"value":   "bar",
					"isRegex": false,
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
			Matchers: []map[string]interface{}{
				{
					"name":    "foo",
					"value":   "bar",
					"isRegex": false,
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
			Matchers:  []map[string]interface{}{},
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
		got := c.request.Valid()
		if got != c.want {
			t.Errorf("APISilenceRequest validation not returning expected result for => %v\n", c.request)
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
		got := c.schedule.Valid()
		if got != c.want {
			t.Errorf("Schedule validation not returning expected result for => %v\n", c.schedule)
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
		got := c.repeat.Valid()
		if got != c.want {
			t.Errorf("Repeat validation not returning expected result for => %v\n", c.repeat)
		}
	}
}
