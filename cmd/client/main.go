package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"payproxy/internal"
)

type subscribePayload struct {
	Key string `json:"key"`
	Url string `json:"url"`
}

func errorHandler(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	proxyServer := flag.String("proxy", "", "the address of the proxy server")
	mirrorServer := flag.String("mirror", "", "the address of the mirror server")

	flag.Parse()

	if len(*proxyServer) == 0 {
		log.Fatal("the address of the proxy server has to be provided with the -proxy tag")
	}

	// Connexion au serveur
	conn, err := net.Dial("tcp", *proxyServer)
	errorHandler(err)
	sp := subscribePayload{Key: "abcdef", Url: "/yo/man"}
	if err = json.NewEncoder(conn).Encode(sp); err != nil {
		log.Fatal(err)
	}

	message, err := bufio.NewReader(conn).ReadString('\n')
	errorHandler(err)
	fmt.Println(message)

	for {
		buf := bufio.NewReader(conn)

		var r internal.Request
		if err = json.NewDecoder(buf).Decode(&r); err != nil {
			log.Fatal(err)
		}

		var buffer bytes.Buffer
		buffer.Write(r.Body)
		reader := io.Reader(&buffer)
		url := fmt.Sprintf("%s%s", *mirrorServer, r.Url)
		if len(r.Params) > 0 {
			url = fmt.Sprintf("%s?%s", url, r.Params)
		}
		req, err := http.NewRequest(
			r.Method,
			url,
			reader,
		)
		if err != nil {
			log.Fatal(err)
		}

		for name, values := range r.Headers {
			for _, value := range values {
				req.Header.Add(name, value)
			}
		}

		response, err := http.DefaultClient.Do(req)
		if err != nil {
			log.Fatal(err)
		}
		respByte, err := io.ReadAll(response.Body)
		if err != nil {
			log.Fatal(err)
		}
		log.Println(string(respByte))
	}
}
