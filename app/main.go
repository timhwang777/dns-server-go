package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"strings"
)

func main() {
	// DNS forwarder
	resolver := flag.String("resolver", "", "Resolver to forward requests to")
	flag.Parse()
	if resolver == nil || *resolver == "" {
		fmt.Println("Please specify a resolver to forward requests to")
		os.Exit(1)
	}

	// Resolve the resolver address and port
	resolverAddr, err := net.ResolveUDPAddr("udp", strings.TrimSpace(*resolver))
	if err != nil {
		fmt.Println("Failed to resolve resolver address:", err)
		os.Exit(1)
	}

	// Resolve the UDP address and port
	udpAddr, err := net.ResolveUDPAddr("udp", "127.0.0.1:2053")
	if err != nil {
		fmt.Println("Failed to resolve UDP address:", err)
		return
	}

	udpConn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		fmt.Println("Failed to bind to address:", err)
		return
	}

	defer udpConn.Close()

	buf := make([]byte, 512)

	for {
		size, source, err := udpConn.ReadFromUDP(buf)
		if err != nil {
			fmt.Println("Error receiving data:", err)
			break
		}

		packet := buf[:size]
		decodedMessage, err := parseDNSMessage(packet)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Falied to parse DNS packet: %s", err)
		}

		go handleConnection(udpConn, source, decodedMessage, resolverAddr)
	}

}

func handleConnection(conn *net.UDPConn, source *net.UDPAddr, message *DNSMessage, resolverAddr *net.UDPAddr) {
	resolverConn, err := net.DialUDP("udp", nil, resolverAddr)
	if err != nil {
		fmt.Println("Failed to connect to resolver:", err)
		return
	}
	defer conn.Close()

	fmt.Println("Received message from", source)
	fmt.Println("Message:", message)

	answers := make([]DNSAnswer, 0)
	for _, question := range message.Question {
		nameServerMsg := DNSMessage{
			Header:   message.Header,
			Question: []DNSQuestion{question},
			Answer:   []DNSAnswer{},
		}

		nameServerMsg.Header.QDCOUNT = 1
		nameServerMsg.Header.ANCOUNT = 0

		nameBytes := nameServerMsg.Encode()
		_, err = resolverConn.Write(nameBytes)
		if err != nil {
			fmt.Println("Failed to write to resolver:", err)
			continue
		}

		buffer := make([]byte, 512)
		size, err := resolverConn.Read(buffer)
		if err != nil {
			fmt.Println("Failed to read from resolver:", err)
			continue
		}

		response, err := parseDNSMessage(buffer[:size])
		if err != nil {
			fmt.Println("Failed to parse response from resolver:", err)
			continue
		}

		answers = append(answers, response.Answer...)
	}

	finalResponse := DNSMessage{
		Header:   message.Header,
		Question: message.Question,
		Answer:   answers,
	}

	if finalResponse.Header.OPCODE != 0 {
		finalResponse.Header.RCODE = 4
	} else {
		finalResponse.Header.RCODE = 0
	}
	finalResponse.Header.QR = true
	finalResponse.Header.ANCOUNT = uint16(len(answers))
	finalResponseBytes := finalResponse.Encode()

	_, err = conn.WriteToUDP(finalResponseBytes, source)
	if err != nil {
		fmt.Println("Failed to write to client:", err)
		return
	}
}
