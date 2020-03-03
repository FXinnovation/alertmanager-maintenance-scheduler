package main

import (
	"net/http"

	"github.com/gorilla/sessions"
)

var (
	cookieName = "ams-session"

	// no heavy encryption until authentication is required
	store = sessions.NewCookieStore([]byte(""))
)

type Flash struct {
	Status  string
	Message string
}

func sessionAddFlash(w http.ResponseWriter, r *http.Request, status, message string) error {
	session, err := store.Get(r, cookieName)
	if err != nil {
		return err
	}

	session.AddFlash(Flash{Status: status, Message: message})

	err = session.Save(r, w)
	if err != nil {
		return err
	}

	return nil
}

func sessionGetFlash(w http.ResponseWriter, r *http.Request) ([]interface{}, error) {
	session, err := store.Get(r, cookieName)
	if err != nil {
		return nil, err
	}

	flashes := session.Flashes()

	err = session.Save(r, w)
	if err != nil {
		return nil, err
	}

	return flashes, nil
}
