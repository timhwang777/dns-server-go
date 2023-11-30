package main

import (
	"bytes"
)

type DNSMessage struct {
	Header   DNSHeader
	Question []DNSQuestion
	Answer   []DNSAnswer
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

func parseDNSMessage(packet []byte) (DNSMessage, error) {
	var message DNSMessage

	// parse the header
	header, err := parseDNSHeader(packet)
	if err != nil {
		return DNSMessage{}, err
	}

	payload := bytes.NewBuffer(packet[12:])
	// parse the questions
	questions, err := parseDNSQuestions(payload, packet, int(header.QDCOUNT))
	if err != nil {
		return DNSMessage{}, err
	}

	// parse the answers
	answers, err := parseDNSAnswers(payload, packet, int(header.ANCOUNT))
	if err != nil {
		return DNSMessage{}, err
	}

	message.Header = header
	message.Question = questions
	message.Answer = answers
	return message, nil
}
