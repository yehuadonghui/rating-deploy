package csv

import (
	"io"
)

// ContestEntry CSV 解析后的单条记录
type ContestEntry struct {
	Rank      int    `json:"rank"`
	Email     string `json:"email"`
	Class     string `json:"class"`
	Name      string `json:"name"`
	StudentID string `json:"student_id"`
	Grade     int    `json:"grade"`
	Solved    int    `json:"solved"`
	TotalTime string `json:"total_time"`
}

// CSVParser CSV 解析器接口
type CSVParser interface {
	// Parse 解析 CSV 文件，返回比赛条目
	Parse(reader io.Reader) ([]ContestEntry, error)

	// Platform 支持的平台名称
	Platform() string
}

// 解析器注册表
var parsers = make(map[string]CSVParser)

// Register 注册解析器
func Register(p CSVParser) {
	parsers[p.Platform()] = p
}

// GetParser 获取解析器
func GetParser(platform string) CSVParser {
	return parsers[platform]
}

// ListParsers 列出所有已注册的解析器
func ListParsers() []string {
	var names []string
	for name := range parsers {
		names = append(names, name)
	}
	return names
}
