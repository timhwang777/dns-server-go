package main

import (
	"encoding/binary"
	"fmt"
	"net"
	"strings"
)

type DNSHeader struct {
	ID      uint16
	QR      bool
	OPCODE  uint8
	AA      bool
	TC      bool
	RD      bool
	RA      bool
	Z       uint8
	RCODE   uint8
	QDCOUNT uint16
	ANCOUNT uint16
	NSCOUNT uint16
	ARCOUNT uint16
}

type DNSQuestion struct {
	Name  string
	Type  int
	Class int
}

type DNSAnswer struct {
	Name   string
	Type   int
	Class  int
	TTL    int
	Length int
	Data   string
}

func (h *DNSHeader) packHeader() uint16 {
	var header uint16 = 0

	if h.QR {
		header += 1 << 15
	}
	if h.OPCODE != 0 {
		header += uint16(h.OPCODE) << 11
	}
	if h.AA {
		header += 1 << 10
	}
	if h.TC {
		header += 1 << 9
	}
	if h.RD {
		header += 1 << 8
	}
	if h.RA {
		header += 1 << 7
	}
	if h.Z != 0 {
		header += uint16(h.Z) << 6
	}
	if h.RCODE != 0 {
		header += uint16(h.RCODE)
	}

	return header
}

func (h *DNSHeader) Encode() []byte {
	buffer := make([]byte, 12)

	binary.BigEndian.PutUint16(buffer[0:], h.ID)
	binary.BigEndian.PutUint16(buffer[2:], h.packHeader())
	binary.BigEndian.PutUint16(buffer[4:], h.QDCOUNT)
	binary.BigEndian.PutUint16(buffer[6:], h.ANCOUNT)
	binary.BigEndian.PutUint16(buffer[8:], h.NSCOUNT)
	binary.BigEndian.PutUint16(buffer[10:], h.ARCOUNT)

	fmt.Println("Header Result:", buffer)
	return buffer
}

func (q *DNSQuestion) Encode() []byte {
	// convert the domain name
	labels := strings.Split(q.Name, ".")
	var sequence []byte

	for _, label := range labels {
		sequence = append(sequence, byte(len(label)))
		sequence = append(sequence, label...)
	}
	sequence = append(sequence, '\x00')

	// convert type and class
	buffer := make([]byte, 4)
	binary.BigEndian.PutUint16(buffer[0:], uint16(q.Type))
	binary.BigEndian.PutUint16(buffer[2:], uint16(q.Class))

	result := append(sequence, buffer...)
	fmt.Println("Question Result:", result)

	return result
}

func (a *DNSAnswer) Encode() []byte {
	labels := strings.Split(a.Name, ".")
	var sequence []byte
	for _, label := range labels {
		sequence = append(sequence, byte(len(label)))
		sequence = append(sequence, label...)
	}
	sequence = append(sequence, '\x00')

	buffer := make([]byte, 10)
	ip := net.ParseIP(a.Data).To4()
	a.Length = len(ip)
	binary.BigEndian.PutUint16(buffer[0:], uint16(a.Type))
	binary.BigEndian.PutUint16(buffer[2:], uint16(a.Class))
	binary.BigEndian.PutUint32(buffer[4:], uint32(a.TTL))
	binary.BigEndian.PutUint16(buffer[8:], uint16(a.Length))

	result := append(sequence, buffer...)
	result = append(result, ip...)

	fmt.Println("Answer Result:", result)

	return result
}

func parseDNSHeader(receivedData []byte) DNSHeader {
	parsedResponse := DNSHeader{
		ID: binary.BigEndian.Uint16(receivedData[0:2]),
	}

	remainValues := binary.BigEndian.Uint16(receivedData[2:4])
	parsedResponse.QR = (remainValues & (1 << 15)) != 0
	parsedResponse.OPCODE = uint8(remainValues >> 11)
	parsedResponse.AA = (remainValues & (1 << 10)) != 0
	parsedResponse.TC = (remainValues & (1 << 9)) != 0
	parsedResponse.RD = (remainValues & (1 << 8)) != 0
	parsedResponse.RA = (remainValues & (1 << 7)) != 0
	parsedResponse.Z = uint8(remainValues >> 4)

	return parsedResponse
}

func main() {
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

		receivedData := string(buf[:size])
		fmt.Printf("Received %d bytes from %s: %s\n", size, source, receivedData)

		parsed := parseDNSHeader([]byte(receivedData))
		// DNS Header
		header := DNSHeader{
			ID:      parsed.ID,
			QR:      true,
			OPCODE:  parsed.OPCODE,
			AA:      false,
			TC:      false,
			RD:      parsed.RD,
			RA:      false,
			Z:       0,
			RCODE:   4,
			QDCOUNT: 1,
			ANCOUNT: 1,
			NSCOUNT: 0,
			ARCOUNT: 0,
		}

		// DNS Question
		question := DNSQuestion{
			Name:  "codecrafters.io",
			Type:  1,
			Class: 1,
		}

		// DNS Answer
		answer := DNSAnswer{
			Name:  "codecrafters.io",
			Type:  1,
			Class: 1,
			TTL:   60,
			Data:  "8.8.8.8",
		}

		response := append(header.Encode(), question.Encode()...)
		response = append(response, answer.Encode()...)
		fmt.Println("Final Response: ", response)

		_, err = udpConn.WriteToUDP(response, source)
		if err != nil {
			fmt.Println("Failed to send response:", err)
		}
	}
}
