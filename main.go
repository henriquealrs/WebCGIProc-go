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
	Length  uint32
	Version int32
	Id      int32
}

type JsonObject = map[string]interface{}

type RequestHandler func(w http.ResponseWriter, req *http.Request)

func genSessionID() string {
	return "0123456789abcdef"
}

func IsSetSessionInfoRequired(body JsonObject) bool {
	if function, ok := body["function"]; ok {
		return function == "isLogged" || function == "logon"
	}
	return false
}

func performRequest(session string, input string) string {
	fmt.Printf("Buffer request: %s\n\n", input)
	c, err := net.Dial("tcp", "127.0.0.1:4444")
	if err != nil {
		log.Panic(err)
	}

	headerLen := uint32(3 * 32 / 8)
	var headerBinBuff bytes.Buffer
	var header ConnectionMessageHeader
	header.Id = 0
	header.Length = uint32(len(input) + len(session) + 1)
	header.Version = 1
	fmt.Printf("Msd len: %d\n", header.Length)
	binary.Write(&headerBinBuff, binary.LittleEndian, header)
	sendBytes := headerBinBuff.Bytes()
	sendBytes = append(sendBytes, []byte(session)...)
	sendBytes = append(sendBytes, []byte("\t")...)
	sendBytes = append(sendBytes, []byte(input)...)
	c.Write(sendBytes)

	response := make([]byte, 2048*8)

	nTotal := uint32(0)
	for nTotal < headerLen {
		n, err := c.Read(response[nTotal:])
		if err != nil {
			log.Panic(err)
		}
		nTotal += uint32(n)
	}
	var respHeader ConnectionMessageHeader
	err = binary.Read(bytes.NewBuffer(response), binary.LittleEndian, &respHeader)
	if err != nil {
		log.Panic(err)
	}

	for nTotal < (headerLen + respHeader.Length) {
		fmt.Printf("%d < %d\n", nTotal, (headerLen + respHeader.Length))
		n, err := c.Read(response[nTotal:])
		if err != nil {
			log.Panic(err)
		}
		nTotal += uint32(n)
	}

	respStr := string(response[headerLen:nTotal])
	fmt.Printf("\nResponse:\n%s\n\n", respStr)
	fmt.Println(hexdump.Dump(response[:nTotal]))
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

		inputJson := make(JsonObject)
		err = json.Unmarshal(input, &inputJson)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			log.Fatal(err)
			return
		}

		cookies := req.Header["Cookie"]
		sessionId := ""
		for _, cookie := range cookies {
			if strings.HasPrefix(cookie, "mobile-access-session-id=") {
				sessionId = cookie[strings.Index(cookie, "=")+1:]
			}
		}
		if sessionId == "" {
			sessionId = genSessionID()
			cookies = append(cookies, sessionId)
			setSessionInfo(cookies[0], req, w)
			w.Header().Add("Set-Cookie", fmt.Sprintf("mobile-access-session-id=%s;HttpOnly", cookies[0]))
		} else {
			i := strings.Index(sessionId, ";")
			if i != -1 {
				sessionId = sessionId[i+2:]
			}
			if IsSetSessionInfoRequired(inputJson) && len(sessionId) > 0 {
				fmt.Printf("SessionIdConfig = %s\n", sessionId)
				setSessionInfo(sessionId, req, w)
			}
		}

		fmt.Println("SessionId = " + sessionId)

		decodedInput := decodeInput(input)
		fmt.Printf("decodedInput: %s\n\n", decodedInput)

		response := performRequest(sessionId, string(decodedInput))
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
