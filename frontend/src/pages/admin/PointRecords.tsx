import { Card, Input, Select, Space, Table, Tag } from "antd"
import { useEffect, useState } from "react"
import { adminApi } from "../../api/admin"
import type { PointCategory, PointRecord, PointSource } from "../../types"

type PointRecordResponse = {
  total: number
  page: number
  size: number
  data: PointRecord[]
}

const categoryOptions: Array<{ value: PointCategory; label: string }> = [
  { value: "contest_award", label: "比赛奖励" },
  { value: "progress_award", label: "进步奖" },
  { value: "organizer_reward", label: "组织工作积分" },
]

const sourceOptions: Array<{ value: PointSource; label: string }> = [
  { value: "auto", label: "自动" },
  { value: "manual", label: "手工" },
]

const categoryLabel = (value?: PointCategory) =>
  categoryOptions.find((item) => item.value === value)?.label || value || "-"

const sourceLabel = (value?: PointSource) =>
  sourceOptions.find((item) => item.value === value)?.label || value || "-"

export default function AdminPointRecords() {
  const [loading, setLoading] = useState(false)
  const [data, setData] = useState<PointRecord[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [size, setSize] = useState(20)
  const [studentId, setStudentId] = useState("")
  const [contestId, setContestId] = useState("")
  const [category, setCategory] = useState<PointCategory | undefined>(undefined)
  const [source, setSource] = useState<PointSource | undefined>(undefined)

  const load = () => {
    setLoading(true)
    adminApi
      .listPointRecords({
        student_id: studentId || undefined,
        contest_id: contestId ? Number(contestId) : undefined,
        category,
        source,
        page,
        size,
      })
      .then((res) => {
        const payload = res.data as PointRecordResponse
        setData(payload.data || [])
        setTotal(payload.total || 0)
      })
      .finally(() => setLoading(false))
  }

  useEffect(() => {
    load()
  }, [page, size, studentId, contestId, category, source])

  return (
    <div>
      <h2>兑奖积分流水</h2>
      <Card style={{ marginBottom: 16 }}>
        <Space wrap>
          <Input
            placeholder="按学号筛选"
            value={studentId}
            onChange={(e) => {
              setPage(1)
              setStudentId(e.target.value.trim())
            }}
            style={{ width: 180 }}
          />
          <Input
            placeholder="按比赛 ID 筛选"
            value={contestId}
            onChange={(e) => {
              setPage(1)
              setContestId(e.target.value.trim())
            }}
            style={{ width: 180 }}
          />
          <Select
            allowClear
            placeholder="积分类型"
            value={category}
            onChange={(value) => {
              setPage(1)
              setCategory(value)
            }}
            options={categoryOptions}
            style={{ width: 180 }}
          />
          <Select
            allowClear
            placeholder="来源"
            value={source}
            onChange={(value) => {
              setPage(1)
              setSource(value)
            }}
            options={sourceOptions}
            style={{ width: 140 }}
          />
        </Space>
      </Card>

      <Table
        rowKey="id"
        dataSource={data}
        loading={loading}
        pagination={{
          current: page,
          pageSize: size,
          total,
          showSizeChanger: true,
          onChange: (nextPage, nextSize) => {
            setPage(nextPage)
            setSize(nextSize)
          },
        }}
        columns={[
          {
            title: "学生",
            render: (_, record) =>
              record.student?.name ? `${record.student.name} (${record.student_id})` : record.student_id,
          },
          { title: "分值", dataIndex: "points", render: (value) => <strong>+{value}</strong> },
          { title: "类型", dataIndex: "category", render: (value: PointCategory) => categoryLabel(value) },
          {
            title: "来源",
            dataIndex: "source",
            render: (value: PointSource) => (
              <Tag color={value === "auto" ? "blue" : "green"}>{sourceLabel(value)}</Tag>
            ),
          },
          { title: "标题", dataIndex: "title" },
          { title: "说明", dataIndex: "description", render: (value) => value || "-" },
          { title: "关联比赛", render: (_, record) => record.contest?.name || "-" },
          {
            title: "时间",
            dataIndex: "created_at",
            render: (value: string) => (value ? new Date(value).toLocaleString() : "-"),
          },
        ]}
      />
    </div>
  )
}
