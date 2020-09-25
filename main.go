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
	"strings"

	"WebCGIProc-go/github.com/augustoroman/hexdump"
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
	length  uint32
	version int32
	id      int32
}

type RequestHandler func(w http.ResponseWriter, req *http.Request)

func genSessionID() string {
	return "0123456789abcdef"
}

func performRequest(session string, input string) string {
	fmt.Printf("Buffer request: %s\n\n", input)
	c, err := net.Dial("tcp", "127.0.0.1:4444")
	if err != nil {
		log.Panic(err)
	}

	var headerBinBuff bytes.Buffer
	var header ConnectionMessageHeader
	header.id = 0
	header.length = uint32(len(input) + len(session) + 1)
	header.version = 1
	fmt.Printf("Msd len: %d\n", header.length)
	binary.Write(&headerBinBuff, binary.LittleEndian, header)
	sendBytes := headerBinBuff.Bytes()
	sendBytes = append(sendBytes, []byte(session)...)
	sendBytes = append(sendBytes, []byte("\t")...)
	sendBytes = append(sendBytes, []byte(input)...)
	c.Write(sendBytes)

	response := make([]byte, 2048*4)
	n, err := c.Read(response)
	if err != nil {
		log.Panic(err)
	}

	respStr := string(response[12 : n-1])
	fmt.Printf("\nResponse:\n%s\n\n", response[12:n-1])
	fmt.Println(hexdump.Dump(response[:n-1]))
	return string(respStr)
}

func decodeInput(input []byte) string {
	ret := string(input)
	fmt.Printf("\n%s\n\n", ret)
	return ret
}

func setSessionInfo(session string, req *http.Request, w http.ResponseWriter) bool {
	body := make(map[string]interface{})
	data := make(map[string]interface{})

	body["id"] = 0
	body["function"] = "setSessionInfo"

	data["remoteHost"] = ""
	data["remoteAddr"] = req.RemoteAddr
	data["serverProtocol"] = "HTTP/1.1"
	data["serverPort"] = ServingPort
	data["https"] = false
	data["browser"] = req.UserAgent()

	body["data"] = data

	bodyJSON, err := json.Marshal(body)

	if err != nil {
		log.Fatal(err)
		return false
	}

	bodyStr := string(bodyJSON)
	response := performRequest(session, bodyStr)
	respJson := make(map[string]interface{})
	err = json.Unmarshal([]byte(response[1:]), &respJson)
	fmt.Println(respJson["resultCode"])
	// if respJson["resultCode"].(int) == 0 {
	// 	return true
	// }
	dumpMap("", respJson)

	return false
	//w.Write(c.Read)
}

func dumpMap(space string, m map[string]interface{}) {
	for k, v := range m {
		if mv, ok := v.(map[string]interface{}); ok {
			fmt.Printf("{ \"%v\": \n", k)
			dumpMap(space+"\t", mv)
			fmt.Printf("}\n")
		} else {
			fmt.Printf("%v %v : %v\n", space, k, v)
		}
	}
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
		cookies := req.Header["Cookie"]
		sessionId := ""
		for _, cookie := range cookies {
			fmt.Printf("Cookie: %s\n", cookie)
			if strings.HasPrefix(cookie, "mobile-access-session-id=") {
				sessionId = cookie[strings.Index(cookie, "=")+1:]
			}
		}
		if sessionId == "" {
			sessionId = genSessionID()
			cookies = append(cookies, sessionId)
			setSessionInfo(cookies[0], req, w)
			w.Header().Add("Set-Cookie", fmt.Sprintf("mobile-access-session-id=%s", cookies[0]))
		}

		fmt.Println("SessionId = " + sessionId)

		decodedInput := decodeInput(input)
		fmt.Printf("decodedInput: %s\n\n", decodedInput)

		response := performRequest(sessionId, string(decodedInput))
		respJson := make(map[string]interface{})
		err = json.Unmarshal([]byte(response[1:]), &respJson)
		dumpMap("", respJson)
		w.Write([]byte(response))
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
