package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strings"
)

type DNSQuestion struct {
	Name  string
	Type  uint16
	Class uint16
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

func parseDNSQuestion(payload *bytes.Buffer, packet []byte) (DNSQuestion, error) {
	var question DNSQuestion

	name, err := parseDomainName(payload, packet)
	if err != nil {
		return DNSQuestion{}, err
	}
	question.Name = name

	// Type and Class
	err = binary.Read(payload, binary.BigEndian, &question.Type)
	if err != nil {
		return DNSQuestion{}, err
	}
	err = binary.Read(payload, binary.BigEndian, &question.Class)
	if err != nil {
		return DNSQuestion{}, err
	}

	return question, nil
}

type Questions []DNSQuestion

func parseDNSQuestions(payload *bytes.Buffer, packet []byte, QDCOUNT int) (Questions, error) {
	questions := make(Questions, 0, QDCOUNT)

	for i := 0; i < QDCOUNT; i++ {
		question, err := parseDNSQuestion(payload, packet)
		if err != nil {
			return nil, err
		}
		questions = append(questions, question)
	}

	return questions, nil
}

func parseDomainName(payload *bytes.Buffer, packet []byte) (string, error) {
	var labels []string

	// begin of the find dns name loop
	for {
		lenByte, err := payload.ReadByte()
		if err != nil {
			return "", err
		}

		if lenByte == 0 {
			break
		}

		// it's a pointer, MSB is b11
		if lenByte&0xC0 == 0xC0 {
			byteRemain, err := payload.ReadByte()
			if err != nil {
				return "", err
			}

			offset := int(lenByte&0x3F)<<8 + int(byteRemain)

			offsetPayload := bytes.NewBuffer(packet[offset:])
			for {
				offsetLen, err := offsetPayload.ReadByte()
				if err != nil {
					return "", err
				}
				if offsetLen == 0 {
					break
				}

				offsetLabel := make([]byte, offsetLen)
				_, err = offsetPayload.Read(offsetLabel)
				if err != nil {
					return "", err
				}
				labels = append(labels, string(offsetLabel))
			}
			break
		}

		label := make([]byte, lenByte)
		_, err = payload.Read(label)
		if err != nil {
			return "", err
		}
		labels = append(labels, string(label))
	} // end of the find dns name loop

	return strings.Join(labels, "."), nil
}
