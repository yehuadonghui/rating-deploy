import { Alert, Button, Card, Form, Input, Space, Switch, message } from "antd"
import { useEffect, useMemo, useState } from "react"
import { adminApi } from "../../api/admin"
import Loading from "../../components/Loading"
import { useSiteConfigStore } from "../../stores/siteConfig"

type SiteConfigFormValues = {
  site_name: string
  site_logo: string
  primary_color: string
  footer_text: string
  chart_show_area: boolean
  chart_smooth: boolean
}

const toBool = (value: unknown) => value === true || value === "true"

export default function AdminSiteConfig() {
  const [form] = Form.useForm<SiteConfigFormValues>()
  const { config, loadConfig, updateLocal } = useSiteConfigStore()
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    loadConfig()
      .catch(() => setError("加载站点配置失败"))
      .finally(() => setLoading(false))
  }, [loadConfig])

  const initialValues = useMemo(
    () => ({
      site_name: config.site_name,
      site_logo: config.site_logo,
      primary_color: config.primary_color,
      footer_text: config.footer_text,
      chart_show_area: toBool(config.chart_show_area),
      chart_smooth: toBool(config.chart_smooth),
    }),
    [config]
  )

  useEffect(() => {
    form.setFieldsValue(initialValues)
  }, [form, initialValues])

  if (loading) {
    return <Loading />
  }

  if (error) {
    return <Alert type="error" message={error} />
  }

  return (
    <div>
      <h2>网站配置</h2>
      <Card>
        <Form form={form} layout="vertical" initialValues={initialValues}>
          <Form.Item label="站点名称" name="site_name" rules={[{ required: true, message: "请输入站点名称" }]}>
            <Input />
          </Form.Item>
          <Form.Item label="Logo 地址" name="site_logo">
            <Input placeholder="可留空" />
          </Form.Item>
          <Form.Item label="主色" name="primary_color" rules={[{ required: true, message: "请输入主色" }]}>
            <Input type="color" style={{ width: 120 }} />
          </Form.Item>
          <Form.Item label="页脚文本" name="footer_text">
            <Input placeholder="可留空，默认显示站点名" />
          </Form.Item>
          <Form.Item label="图表显示面积" name="chart_show_area" valuePropName="checked">
            <Switch />
          </Form.Item>
          <Form.Item label="图表平滑曲线" name="chart_smooth" valuePropName="checked">
            <Switch />
          </Form.Item>
          <Space>
            <Button
              type="primary"
              loading={saving}
              onClick={async () => {
                try {
                  setSaving(true)
                  const values = await form.validateFields()
                  await adminApi.updateSiteConfig(values)
                  updateLocal(values)
                  message.success("网站配置已更新")
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
        </Form>
      </Card>
    </div>
  )
}
