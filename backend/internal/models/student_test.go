package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseGrade(t *testing.T) {
	tests := []struct {
		name      string
		studentID string
		expected  int
	}{
		{"正常学号25级", "2508090301024", 25},
		{"正常学号24级", "2408090301024", 24},
		{"正常学号23级", "2308090301024", 23},
		{"带空格学号", " 2508090301024 ", 25},
		{"空学号", "", 0},
		{"单字符学号", "2", 0},
		{"非数字开头", "AB08090301024", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseGrade(tt.studentID)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestStudent_UpdateRating(t *testing.T) {
	tests := []struct {
		name              string
		currentRating     float64
		maxRating         float64
		newRating         float64
		expectedCurrent   float64
		expectedMax       float64
	}{
		{"Rating 上升", 100, 100, 150, 150, 150},
		{"Rating 不变（新值更低）", 100, 100, 80, 100, 100},
		{"Rating 上升但未超过历史最高", 100, 200, 150, 150, 200},
		{"Rating 上升且超过历史最高", 100, 100, 200, 200, 200},
		{"从0开始", 0, 0, 50, 50, 50},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Student{
				CurrentRating: tt.currentRating,
				MaxRating:     tt.maxRating,
			}
			s.UpdateRating(tt.newRating)
			assert.Equal(t, tt.expectedCurrent, s.CurrentRating)
			assert.Equal(t, tt.expectedMax, s.MaxRating)
		})
	}
}
