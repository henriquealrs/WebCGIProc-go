package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
)

const ServingPort int = 8000

type HttpData struct {
	method      string
	contentType string
	contentLen  int
	body        []byte
}

type SessionJson struct {
	remoteHost     string
	remoteAddress  string
	serverProtocol string
	serverPort     string
	https          bool
	browser        string
}

type RequestHandler func(w http.ResponseWriter, req *http.Request)

func genSessionID() string {
	return "0123456789abcdef"
}

func decodeInput(input []byte) string {
	ret := string(input)
	fmt.Printf("\n%s\n\n", ret)
	return ret
}

func setSessionInfo(sessionInfo string, input string, req *http.Request) {
	body := make(map[string]interface{})

	body["remoteHost"] = ""
	body["remoteAddr"] = req.RemoteAddr
	body["serverProtocol"] = "HTTP/1.1"
	body["ServerPort"] = ServingPort
	body["https"] = false
	body["browser"] = req.UserAgent()

	bodyJson, err := json.Marshal(body)

	if err != nil {
		log.Fatal(err)
		return
	}

	bodyStr := string(bodyJson)
}

func getPOSTHandler() RequestHandler {
	return func(w http.ResponseWriter, req *http.Request) {
		fmt.Println("Starting POST Handler")

		input, err := ioutil.ReadAll(req.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			log.Fatal(err)
			return
		}
		cookie := req.Header.Get("Cookie")
		if cookie == "" {
			cookie = genSessionID()
			w.Header().Add("Set-Cookie", fmt.Sprintf("mobile-access-session-id=%s", cookie))
		}

		decodedInput := decodeInput(input)
	}
}

func getGETRequestHandler() RequestHandler {
	return func(w http.ResponseWriter, req *http.Request) {
		fmt.Println("Starting GET Handler")

		params, err := url.ParseQuery(req.RequestURI)

		if err != nil {
			return
		}

		var first string
		query := ""
		for key, val := range params {
			if key[0] == '/' {
				first = key + val[0]
				continue
			}
			query = query + key + val[0]
		}
		query = first + query

		cookie := req.Header["Cookie"][0]
		if cookie == "" {
			cookie = genSessionID()
		}

		//newReq = http.NewRequest()
		fmt.Println(cookie)
		//fmt.Printf("%s\n", query)

		w.Write([]byte(query))
	}
}

func main() {

	handlers := make(map[string]RequestHandler)
	handlers["GET"] = getGETRequestHandler()
	handlers["POST"] = getPOSTHandler()

	fmt.Println("Starting")
	http.Handle("/MA", http.FileServer(http.Dir("MA")))
	http.HandleFunc("/MA/service", func(w http.ResponseWriter, req *http.Request) {
		handlers[req.Method](w, req)
	})

	listenStr := fmt.Sprintf(":%d", ServingPort)
	http.ListenAndServe(":8000", nil)
}
