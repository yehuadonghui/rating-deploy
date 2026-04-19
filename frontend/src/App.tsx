import { Layout, Menu } from "antd"
import { useEffect } from "react"
import { Link, Navigate, Outlet, Route, Routes, useLocation } from "react-router-dom"
import AdminLayout from "./pages/admin/Layout"
import AdminLogin from "./pages/admin/Login"
import AdminContests from "./pages/admin/Contests"
import AdminStudents from "./pages/admin/Students"
import AdminPointRecords from "./pages/admin/PointRecords"
import AdminLab from "./pages/admin/Lab"
import AdminTiers from "./pages/admin/Tiers"
import AdminSiteConfig from "./pages/admin/SiteConfig"
import Leaderboard from "./pages/Leaderboard"
import Statistics from "./pages/Statistics"
import StudentDetail from "./pages/StudentDetail"
import ProtectedRoute from "./components/ProtectedRoute"
import { useSiteConfigStore } from "./stores/siteConfig"

const { Header, Content, Footer } = Layout

function PublicLayout() {
  const location = useLocation()
  const { config, loadConfig } = useSiteConfigStore()

  useEffect(() => {
    loadConfig()
  }, [loadConfig])

  const items = [
    { key: "/", label: <Link to="/">排行榜</Link> },
    { key: "/statistics", label: <Link to="/statistics">统计</Link> },
  ]

  return (
    <Layout style={{ minHeight: "100vh" }}>
      <Header
        style={{
          display: "flex",
          alignItems: "center",
          padding: "0 24px",
          background: config.primary_color,
        }}
      >
        <Link
          to="/"
          style={{ textDecoration: "none", marginRight: 48, display: "flex", alignItems: "center" }}
        >
          {config.site_logo ? (
            <img src={config.site_logo} alt={config.site_name} style={{ height: 32, marginRight: 8 }} />
          ) : null}
          <span style={{ color: "#fff", fontSize: 18, fontWeight: 600 }}>{config.site_name}</span>
        </Link>
        <Menu
          theme="dark"
          mode="horizontal"
          selectedKeys={[location.pathname]}
          items={items}
          style={{ flex: 1, background: "transparent" }}
        />
        <Link to="/admin/login" style={{ color: "#fff" }}>
          管理员登录
        </Link>
      </Header>
      <Content style={{ padding: "24px 48px" }}>
        <Outlet />
      </Content>
      <Footer style={{ textAlign: "center" }}>
        {config.footer_text || `${config.site_name} ©${new Date().getFullYear()}`}
      </Footer>
    </Layout>
  )
}

export default function App() {
  return (
    <Routes>
      <Route element={<PublicLayout />}>
        <Route index element={<Leaderboard />} />
        <Route path="statistics" element={<Statistics />} />
        <Route path="students/:id" element={<StudentDetail />} />
      </Route>

      <Route path="/admin/login" element={<AdminLogin />} />

      <Route path="/admin" element={<AdminLayout />}>
        <Route
          path="contests"
          element={
            <ProtectedRoute>
              <AdminContests />
            </ProtectedRoute>
          }
        />
        <Route
          path="lab"
          element={
            <ProtectedRoute>
              <AdminLab />
            </ProtectedRoute>
          }
        />
        <Route
          path="students"
          element={
            <ProtectedRoute>
              <AdminStudents />
            </ProtectedRoute>
          }
        />
        <Route
          path="points"
          element={
            <ProtectedRoute>
              <AdminPointRecords />
            </ProtectedRoute>
          }
        />
        <Route
          path="tiers"
          element={
            <ProtectedRoute>
              <AdminTiers />
            </ProtectedRoute>
          }
        />
        <Route
          path="site"
          element={
            <ProtectedRoute>
              <AdminSiteConfig />
            </ProtectedRoute>
          }
        />
        <Route index element={<Navigate to="/admin/contests" replace />} />
      </Route>

      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  )
}
