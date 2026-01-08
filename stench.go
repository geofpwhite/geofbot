package main

import (
	"fmt"
	"net"
)

type stenchHandler struct {
	conn net.Conn
}

func starttcp() net.Conn {
	conn, err := net.Dial("tcp", ":4040")
	i := 0
	for err != nil && i < 100 {
		conn, err = net.Dial("tcp", ":4040")
	}
	if err != nil {
		return nil
	}
	return conn
}

func newStenchHandler() *stenchHandler {
	return &stenchHandler{
		conn: starttcp(),
	}
}

func (s *stenchHandler) eval(input string) string {
	_, err := s.conn.Write([]byte(input + "\n"))
	fmt.Println("written")
	if err != nil {
		panic(err)
	}
	buffer := make([]byte, 1028)
	n, err := s.conn.Read(buffer)
	fmt.Println("done reading")
	fmt.Println(string(buffer))
	if err != nil {
		panic(err)
	}
	return string(buffer[:n:n])
}
