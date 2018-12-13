package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strconv"

	"github.com/gorilla/websocket"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"golang.org/x/oauth2/jwt"
	"google.golang.org/api/analytics/v3"
)

const (
	gaCreds string = "./config.json"
)

func auth() *http.Client {
	creds := getAnalyticsCredsFromJSON(gaCreds)

	conf := &jwt.Config{
		Email:        creds.ClientEmail,
		PrivateKey:   []byte(creds.PrivateKey),
		PrivateKeyID: creds.PrivateKeyID,
		Scopes:       []string{analytics.AnalyticsReadonlyScope},
		TokenURL:     google.JWTTokenURL,
	}

	return conf.Client(oauth2.NoContext)
}

type analyticsCred struct {
	ClientEmail  string `json:"client_email"`
	PrivateKey   string `json:"private_key"`
	PrivateKeyID string `json:"private_key_id"`
}

type usersByBrowserVerion = map[string]map[string]int

func getAnalyticsCredsFromJSON(file string) analyticsCred {
	byteValue, _ := ioutil.ReadFile(file)

	var res analyticsCred

	json.Unmarshal(byteValue, &res)

	return res
}

func getReport() [][]string {
	startDate := "2016-10-12"
	endDate := "2018-10-12"
	dimensions := "ga:browser,ga:browserVersion"
	sortBy := "ga:sessions"
	viewID := "ga:62539387"
	metrics := "ga:sessions"

	analyticsService, err := analytics.New(auth())

	if err != nil {
		fmt.Println(err)
	}

	data, err := analyticsService.Data.Ga.Get(viewID, startDate, endDate, metrics).Dimensions(dimensions).Sort(sortBy).Do()

	if err != nil {
		fmt.Println(err)
	}

	return data.Rows
}

func formatReportData(data [][]string) usersByBrowserVerion {
	result := make(usersByBrowserVerion)

	for _, browser := range data {
		_, exists := result[browser[0]]

		if exists {
			r, _ := regexp.Compile("^([^.]+)")

			majorRelease := r.FindString(browser[1])

			sessions, err := strconv.Atoi(browser[2])

			if err != nil {
				fmt.Println(err)
			}

			result[browser[0]][majorRelease] = result[browser[0]][majorRelease] + sessions
		} else {
			versions := make(map[string]int)

			result[browser[0]] = versions
		}
	}

	return result
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func main() {
	result, _ := json.Marshal(formatReportData(getReport()))

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")

		w.Write(result)
	})

	http.ListenAndServe(":5060", nil)
}
