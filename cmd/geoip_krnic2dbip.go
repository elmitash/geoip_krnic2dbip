package main

import (
	"encoding/binary"
	"encoding/csv"
	"io"
	"net"
	"os"
	"sort"

	"golang.org/x/text/encoding/korean"
	"golang.org/x/text/transform"
)

type countryIprange struct {
	countryCode string
	firstIp     string
	lastIp      string
	firstNum    uint32
	lastNum     uint32
}

func (c countryIprange) getCsv() []string {
	return []string{c.firstIp, c.lastIp, c.countryCode}
}

func main() {
	// read ipv4.csv
	fi, err := os.OpenFile("ipv4.csv", os.O_RDONLY, 0444)
	if err != nil {
		panic(err)
	}
	defer fi.Close()

	// get dtos
	dtos := filtering(transform.NewReader(fi, korean.EUCKR.NewDecoder()))

	// sort dtos
	sort.Slice(dtos, func(i, j int) bool {
		return dtos[i].firstNum < dtos[j].firstNum
	})

	// merge sequential ip
	i := 0
	for j, dto := range dtos {
		if j == 0 {
			dtos[i] = dto
			i++
			continue
		}

		prevDto := dtos[i-1]
		if prevDto.countryCode == dto.countryCode && prevDto.lastNum == dto.firstNum-1 {
			prevDto.lastIp = dto.lastIp
			prevDto.lastNum = dto.lastNum
			dtos[i-1] = prevDto
		} else {
			dtos[i] = dto
			i++
		}
	}

	dtos = dtos[:i]

	// write file
	outputFilePath := "dbip-country-lite.csv"
	fo, err := os.OpenFile(outputFilePath, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		panic(err)
	}
	defer fo.Close()

	w := csv.NewWriter(fo)
	for _, dto := range dtos {
		if err := w.Write(dto.getCsv()); err != nil {
			panic(err)
		}
	}
	w.Flush()
}

func filtering(rd io.Reader) (dtos []countryIprange) {
	rows, err := csv.NewReader(rd).ReadAll()
	if err != nil {
		panic(err)
	}

	for _, row := range rows {
		if len(row) == 6 && len(row[0]) == 8 {
			countryCode := row[1]
			firstIp := row[2]
			lastIp := row[3]
			firstNum := ip2int(net.ParseIP(firstIp))
			lastNum := ip2int(net.ParseIP(lastIp))

			dtos = append(dtos, countryIprange{countryCode, firstIp, lastIp, firstNum, lastNum})
		}
	}
	return dtos
}

func remove(dtos []countryIprange, i int) []countryIprange {
	return append(dtos[:i], dtos[i+1:]...)
}

func ip2int(ip net.IP) uint32 {
	if len(ip) == 16 {
		return binary.BigEndian.Uint32(ip[12:16])
	}
	return binary.BigEndian.Uint32(ip)
}

func int2ip(nn uint32) net.IP {
	ip := make(net.IP, 4)
	binary.BigEndian.PutUint32(ip, nn)
	return ip
}
