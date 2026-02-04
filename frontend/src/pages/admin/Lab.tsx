import { Alert, Button, Card, Form, InputNumber, Progress, Space, Table, Tag, message } from "antd"
import { useEffect, useMemo, useRef, useState } from "react"
import { adminApi } from "../../api/admin"
import Loading from "../../components/Loading"

type ReplayPreviewItem = {
  student_id: string
  name: string
  class: string
  current_rating: number
  max_rating: number
  match_count: number
}

type RatingConfigValues = {
  k_factor: number
  growth_inertia: number
  history_decay: number
  initial_rating: number
  top_bonus_threshold: number
  top_bonus_multiplier: number
}

const defaultConfig: RatingConfigValues = {
  k_factor: 48,
  growth_inertia: 0.8,
  history_decay: 0.95,
  initial_rating: 0,
  top_bonus_threshold: 0.1,
  top_bonus_multiplier: 0.5,
}

export default function AdminLab() {
  const [form] = Form.useForm()
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [running, setRunning] = useState(false)
  const [preview, setPreview] = useState<ReplayPreviewItem[]>([])
  const [progress, setProgress] = useState<{ current: number; total: number; name: string; status: string }>({
    current: 0,
    total: 0,
    name: "",
    status: "idle",
  })
  const pollRef = useRef<number | null>(null)

  const loadConfig = async () => {
    setLoading(true)
    setError(null)
    try {
      const [configRes, statusRes] = await Promise.all([adminApi.getConfig(), adminApi.getReplayStatus()])
      const raw = configRes.data || {}
      const values: RatingConfigValues = { ...defaultConfig }
      Object.keys(defaultConfig).forEach((key) => {
        const rawValue = raw[key]
        const parsed = typeof rawValue === "string" ? Number(rawValue) : Number(rawValue)
        if (!Number.isNaN(parsed)) {
          values[key as keyof RatingConfigValues] = parsed
        }
      })
      form.setFieldsValue(values)
      setRunning(!!statusRes.data?.running)
      const rawProgress = statusRes.data?.progress || {}
      setProgress({
        current: Number(rawProgress.current_contest || 0),
        total: Number(rawProgress.total_contests || 0),
        name: String(rawProgress.contest_name || ""),
        status: String(rawProgress.status || "idle"),
      })
    } catch {
      setError("加载配置失败")
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    loadConfig()
  }, [])

  useEffect(() => {
    if (!running) {
      if (pollRef.current) {
        window.clearInterval(pollRef.current)
        pollRef.current = null
      }
      return
    }

    if (!pollRef.current) {
      pollRef.current = window.setInterval(async () => {
        try {
          const { data } = await adminApi.getReplayStatus()
          setRunning(!!data?.running)
          const rawProgress = data?.progress || {}
          setProgress({
            current: Number(rawProgress.current_contest || 0),
            total: Number(rawProgress.total_contests || 0),
            name: String(rawProgress.contest_name || ""),
            status: String(rawProgress.status || "idle"),
          })
        } catch {
          // ignore polling errors
        }
      }, 2000)
    }

    return () => {
      if (pollRef.current) {
        window.clearInterval(pollRef.current)
        pollRef.current = null
      }
    }
  }, [running])

  const previewColumns = useMemo(
    () => [
      { title: "#", render: (_: unknown, __: ReplayPreviewItem, index: number) => index + 1, width: 60 },
      { title: "姓名", dataIndex: "name" },
      { title: "学号", dataIndex: "student_id" },
      { title: "班级", dataIndex: "class", render: (value: string) => value || "-" },
      { title: "Rating", dataIndex: "current_rating", render: (value: number) => Math.round(value) },
      { title: "参赛次数", dataIndex: "match_count" },
    ],
    []
  )

  if (loading) {
    return <Loading />
  }

  if (error) {
    return <Alert type="error" message={error} />
  }

  return (
    <div>
      <h2>算法实验室</h2>
      <Card title="Rating 参数" style={{ marginBottom: 16 }}>
        <Form form={form} layout="vertical">
          <Form.Item label="K Factor" name="k_factor" rules={[{ required: true, message: "请输入 K Factor" }]}>
            <InputNumber min={0} step={1} style={{ width: "100%" }} />
          </Form.Item>
          <Form.Item
            label="成长阻尼"
            name="growth_inertia"
            rules={[{ required: true, message: "请输入成长阻尼" }]}
          >
            <InputNumber min={0} step={0.01} style={{ width: "100%" }} />
          </Form.Item>
          <Form.Item
            label="历史衰减"
            name="history_decay"
            rules={[{ required: true, message: "请输入历史衰减" }]}
          >
            <InputNumber min={0} step={0.01} style={{ width: "100%" }} />
          </Form.Item>
          <Form.Item
            label="初始 Rating"
            name="initial_rating"
            rules={[{ required: true, message: "请输入初始 Rating" }]}
          >
            <InputNumber min={0} step={10} style={{ width: "100%" }} />
          </Form.Item>
          <Form.Item
            label="Top 奖励阈值"
            name="top_bonus_threshold"
            rules={[{ required: true, message: "请输入 Top 奖励阈值" }]}
          >
            <InputNumber min={0} max={1} step={0.01} style={{ width: "100%" }} />
          </Form.Item>
          <Form.Item
            label="Top 奖励倍率"
            name="top_bonus_multiplier"
            rules={[{ required: true, message: "请输入 Top 奖励倍率" }]}
          >
            <InputNumber min={0} step={0.1} style={{ width: "100%" }} />
          </Form.Item>
        </Form>
        <Space>
          <Button
            type="primary"
            onClick={async () => {
              try {
                const values = await form.validateFields()
                await adminApi.updateConfig(values)
                message.success("配置已保存")
              } catch {
                message.error("保存失败")
              }
            }}
          >
            保存配置
          </Button>
          <Button
            onClick={async () => {
              try {
                const values = await form.validateFields()
                const { data } = await adminApi.previewReplay(values)
                setPreview((data?.preview || []).slice(0, 20))
                message.success("预览完成")
              } catch {
                message.error("预览失败")
              }
            }}
          >
            预览重算
          </Button>
          <Button
            danger
            onClick={async () => {
              try {
                await adminApi.triggerReplay()
                setRunning(true)
                message.success("已触发重算")
              } catch {
                message.error("触发失败")
              }
            }}
          >
            执行重算
          </Button>
          <Button
            onClick={async () => {
              try {
                const { data } = await adminApi.getReplayStatus()
                setRunning(!!data?.running)
                const rawProgress = data?.progress || {}
                setProgress({
                  current: Number(rawProgress.current_contest || 0),
                  total: Number(rawProgress.total_contests || 0),
                  name: String(rawProgress.contest_name || ""),
                  status: String(rawProgress.status || "idle"),
                })
              } catch {
                message.error("获取状态失败")
              }
            }}
          >
            刷新状态
          </Button>
          <Tag color={running ? "processing" : "default"}>{running ? "重算进行中" : "重算空闲"}</Tag>
        </Space>
        <div style={{ marginTop: 12 }}>
          <Progress
            percent={progress.total > 0 ? Math.round((progress.current / progress.total) * 100) : 0}
            status={running ? "active" : progress.status === "completed" ? "success" : "normal"}
          />
          <div style={{ marginTop: 4, color: "#666" }}>
            {progress.total > 0
              ? `当前进度：${progress.current}/${progress.total} ${progress.name ? `（${progress.name}）` : ""}`
              : "暂无进度信息"}
          </div>
        </div>
      </Card>

      <Card title="预览排行（前 20 名）">
        <Table
          rowKey="student_id"
          columns={previewColumns}
          dataSource={preview}
          pagination={false}
          locale={{ emptyText: "暂无预览数据" }}
        />
      </Card>
    </div>
  )
}
