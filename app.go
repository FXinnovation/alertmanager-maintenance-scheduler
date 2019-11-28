package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"text/template"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/schema"
	"gopkg.in/alecthomas/kingpin.v2"
)

var (
	configFile    = kingpin.Flag("config.file", "Path to config file.").Short('c').Required().String()
	listenAddress = kingpin.Flag("web.listen-address", "Address for the application to listen on").Default("8080").Short('p').Int()
	genericError  = 1

	requestScheduleReg = regexp.MustCompile(`^(h|d|w)?$`)
)

const (
	errorStatus       = "error"
	requestTimeLayout = "2006-01-02T15:04:05.000Z"
)

// App receiver for app methods
type App struct {
	config *Config
	client AlertmanagerAPI
}

// APIResponse classical response of the API
type APIResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

func writeError(msg string, w http.ResponseWriter) {
	resp := APIResponse{Status: errorStatus, Message: msg}
	w.WriteHeader(http.StatusInternalServerError)
	json.NewEncoder(w).Encode(resp)
}

func (a *App) getAlerts(w http.ResponseWriter, r *http.Request) {
	alerts, err := a.client.ListAlerts()
	if err != nil {
		msg := fmt.Sprintf("unable to retrieve alerts: %s", err.Error())
		writeError(msg, w)
		return
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(alerts)
}

// APISilenceRequest request for silence
type APISilenceRequest struct {
	ID        string    `json:"id";schema:"ID"`
	Comment   string    `json:"comment";schema:"Comment"`
	CreatedBy string    `json:"createdBy";schema:"CreatedBy"`
	Matchers  []Matcher `json:"matchers";schema:"Matchers"`
	Schedule  Schedule  `json:"schedule";schema:"Schedule"`
}

type Matcher struct {
	Name    string `json:"name";schema:"Name"`
	Value   string `json:"value";schema:"Value"`
	IsRegex bool   `json:"isRegex";schema:"IsRegex"`
}

// Schedule structure
type Schedule struct {
	StartTime string `json:"start_time";schema:"StartTime"`
	EndTime   string `json:"end_time";schema:"EndTime"`
	Repeat    Repeat `json:"repeat";schema:"Repeat"`
}

// Repeat structure
type Repeat struct {
	Enabled  bool   `json:"enabled";schema:"-"`
	Interval string `json:"interval";schema:"Interval"`
	Count    int    `json:"count";schema:"Count"`
}

// Valid validates a silence request
func (r APISilenceRequest) Valid() bool {
	if r.Comment == "" || r.CreatedBy == "" {
		return false
	}
	if len(r.Matchers) < 1 {
		return false
	}

	for _, m := range r.Matchers {
		if !m.Valid() {
			return false
		}
	}

	if r.Schedule.Valid() != true {
		return false
	}

	if r.Schedule.Repeat.Valid() != true {
		return false
	}
	return true
}

func (m Matcher) Valid() bool {
	return true
}

//Valid returns true if the schedule is valid
func (s Schedule) Valid() bool {
	if s == (Schedule{}) {
		return false
	}
	_, err := time.Parse(requestTimeLayout, s.StartTime)
	if err != nil {
		return false
	}
	_, err = time.Parse(requestTimeLayout, s.EndTime)
	if err != nil {
		return false
	}
	return true
}

//Valid returns true if the repeat is valid
func (r Repeat) Valid() bool {
	if r == (Repeat{}) {
		return false
	}
	if r.Count > 50 {
		return false
	}
	if !requestScheduleReg.MatchString(r.Interval) {
		return false
	}
	return true
}

var intervalTable = map[string]time.Duration{
	"h": 1,
	"d": 24,
	"w": 168,
}

func addDuration(timestamp, interval string, count int) (string, error) {
	parsed, err := time.Parse(requestTimeLayout, timestamp)
	if err != nil {
		return "", err
	}
	next := parsed.Add(time.Hour * intervalTable[interval] * time.Duration(count))
	return next.Format(requestTimeLayout), nil
}

func (a *App) createSilence(w http.ResponseWriter, r *http.Request) {
	var silenceRequest APISilenceRequest
	var decoder = schema.NewDecoder()

	err := r.ParseForm()
	if err != nil {
		msg := fmt.Sprintf("unable to parse form: %s", err.Error())
		writeError(msg, w)
		return
	}

	err = decoder.Decode(&silenceRequest, r.PostForm)
	if err != nil {
		msg := fmt.Sprintf("unable to read silence request: %s", err.Error())
		writeError(msg, w)
		return
	}

	if !silenceRequest.Valid() {
		msg := "silence request is invalid"
		writeError(msg, w)
		return
	}

	var requestErr = 0
	for i := 0; i < silenceRequest.Schedule.Repeat.Count; i++ {
		nextStart, err := addDuration(silenceRequest.Schedule.StartTime, silenceRequest.Schedule.Repeat.Interval, i)
		if err != nil {
			log.Println(err)
			requestErr++
			continue
		}

		nextEnd, err := addDuration(silenceRequest.Schedule.EndTime, silenceRequest.Schedule.Repeat.Interval, i)
		if err != nil {
			log.Println(err)
			requestErr++
			continue
		}

		_, err = a.client.CreateSilenceWith(nextStart, nextEnd, silenceRequest)
		if err != nil {
			log.Println(err)
			requestErr++
			continue
		}
	}

	msg := fmt.Sprintf("%d/%d new silences created", silenceRequest.Schedule.Repeat.Count-requestErr, silenceRequest.Schedule.Repeat.Count)
	if requestErr != 0 {
		writeError(msg, w)
		return
	}

	resp := APIResponse{
		Status:  "success",
		Message: msg,
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

func (a *App) updateSilence(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	err := a.client.ExpireSilenceWithID(id)
	if err != nil {
		msg := fmt.Sprintf("unable to expire silence '%s': %s\n", id, err.Error())
		writeError(msg, w)
		return
	}

	url, err := mux.CurrentRoute(r).Subrouter().Get("createSilence").URL()
	if err != nil {
		msg := fmt.Sprintf("unable to update silence '%s'", id)
		writeError(msg, w)
		return
	}

	http.Redirect(w, r, url.String(), 307)
}

func (a *App) getSilenceWithID(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	silence, err := a.client.GetSilenceWithID(id)
	if err != nil {
		msg := fmt.Sprintf("unable to retrieve silence from Alertmanager: %s\n", err.Error())
		writeError(msg, w)
		return
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(silence)
}

func (a *App) getAllSilences(w http.ResponseWriter, r *http.Request) {
	silences, err := a.client.ListSilences()
	if err != nil {
		msg := fmt.Sprintf("unable to retrieve silences: %s\n", err.Error())
		writeError(msg, w)
		return
	}

	if strings.Contains(r.RequestURI, "filtered") {
		silences = FilterExpired(silences)
	}

	w.WriteHeader(http.StatusOK)
	err = json.NewEncoder(w).Encode(silences)
	if err != nil {
		msg := fmt.Sprintf("unable to retrieve silences: %s\n", err.Error())
		writeError(msg, w)
		return
	}
}

func (a *App) expireSilence(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	err := a.client.ExpireSilenceWithID(id)
	if err != nil {
		msg := fmt.Sprintf("unable to expire silence '%s': %s\n", id, err.Error())
		writeError(msg, w)
		return
	}
	resp := APIResponse{
		Status:  "success",
		Message: fmt.Sprintf("expired silence with ID: %s", id),
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

var templates *template.Template

func renderTemplate(w http.ResponseWriter, tmpl string, data interface{}) error {
	return templates.ExecuteTemplate(w, tmpl, data)
}

type IndexData struct {
	Data string
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	data := IndexData{
		Data: "sample data",
	}

	err := renderTemplate(w, "layout.gohtml", data)
	if err != nil {
		msg := fmt.Sprintf("Internal error rendering page: %s", err.Error())
		writeError(msg, w)
		return
	}
}

func main() {
	kingpin.Parse()

	appConf, err := loadConfig(*configFile)
	if err != nil {
		log.Printf("error loading config: %s\n", err.Error())
		os.Exit(genericError)
	}
	application := App{
		config: appConf,
		client: NewAlertManagerClient(appConf.AlertmanagerAPI),
	}

	templates, err = template.ParseGlob("templates/*")
	if err != nil {
		log.Printf("error loading templates: %s\n", err.Error())
		os.Exit(genericError)
	}

	r := mux.NewRouter().StrictSlash(true)

	s := r.PathPrefix("/api/v1/").Subrouter()
	s.HandleFunc("/alerts", application.getAlerts).Methods("GET").Name("getAlerts")
	s.HandleFunc("/silence", application.createSilence).Methods("POST").Name("createSilence")
	s.HandleFunc("/silences", application.getAllSilences).Methods("GET").Name("getAllSilences")
	s.HandleFunc("/silences_filtered", application.getAllSilences).Methods("GET").Name("getAllSilencesFiltered")
	s.HandleFunc("/silence/{id}", application.getSilenceWithID).Methods("GET").Name("getSilence")
	s.HandleFunc("/silence/{id}", application.updateSilence).Methods("POST").Name("updateSilence")
	s.HandleFunc("/silence/{id}", application.expireSilence).Methods("DELETE").Name("expireSilence")

	r.HandleFunc("/", indexHandler)
	http.Handle("/", r)

	log.Printf("Starting server on port %d\n", *listenAddress)
	err = http.ListenAndServe(fmt.Sprintf(":%d", *listenAddress), nil)
	if err != nil {
		log.Printf("error running server: %s\n", err.Error())
		os.Exit(genericError)
	}
}
