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

	"github.com/go-openapi/strfmt"
	"github.com/prometheus/alertmanager/api/v2/models"
)

// AlertmanagerAPI interface to hold api methods
type AlertmanagerAPI interface {
	ListAlerts() (models.GettableAlerts, error)
	CreateSilenceWith(start, end string, request APISilenceRequest) (string, error)
	UpdateSilenceWith(uuid, start, end string, request APISilenceRequest) (string, error)
	GetSilenceWithID(uuid string) (models.GettableSilence, error)
	ListSilences() (models.GettableSilences, error)
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

// ListAlerts list all alerts
func (ac *AlertmanagerClient) ListAlerts() (models.GettableAlerts, error) {
	var alerts models.GettableAlerts

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

// FilterExpired filters expired silences
func FilterExpired(list models.GettableSilences) models.GettableSilences {
	var out models.GettableSilences
	for _, e := range list {
		if *e.Status.State != "expired" {
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

func constructSilence(start, end string, request APISilenceRequest) (models.Silence, error) {
	var silence models.Silence

	startDatetime, err := strfmt.ParseDateTime(start)
	if err != nil {
		return silence, err
	}
	silence.StartsAt = &startDatetime

	endDatetime, err := strfmt.ParseDateTime(end)
	if err != nil {
		return silence, err
	}
	silence.EndsAt = &endDatetime

	silence.CreatedBy = &request.CreatedBy
	silence.Comment = &request.Comment

	for _, m := range request.Matchers {
		silence.Matchers = append(silence.Matchers,
			&models.Matcher{
				Name:    &m.Name,
				Value:   &m.Value,
				IsRegex: &m.IsRegex,
			})
	}

	return silence, nil
}

// CreateSilenceWith creates a silence
func (ac *AlertmanagerClient) CreateSilenceWith(start, end string, request APISilenceRequest) (string, error) {
	url, err := ac.constructURL("silences")
	if err != nil {
		return "", err
	}

	silence, err := constructSilence(start, end, request)
	if err != nil {
		return "", err
	}

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
func (ac *AlertmanagerClient) UpdateSilenceWith(uuid, start, end string, request APISilenceRequest) (string, error) {
	err := ac.ExpireSilenceWithID(uuid)
	if err != nil {
		return "", err
	}

	silenceID, err := ac.CreateSilenceWith(start, end, request)
	if err != nil {
		return "", err
	}
	return silenceID, nil
}

// GetSilenceWithID returns a silence with the specified ID
func (ac *AlertmanagerClient) GetSilenceWithID(uuid string) (models.GettableSilence, error) {
	var silence models.GettableSilence

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
func (ac *AlertmanagerClient) ListSilences() (models.GettableSilences, error) {
	var silences models.GettableSilences

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
