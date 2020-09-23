package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
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

type ConnectionMessageHeader struct {
	length  uint64
	version int16
	id      int64
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

func setSessionInfo(input string, req *http.Request) {
	body := make(map[string]interface{})

	body["remoteHost"] = ""
	body["remoteAddr"] = req.RemoteAddr
	body["serverProtocol"] = "HTTP/1.1"
	body["ServerPort"] = ServingPort
	body["https"] = false
	body["browser"] = req.UserAgent()

	bodyJSON, err := json.Marshal(body)

	if err != nil {
		log.Fatal(err)
		return
	}

	bodyStr := string(bodyJSON)
	fmt.Printf("Buffer request: %s\n\n", bodyStr)
	c, err := net.Dial("tcp", "127.0.0.1:4444")
	if err != nil {
		log.Panic(err)
	}

	// c.Write([]byte{0, 0, 0, 0, 0, 0, 0, 0, 0})
	// c.Write([]byte{0, 0, 0, 0, 0, 0, 0, 0, 0})
	// c.Write([]byte{0, 0, 0, 0, 0, 0, 0, 0, 0})
	var binBuff bytes.Buffer
	var header ConnectionMessageHeader
	header.id = 0
	header.length = 10
	header.version = 1
	fmt.Printf("header.id: %d\n", header.id)
	binary.Write(&binBuff, binary.LittleEndian, header)
	c.Write(binBuff.Bytes())

	fmt.Fprintf(c, "%s\t%s\n", input, bodyStr)
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
		fmt.Println("0")
		cookie := req.Header["Cookie"]
		if len(cookie) == 0 { // remember, cookie is []string
			newCookie := genSessionID()
			setSessionInfo(newCookie, req)
			//w.Header().Add("Set-Cookie", fmt.Sprintf("mobile-access-session-id=%s", cookie))
		} else {
			fmt.Printf("Cookie: %s\n", cookie)
		}
		//cookie[0] = genSessionID()

		decodedInput := decodeInput(input)
		fmt.Printf("decodedInput: %s\n\n", decodedInput)
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
	http.Handle("/", http.FileServer(http.Dir("public")))
	http.HandleFunc("/MA/service", func(w http.ResponseWriter, req *http.Request) {
		fmt.Println("Service")
		handlers[req.Method](w, req)
	})

	//listenStr := fmt.Sprintf(":%d", ServingPort)
	http.ListenAndServe(":8000", nil)
}
