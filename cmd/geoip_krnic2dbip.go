package main

import (
	"bufio"
	"crypto/sha256"
	"encoding/binary"
	"encoding/csv"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"sort"
	"strings"
	"time"

	"golang.org/x/text/encoding/korean"
	"golang.org/x/text/transform"
)

const (
	// RecordSize represents the fixed binary record size in bytes
	// [0:4] uint32 BigEndian: firstNum (start IP)
	// [4:8] uint32 BigEndian: lastNum (end IP)
	// [8:10] byte[2] ASCII: 2-letter country code
	RecordSize = 10
)

var (
	// ErrNotFound is returned when an IP is not found in the binary database
	ErrNotFound = errors.New("IP not found in database")
)

// Record represents a single GeoIP range entry
type Record struct {
	FirstNum    uint32
	LastNum     uint32
	CountryCode [2]byte
}

// Country returns the 2-letter country code as a string
func (r Record) Country() string {
	return string(r.CountryCode[:])
}

// FirstIP returns the start IP as net.IP
func (r Record) FirstIP() net.IP {
	return int2ip(r.FirstNum)
}

// LastIP returns the end IP as net.IP
func (r Record) LastIP() net.IP {
	return int2ip(r.LastNum)
}

// Bytes serializes the record into exactly 10 bytes (Big-Endian)
func (r Record) Bytes() []byte {
	buf := make([]byte, RecordSize)
	binary.BigEndian.PutUint32(buf[0:4], r.FirstNum)
	binary.BigEndian.PutUint32(buf[4:8], r.LastNum)
	buf[8] = r.CountryCode[0]
	buf[9] = r.CountryCode[1]
	return buf
}

// ReadFrom deserializes a 10-byte slice into the Record
func (r *Record) ReadFrom(b []byte) error {
	if len(b) < RecordSize {
		return errors.New("insufficient bytes for record")
	}
	r.FirstNum = binary.BigEndian.Uint32(b[0:4])
	r.LastNum = binary.BigEndian.Uint32(b[4:8])
	r.CountryCode[0] = b[8]
	r.CountryCode[1] = b[9]
	return nil
}

// ToCSV returns slice of strings for CSV representation
func (r Record) ToCSV() []string {
	return []string{r.FirstIP().String(), r.LastIP().String(), r.Country()}
}

func ip2int(ip net.IP) uint32 {
	ipv4 := ip.To4()
	if ipv4 == nil {
		return 0
	}
	return binary.BigEndian.Uint32(ipv4)
}

func int2ip(nn uint32) net.IP {
	ip := make(net.IP, 4)
	binary.BigEndian.PutUint32(ip, nn)
	return ip
}

// ParseKRNIC reads and filters records from KRNIC CSV reader
func ParseKRNIC(rd io.Reader) ([]Record, error) {
	reader := csv.NewReader(rd)
	// Allow variable number of fields if needed
	reader.FieldsPerRecord = -1

	var records []Record
	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read CSV row: %w", err)
		}

		// KRNIC format: 기준일자, 국가코드, 시작IP, 끝IP, PREFIX, 할당일자
		// row[0] length 8 (YYYYMMDD), row[1] country code (2 chars)
		if len(row) >= 4 && len(row[0]) == 8 && len(row[1]) == 2 {
			country := strings.ToUpper(strings.TrimSpace(row[1]))
			firstIP := net.ParseIP(strings.TrimSpace(row[2])).To4()
			lastIP := net.ParseIP(strings.TrimSpace(row[3])).To4()

			if firstIP == nil || lastIP == nil {
				continue
			}

			firstNum := binary.BigEndian.Uint32(firstIP)
			lastNum := binary.BigEndian.Uint32(lastIP)

			if firstNum > lastNum {
				continue
			}

			rec := Record{
				FirstNum:    firstNum,
				LastNum:     lastNum,
				CountryCode: [2]byte{country[0], country[1]},
			}
			records = append(records, rec)
		}
	}

	return records, nil
}

// SortAndMerge sorts records by FirstNum and merges contiguous ranges with the same country
func SortAndMerge(records []Record) []Record {
	if len(records) <= 1 {
		return records
	}

	sort.Slice(records, func(i, j int) bool {
		if records[i].FirstNum == records[j].FirstNum {
			return records[i].LastNum < records[j].LastNum
		}
		return records[i].FirstNum < records[j].FirstNum
	})

	merged := make([]Record, 0, len(records))
	merged = append(merged, records[0])

	for i := 1; i < len(records); i++ {
		curr := records[i]
		lastIdx := len(merged) - 1
		prev := &merged[lastIdx]

		if prev.CountryCode == curr.CountryCode && prev.LastNum >= curr.FirstNum-1 {
			if curr.LastNum > prev.LastNum {
				prev.LastNum = curr.LastNum
			}
		} else {
			merged = append(merged, curr)
		}
	}

	return merged
}

// SearchIP performs an O(log N) binary search on an io.ReadSeeker containing 10-byte records
func SearchIP(rs io.ReadSeeker, recordCount int64, ip net.IP) (*Record, error) {
	ipv4 := ip.To4()
	if ipv4 == nil {
		return nil, errors.New("invalid IPv4 address")
	}
	targetNum := binary.BigEndian.Uint32(ipv4)

	low := int64(0)
	high := recordCount - 1
	buf := make([]byte, RecordSize)

	for low <= high {
		mid := low + (high-low)/2
		offset := mid * int64(RecordSize)

		if _, err := rs.Seek(offset, io.SeekStart); err != nil {
			return nil, fmt.Errorf("seek failed at offset %d: %w", offset, err)
		}
		if _, err := io.ReadFull(rs, buf); err != nil {
			return nil, fmt.Errorf("read record failed at offset %d: %w", offset, err)
		}

		var rec Record
		_ = rec.ReadFrom(buf)

		if targetNum < rec.FirstNum {
			high = mid - 1
		} else if targetNum > rec.LastNum {
			low = mid + 1
		} else {
			return &rec, nil
		}
	}

	return nil, ErrNotFound
}

// SearchIPInFile opens the .dat file and searches for target IP using fseek binary search
func SearchIPInFile(filePath string, ipStr string) (*Record, time.Duration, error) {
	start := time.Now()
	f, err := os.Open(filePath)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to open file %s: %w", filePath, err)
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		return nil, 0, fmt.Errorf("failed to stat file %s: %w", filePath, err)
	}

	if fi.Size()%int64(RecordSize) != 0 {
		return nil, 0, fmt.Errorf("invalid file size %d: not multiple of %d", fi.Size(), RecordSize)
	}

	parsedIP := net.ParseIP(strings.TrimSpace(ipStr))
	if parsedIP == nil || parsedIP.To4() == nil {
		return nil, 0, fmt.Errorf("invalid IPv4 address string: %s", ipStr)
	}

	recordCount := fi.Size() / int64(RecordSize)
	rec, err := SearchIP(f, recordCount, parsedIP)
	elapsed := time.Since(start)
	return rec, elapsed, err
}

// WriteBinary writes records to a writer in fixed 10-byte binary format
func WriteBinary(w io.Writer, records []Record) error {
	bw := bufio.NewWriter(w)
	buf := make([]byte, RecordSize)

	for _, r := range records {
		binary.BigEndian.PutUint32(buf[0:4], r.FirstNum)
		binary.BigEndian.PutUint32(buf[4:8], r.LastNum)
		buf[8] = r.CountryCode[0]
		buf[9] = r.CountryCode[1]

		if _, err := bw.Write(buf); err != nil {
			return err
		}
	}

	return bw.Flush()
}

// WriteBinaryWithChecksum writes records to a file and computes its SHA-256 hash simultaneously
func WriteBinaryWithChecksum(filePath string, records []Record) (string, error) {
	f, err := os.Create(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to create file %s: %w", filePath, err)
	}
	defer f.Close()

	hasher := sha256.New()
	mw := io.MultiWriter(f, hasher)

	if err := WriteBinary(mw, records); err != nil {
		return "", fmt.Errorf("failed to write records to %s: %w", filePath, err)
	}

	hashStr := hex.EncodeToString(hasher.Sum(nil))
	return hashStr, nil
}

// WriteCSV exports records to DB-IP compatible CSV format
func WriteCSV(filePath string, records []Record) error {
	f, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create csv file %s: %w", filePath, err)
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	for _, r := range records {
		if err := w.Write(r.ToCSV()); err != nil {
			return fmt.Errorf("failed to write csv record: %w", err)
		}
	}
	return nil
}

// ParseCountryList parses comma-separated 2-letter country codes
func ParseCountryList(str string) ([]string, error) {
	if strings.TrimSpace(str) == "" {
		return nil, nil
	}
	parts := strings.Split(str, ",")
	seen := make(map[string]bool)
	var list []string

	for _, p := range parts {
		cc := strings.ToUpper(strings.TrimSpace(p))
		if cc == "" {
			continue
		}
		if len(cc) != 2 {
			return nil, fmt.Errorf("invalid country code '%s': must be 2 uppercase ASCII letters", cc)
		}
		if !seen[cc] {
			seen[cc] = true
			list = append(list, cc)
		}
	}
	return list, nil
}

// PartitionByCountries dynamically partitions records by the requested country list in a single pass
func PartitionByCountries(records []Record, targetCountries []string) map[string][]Record {
	targetSet := make(map[string]bool, len(targetCountries))
	result := make(map[string][]Record, len(targetCountries))
	for _, cc := range targetCountries {
		targetSet[cc] = true
		result[cc] = make([]Record, 0)
	}

	for _, r := range records {
		cc := r.Country()
		if targetSet[cc] {
			result[cc] = append(result[cc], r)
		}
	}
	return result
}

func runBuild(inputPath string, targetCountries []string, buildGlobal bool, singleOutPath string, csvOutPath string) error {
	fmt.Printf("[1/4] Reading and decoding KRNIC CSV from: %s\n", inputPath)
	fi, err := os.Open(inputPath)
	if err != nil {
		return fmt.Errorf("cannot open input file: %w", err)
	}
	defer fi.Close()

	records, err := ParseKRNIC(transform.NewReader(fi, korean.EUCKR.NewDecoder()))
	if err != nil {
		return fmt.Errorf("failed to parse KRNIC CSV: %w", err)
	}
	fmt.Printf("      - Parsed raw records: %d\n", len(records))

	fmt.Println("[2/4] Sorting and merging contiguous IP ranges...")
	merged := SortAndMerge(records)
	fmt.Printf("      - Merged total records: %d\n", len(merged))

	fmt.Println("[3/4] Partitioning and writing binary files...")
	type targetFile struct {
		name    string
		records []Record
	}
	var targets []targetFile

	if buildGlobal {
		targets = append(targets, targetFile{"global.dat", merged})
	}

	if len(targetCountries) > 0 {
		partitioned := PartitionByCountries(merged, targetCountries)
		for _, cc := range targetCountries {
			outName := strings.ToLower(cc) + ".dat"
			if len(targetCountries) == 1 && singleOutPath != "" {
				outName = singleOutPath
			}
			targets = append(targets, targetFile{outName, partitioned[cc]})
		}
	}

	if len(targets) == 0 {
		return errors.New("no build targets specified (use -all or -country <codes>)")
	}

	var checksumLines []string
	for _, t := range targets {
		hash, err := WriteBinaryWithChecksum(t.name, t.records)
		if err != nil {
			return err
		}
		fiStat, _ := os.Stat(t.name)
		fmt.Printf("      - Created %-15s : %7d records, %8d bytes (SHA256: %s)\n",
			t.name, len(t.records), fiStat.Size(), hash[:12]+"...")
		checksumLines = append(checksumLines, fmt.Sprintf("%s  %s", hash, t.name))
	}

	checksumFile := "checksum.sha256"
	checksumContent := strings.Join(checksumLines, "\n") + "\n"
	if err := os.WriteFile(checksumFile, []byte(checksumContent), 0644); err != nil {
		return fmt.Errorf("failed to write %s: %w", checksumFile, err)
	}
	fmt.Printf("      - Created %-15s with SHA256 hashes for all targets.\n", checksumFile)

	if csvOutPath != "" {
		fmt.Printf("[4/4] Writing DB-IP compatible CSV to: %s\n", csvOutPath)
		if err := WriteCSV(csvOutPath, merged); err != nil {
			return err
		}
		fmt.Printf("      - CSV file saved successfully.\n")
	} else {
		fmt.Println("[4/4] Done! All targets created successfully.")
	}

	return nil
}

func runVerify(datPath, ipStr string) error {
	if datPath == "" || ipStr == "" {
		return errors.New("both -verify <file.dat> and -ip <address> are required for verification")
	}

	fmt.Printf("Verifying IP %s in binary file: %s ...\n", ipStr, datPath)
	rec, elapsed, err := SearchIPInFile(datPath, ipStr)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			fmt.Printf("Result: IP %s NOT FOUND in %s (Lookup took %s)\n", ipStr, datPath, elapsed)
			return err
		}
		return fmt.Errorf("verification error: %w", err)
	}

	fmt.Printf("Result: MATCH FOUND!\n")
	fmt.Printf("  - Country   : %s\n", rec.Country())
	fmt.Printf("  - IP Range  : %s - %s\n", rec.FirstIP().String(), rec.LastIP().String())
	fmt.Printf("  - Range (u32): %d - %d\n", rec.FirstNum, rec.LastNum)
	fmt.Printf("  - Lookup Time: %s\n", elapsed)
	return nil
}

func runLegacyCSV(inputPath, csvOutPath string) error {
	if csvOutPath == "" {
		csvOutPath = "dbip-country-lite.csv"
	}
	fmt.Printf("[1/2] Reading and decoding KRNIC CSV from: %s\n", inputPath)
	fi, err := os.Open(inputPath)
	if err != nil {
		return fmt.Errorf("cannot open input file: %w", err)
	}
	defer fi.Close()

	records, err := ParseKRNIC(transform.NewReader(fi, korean.EUCKR.NewDecoder()))
	if err != nil {
		return fmt.Errorf("failed to parse KRNIC CSV: %w", err)
	}
	fmt.Printf("      - Parsed raw records: %d\n", len(records))

	fmt.Printf("[2/2] Sorting, merging contiguous ranges and writing CSV to: %s\n", csvOutPath)
	merged := SortAndMerge(records)
	fmt.Printf("      - Merged total records: %d\n", len(merged))

	if err := WriteCSV(csvOutPath, merged); err != nil {
		return fmt.Errorf("failed to write CSV: %w", err)
	}

	fiStat, _ := os.Stat(csvOutPath)
	fmt.Printf("Successfully created %s (%d bytes, %d records)\n", csvOutPath, fiStat.Size(), len(merged))
	return nil
}

func printUsage() {
	fmt.Println("Usage of geoip_krnic2dbip:")
	fmt.Println("  1. Legacy CSV mode (default when run without flags or with -csv):")
	fmt.Println("     ./geoip_krnic2dbip                             # Converts ipv4.csv -> dbip-country-lite.csv")
	fmt.Println("     ./geoip_krnic2dbip -csv custom-lite.csv        # Converts to custom CSV file")
	fmt.Println()
	fmt.Println("  2. One-pass build for binary targets + CSV (recommended):")
	fmt.Println("     ./geoip_krnic2dbip -all [-country JP,KR,CN] [-in ipv4.csv] [-csv dbip-country-lite.csv]")
	fmt.Println("     Outputs: global.dat, <country>.dat files, checksum.sha256, dbip-country-lite.csv")
	fmt.Println()
	fmt.Println("  3. Build binary for specific country or multiple countries (comma-separated):")
	fmt.Println("     ./geoip_krnic2dbip -country JP,KR,CN [-in ipv4.csv]")
	fmt.Println("     ./geoip_krnic2dbip -country US -out us.dat [-in ipv4.csv]")
	fmt.Println()
	fmt.Println("  4. Verification mode (binary search query):")
	fmt.Println("     ./geoip_krnic2dbip -verify jp.dat -ip 182.22.59.229")
	fmt.Println()
	fmt.Println("Flags:")
	flag.PrintDefaults()
}

func main() {
	var (
		flagAll       = flag.Bool("all", false, "Build global.dat + specified countries (default: JP,KR,CN) + dbip-country-lite.csv + checksum.sha256")
		flagCountry   = flag.String("country", "", "Filter and build for specific country code(s), comma-separated (e.g. JP,KR,CN or US)")
		flagCountries = flag.String("countries", "", "Alias for -country")
		flagOut       = flag.String("out", "", "Output path when building a single country (e.g. jp.dat)")
		flagIn        = flag.String("in", "ipv4.csv", "Input KRNIC CSV file path (EUC-KR)")
		flagCsv       = flag.String("csv", "", "Output path for DB-IP compatible CSV (default: dbip-country-lite.csv)")
		flagVerify    = flag.String("verify", "", "Path to .dat file for binary search verification")
		flagIP        = flag.String("ip", "", "Target IP address to verify in binary search mode")
		flagHelp      = flag.Bool("help", false, "Show help message")
	)

	flag.Usage = printUsage
	flag.Parse()

	if *flagHelp {
		printUsage()
		os.Exit(0)
	}

	// [레거시 호환 모드] 인자 없이 실행 시: 기존처럼 ipv4.csv -> dbip-country-lite.csv 생성
	if flag.NFlag() == 0 {
		if _, err := os.Stat(*flagIn); err == nil {
			fmt.Printf("인자 없이 실행되어 기존 레거시 모드로 동작합니다 (%s -> dbip-country-lite.csv)\n", *flagIn)
			if err := runLegacyCSV(*flagIn, "dbip-country-lite.csv"); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			return
		}
		printUsage()
		os.Exit(0)
	}

	// 검증 모드
	if *flagVerify != "" {
		if *flagIP == "" {
			fmt.Fprintln(os.Stderr, "Error: -ip <ip_address> is required with -verify")
			os.Exit(1)
		}
		if err := runVerify(*flagVerify, *flagIP); err != nil {
			if errors.Is(err, ErrNotFound) {
				os.Exit(2)
			}
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	countryInput := *flagCountry
	if countryInput == "" {
		countryInput = *flagCountries
	}

	// 원패스 전체 빌드 모드 (-all)
	if *flagAll {
		if countryInput == "" {
			countryInput = "JP,KR,CN" // Default targets when -all is specified
		}
		targetCountries, err := ParseCountryList(countryInput)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing country list: %v\n", err)
			os.Exit(1)
		}

		// -all 실행 시 DB-IP CSV 파일도 기본으로 함께 생성
		csvOut := *flagCsv
		if csvOut == "" {
			csvOut = "dbip-country-lite.csv"
		}

		if err := runBuild(*flagIn, targetCountries, true, *flagOut, csvOut); err != nil {
			fmt.Fprintf(os.Stderr, "Build error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// 특정 국가 바이너리 빌드 모드
	if countryInput != "" {
		targetCountries, err := ParseCountryList(countryInput)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing country list: %v\n", err)
			os.Exit(1)
		}

		if len(targetCountries) > 1 && *flagOut != "" {
			fmt.Fprintln(os.Stderr, "Warning: -out is ignored when multiple countries are specified. Files will be named <country>.dat.")
		}

		if err := runBuild(*flagIn, targetCountries, false, *flagOut, *flagCsv); err != nil {
			fmt.Fprintf(os.Stderr, "Build error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// 단독 CSV 변환 모드 (-csv 지정 시)
	if *flagCsv != "" {
		if err := runLegacyCSV(*flagIn, *flagCsv); err != nil {
			fmt.Fprintf(os.Stderr, "CSV conversion error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	printUsage()
}
