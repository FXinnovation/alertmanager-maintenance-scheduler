package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
)

// AlertmanagerAPI interface to hold api methods
type AlertmanagerAPI interface {
	ListAlerts() ([]AlertmanagerAlert, error)
	CreateSilenceWith(start, end string, matchers []map[string]interface{}) (string, error)
	UpdateSilenceWith(uuid, start, end string, matchers []map[string]interface{}) (string, error)
	GetSilenceWithID(uuid string) (AlertmanagerSilence, error)
	ListSilences() (AlertmanagerSilenceList, error)
	ExpireSilenceWithID(uuid string) error
}

// AlertmanagerClient is the concrete implementation of the client object for methods calling the Alertmanager API
type AlertmanagerClient struct {
	AlertManagerAPIURL string
}

func (ac *AlertmanagerClient) constructURL(pairs ...string) (string, error) {
	u, err := url.Parse(ac.AlertManagerAPIURL)
	if err != nil {
		return "", err
	}
	p := path.Join(pairs...)
	u.Path = path.Join(u.Path, p)

	return u.String(), nil
}

func (ac *AlertmanagerClient) doRequest(method, url string, requestBody io.Reader) ([]byte, error) {
	var client = &http.Client{}
	req, err := http.NewRequest(method, url, requestBody)
	if err != nil {
		return nil, fmt.Errorf("unable to create HTTP request: %s", err.Error())
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("unable to get response: %s", err.Error())
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("Alertmanager returned an HTTP error code: %d", resp.StatusCode)
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("unable to read response body: %s", err.Error())
	}
	return body, nil
}

// AlertmanagerAlert is the Alertmanager alert object returned by the API
type AlertmanagerAlert struct {
	Annotations map[string]string   `json:"annotations"`
	EndsAt      string              `json:"endsAt"`
	FingerPrint string              `json:"fingerprint"`
	Receivers   []map[string]string `json:"receivers"`
	StartsAt    string              `json:"startsAt"`
	Status      struct {
		InhibitedBy []string `json:"inhibitedBy"`
		SilencedBy  []string `json:"silencedBy"`
		State       string   `json:"state"`
	} `json:"status"`
	UpdatedAt    string            `json:"updatedAt"`
	GeneratorURL string            `json:"generatorURL"`
	Labels       map[string]string `json:"labels"`
}

// ListAlerts list all alerts
func (ac *AlertmanagerClient) ListAlerts() ([]AlertmanagerAlert, error) {
	var alerts []AlertmanagerAlert

	url, err := ac.constructURL("alerts")
	if err != nil {
		return alerts, err
	}

	body, err := ac.doRequest("GET", url, nil)
	if err != nil {
		return alerts, fmt.Errorf("unable to create HTTP request: %s", err.Error())
	}

	err = json.Unmarshal(body, &alerts)
	if err != nil {
		return alerts, fmt.Errorf("unable to unmarshal body: %s", err.Error())
	}
	return alerts, nil
}

// AlertmanagerSilence is the Alertmanager silence object returned by the API
type AlertmanagerSilence struct {
	ID     string `json:"id"`
	Status struct {
		State string `json:"state"`
	} `json:"status"`
	UpdatedAt string                   `json:"updatedAt"`
	Comment   string                   `json:"comment"`
	CreatedBy string                   `json:"createdBy"`
	EndsAt    string                   `json:"endsAt"`
	Matchers  []map[string]interface{} `json:"matchers"`
	StartsAt  string                   `json:"startsAt"`
}

// AlertmanagerSilenceList list of silences
type AlertmanagerSilenceList []AlertmanagerSilence

// FilterExpired filters expired silences
func FilterExpired(list AlertmanagerSilenceList) AlertmanagerSilenceList {
	var out AlertmanagerSilenceList
	for _, e := range list {
		if e.Status.State != "expired" {
			out = append(out, e)
		}
	}
	return out
}

// AlertmanagerSilenceResponse response received
type AlertmanagerSilenceResponse struct {
	SilenceID string `json:"silenceID"`
	Code      *int   `json:"code"`
	Message   string `json:"message"`
}

func constructSilence(start, end string, matchers []map[string]interface{}) AlertmanagerSilence {
	return AlertmanagerSilence{
		StartsAt:  start,
		EndsAt:    end,
		CreatedBy: "Maintenance Scheduler",
		Matchers:  matchers,
	}
}

// CreateSilenceWith creates a silence
func (ac *AlertmanagerClient) CreateSilenceWith(start, end string, matchers []map[string]interface{}) (string, error) {
	url, err := ac.constructURL("silences")
	if err != nil {
		return "", err
	}
	var silence = constructSilence(start, end, matchers)

	var b = new(bytes.Buffer)
	err = json.NewEncoder(b).Encode(silence)
	if err != nil {
		return "", fmt.Errorf("unable to encode request body: %s", err.Error())
	}

	body, err := ac.doRequest("POST", url, b)
	if err != nil {
		return "", fmt.Errorf("unable to create HTTP request: %s", err.Error())
	}

	var silenceResp AlertmanagerSilenceResponse
	err = json.Unmarshal(body, &silenceResp)
	if err != nil {
		return "", fmt.Errorf("unable to unmarshal body: %s", err.Error())
	}

	if silenceResp.Code != nil {
		return "", fmt.Errorf("unable to create silence: '%d %s'", silenceResp.Code, silenceResp.Message)
	}
	return silenceResp.SilenceID, nil
}

// UpdateSilenceWith updates a silence
func (ac *AlertmanagerClient) UpdateSilenceWith(uuid, start, end string, matchers []map[string]interface{}) (string, error) {
	err := ac.ExpireSilenceWithID(uuid)
	if err != nil {
		return "", err
	}

	silenceID, err := ac.CreateSilenceWith(start, end, matchers)
	if err != nil {
		return "", err
	}
	return silenceID, nil
}

// GetSilenceWithID get the silence asked
func (ac *AlertmanagerClient) GetSilenceWithID(uuid string) (AlertmanagerSilence, error) {
	var silence AlertmanagerSilence

	url, err := ac.constructURL("silence", uuid)
	if err != nil {
		return silence, err
	}

	body, err := ac.doRequest("GET", url, nil)
	if err != nil {
		return silence, fmt.Errorf("unable to create HTTP request: %s", err.Error())
	}

	err = json.Unmarshal(body, &silence)
	if err != nil {
		return silence, fmt.Errorf("unable to unmarshal body: %s", err.Error())
	}
	return silence, nil
}

// ListSilences list all silences
func (ac *AlertmanagerClient) ListSilences() (AlertmanagerSilenceList, error) {
	var silences AlertmanagerSilenceList

	url, err := ac.constructURL("silences")
	if err != nil {
		return silences, err
	}

	body, err := ac.doRequest("GET", url, nil)
	if err != nil {
		return silences, fmt.Errorf("unable to create HTTP request: %s", err.Error())
	}

	err = json.Unmarshal(body, &silences)
	if err != nil {
		return silences, fmt.Errorf("unable to unmarshal body: %s", err.Error())
	}
	return silences, nil
}

// ExpireSilenceWithID expires a silence
func (ac *AlertmanagerClient) ExpireSilenceWithID(uuid string) error {
	url, err := ac.constructURL("silence", uuid)
	if err != nil {
		return err
	}

	_, err = ac.doRequest("DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("unable to create HTTP request: %s", err.Error())
	}
	return nil
}

// NewAlertManagerClient creates a client to work with
func NewAlertManagerClient(apiURL string) *AlertmanagerClient {
	return &AlertmanagerClient{AlertManagerAPIURL: apiURL}
}
