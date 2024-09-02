package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"payproxy/internal"
	"strings"
	"sync"
)

type server struct {
	key       string
	clientMap sync.Map
}

type subscribePayload struct {
	Key string `json:"key"`
	Url string `json:"url"`
}

type broadcastChan chan internal.Request

func (s *server) root(w http.ResponseWriter, r *http.Request) {
	urlStr := fmt.Sprintf("%s", r.URL)
	urlStrSlice := strings.Split(urlStr, "?")
	params := ""
	url := urlStrSlice[0]
	if len(urlStrSlice) == 2 {
		params = urlStrSlice[1]
	}

	value, ok := s.clientMap.Load(url)
	if ok {
		chanList, ok := value.([]broadcastChan)
		if ok {
			body, err := io.ReadAll(r.Body)
			if err == nil {
				req := internal.Request{
					Method:  r.Method,
					Body:    body,
					Headers: r.Header,
					Params:  params,
					Url:     url,
				}
				for _, channel := range chanList {
					channel <- req
				}
			}
		}
	}
}

func (s *server) start(port string) {
	ln, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
	if err != nil {
		log.Fatal(err)
	}
	for {
		conn, err := ln.Accept()
		defer conn.Close()

		if err != nil {
			log.Println("client missconnected")
		}
		go func() {
			buf := bufio.NewReader(conn)
			channel := make(broadcastChan)

			// Authorization
			var sp subscribePayload
			if err = json.NewDecoder(buf).Decode(&sp); err != nil {
				log.Fatal(err)
			}
			if sp.Key == s.key {
				conn.Write([]byte("success\n"))
				var connList []broadcastChan
				value, ok := s.clientMap.Load(sp.Url)
				if !ok {
					connList = []broadcastChan{}
				}
				connList, ok = value.([]broadcastChan)
				if !ok {
					connList = []broadcastChan{}
				}
				s.clientMap.Store(sp.Url, append(connList, channel))
			} else {
				conn.Write([]byte("unauthorized\n"))
				return
			}

			// Main loop
			for {
				select {
				case req := <-channel:
					fmt.Println(req)
					json.NewEncoder(conn).Encode(req)
				}
			}
		}()
	}
}

func (s *server) listenAndServe(port string) {
	go s.start(port)
}

func checkFlags(tcpPort, httpPort, secret string) error {
	err := []string{}
	if len(tcpPort) == 0 {
		err = append(err, "-tcpPort tag is mandatory")
	}
	if len(httpPort) == 0 {
		err = append(err, "-httpPort tag is mandatory")
	}
	if len(secret) == 0 {
		err = append(err, "-secret tag is mandatory")
	}
	if len(err) > 0 {
		joined := strings.Join(err, ",\n")
		return fmt.Errorf("%s%s", joined, ".\n\n type -h for help")
	}
	return nil
}

func main() {
	secret := flag.String("secret", "", "the secret password")
	tcpPort := flag.String("tcpPort", "", "the port used for the tcp server")
	httpPort := flag.String("httpPort", "", "the port used for the http server")
	flag.Parse()
	if err := checkFlags(*tcpPort, *httpPort, *secret); err != nil {
		log.Fatal(err)
	}
	mux := http.NewServeMux()
	serv := server{
		key:       *secret,
		clientMap: sync.Map{},
	}
	serv.listenAndServe(*tcpPort)
	mux.HandleFunc("/", serv.root)
	http.ListenAndServe(fmt.Sprintf(":%s", *httpPort), mux)
}
