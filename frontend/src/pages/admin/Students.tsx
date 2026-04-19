import { Button, Card, Form, Input, InputNumber, Modal, Select, Space, Table, message } from "antd"
import { useEffect, useState } from "react"
import { adminApi } from "../../api/admin"
import type { PointCategory, Student } from "../../types"

type StudentResponse = {
  total: number
  page: number
  size: number
  data: Student[]
}

type ManualPointValues = {
  category: PointCategory
  points: number
  title: string
  description?: string
}

const manualPointOptions: Array<{ value: PointCategory; label: string }> = [
  { value: "progress_award", label: "进步奖" },
  { value: "organizer_reward", label: "组织工作积分" },
]

export default function AdminStudents() {
  const [loading, setLoading] = useState(false)
  const [data, setData] = useState<Student[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [size, setSize] = useState(20)
  const [query, setQuery] = useState("")
  const [modalOpen, setModalOpen] = useState(false)
  const [editing, setEditing] = useState<Student | null>(null)
  const [pointModalOpen, setPointModalOpen] = useState(false)
  const [pointTarget, setPointTarget] = useState<Student | null>(null)
  const [form] = Form.useForm()
  const [pointForm] = Form.useForm()
  const [selectedRowKeys, setSelectedRowKeys] = useState<string[]>([])

  const load = () => {
    setLoading(true)
    adminApi
      .listStudents({ query: query || undefined, page, size })
      .then((res) => {
        const payload = res.data as StudentResponse
        setData(payload.data || [])
        setTotal(payload.total || 0)
      })
      .catch(() => message.error("获取学生列表失败"))
      .finally(() => setLoading(false))
  }

  useEffect(() => {
    load()
  }, [page, size, query])

  const openCreate = () => {
    setEditing(null)
    form.resetFields()
    setModalOpen(true)
  }

  const openEdit = (student: Student) => {
    setEditing(student)
    form.setFieldsValue({
      student_id: student.student_id,
      name: student.name,
      email: student.email,
      class: student.class,
      grade: student.grade,
    })
    setModalOpen(true)
  }

  const openPointModal = (student: Student) => {
    setPointTarget(student)
    pointForm.resetFields()
    pointForm.setFieldsValue({ category: "progress_award" })
    setPointModalOpen(true)
  }

  const handleBatchDelete = () => {
    if (selectedRowKeys.length === 0) return
    Modal.confirm({
      title: `确认删除选中的 ${selectedRowKeys.length} 名学生吗？`,
      content: "将同时删除这些学生的比赛结果和兑奖积分记录。",
      okText: "删除",
      okButtonProps: { danger: true },
      onOk: async () => {
        const key = "batch-delete"
        message.loading({ content: "正在删除...", key })
        const results = await Promise.allSettled(selectedRowKeys.map((id) => adminApi.deleteStudent(id)))
        const failed = results.filter((r) => r.status === "rejected").length
        if (failed === 0) {
          message.success({ content: "删除成功", key })
        } else {
          message.warning({ content: `删除完成，失败 ${failed} 个`, key })
        }
        setSelectedRowKeys([])
        load()
      },
    })
  }

  const handleDeleteAll = () => {
    Modal.confirm({
      title: "确认删除全部学生吗？",
      content: "该操作不可恢复，将同时删除全部比赛结果和兑奖积分记录。",
      okText: "删除全部",
      okButtonProps: { danger: true },
      onOk: async () => {
        const key = "delete-all"
        message.loading({ content: "正在删除全部学生...", key })
        try {
          await adminApi.deleteAllStudents()
          message.success({ content: "已删除全部学生", key })
          setSelectedRowKeys([])
          setPage(1)
          load()
        } catch {
          message.error({ content: "删除失败", key })
        }
      },
    })
  }

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields()
      if (editing) {
        await adminApi.updateStudent(editing.student_id, {
          name: values.name,
          email: values.email,
          class: values.class,
          grade: values.grade,
        })
        message.success("学生已更新")
      } else {
        await adminApi.createStudent(values)
        message.success("学生已创建")
      }
      setModalOpen(false)
      load()
    } catch {
      message.error("保存失败")
    }
  }

  const handleCreatePoint = async () => {
    try {
      const values = (await pointForm.validateFields()) as ManualPointValues
      if (!pointTarget) return
      await adminApi.createStudentPointRecord(pointTarget.student_id, values)
      message.success("积分录入成功")
      setPointModalOpen(false)
      load()
    } catch {
      message.error("积分录入失败")
    }
  }

  return (
    <div>
      <h2>学生管理</h2>
      <Card style={{ marginBottom: 16 }}>
        <Space wrap>
          <Input.Search
            placeholder="搜索学号/姓名/班级/邮箱"
            allowClear
            onSearch={(value) => {
              setPage(1)
              setQuery(value.trim())
            }}
            onChange={(e) => {
              if (e.target.value === "") {
                setPage(1)
                setQuery("")
              }
            }}
            style={{ width: 260 }}
          />
          <Button type="primary" onClick={openCreate}>
            添加学生
          </Button>
          <Button danger disabled={selectedRowKeys.length === 0} onClick={handleBatchDelete}>
            删除选中{selectedRowKeys.length ? `（${selectedRowKeys.length}）` : ""}
          </Button>
          <Button danger disabled={total === 0} onClick={handleDeleteAll}>
            一键全删
          </Button>
        </Space>
      </Card>

      <Table
        rowKey="student_id"
        dataSource={data}
        loading={loading}
        rowSelection={{
          selectedRowKeys,
          onChange: (keys) => setSelectedRowKeys(keys as string[]),
        }}
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
          { title: "学号", dataIndex: "student_id" },
          { title: "姓名", dataIndex: "name" },
          { title: "班级", dataIndex: "class", render: (value) => value || "-" },
          { title: "年级", dataIndex: "grade", render: (value) => (value ? `20${value}级` : "-") },
          { title: "邮箱", dataIndex: "email", render: (value) => value || "-" },
          { title: "Rating", dataIndex: "current_rating", render: (value) => Math.round(value || 0) },
          { title: "兑奖积分", dataIndex: "redeem_points", render: (value) => value || 0 },
          { title: "参赛次数", dataIndex: "match_count" },
          {
            title: "操作",
            render: (_value, record) => (
              <Space>
                <Button size="small" onClick={() => openEdit(record)}>
                  编辑
                </Button>
                <Button size="small" onClick={() => openPointModal(record)}>
                  录入积分
                </Button>
                <Button
                  size="small"
                  danger
                  onClick={async () => {
                    try {
                      await adminApi.deleteStudent(record.student_id)
                      message.success("删除成功")
                      load()
                    } catch {
                      message.error("删除失败")
                    }
                  }}
                >
                  删除
                </Button>
              </Space>
            ),
          },
        ]}
      />

      <Modal
        open={modalOpen}
        title={editing ? "编辑学生" : "新增学生"}
        onCancel={() => setModalOpen(false)}
        onOk={handleSubmit}
        okText="保存"
      >
        <Form form={form} layout="vertical">
          <Form.Item label="学号" name="student_id" rules={[{ required: true, message: "请输入学号" }]}>
            <Input disabled={!!editing} />
          </Form.Item>
          <Form.Item label="姓名" name="name" rules={[{ required: true, message: "请输入姓名" }]}>
            <Input />
          </Form.Item>
          <Form.Item label="班级" name="class">
            <Input />
          </Form.Item>
          <Form.Item label="邮箱" name="email">
            <Input />
          </Form.Item>
          <Form.Item label="年级" name="grade">
            <InputNumber min={0} step={1} style={{ width: "100%" }} />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        open={pointModalOpen}
        title={pointTarget ? `录入积分 - ${pointTarget.name}` : "录入积分"}
        onCancel={() => setPointModalOpen(false)}
        onOk={handleCreatePoint}
        okText="保存"
      >
        <Form form={pointForm} layout="vertical">
          <Form.Item label="积分类型" name="category" rules={[{ required: true, message: "请选择积分类型" }]}>
            <Select options={manualPointOptions} />
          </Form.Item>
          <Form.Item label="积分分值" name="points" rules={[{ required: true, message: "请输入积分分值" }]}>
            <InputNumber min={1} step={1} style={{ width: "100%" }} />
          </Form.Item>
          <Form.Item label="标题" name="title" rules={[{ required: true, message: "请输入标题" }]}>
            <Input />
          </Form.Item>
          <Form.Item label="说明" name="description">
            <Input.TextArea rows={3} />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  )
}
