package main

import (
	"bufio"
	"encoding/json"
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

func (s *server) start(ip, port string) {
	ln, err := net.Listen("tcp", fmt.Sprintf("%s:%s", ip, port))
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

func (s *server) listenAndServe(addr string) {
	addrSplit := strings.Split(addr, ":")
	if len(addrSplit) != 2 {
		log.Fatal("invalid addr for start Method")
	}
	ip := addrSplit[0]
	port := addrSplit[1]

	go s.start(ip, port)
}

func main() {
	mux := http.NewServeMux()
	serv := server{
		key:       "abcdef",
		clientMap: sync.Map{},
	}
	serv.listenAndServe("127.0.0.1:9999")
	mux.HandleFunc("/", serv.root)
	http.ListenAndServe(":8888", mux)
}
