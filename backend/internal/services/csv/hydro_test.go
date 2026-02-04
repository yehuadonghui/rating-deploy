package csv

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHydroParser_Parse(t *testing.T) {
	parser := &HydroParser{}

	t.Run("正常解析", func(t *testing.T) {
		csvData := `#,用户,电子邮件,学校,名称,学号,"解决
总耗时",#1 题目
1,user1,test@test.com,ACM组,张三,2508090301024,"10
17:40",100
2,user2,test2@test.com,ACM组,李四,2408090301025,"8
12:30",80`

		entries, err := parser.Parse(strings.NewReader(csvData))
		require.NoError(t, err)
		assert.Len(t, entries, 2)

		// 验证第一条记录
		assert.Equal(t, 1, entries[0].Rank)
		assert.Equal(t, "张三", entries[0].Name)
		assert.Equal(t, "2508090301024", entries[0].StudentID)
		assert.Equal(t, 25, entries[0].Grade)
		assert.Equal(t, 10, entries[0].Solved)
		assert.Equal(t, "17:40", entries[0].TotalTime)

		// 验证第二条记录
		assert.Equal(t, 2, entries[1].Rank)
		assert.Equal(t, "李四", entries[1].Name)
		assert.Equal(t, 24, entries[1].Grade)
		assert.Equal(t, 8, entries[1].Solved)
	})

	t.Run("过滤爆0记录", func(t *testing.T) {
		csvData := `#,用户,电子邮件,学校,名称,学号,"解决
总耗时",#1 题目
1,user1,test@test.com,ACM组,张三,2508090301024,"5
10:00",100
2,user2,test2@test.com,ACM组,李四,2408090301025,"0
0:00",0
3,user3,test3@test.com,ACM组,王五,2308090301026,"3
8:00",60`

		entries, err := parser.Parse(strings.NewReader(csvData))
		require.NoError(t, err)
		assert.Len(t, entries, 2) // 只有2条，爆0的被过滤

		assert.Equal(t, "张三", entries[0].Name)
		assert.Equal(t, "王五", entries[1].Name)
	})

	t.Run("空文件", func(t *testing.T) {
		csvData := `#,用户,电子邮件,学校,名称,学号,"解决
总耗时",#1 题目`

		entries, err := parser.Parse(strings.NewReader(csvData))
		require.NoError(t, err)
		assert.Len(t, entries, 0)
	})

	t.Run("学号带空格", func(t *testing.T) {
		csvData := `#,用户,电子邮件,学校,名称,学号,"解决
总耗时",#1 题目
1,user1,test@test.com,ACM组,张三, 2508090301024 ,"5
10:00",100`

		entries, err := parser.Parse(strings.NewReader(csvData))
		require.NoError(t, err)
		assert.Len(t, entries, 1)
		assert.Equal(t, "2508090301024", entries[0].StudentID)
		assert.Equal(t, 25, entries[0].Grade)
	})
}

func TestHydroParser_Platform(t *testing.T) {
	parser := &HydroParser{}
	assert.Equal(t, "hydro", parser.Platform())
}
