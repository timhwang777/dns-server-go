package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
	"os"
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
	Type  uint16
	Class uint16
}

type DNSAnswer struct {
	Name   string
	Type   uint16
	Class  uint16
	TTL    int
	Length int
	Data   string
}

type DNSMessage struct {
	Header   DNSHeader
	Question []DNSQuestion
	Answer   []DNSAnswer
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

func (h *DNSHeader) parseDNSHeader(data []byte) {
	h.ID = binary.BigEndian.Uint16(data[0:2])

	remainValues := binary.BigEndian.Uint16(data[2:4])
	h.QR = (remainValues>>15)&1 == 1
	h.OPCODE = uint8((remainValues >> 11) & 0xF)
	h.AA = (remainValues>>10)&1 == 1
	h.TC = (remainValues>>9)&1 == 1
	h.RD = (remainValues>>8)&1 == 1
	h.RA = (remainValues>>7)&1 == 0
	h.Z = uint8((remainValues >> 4) & 0x7) // reserved value
	h.RCODE = uint8(remainValues & 0xF)
	h.QDCOUNT = binary.BigEndian.Uint16(data[4:6])
	h.ANCOUNT = binary.BigEndian.Uint16(data[4:8])
	h.NSCOUNT = binary.BigEndian.Uint16(data[8:10])
	h.ARCOUNT = binary.BigEndian.Uint16(data[10:12])
}

func (m *DNSMessage) Encode() []byte {
	buffer := new(bytes.Buffer)
	buffer.Write(m.Header.Encode())
	for _, q := range m.Question {
		buffer.Write(q.Encode())
	}
	for _, a := range m.Answer {
		buffer.Write(a.Encode())
	}

	return buffer.Bytes()
}

func parseQuestion(payload *bytes.Buffer, packet []byte) (DNSQuestion, error) {
	var question DNSQuestion
	var labels []string

	// begin of the find dns name loop
	for {
		lenByte, err := payload.ReadByte()
		if err != nil {
			return DNSQuestion{}, err
		}

		if lenByte == 0 {
			break
		}

		// it's a pointer, MSB is b11
		if lenByte&0xC0 == 0xC0 {
			byteRemain, err := payload.ReadByte()
			if err != nil {
				return DNSQuestion{}, err
			}

			offset := int(lenByte&0x3F)<<8 + int(byteRemain)

			offsetPayload := bytes.NewBuffer(packet[offset:])
			for {
				offsetLen, err := offsetPayload.ReadByte()
				if err != nil {
					return DNSQuestion{}, err
				}
				if offsetLen == 0 {
					break
				}

				offsetLabel := make([]byte, offsetLen)
				_, err = offsetPayload.Read(offsetLabel)
				if err != nil {
					return DNSQuestion{}, err
				}
				labels = append(labels, string(offsetLabel))
			}
			break
		}

		label := make([]byte, lenByte)
		_, err = payload.Read(label)
		if err != nil {
			return DNSQuestion{}, err
		}
		labels = append(labels, string(label))
	} // end of the find dns name loop

	question.Name = strings.Join(labels, ".")

	// Type and Class
	err := binary.Read(payload, binary.BigEndian, &question.Type)
	if err != nil {
		return DNSQuestion{}, err
	}
	err = binary.Read(payload, binary.BigEndian, &question.Class)
	if err != nil {
		return DNSQuestion{}, err
	}

	return question, nil
}

func parseDNSPacket(packet []byte) (*DNSMessage, error) {
	var header DNSHeader
	header.parseDNSHeader(packet[:12])

	questions := make([]DNSQuestion, 0, header.QDCOUNT)
	payload := bytes.NewBuffer(packet[12:])

	for i := 0; i < int(header.QDCOUNT); i++ {
		question, err := parseQuestion(payload, packet)
		if err != nil {
			return nil, err
		}
		questions = append(questions, question)
	}

	return &DNSMessage{
		Header:   header,
		Question: questions,
		Answer:   make([]DNSAnswer, 0),
	}, nil
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

		packet := buf[:size]
		decodedMessage, err := parseDNSPacket(packet)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Falied to parse DNS packet: %s", err)
		}

		// response
		var response DNSMessage

		// DNS Header
		header := DNSHeader{
			ID:      decodedMessage.Header.ID,
			QR:      true,
			OPCODE:  decodedMessage.Header.OPCODE,
			AA:      false,
			TC:      false,
			RD:      decodedMessage.Header.RD,
			RA:      false,
			Z:       0,
			RCODE:   4,
			QDCOUNT: decodedMessage.Header.QDCOUNT,
			ANCOUNT: decodedMessage.Header.ANCOUNT,
			NSCOUNT: 0,
			ARCOUNT: 0,
		}
		response.Header = header

		// DNS Question
		question := decodedMessage.Question
		response.Question = question

		response.Answer = make([]DNSAnswer, 0)
		for _, q := range decodedMessage.Question {
			ans := DNSAnswer{
				Name:  q.Name,
				Type:  q.Type,
				Class: q.Class,
				TTL:   1000,
				Data:  "8.8.8.8",
			}
			response.Answer = append(response.Answer, ans)
		}

		fmt.Println("Final Response: ", response)

		_, err = udpConn.WriteToUDP(response.Encode(), source)
		if err != nil {
			fmt.Println("Failed to send response:", err)
		}
	}
}
