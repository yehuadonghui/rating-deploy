import { Layout, Menu, Button } from "antd"
import { Link, Outlet, useLocation, useNavigate } from "react-router-dom"
import { useAdminAuthStore } from "../../stores/adminAuth"

const { Header, Content, Sider } = Layout

export default function AdminLayout() {
  const location = useLocation()
  const navigate = useNavigate()
  const logout = useAdminAuthStore((state) => state.logout)

  const items = [
    { key: "/admin/contests", label: <Link to="/admin/contests">比赛管理</Link> },
    { key: "/admin/students", label: <Link to="/admin/students">学生管理</Link> },
    { key: "/admin/lab", label: <Link to="/admin/lab">算法实验室</Link> },
    { key: "/admin/tiers", label: <Link to="/admin/tiers">段位配置</Link> },
    { key: "/admin/site", label: <Link to="/admin/site">网站配置</Link> },
  ]

  return (
    <Layout style={{ minHeight: "100vh" }}>
      <Sider theme="light" width={200}>
        <div style={{ padding: 16, fontWeight: 600 }}>管理后台</div>
        <Menu mode="inline" selectedKeys={[location.pathname]} items={items} />
      </Sider>
      <Layout>
        <Header
          style={{
            background: "#fff",
            padding: "0 24px",
            display: "flex",
            justifyContent: "flex-end",
            gap: 12,
          }}
        >
          <Button onClick={() => navigate("/")}>返回主页</Button>
          <Button
            type="primary"
            onClick={() => {
              logout()
              navigate("/admin/login")
            }}
          >
            退出登录
          </Button>
        </Header>
        <Content style={{ padding: 24, background: "#f5f5f5" }}>
          <div className="page-card">
            <Outlet />
          </div>
        </Content>
      </Layout>
    </Layout>
  )
}
