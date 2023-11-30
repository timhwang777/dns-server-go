package main

import (
	"encoding/binary"
	"fmt"
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
	/*if h.Z != 0 {
		header += uint16(h.Z) << 6
	}*/
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

func parseDNSHeader(data []byte) (DNSHeader, error) {
	var header DNSHeader

	header.ID = binary.BigEndian.Uint16(data[0:2])

	remainValues := binary.BigEndian.Uint16(data[2:4])
	header.QR = (remainValues>>15)&1 == 1
	header.OPCODE = uint8((remainValues >> 11) & 0xF)
	header.AA = (remainValues>>10)&1 == 1
	header.TC = (remainValues>>9)&1 == 1
	header.RD = (remainValues>>8)&1 == 1
	header.RA = (remainValues>>7)&1 == 0
	header.Z = uint8((remainValues >> 4) & 0x7) // reserved value
	header.RCODE = uint8(remainValues & 0xF)
	header.QDCOUNT = binary.BigEndian.Uint16(data[4:6])
	header.ANCOUNT = binary.BigEndian.Uint16(data[6:8])
	header.NSCOUNT = binary.BigEndian.Uint16(data[8:10])
	header.ARCOUNT = binary.BigEndian.Uint16(data[10:12])

	return header, nil
}
