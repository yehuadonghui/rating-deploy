import { Alert, Card, Descriptions, Table } from "antd"
import { useEffect, useMemo, useState } from "react"
import { useParams } from "react-router-dom"
import { Line, LineChart, ResponsiveContainer, Tooltip, XAxis, YAxis } from "recharts"
import { publicApi } from "../api/public"
import Loading from "../components/Loading"
import type { PointCategory, PointRecord, Result, Student } from "../types"

type StudentResponse = {
  student: Student
  tier_name: string
  tier_color: string
}

type StudentPointsResponse = {
  student_id: string
  redeem_points: number
  point_records: PointRecord[]
}

const pointCategoryLabel = (value?: PointCategory) => {
  switch (value) {
    case "contest_award":
      return "比赛奖励"
    case "progress_award":
      return "进步奖"
    case "organizer_reward":
      return "组织工作积分"
    default:
      return "-"
  }
}

export default function StudentDetail() {
  const { id } = useParams()
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [student, setStudent] = useState<StudentResponse | null>(null)
  const [history, setHistory] = useState<Result[]>([])
  const [redeemPoints, setRedeemPoints] = useState(0)
  const [pointRecords, setPointRecords] = useState<PointRecord[]>([])

  useEffect(() => {
    if (!id) return
    setLoading(true)
    Promise.all([publicApi.getStudent(id), publicApi.getStudentHistory(id), publicApi.getStudentPoints(id)])
      .then(([studentRes, historyRes, pointsRes]) => {
        setStudent(studentRes.data)
        setHistory(historyRes.data || [])
        const pointPayload = pointsRes.data as StudentPointsResponse
        setRedeemPoints(Number(pointPayload.redeem_points || 0))
        setPointRecords(pointPayload.point_records || [])
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
        <Descriptions.Item label="兑奖积分">{redeemPoints}</Descriptions.Item>
        <Descriptions.Item label="参赛次数">{student.student.match_count}</Descriptions.Item>
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

      <Card title="比赛记录" style={{ marginBottom: 16 }}>
        <Table
          rowKey="id"
          dataSource={history}
          pagination={false}
          columns={[
            { title: "比赛", dataIndex: ["contest", "name"], render: (value) => value || "-" },
            {
              title: "日期",
              dataIndex: ["contest", "date"],
              render: (value) => (value ? new Date(value).toLocaleDateString() : "-"),
            },
            { title: "排名", dataIndex: "rank" },
            { title: "解题数", dataIndex: "solved" },
            {
              title: "Rating",
              render: (_, record) =>
                `${Math.round(record.rating_before || 0)} → ${Math.round(record.rating_after || 0)}`,
            },
            { title: "变化", dataIndex: "delta", render: (value) => Math.round(value || 0) },
          ]}
        />
      </Card>

      <Card title="兑奖积分明细">
        <Table
          rowKey="id"
          dataSource={pointRecords}
          pagination={false}
          locale={{ emptyText: "暂无兑奖积分记录" }}
          columns={[
            { title: "分值", dataIndex: "points", render: (value: number) => `+${value}` },
            { title: "标题", dataIndex: "title" },
            {
              title: "类型",
              dataIndex: "category",
              render: (value: PointCategory) => pointCategoryLabel(value),
            },
            { title: "说明", dataIndex: "description", render: (value: string) => value || "-" },
            { title: "关联比赛", render: (_, record: PointRecord) => record.contest?.name || "-" },
            {
              title: "时间",
              dataIndex: "created_at",
              render: (value: string) => (value ? new Date(value).toLocaleString() : "-"),
            },
          ]}
        />
      </Card>
    </div>
  )
}
