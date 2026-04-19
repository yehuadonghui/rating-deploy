package csv

import (
	"bytes"
	"encoding/csv"
	"io"
	"strconv"
	"strings"
	"unicode/utf8"

	"golang.org/x/text/encoding/simplifiedchinese"
	"rating-system/internal/models"
)

// HydroParser parses Hydro OJ CSV exports.
type HydroParser struct{}

func init() {
	Register(&HydroParser{})
}

func (p *HydroParser) Platform() string {
	return "hydro"
}

// Parse reads Hydro OJ CSV export.
// CSV format: #,用户,电子邮件,学校,名称,学号,"解决\n总耗时",#1 题目名,#1 罚时（分钟）,...
func (p *HydroParser) Parse(reader io.Reader) ([]ContestEntry, error) {
	// Read all to handle BOM and encoding.
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	// Remove UTF-8 BOM if present.
	data = bytes.TrimPrefix(data, []byte{0xEF, 0xBB, 0xBF})

	// Decode GBK/GB18030 if data is not valid UTF-8.
	if !utf8.Valid(data) {
		if decoded, err := simplifiedchinese.GB18030.NewDecoder().Bytes(data); err == nil {
			data = decoded
		}
	} else if !looksLikeHydroHeader(data) {
		// UTF-8 is valid but likely mis-decoded GBK. Try GB18030 and verify header.
		if decoded, err := simplifiedchinese.GB18030.NewDecoder().Bytes(data); err == nil {
			if looksLikeHydroHeader(decoded) {
				data = decoded
			}
		}
	}

	csvReader := csv.NewReader(bytes.NewReader(data))
	csvReader.FieldsPerRecord = -1 // allow variable fields per row
	csvReader.LazyQuotes = true

	records, err := csvReader.ReadAll()
	if err != nil {
		return nil, err
	}

	if len(records) < 2 {
		return nil, nil // header only or empty
	}

	var entries []ContestEntry

	// Skip header row.
	for _, record := range records[1:] {
		if len(record) < 7 {
			continue
		}

		rank, err := strconv.Atoi(strings.TrimSpace(record[0]))
		if err != nil {
			continue
		}

		// Student ID: use 学号; fallback to 用户 if empty.
		studentID := normalizeID(record[5])
		if studentID == "" {
			studentID = normalizeID(record[1])
			if studentID == "" {
				continue
			}
		}

		// "解决\n总耗时" like "10\n17:40"
		solvedTimeStr := strings.TrimSpace(record[6])
		parts := strings.Split(solvedTimeStr, "\n")
		if len(parts) < 1 {
			continue
		}

		solved, err := strconv.Atoi(strings.TrimSpace(parts[0]))
		if err != nil {
			solved = 0
		}

		totalTime := ""
		if len(parts) > 1 {
			totalTime = strings.TrimSpace(parts[1])
		}

		entry := ContestEntry{
			Rank:      rank,
			Email:     strings.TrimSpace(record[2]),
			Class:     strings.TrimSpace(record[3]),
			Name:      strings.TrimSpace(record[4]),
			StudentID: studentID,
			Grade:     models.ParseGrade(studentID),
			Solved:    solved,
			TotalTime: totalTime,
		}
		entries = append(entries, entry)
	}

	return entries, nil
}

func looksLikeHydroHeader(data []byte) bool {
	line := string(data)
	if idx := strings.Index(line, "\n"); idx >= 0 {
		line = line[:idx]
	}
	line = strings.TrimSpace(strings.TrimRight(line, "\r"))
	return strings.Contains(line, "用户") &&
		strings.Contains(line, "电子邮件") &&
		strings.Contains(line, "学号")
}

func normalizeID(value string) string {
	s := strings.TrimSpace(value)
	if strings.HasPrefix(s, "=\"") && strings.HasSuffix(s, "\"") && len(s) >= 3 {
		s = s[2 : len(s)-1]
	}
	if strings.HasPrefix(s, "'") {
		s = strings.TrimPrefix(s, "'")
	}
	return strings.TrimSpace(s)
}
