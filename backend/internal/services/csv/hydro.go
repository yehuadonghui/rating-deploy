package csv

import (
	"bytes"
	"encoding/csv"
	"io"
	"strconv"
	"strings"

	"rating-system/internal/models"
)

// HydroParser Hydro OJ CSV 解析器
type HydroParser struct{}

func init() {
	Register(&HydroParser{})
}

func (p *HydroParser) Platform() string {
	return "hydro"
}

// Parse 解析 Hydro OJ 导出的 CSV 文件
// CSV 格式：#,用户,电子邮件,学校,名称,学号,"解决\n总耗时",#1 题目名,#1 罚时（分钟）,...
func (p *HydroParser) Parse(reader io.Reader) ([]ContestEntry, error) {
	// 读取全部内容以处理 BOM
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	// 去除 UTF-8 BOM
	data = bytes.TrimPrefix(data, []byte{0xEF, 0xBB, 0xBF})

	csvReader := csv.NewReader(bytes.NewReader(data))
	csvReader.FieldsPerRecord = -1 // 允许不同行有不同数量的字段
	csvReader.LazyQuotes = true    // 宽松引号处理

	// 读取所有行
	records, err := csvReader.ReadAll()
	if err != nil {
		return nil, err
	}

	if len(records) < 2 {
		return nil, nil // 只有表头或空文件
	}

	var entries []ContestEntry

	// 跳过表头，从第二行开始
	for _, record := range records[1:] {
		if len(record) < 7 {
			continue
		}

		// 解析排名
		rank, err := strconv.Atoi(strings.TrimSpace(record[0]))
		if err != nil {
			continue
		}

		// 解析学号（去除空格）
		studentID := strings.TrimSpace(record[5])
		if studentID == "" {
			// 学号为空时，回退使用用户名作为唯一标识
			studentID = strings.TrimSpace(record[1])
			if studentID == "" {
				continue
			}
		}

		// 解析 "解决\n总耗时" 字段，格式如 "10\n17:40"
		solvedTimeStr := strings.TrimSpace(record[6])
		parts := strings.Split(solvedTimeStr, "\n")
		if len(parts) < 1 {
			continue
		}

		solved, err := strconv.Atoi(strings.TrimSpace(parts[0]))
		if err != nil {
			solved = 0
		}

		// 过滤爆0记录
		if solved == 0 {
			continue
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
