import { Button, Input, InputNumber, Popconfirm, Space, Table, message } from "antd"
import { useEffect, useMemo, useState } from "react"
import { adminApi } from "../../api/admin"
import { publicApi } from "../../api/public"
import Loading from "../../components/Loading"
import type { TierConfig } from "../../types"

export default function AdminTiers() {
  const [tiers, setTiers] = useState<TierConfig[]>([])
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)

  const loadTiers = async () => {
    setLoading(true)
    try {
      const { data } = await publicApi.getTiers()
      setTiers(data || [])
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    loadTiers()
  }, [])

  const updateTier = (index: number, patch: Partial<TierConfig>) => {
    setTiers((prev) => prev.map((item, idx) => (idx === index ? { ...item, ...patch } : item)))
  }

  const columns = useMemo(
    () => [
      { title: "顺序", render: (_: unknown, __: TierConfig, index: number) => index + 1, width: 80 },
      {
        title: "名称",
        render: (_: unknown, record: TierConfig, index: number) => (
          <Input value={record.name} onChange={(e) => updateTier(index, { name: e.target.value })} />
        ),
      },
      {
        title: "最低 Rating",
        render: (_: unknown, record: TierConfig, index: number) => (
          <InputNumber
            value={record.min_rating}
            onChange={(value) => updateTier(index, { min_rating: Number(value) })}
          />
        ),
      },
      {
        title: "最高 Rating",
        render: (_: unknown, record: TierConfig, index: number) => (
          <InputNumber
            value={record.max_rating}
            onChange={(value) => updateTier(index, { max_rating: Number(value) })}
          />
        ),
      },
      {
        title: "颜色",
        render: (_: unknown, record: TierConfig, index: number) => (
          <Input
            type="color"
            value={record.color || "#000000"}
            onChange={(e) => updateTier(index, { color: e.target.value })}
            style={{ width: 80, padding: 0 }}
          />
        ),
      },
      {
        title: "操作",
        render: (_: unknown, record: TierConfig, index: number) => (
          <Space>
            <Popconfirm
              title="确定删除该段位吗？"
              onConfirm={() => setTiers((prev) => prev.filter((_item, idx) => idx !== index))}
            >
              <Button danger>删除</Button>
            </Popconfirm>
          </Space>
        ),
      },
    ],
    []
  )

  if (loading) {
    return <Loading />
  }

  return (
    <div>
      <h2>段位配置</h2>
      <Space style={{ marginBottom: 16 }}>
        <Button
          onClick={() =>
            setTiers((prev) => [
              ...prev,
              {
                name: "New Tier",
                min_rating: 0,
                max_rating: 100,
                color: "#999999",
                order: prev.length + 1,
              },
            ])
          }
        >
          新增段位
        </Button>
        <Button
          type="primary"
          loading={saving}
          onClick={async () => {
            setSaving(true)
            try {
              const payload = tiers.map((tier, index) => ({
                ...tier,
                order: index + 1,
              }))
              await adminApi.updateTiers(payload)
              message.success("段位配置已保存")
              loadTiers()
            } catch {
              message.error("保存失败")
            } finally {
              setSaving(false)
            }
          }}
        >
          保存配置
        </Button>
      </Space>
      <Table rowKey={(record) => `${record.id || "new"}-${record.name}`} columns={columns} dataSource={tiers} pagination={false} />
    </div>
  )
}
