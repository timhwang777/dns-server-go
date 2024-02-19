# DNS Server in Golang

![Static Badge](https://img.shields.io/badge/Go-Solutions-blue?logo=Go
)

## Table of Contents
1. [About the Project](#about-the-project)
2. [Getting Started](#getting-started)
    - [Prerequisites](#prerequisites)
3. [Author](#author)

## About the Project
This project is a simple DNS forwarder written in Go. It listens for DNS requests on a specified UDP port, forwards the requests to a specified DNS resolver, and then sends the responses back to the client. The DNS forwarder is capable of handling multiple DNS queries in a single request and can handle concurrent requests using Go routines.

The main function of the program parses command-line flags to get the DNS resolver to forward requests to. It then sets up a UDP listener on port 2053 of the localhost. The program reads from the UDP connection in an infinite loop, parsing each DNS message it receives and then handling the connection in a separate Go routine.

The `handleConnection` function takes a UDP connection, the source address of the DNS request, the parsed DNS message, and the address of the DNS resolver. It opens a new UDP connection to the DNS resolver, sends the DNS queries to the resolver one by one, and reads the responses. It then constructs a final DNS message containing all the answers and sends this message back to the client.

## Getting Started
To get a local copy up and running, follow these steps:

1. Clone the repository
2. Install Go if you haven't already
3. Run the program with the `-resolver` flag to specify the DNS resolver to forward requests to. For example: `go run main.go -resolver=8.8.8.8:53`

### Prerequisites
This project requires Go 1.16 or later.

## Author
Timothy Hwang