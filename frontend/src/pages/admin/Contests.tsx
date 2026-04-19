import {
  Alert,
  Button,
  DatePicker,
  Form,
  Input,
  InputNumber,
  Modal,
  Popconfirm,
  Progress,
  Select,
  Space,
  Table,
  Tag,
  message,
} from "antd"
import dayjs from "dayjs"
import { useEffect, useMemo, useState } from "react"
import { adminApi } from "../../api/admin"
import { publicApi } from "../../api/public"
import Loading from "../../components/Loading"
import type { Contest, ContestRewardType } from "../../types"

type ContestFormValues = {
  name: string
  date?: dayjs.Dayjs
  weight?: number
  reward_type?: ContestRewardType
}

const rewardTypeOptions: Array<{ value: ContestRewardType; label: string; color?: string }> = [
  { value: "none", label: "不计兑奖积分" },
  { value: "foundation_exam", label: "强基月考", color: "blue" },
  { value: "brand_monthly", label: "品牌月赛", color: "green" },
  { value: "brand_final", label: "品牌决赛", color: "gold" },
]

function renderRewardType(value?: ContestRewardType) {
  const matched = rewardTypeOptions.find((item) => item.value === (value || "none"))
  if (!matched) return "-"
  return matched.color ? <Tag color={matched.color}>{matched.label}</Tag> : matched.label
}

export default function AdminContests() {
  const [contests, setContests] = useState<Contest[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [uploading, setUploading] = useState(false)
  const [uploadProgress, setUploadProgress] = useState<number | null>(null)
  const [uploadFile, setUploadFile] = useState<File | null>(null)
  const [fileInputKey, setFileInputKey] = useState(0)
  const [editTarget, setEditTarget] = useState<Contest | null>(null)
  const [editForm] = Form.useForm()
  const [uploadForm] = Form.useForm()

  const loadContests = async () => {
    setLoading(true)
    setError(null)
    try {
      const { data } = await publicApi.getContests()
      setContests(data || [])
    } catch {
      setError("获取比赛列表失败")
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    loadContests()
  }, [])

  const columns = useMemo(
    () => [
      { title: "名称", dataIndex: "name" },
      {
        title: "日期",
        dataIndex: "date",
        render: (value: string) => (value ? dayjs(value).format("YYYY-MM-DD") : "-"),
      },
      { title: "权重", dataIndex: "weight" },
      {
        title: "兑奖类型",
        dataIndex: "reward_type",
        render: (value: ContestRewardType) => renderRewardType(value),
      },
      { title: "参赛人数", dataIndex: "participant_count" },
      {
        title: "操作",
        render: (_: unknown, record: Contest) => (
          <Space>
            <Button
              onClick={() => {
                setEditTarget(record)
                editForm.setFieldsValue({
                  name: record.name,
                  weight: record.weight,
                  reward_type: record.reward_type || "none",
                })
              }}
            >
              编辑
            </Button>
            <Popconfirm
              title="确定删除该比赛吗？"
              description="删除后会同时移除该场自动兑奖积分，并需要重新计算 Rating。"
              onConfirm={async () => {
                try {
                  await adminApi.deleteContest(record.id)
                  message.success("比赛已删除")
                  loadContests()
                } catch {
                  message.error("删除失败")
                }
              }}
            >
              <Button danger>删除</Button>
            </Popconfirm>
          </Space>
        ),
      },
    ],
    [editForm]
  )

  if (loading && contests.length === 0) {
    return <Loading />
  }

  if (error) {
    return <Alert type="error" message={error} />
  }

  return (
    <div>
      <h2>比赛管理</h2>
      <Form
        form={uploadForm}
        layout="inline"
        onFinish={async (values: ContestFormValues) => {
          if (!uploadFile) {
            message.error("请先选择 CSV 文件")
            return
          }
          const formData = new FormData()
          formData.append("file", uploadFile)
          if (values.name) {
            formData.append("name", values.name)
          }
          if (values.date) {
            formData.append("date", values.date.format("YYYY-MM-DD"))
          }
          if (values.weight !== undefined) {
            formData.append("weight", String(values.weight))
          }
          formData.append("reward_type", values.reward_type || "none")

          setUploading(true)
          setUploadProgress(null)
          try {
            await adminApi.uploadContest(formData, (p) => {
              setUploadProgress(Math.round(p * 100))
            })
            message.success("比赛上传成功")
            uploadForm.resetFields()
            setUploadFile(null)
            setFileInputKey((prev) => prev + 1)
            loadContests()
          } catch {
            message.error("上传失败")
          } finally {
            setUploading(false)
          }
        }}
        style={{ marginBottom: 24, alignItems: "flex-end", flexWrap: "wrap", gap: 16 }}
      >
        <Form.Item label="比赛名称" name="name">
          <Input placeholder="默认使用文件名" style={{ width: 220 }} />
        </Form.Item>
        <Form.Item label="日期" name="date">
          <DatePicker />
        </Form.Item>
        <Form.Item label="权重" name="weight" initialValue={1}>
          <InputNumber min={0.1} step={0.1} />
        </Form.Item>
        <Form.Item label="兑奖类型" name="reward_type" initialValue="none">
          <Select style={{ width: 180 }} options={rewardTypeOptions} />
        </Form.Item>
        <Form.Item label="CSV 文件" required>
          <input
            key={fileInputKey}
            type="file"
            accept=".csv"
            onChange={(event) => {
              const file = event.target.files?.[0] || null
              setUploadFile(file)
            }}
          />
        </Form.Item>
        <Form.Item>
          <Button type="primary" htmlType="submit" loading={uploading}>
            上传比赛
          </Button>
        </Form.Item>
      </Form>

      {uploading && uploadProgress !== null ? (
        <div style={{ maxWidth: 420, marginBottom: 16 }}>
          <Progress percent={uploadProgress} />
        </div>
      ) : null}

      <Table rowKey="id" columns={columns} dataSource={contests} pagination={{ pageSize: 10 }} />

      <Modal
        title="编辑比赛"
        open={!!editTarget}
        onCancel={() => setEditTarget(null)}
        onOk={async () => {
          try {
            const values = await editForm.validateFields()
            if (!editTarget) return
            await adminApi.updateContest(editTarget.id, {
              name: values.name,
              weight: values.weight,
              reward_type: values.reward_type,
            })
            message.success("比赛更新成功")
            setEditTarget(null)
            loadContests()
          } catch {
            message.error("更新失败")
          }
        }}
      >
        <Form layout="vertical" form={editForm}>
          <Form.Item label="比赛名称" name="name" rules={[{ required: true, message: "请输入比赛名称" }]}>
            <Input />
          </Form.Item>
          <Form.Item label="权重" name="weight" rules={[{ required: true, message: "请输入权重" }]}>
            <InputNumber min={0.1} step={0.1} style={{ width: "100%" }} />
          </Form.Item>
          <Form.Item label="兑奖类型" name="reward_type" rules={[{ required: true, message: "请选择兑奖类型" }]}>
            <Select options={rewardTypeOptions} />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  )
}
