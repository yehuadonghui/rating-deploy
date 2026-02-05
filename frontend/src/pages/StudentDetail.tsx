import { Alert, Card, Descriptions, Table } from "antd"
import { useEffect, useMemo, useState } from "react"
import { useParams } from "react-router-dom"
import { Line, LineChart, ResponsiveContainer, Tooltip, XAxis, YAxis } from "recharts"
import { publicApi } from "../api/public"
import Loading from "../components/Loading"
import type { Result, Student } from "../types"

type StudentResponse = {
  student: Student
  tier_name: string
  tier_color: string
}

export default function StudentDetail() {
  const { id } = useParams()
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [student, setStudent] = useState<StudentResponse | null>(null)
  const [history, setHistory] = useState<Result[]>([])

  useEffect(() => {
    if (!id) return
    setLoading(true)
    Promise.all([publicApi.getStudent(id), publicApi.getStudentHistory(id)])
      .then(([studentRes, historyRes]) => {
        setStudent(studentRes.data)
        setHistory(historyRes.data || [])
      })
      .catch(() => setError("获取学生信息失败"))
      .finally(() => setLoading(false))
  }, [id])

  const chartData = useMemo(() => {
    return history.map((item) => ({
      name: item.contest?.name || `比赛 ${item.contest_id}`,
      rating: Math.round(item.rating_after || 0),
    }))
  }, [history])

  if (loading) {
    return <Loading />
  }

  if (error || !student) {
    return <Alert type="error" message={error || "未找到学生"} />
  }

  return (
    <div className="page-card">
      <h2>学生详情</h2>
      <Descriptions column={2} bordered style={{ marginBottom: 16 }}>
        <Descriptions.Item label="姓名">{student.student.name}</Descriptions.Item>
        <Descriptions.Item label="学号">{student.student.student_id}</Descriptions.Item>
        <Descriptions.Item label="班级">{student.student.class || "-"}</Descriptions.Item>
        <Descriptions.Item label="年级">
          {student.student.grade ? `20${student.student.grade}级` : "-"}
        </Descriptions.Item>
        <Descriptions.Item label="当前 Rating">
          {Math.round(student.student.current_rating)}
        </Descriptions.Item>
        <Descriptions.Item label="最高 Rating">
          {Math.round(student.student.max_rating)}
        </Descriptions.Item>
        <Descriptions.Item label="段位" span={2}>
          <span style={{ color: student.tier_color || "#333" }}>{student.tier_name || "-"}</span>
        </Descriptions.Item>
      </Descriptions>

      <Card title="Rating 曲线" style={{ marginBottom: 16 }}>
        {chartData.length === 0 ? (
          <div>暂无比赛记录</div>
        ) : (
          <ResponsiveContainer width="100%" height={260}>
            <LineChart data={chartData}>
              <XAxis dataKey="name" hide />
              <YAxis />
              <Tooltip />
              <Line type="monotone" dataKey="rating" stroke="#1677ff" />
            </LineChart>
          </ResponsiveContainer>
        )}
      </Card>

      <Card title="比赛记录">
        <Table
          rowKey="id"
          dataSource={history}
          pagination={false}
          columns={[
            { title: "比赛", dataIndex: ["contest", "name"], render: (value) => value || "-" },
            {
              title: "日期",
              dataIndex: ["contest", "date"],
              render: (value) => (value ? new Date(value).toLocaleDateString() : "-") ,
            },
            { title: "排名", dataIndex: "rank" },
            { title: "解题数", dataIndex: "solved" },
            {
              title: "Rating",
              render: (_, record) => `${Math.round(record.rating_before || 0)} → ${Math.round(record.rating_after || 0)}`,
            },
            { title: "变化", dataIndex: "delta", render: (value) => Math.round(value || 0) },
          ]}
        />
      </Card>
    </div>
  )
}
