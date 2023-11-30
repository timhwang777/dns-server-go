package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
	"strings"
)

type DNSAnswer struct {
	Name   string
	Type   uint16
	Class  uint16
	TTL    int
	Length int
	Data   string
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

func parseDNSAnswer(payload *bytes.Buffer, packet []byte) (DNSAnswer, error) {
	var answer DNSAnswer

	// parse the name
	name, err := parseDomainName(payload, packet)
	if err != nil {
		return DNSAnswer{}, nil
	}
	answer.Name = name

	// parse the type and class
	typeAndClass := payload.Next(4)
	answer.Type = binary.BigEndian.Uint16(typeAndClass[0:2])
	answer.Class = binary.BigEndian.Uint16(typeAndClass[2:4])

	// parse the ttl and length
	ttlAndLength := payload.Next(6)
	answer.TTL = int(binary.BigEndian.Uint32(ttlAndLength[0:4]))
	answer.Length = int(binary.BigEndian.Uint16(ttlAndLength[4:6]))

	// parse the data
	answer.Data = net.IP(payload.Next(answer.Length)).String()

	return answer, nil
}

type Answers []DNSAnswer

func parseDNSAnswers(payload *bytes.Buffer, packet []byte, ANCOUNT int) (Answers, error) {
	var answers Answers

	for i := 0; i < ANCOUNT; i++ {
		answer, err := parseDNSAnswer(payload, packet)
		if err != nil {
			return nil, err
		}
		answers = append(answers, answer)
	}

	return answers, nil
}
