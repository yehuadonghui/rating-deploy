import { Alert, Button, Card, Form, Input } from "antd"
import { useState } from "react"
import { useNavigate } from "react-router-dom"
import { useAdminAuthStore } from "../../stores/adminAuth"

export default function AdminLogin() {
  const [error, setError] = useState<string | null>(null)
  const navigate = useNavigate()
  const login = useAdminAuthStore((state) => state.login)
  const loading = useAdminAuthStore((state) => state.loading)

  return (
    <div style={{ display: "flex", justifyContent: "center", padding: 48 }}>
      <Card title="管理员登录" style={{ width: 360 }}>
        {error ? <Alert type="error" message={error} style={{ marginBottom: 16 }} /> : null}
        <Form
          layout="vertical"
          onFinish={async (values) => {
            const ok = await login(values.password)
            if (ok) {
              navigate("/admin/contests")
            } else {
              setError("密码错误")
            }
          }}
        >
          <Form.Item label="密码" name="password" rules={[{ required: true, message: "请输入密码" }]}>
            <Input.Password />
          </Form.Item>
          <Button type="primary" htmlType="submit" block loading={loading}>
            登录
          </Button>
        </Form>
      </Card>
    </div>
  )
}
