package main

import(
	"fmt"
	"net"
	"os"
	"strings"
	"io/ioutil"
	"path/filepath"
)

const(
	CRLF = "\r\n"
	serverFilePath = "./files/"
)

const(
	connectionType = "tcp"
	host = "0.0.0.0"
	port = "4221"
)

const(
	OK = "HTTP/1.1 200 OK"
	CREATED = "HTTP/1.1 201 Created"
	BAD_REQUEST = "HTTP/1.1 400 Bad Request"
	NOT_FOUND = "HTTP/1.1 404 Not Found"
	INTERNAL_SERVER_ERROR = "HTTP/1.1 500 Internal Server Error"
	NOT_IMPLEMENTED = "HTTP/1.1 501 Not Implemented"
)

const(
	ContentTypePlainText = "Content-Type: text/plain"
	ContentTypeJSON = "Content-Type: application/json"
)

func parseRequestStartLine(startLine string)(method string, path string){
	parts := strings.Split(startLine, " ")
	if len(parts) != 3{
		fmt.Println("Incorrect format of request start line.")
		fmt.Println(startLine)
		os.Exit(1)
	}

	method = parts[0]
	path = parts[1]

	return method, path
}

func parseRequestHeaders(requestHeaders []string) map[string]string{
	headers := make(map[string]string)
	for _, line := range requestHeaders{
		if line == ""{
			break
		}
		parts := strings.SplitN(line, ": ", 2)
		if len(parts) == 2{
			headers[strings.ToLower(parts[0])] = parts[1]
		}
	}

	return headers
}

func parseRequest(buffer []byte, n int)(string, string, map[string]string, string){
	parts := strings.Split(string(buffer[:n]), "\r\n\r\n");
	head := strings.Split(parts[0], "\r\n");

	method, path := parseRequestStartLine(head[0]);
	headers := parseRequestHeaders(head[1:]);

	var body string
	if len(parts) > 1{
		body = parts[1]
	}else{
		body = ""
	}

	return method, path, headers, body;
}

func getFileExtensionFromContentType(contentType string) string{
	switch contentType{
		case "text/plain":
			return ".txt"
		case "application/json":
			return ".json"
		default:
			return ""
	}
}

func handleRequest(method string, path string, headers map[string]string, body string) string{
	if(method == "GET"){
		file := filepath.Join(serverFilePath, path)
		content, err := ioutil.ReadFile(file)
		if err == nil {
			if(strings.HasSuffix(path, ".txt")){
				response := OK + CRLF + ContentTypePlainText + CRLF + "Content-Length: " + fmt.Sprint(len(content)) + CRLF + CRLF + string(content)
				return response
			}else if(strings.HasSuffix(path, ".json")){
				response := OK + CRLF + ContentTypeJSON + CRLF + "Content-Length: " + fmt.Sprint(len(content)) + CRLF + CRLF + string(content)
				return response
			}else{
				content := "Please specify a .txt or .json in the file path."
				response := NOT_FOUND + CRLF + ContentTypePlainText + CRLF + "Content-Length: " + fmt.Sprint(len(content)) + CRLF + CRLF + string(content)
				return response
			}
		}else{
			content := "Not Found. Make sure the file you are looking for exists, and make sure you include either .txt or .json in the path."
			response := NOT_FOUND + CRLF + ContentTypePlainText + CRLF + "Content-Length: " + fmt.Sprint(len(content)) + CRLF + CRLF + string(content)
			return response
		}
	}else if(method == "POST"){
		contentType, exists := headers["content-type"]
		if !exists{
			content := "Please provide a Content-Type header of either text/plain or application/json."
			response := BAD_REQUEST + CRLF + ContentTypePlainText + CRLF +  "Content-Length: " + fmt.Sprint(len(content)) + CRLF + CRLF + string(content)
			return response
		}else{
			fileExtention := getFileExtensionFromContentType(contentType)
			if fileExtention == ""{
				content := "Please provide a Content-Type header of either text/plain or application/json."
				response := BAD_REQUEST + CRLF + ContentTypePlainText + CRLF + "Content-Length: " + fmt.Sprint(len(content)) + CRLF + CRLF + string(content)
				return response
			}

			dir := filepath.Dir(filepath.Join(serverFilePath, path))
			if err := os.MkdirAll(dir, 0755); err != nil{
				content := "ERROR: Could not create file."
				response := INTERNAL_SERVER_ERROR + CRLF + ContentTypePlainText + CRLF + "Content-Length: " + fmt.Sprint(len(content)) + CRLF + CRLF + string(content)
				return response
			}

			file := filepath.Join(serverFilePath, path + fileExtention)
			if err := ioutil.WriteFile(file, []byte(body), 0666); err != nil{
				content := "ERROR: Could not create or write to file."
				response := INTERNAL_SERVER_ERROR + CRLF + ContentTypePlainText + CRLF + "Content-Length: " + fmt.Sprint(len(content)) + CRLF + CRLF + string(content)
				return response
			}
			response := CREATED + CRLF + CRLF
			return response
		}
	}else{
		content := "ERROR: Method not implemented. Implemented methods include GET and POST."
		response := NOT_IMPLEMENTED + CRLF + ContentTypePlainText + CRLF + "Content-Length: " + fmt.Sprint(len(content)) + CRLF + CRLF + string(content)
		return response
	}
}

func handleConnection(connection net.Conn){
	defer connection.Close()

	buffer := make([]byte, 1024)
	n, err := connection.Read(buffer)
	if err != nil{
		fmt.Println("Error reading request: ", err)
		os.Exit(1)
	}

	method, path, headers, body := parseRequest(buffer, n);

	response := handleRequest(method, path, headers, body);
	
	_, err = connection.Write([]byte(response))
	if err != nil{
		fmt.Println("Error sending response: ", err)
		os.Exit(1)
	}
}

func main(){
	if(len(os.Args) > 1){
		fmt.Println("ERROR: Too Many Arguments");
		os.Exit(1);
	}

	listener, err := net.Listen(connectionType, host + ":" + port)
	if err != nil {
		fmt.Println("Failed to bind to port " + port)
		os.Exit(1)
	}
	defer listener.Close()

	fmt.Println("Server started. Listening on port " + port + "...")

	for{
		connection, err := listener.Accept()
		if err != nil {
			fmt.Println("Error accepting connection: ", err.Error())
			os.Exit(1)
		}

		go handleConnection(connection)
	}
}