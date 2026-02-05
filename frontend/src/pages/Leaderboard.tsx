import { Alert, Input, Select, Table } from "antd"
import { useEffect, useMemo, useState } from "react"
import { Link, useSearchParams } from "react-router-dom"
import { publicApi } from "../api/public"
import Loading from "../components/Loading"
import type { Student } from "../types"

type LeaderboardResponse = {
  total: number
  page: number
  size: number
  data: Student[]
}

export default function Leaderboard() {
  const [searchParams, setSearchParams] = useSearchParams()
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [data, setData] = useState<Student[]>([])
  const [total, setTotal] = useState(0)

  const page = Number(searchParams.get("page") || "1")
  const size = Number(searchParams.get("size") || "20")
  const gradeParam = searchParams.get("grade")
  const grade = gradeParam ? Number(gradeParam) : undefined
  const queryParam = (searchParams.get("query") || "").trim()

  useEffect(() => {
    setLoading(true)
    setError(null)
    const request = queryParam
      ? publicApi.searchStudents({ query: queryParam, page, size })
      : publicApi.getLeaderboard({ grade, page, size })

    request
      .then((res) => {
        const payload = res.data as LeaderboardResponse
        setData(payload.data || [])
        setTotal(payload.total || 0)
      })
      .catch(() => setError("获取排行榜失败"))
      .finally(() => setLoading(false))
  }, [grade, page, size, queryParam])

  const gradeOptions = useMemo(() => {
    const year = new Date().getFullYear() % 100
    return [
      { value: "", label: "全部年级" },
      ...Array.from({ length: 6 }, (_, idx) => {
        const value = String(year - idx)
        return { value, label: `20${value}级` }
      }),
    ]
  }, [])

  if (loading && data.length === 0) {
    return <Loading />
  }

  if (error) {
    return <Alert type="error" message={error} />
  }

  return (
    <div className="page-card">
      <h2>排行榜</h2>
      <div style={{ marginBottom: 16 }}>
        <div style={{ display: "flex", gap: 12, flexWrap: "wrap" }}>
          <Select
            style={{ width: 140 }}
            value={gradeParam || ""}
            options={gradeOptions}
            onChange={(value) => {
              const params = new URLSearchParams(searchParams)
              if (value) {
                params.set("grade", value)
              } else {
                params.delete("grade")
              }
              params.set("page", "1")
              setSearchParams(params)
            }}
          />
          <Input.Search
            allowClear
            placeholder="????/??/??/??"
            defaultValue={queryParam}
            onSearch={(value) => {
              const params = new URLSearchParams(searchParams)
              const trimmed = value.trim()
              if (trimmed) {
                params.set("query", trimmed)
                params.delete("grade")
              } else {
                params.delete("query")
              }
              params.set("page", "1")
              setSearchParams(params)
            }}
            onChange={(e) => {
              if (e.target.value === "") {
                const params = new URLSearchParams(searchParams)
                params.delete("query")
                params.set("page", "1")
                setSearchParams(params)
              }
            }}
            style={{ width: 260 }}
          />
        </div>
      </div>
      <Table
        rowKey="student_id"
        dataSource={data}
        loading={loading}
        pagination={{
          current: page,
          pageSize: size,
          total,
          showSizeChanger: true,
          onChange: (nextPage, nextSize) => {
            const params = new URLSearchParams(searchParams)
            params.set("page", String(nextPage))
            params.set("size", String(nextSize))
            setSearchParams(params)
          },
        }}
        columns={[
          {
            title: "#",
            dataIndex: "rank",
            width: 60,
            render: (_value, _record, index) => (page - 1) * size + index + 1,
          },
          {
            title: "姓名",
            dataIndex: "name",
            render: (value, record) => <Link to={`/students/${record.student_id}`}>{value}</Link>,
          },
          { title: "学号", dataIndex: "student_id" },
          { title: "班级", dataIndex: "class", render: (value) => value || "-" },
          { title: "年级", dataIndex: "grade", render: (value) => (value ? `20${value}级` : "-") },
          { title: "Rating", dataIndex: "current_rating", render: (value) => Math.round(value) },
          { title: "参赛次数", dataIndex: "match_count" },
        ]}
      />
    </div>
  )
}
