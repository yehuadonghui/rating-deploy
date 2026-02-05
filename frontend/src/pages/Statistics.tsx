import { Alert, Card, Col, Row } from "antd"
import { useEffect, useMemo, useState } from "react"
import {
  Bar,
  BarChart,
  CartesianGrid,
  Pie,
  PieChart,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts"
import { publicApi } from "../api/public"
import Loading from "../components/Loading"
import { useSiteConfigStore } from "../stores/siteConfig"

type GradeItem = {
  grade: number
  count: number
  avg_rating: number
}

type TierItem = {
  tier: string
  color: string
  count: number
}

type StatsResponse = {
  total_students: number
  total_contests: number
  grade_distribution: GradeItem[]
  tier_distribution: TierItem[]
}

export default function Statistics() {
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [stats, setStats] = useState<StatsResponse | null>(null)
  const { config } = useSiteConfigStore()

  useEffect(() => {
    publicApi
      .getStatistics()
      .then((res) => setStats(res.data))
      .catch(() => setError("获取统计数据失败"))
      .finally(() => setLoading(false))
  }, [])

  const gradeData = useMemo(() => {
    if (!stats) return []
    return stats.grade_distribution.map((item) => ({
      name: `20${item.grade}级`,
      count: item.count,
      avg: Math.round(item.avg_rating || 0),
    }))
  }, [stats])

  if (loading && !stats) {
    return <Loading />
  }

  if (error) {
    return <Alert type="error" message={error} />
  }

  if (!stats) {
    return <Alert type="info" message="暂无统计数据" />
  }

  return (
    <div className="page-card">
      <h2>统计</h2>
      <Row gutter={16} style={{ marginBottom: 16 }}>
        <Col span={12}>
          <Card title="学生总数">{stats.total_students}</Card>
        </Col>
        <Col span={12}>
          <Card title="比赛总数">{stats.total_contests}</Card>
        </Col>
      </Row>
      <Row gutter={16}>
        <Col span={14}>
          <Card title="年级分布">
            <ResponsiveContainer width="100%" height={320}>
              <BarChart data={gradeData}>
                <CartesianGrid strokeDasharray="3 3" />
                <XAxis dataKey="name" />
                <YAxis />
                <Tooltip />
                <Bar dataKey="count" fill={config.primary_color} />
              </BarChart>
            </ResponsiveContainer>
          </Card>
        </Col>
        <Col span={10}>
          <Card title="段位分布">
            <ResponsiveContainer width="100%" height={320}>
              <PieChart>
                <Pie
                  data={stats.tier_distribution}
                  dataKey="count"
                  nameKey="tier"
                  outerRadius={100}
                  label
                />
                <Tooltip />
              </PieChart>
            </ResponsiveContainer>
          </Card>
        </Col>
      </Row>
    </div>
  )
}
