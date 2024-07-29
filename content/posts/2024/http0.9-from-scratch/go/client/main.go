package main

import (
	"fmt"
	"io"
	"log"
	"net"
)

func main() {
	conn, err := net.Dial("tcp", "127.0.0.1:9000")
	if err != nil {
		log.Fatalf("err: %s", err)
	}

	if _, err := conn.Write([]byte("GET /this/is/a/test\r\n")); err != nil {
		log.Fatalf("err: %s", err)
	}

	body, err := io.ReadAll(conn)
	if err != nil {
		log.Fatalf("err: %s", err)
	}

	fmt.Println(string(body))
}
