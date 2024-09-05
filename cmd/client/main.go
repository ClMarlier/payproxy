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
	"os"
	"payproxy/internal"
	"strings"
	"time"
)

type subscribePayload struct {
	Key string `json:"key"`
	Url string `json:"url"`
}

func fatalHandler(err error) {
	if err != nil {
		printError(err)
		os.Exit(1)
	}
}

func printError(err error) {
	log.Println("\033[0;31m" + err.Error() + "\033[0m")
}

func checkFlags(proxyServer, targetServer, trackPath, secret string) error {
	err := []string{}
	if len(proxyServer) == 0 {
		err = append(err, "-proxy tag is mandatory")
	}
	if len(targetServer) == 0 {
		err = append(err, "-target tag is mandatory")
	}
	if len(trackPath) == 0 {
		err = append(err, "-path tag is mandatory")
	}
	if len(secret) == 0 {
		err = append(err, "-secret tag is mandatory")
	}
	if len(err) > 0 {
		joined := strings.Join(err, ",\n")
		return fmt.Errorf("%s%s", joined, ".\n type -h for help")
	}
	return nil
}

func main() {
	proxyServer := flag.String("proxy", "", "the uri of the proxy server")
	targetServer := flag.String("target", "", "the uri of the target server")
	trackPath := flag.String("path", "", "the path to listen")
	secret := flag.String("secret", "", "the proxy secret password")

	flag.Parse()
	err := checkFlags(*proxyServer, *targetServer, *trackPath, *secret)
	fatalHandler(err)

	for {
		conn, err := net.Dial("tcp", *proxyServer)
		if err != nil {
			log.Println("\033[0;33m proxy server unreachable, retry in 2sec\033[0m")
			time.Sleep(time.Second * 2)
			continue
		}
		defer conn.Close()

		fatalHandler(err)
		sp := subscribePayload{Key: *secret, Url: *trackPath}
		err = json.NewEncoder(conn).Encode(sp)
		fatalHandler(err)

		message, err := bufio.NewReader(conn).ReadString('\n')
		fatalHandler(err)
		fmt.Println(message)

		for {
			buf := bufio.NewReader(conn)

			var r internal.Request
			if err = json.NewDecoder(buf).Decode(&r); err != nil {
				printError(err)
				break
			}

			var buffer bytes.Buffer
			fmt.Println(string(r.Body))

			buffer.Write(r.Body)
			reader := io.Reader(&buffer)
			url := fmt.Sprintf("%s%s", *targetServer, r.Url)
			if len(r.Params) > 0 {
				url = fmt.Sprintf("%s?%s", url, r.Params)
			}
			req, err := http.NewRequest(
				r.Method,
				url,
				reader,
			)
			fatalHandler(err)

			for name, values := range r.Headers {
				for _, value := range values {
					req.Header.Add(name, value)
				}
			}

			response, err := http.DefaultClient.Do(req)
			fatalHandler(err)
			respByte, err := io.ReadAll(response.Body)
			fatalHandler(err)

			log.Println(string(respByte))
		}
	}
}
