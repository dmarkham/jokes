package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/julienschmidt/httprouter"
	"github.com/pkg/errors"
)

// work time 23:22 - 01.19
type name struct {
	FirstName string `json:"first_name,omitempty"`
	LastName  string `json:"last_name,omitempty"`
}

type joke struct {
	ID         int      `json:"id,omitempty"`
	Joke       string   `json:"joke,omitempty"`
	Categories []string `json:"categories,omitempty"`
}

type jokeResult struct {
	Type  string `json:"type,omitempty"`
	Value joke   `json:"value,omitempty"`
}

func combineJoke(netClient *http.Client) func(http.ResponseWriter, *http.Request, httprouter.Params) {

	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {

		var wg sync.WaitGroup
		wg.Add(2)
		var jokeResult *jokeResult
		var nameResult *name

		go func() {
			// TODO handle this error better should maybe use channels vs a waitgroup?
			jokeResult, _ = requestJoke(netClient)
			wg.Done()
		}()
		go func() {
			// TODO handle this error better should maybe use channels vs a waitgroup?
			nameResult, _ = requestName(netClient)
			wg.Done()
		}()
		wg.Wait()
		if nameResult != nil && jokeResult != nil {
			jokeResult.Value.Joke = strings.Replace(jokeResult.Value.Joke, "--Zippy--", nameResult.FirstName, 1)
			jokeResult.Value.Joke = strings.Replace(jokeResult.Value.Joke, "--Zippy2--", nameResult.LastName, 1)
			data, err := json.Marshal(jokeResult)
			if err != nil {
				http.Error(w, "Sorry we had issue #1", http.StatusInternalServerError)
			}
			fmt.Fprintf(w, string(data))
			return
		}
		http.Error(w, "Sorry we had issue $2", http.StatusInternalServerError)
	}
}

func requestName(netClient *http.Client) (*name, error) {
	resp, err := netClient.Get("https://names.mcquay.me/api/v0/")
	if err != nil {
		return nil, errors.WithStack(err)
	}

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	result := &name{}
	err = json.Unmarshal(data, &result)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return result, nil
}

func requestJoke(netClient *http.Client) (*jokeResult, error) {
	resp, err := netClient.Get("http://api.icndb.com/jokes/random?firstName=--Zippy--&lastName=--Zippy2--&limitTo=[nerdy]")
	if err != nil {
		return nil, errors.WithStack(err)
	}

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	result := &jokeResult{}
	err = json.Unmarshal(data, &result)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return result, nil
}

func main() {
	router := httprouter.New()
	// set up a custom transport
	tr := &http.Transport{
		MaxIdleConns:           40, // We have 2 hosts so ~20 each
		MaxConnsPerHost:        20,
		MaxResponseHeaderBytes: 1024 * 1024,

		Dial: (&net.Dialer{
			Timeout: 5 * time.Second,
		}).Dial,
	}
	// set up a custom client
	var netClient = &http.Client{
		Timeout:   time.Second * 10,
		Transport: tr,
	}

	// TODO Logging
	// TODO Rate Limiting
	router.GET("/", combineJoke(netClient))

	log.Fatal(http.ListenAndServe(":8080", router))
}
