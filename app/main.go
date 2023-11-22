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
	Name  string
	Type  int
	Class int
	TTL   int
	Data  string
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
	binary.BigEndian.PutUint16(buffer, uint16(q.Type))
	binary.BigEndian.PutUint16(buffer, uint16(q.Class))

	result := append(sequence, buffer...)

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
	binary.BigEndian.PutUint16(buffer, uint16(a.Type))
	binary.BigEndian.PutUint16(buffer, uint16(a.Class))
	binary.BigEndian.PutUint32(buffer, uint32(a.TTL))
	binary.BigEndian.PutUint16(buffer, uint16(a.Length))

	result := append(sequence, buffer...)
	result = append(result, []byte(a.Data)...)

	return result
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

		// DNS Header
		header := DNSHeader{
			ID:      1234,
			QR:      true,
			OPCODE:  0,
			AA:      false,
			TC:      false,
			RD:      false,
			RA:      false,
			Z:       0,
			RCODE:   0,
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

		_, err = udpConn.WriteToUDP(response, source)
		if err != nil {
			fmt.Println("Failed to send response:", err)
		}
	}
}
