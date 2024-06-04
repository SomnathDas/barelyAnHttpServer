/*
A really stupid implmentation of HTTP Server created for learning purpose.
Used a lot of go libs except "http", obviously lol
*/

package main

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"log"
	"net"
	"net/textproto"
	"os"
	"regexp"
	"strconv"
	"strings"
)

func main() {
	/* Starting a tcp listener */
	l, err := net.Listen("tcp", "0.0.0.0:4221")
	if err != nil {
		fmt.Println("Failed to bind to port 4221")
		os.Exit(1)
	} else {
		fmt.Println("TCP Server Listening At Port 4221")
	}
	defer l.Close()

	/* Listening for connections */
	for {
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting connection: ", err.Error())
			os.Exit(1)
		}

		/* Passing off tcp connection to the http handler */
		go HttpHandler(conn)

	}

}

func HttpHandler(conn net.Conn) {
	/* Reading bytes from the connection */
	reader := bufio.NewReader(conn)
	textproto := textproto.NewReader(reader)

	// parse Request Method, Path and Protocol Version
	methodpathproto, err := textproto.ReadLineBytes()
	if err != nil {
		fmt.Println("Error while reading from the connection.")
		conn.Close()
	}

	// parse Request Headers
	headers, err := textproto.ReadMIMEHeader()
	if err != nil {
		fmt.Println("Error while reading from the connection.")
		conn.Close()
	}

	/* Extracting common HTTP attributes */
	/* METHOD PATH HTTP/VERSION */
	var requestMethod string = strings.Split(string(methodpathproto), " ")[0]
	var requestPath string = strings.Split(string(methodpathproto), " ")[1]
	/* Headers */
	var userAgent string = headers.Get("User-Agent")

	/* regexes */
	echoPathRegex := regexp.MustCompile(`^/echo/[^/]+$`)
	filePathRegex := regexp.MustCompile(`^/files/[^/]+$`)

	// parse Request Body
	length, err := strconv.Atoi(headers.Get("Content-Length"))
	if err != nil {
		fmt.Println("Invalid or no Content-Length header.")
	}

	var requestBody = make([]byte, length)
	if length > 0 {
		_, err = io.ReadFull(reader, requestBody)
		if err != nil {
			fmt.Println("Failed to read complete request body.")
		}
	}

	/* Handling functions for different routes and methods*/
	if requestMethod == "GET" {
		if requestPath == "/" {
			ServeHome(conn)
		} else if echoPathRegex.Match([]byte(requestPath)) {
			/* Extracting /echo/{keyword} */
			var keyword string = strings.Split(requestPath, "/")[2]
			// Support for gzip encoding
			var encoding string
			var acceptEncoding string = headers.Get("Accept-Encoding")
			if strings.Contains(acceptEncoding, "gzip") {
				encoding = "gzip"
			}
			EchoEndpoint(conn, keyword, encoding)
		} else if requestPath == "/user-agent" {
			UserAgentEndpoint(conn, userAgent)
		} else if filePathRegex.Match([]byte(requestPath)) {
			var filename string = strings.Split(requestPath, "/")[2]
			SendFile(conn, filename)
		} else {
			NotFound(conn)
		}
	} else if requestMethod == "POST" {
		if filePathRegex.Match([]byte(requestPath)) {
			if err != nil {
				fmt.Println("Error. No Request Body Received in a POST request.")
			} else {
				var filename string = strings.Split(requestPath, "/")[2]
				ReceiveFile(conn, filename, requestBody)
			}
		}
	}
}

func ServeHome(conn net.Conn) {
	var response string = "HTTP/1.1 200 OK\r\n\r\n"
	SendVerifyCloseResponse(conn, response)
}

func NotFound(conn net.Conn) {
	var response string = "HTTP/1.1 404 Not Found\r\n\r\n"
	SendVerifyCloseResponse(conn, response)
}

func EchoEndpoint(conn net.Conn, echoKeyword string, encoding string) {
	var response string

	if encoding == "" {
		response = fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: %d\r\n\r\n"+echoKeyword, len(echoKeyword))
	} else {
		var gzipCompressedEcho bytes.Buffer
		gzipWriter := gzip.NewWriter(&gzipCompressedEcho)
		gzipWriter.Write([]byte(echoKeyword))
		gzipWriter.Close()

		response = fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: %d\r\nContent-Encoding: %v\r\n\r\n"+gzipCompressedEcho.String(), len(gzipCompressedEcho.Bytes()), encoding)
	}
	fmt.Println(response)
	SendVerifyCloseResponse(conn, response)
}

func UserAgentEndpoint(conn net.Conn, userAgent string) {
	var response string = fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: %d\r\n\r\n"+userAgent, len(userAgent))
	SendVerifyCloseResponse(conn, response)
}

func SendFile(conn net.Conn, filename string) {
	dir := os.Args[2]
	data, err := os.ReadFile(dir + filename)
	var response string
	if err != nil {
		response = "HTTP/1.1 404 Not Found\r\n\r\n"
	} else {
		response = fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: application/octet-stream\r\nContent-Length: %d\r\n\r\n%s", len(data), data)
	}
	SendVerifyCloseResponse(conn, response)
}

func ReceiveFile(conn net.Conn, filename string, filecontents []byte) {
	/* Absolute Path :: /some/file/path */
	dir := os.Args[2]
	err := os.WriteFile(dir+filename, []byte(filecontents), 0644)
	var response string
	if err != nil {
		response = "HTTP/1.1 404 Not Found\r\n\r\n"
	} else {
		response = "HTTP/1.1 201 Created\r\n\r\n" + string(filecontents)
	}
	SendVerifyCloseResponse(conn, response)
}

func SendVerifyCloseResponse(conn net.Conn, response string) {
	len, err := conn.Write([]byte(response))
	if err != nil {
		log.Fatal("An error occured writing to tcp connection.")
	} else {
		fmt.Println("Wrote response of length :: " + strconv.Itoa(len))
	}
	conn.Close()
}
