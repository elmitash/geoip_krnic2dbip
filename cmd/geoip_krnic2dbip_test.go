package main

import (
	"bytes"
	"math/rand"
	"net"
	"os"
	"testing"
	"time"
)

func TestRecordSerialization(t *testing.T) {
	orig := Record{
		FirstNum:    ip2int(net.ParseIP("192.168.1.0")),
		LastNum:     ip2int(net.ParseIP("192.168.1.255")),
		CountryCode: [2]byte{'K', 'R'},
	}

	b := orig.Bytes()
	if len(b) != RecordSize {
		t.Fatalf("expected record size %d, got %d", RecordSize, len(b))
	}

	var decoded Record
	if err := decoded.ReadFrom(b); err != nil {
		t.Fatalf("unexpected read error: %v", err)
	}

	if decoded.FirstNum != orig.FirstNum || decoded.LastNum != orig.LastNum || decoded.Country() != orig.Country() {
		t.Fatalf("mismatch: %+v vs %+v", orig, decoded)
	}
	if decoded.FirstIP().String() != "192.168.1.0" {
		t.Fatalf("expected FirstIP 192.168.1.0, got %s", decoded.FirstIP().String())
	}
	if decoded.LastIP().String() != "192.168.1.255" {
		t.Fatalf("expected LastIP 192.168.1.255, got %s", decoded.LastIP().String())
	}
}

func TestSortAndMerge(t *testing.T) {
	records := []Record{
		{
			FirstNum:    ip2int(net.ParseIP("1.1.1.0")),
			LastNum:     ip2int(net.ParseIP("1.1.1.127")),
			CountryCode: [2]byte{'A', 'U'},
		},
		{
			FirstNum:    ip2int(net.ParseIP("1.1.1.128")),
			LastNum:     ip2int(net.ParseIP("1.1.1.255")),
			CountryCode: [2]byte{'A', 'U'},
		},
		{
			FirstNum:    ip2int(net.ParseIP("2.0.0.0")),
			LastNum:     ip2int(net.ParseIP("2.0.0.255")),
			CountryCode: [2]byte{'F', 'R'},
		},
		{
			FirstNum:    ip2int(net.ParseIP("2.0.1.0")),
			LastNum:     ip2int(net.ParseIP("2.0.1.255")),
			CountryCode: [2]byte{'U', 'S'}, // different country, should NOT merge
		},
	}

	merged := SortAndMerge(records)
	if len(merged) != 3 {
		t.Fatalf("expected 3 merged records, got %d", len(merged))
	}

	if merged[0].Country() != "AU" || merged[0].FirstIP().String() != "1.1.1.0" || merged[0].LastIP().String() != "1.1.1.255" {
		t.Fatalf("unexpected merged AU range: %+v", merged[0])
	}
	if merged[1].Country() != "FR" {
		t.Fatalf("expected FR, got %s", merged[1].Country())
	}
	if merged[2].Country() != "US" {
		t.Fatalf("expected US, got %s", merged[2].Country())
	}
}

func TestParseCountryList(t *testing.T) {
	cases := []struct {
		input       string
		expected    []string
		expectError bool
	}{
		{"JP,KR,CN", []string{"JP", "KR", "CN"}, false},
		{" jp,  kr, cn , kr ", []string{"JP", "KR", "CN"}, false}, // lowercase, whitespace, duplicate
		{"US", []string{"US"}, false},
		{"", nil, false},
		{"   ", nil, false},
		{"USA", nil, true}, // 3 letters invalid
		{"J", nil, true},   // 1 letter invalid
	}

	for _, tc := range cases {
		res, err := ParseCountryList(tc.input)
		if tc.expectError && err == nil {
			t.Errorf("input '%s' expected error, got nil", tc.input)
		}
		if !tc.expectError && err != nil {
			t.Errorf("input '%s' unexpected error: %v", tc.input, err)
		}
		if len(res) != len(tc.expected) {
			t.Errorf("input '%s' expected len %d, got %d", tc.input, len(tc.expected), len(res))
			continue
		}
		for i := range res {
			if res[i] != tc.expected[i] {
				t.Errorf("input '%s' index %d expected %s, got %s", tc.input, i, tc.expected[i], res[i])
			}
		}
	}
}

func TestPartitionByCountries(t *testing.T) {
	records := []Record{
		{CountryCode: [2]byte{'J', 'P'}, FirstNum: 1, LastNum: 10},
		{CountryCode: [2]byte{'K', 'R'}, FirstNum: 11, LastNum: 20},
		{CountryCode: [2]byte{'C', 'N'}, FirstNum: 21, LastNum: 30},
		{CountryCode: [2]byte{'U', 'S'}, FirstNum: 31, LastNum: 40},
		{CountryCode: [2]byte{'J', 'P'}, FirstNum: 41, LastNum: 50},
	}

	targets := []string{"JP", "CN"}
	part := PartitionByCountries(records, targets)

	if len(part["JP"]) != 2 {
		t.Fatalf("expected 2 JP records, got %d", len(part["JP"]))
	}
	if len(part["CN"]) != 1 {
		t.Fatalf("expected 1 CN record, got %d", len(part["CN"]))
	}
	if _, exists := part["KR"]; exists {
		t.Fatalf("KR should not be in partitioned map")
	}
}

func TestBinarySearchMockData(t *testing.T) {
	records := []Record{
		// Google DNS: 8.8.8.8
		{
			FirstNum:    ip2int(net.ParseIP("8.8.8.0")),
			LastNum:     ip2int(net.ParseIP("8.8.8.255")),
			CountryCode: [2]byte{'U', 'S'},
		},
		// Yahoo Japan: 182.22.59.229
		{
			FirstNum:    ip2int(net.ParseIP("182.22.0.0")),
			LastNum:     ip2int(net.ParseIP("182.22.255.255")),
			CountryCode: [2]byte{'J', 'P'},
		},
		// Naver / KT Korea: 211.249.220.24
		{
			FirstNum:    ip2int(net.ParseIP("211.249.0.0")),
			LastNum:     ip2int(net.ParseIP("211.249.255.255")),
			CountryCode: [2]byte{'K', 'R'},
		},
	}

	merged := SortAndMerge(records)
	var buf bytes.Buffer
	if err := WriteBinary(&buf, merged); err != nil {
		t.Fatalf("failed to write binary: %v", err)
	}

	rawBytes := buf.Bytes()
	reader := bytes.NewReader(rawBytes)
	recordCount := int64(len(rawBytes) / RecordSize)

	testCases := []struct {
		name        string
		ip          string
		expectedCC  string
		expectFound bool
	}{
		{"Yahoo Japan IP", "182.22.59.229", "JP", true},
		{"Yahoo Japan Start Boundary", "182.22.0.0", "JP", true},
		{"Yahoo Japan End Boundary", "182.22.255.255", "JP", true},
		{"Korea Naver IP", "211.249.220.24", "KR", true},
		{"Google DNS", "8.8.8.8", "US", true},
		{"Not Found Low", "1.1.1.1", "", false},
		{"Not Found High", "220.0.0.1", "", false},
		{"Not Found Gap", "100.0.0.1", "", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rec, err := SearchIP(reader, recordCount, net.ParseIP(tc.ip))
			if tc.expectFound {
				if err != nil {
					t.Fatalf("expected to find %s, got error: %v", tc.ip, err)
				}
				if rec.Country() != tc.expectedCC {
					t.Fatalf("expected country %s, got %s", tc.expectedCC, rec.Country())
				}
			} else {
				if err != ErrNotFound {
					t.Fatalf("expected ErrNotFound for %s, got: %v", tc.ip, err)
				}
			}
		})
	}
}

func BenchmarkSearchIP(b *testing.B) {
	// Generate 100,000 synthetic records
	n := 100000
	records := make([]Record, n)
	step := uint32(4294967295 / uint32(n*2))

	for i := 0; i < n; i++ {
		start := uint32(i*2) * step
		end := start + step - 1
		records[i] = Record{
			FirstNum:    start,
			LastNum:     end,
			CountryCode: [2]byte{'J', 'P'},
		}
	}

	var buf bytes.Buffer
	if err := WriteBinary(&buf, records); err != nil {
		b.Fatalf("failed to write binary: %v", err)
	}

	reader := bytes.NewReader(buf.Bytes())
	count := int64(n)

	// Pick random test IPs within range
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	targetIPs := make([]net.IP, 100)
	for i := 0; i < 100; i++ {
		randIdx := r.Intn(n)
		targetIPs[i] = int2ip(records[randIdx].FirstNum + 1)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ip := targetIPs[i%len(targetIPs)]
		rec, err := SearchIP(reader, count, ip)
		if err != nil || rec == nil {
			b.Fatalf("lookup failed for %s: %v", ip, err)
		}
	}
}

func findFile(name string) string {
	paths := []string{name, "../" + name}
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

func TestRealFilesIfExist(t *testing.T) {
	// If global.dat exists, test Google DNS
	if p := findFile("global.dat"); p != "" {
		rec, _, err := SearchIPInFile(p, "8.8.8.8")
		if err != nil || rec.Country() != "US" {
			t.Errorf("global.dat: 8.8.8.8 expected US, got %v (err: %v)", rec, err)
		}
	}

	// If jp.dat exists, test Yahoo Japan
	if p := findFile("jp.dat"); p != "" {
		rec, _, err := SearchIPInFile(p, "182.22.59.229")
		if err != nil || rec.Country() != "JP" {
			t.Errorf("jp.dat: 182.22.59.229 expected JP, got %v (err: %v)", rec, err)
		}
	}

	// If kr.dat exists, test Naver
	if p := findFile("kr.dat"); p != "" {
		rec, _, err := SearchIPInFile(p, "211.249.220.24")
		if err != nil || rec.Country() != "KR" {
			t.Errorf("kr.dat: 211.249.220.24 expected KR, got %v (err: %v)", rec, err)
		}
	}

	// If cn.dat exists, test Baidu / China IP
	if p := findFile("cn.dat"); p != "" {
		rec, _, err := SearchIPInFile(p, "220.181.38.148")
		if err != nil || rec.Country() != "CN" {
			t.Errorf("cn.dat: 220.181.38.148 expected CN, got %v (err: %v)", rec, err)
		}
	}
}

