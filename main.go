package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strconv"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"golang.org/x/oauth2/jwt"
	"google.golang.org/api/analytics/v3"

	s "strings"
)

const (
	acceptedBrowsers string = "^Chrome|Edge|Firefox|Internet Explorer|Opera|Safari$"
	gaCreds          string = "./config.json"
)

func auth() *http.Client {
	creds := getCreds(gaCreds)

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

type browserSessions = map[string]int

type analyticsData = [][]string

type responseData = map[string]map[string]int

func getCreds(file string) analyticsCred {
	byteValue, _ := ioutil.ReadFile(file)

	var res analyticsCred

	json.Unmarshal(byteValue, &res)

	return res
}

func getReport(startDate string, endDate string) analyticsData {
	dimensions := "ga:browser,ga:browserVersion"
	sortBy := "ga:sessions"
	viewID := "ga:62539387"
	metrics := "ga:sessions"

	analytics, _ := analytics.New(auth())

	data, _ := analytics.Data.Ga.Get(viewID, startDate, endDate, metrics).Dimensions(dimensions).Sort(sortBy).Do()

	return data.Rows
}

func compare(h browserSessions, r browserSessions) browserSessions {
	result := make(browserSessions)

	for k := range r {
		result[k] = r[k] - h[k]
	}

	return result
}

func format(data analyticsData) browserSessions {
	result := make(browserSessions)

	for _, bv := range data {
		r, _ := regexp.Compile("^([^.]+)")

		major := r.FindString(bv[1])

		sessions, _ := strconv.Atoi(bv[2])

		bv := s.Join([]string{bv[0], major}, "_")

		result[bv] = result[bv] + sessions
	}

	return result
}

func clean(data analyticsData) analyticsData {
	filtered := make(analyticsData, 0)
	browsers, _ := regexp.Compile(acceptedBrowsers)

	for _, value := range data {
		if browsers.MatchString(value[0]) {
			filtered = append(filtered, value)
		}
	}

	return filtered
}

func getJSON(data browserSessions) responseData {
	result := make(responseData, 0)

	for k, v := range data {
		bv := s.Split(k, "_")

		_, exists := result[bv[0]]

		if !exists {
			result[bv[0]] = make(map[string]int)
		}

		result[bv[0]][bv[1]] = v
	}

	return result
}

func main() {
	historical := clean(getReport("2014-10-12", "2018-10-12"))
	recent := getJSON(format(clean(getReport("2017-10-12", "2018-10-12"))))

	fHistorical := format(historical)
	fRecent := format(recent)

	compared := compare(fHistorical, fRecent)

	response := getJSON(format(clean(getReport("2017-10-12", "2018-10-12"))))

	result, err := json.Marshal(response)

	if err != nil {
		fmt.Println(err)
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	})

	http.HandleFunc("/api", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")

		w.Write(result)
	})

	const port = "5060"

	fmt.Printf("Listening on port: %s", port)

	http.ListenAndServe(":"+port, nil)
}
