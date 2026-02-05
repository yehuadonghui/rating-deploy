import type { ReactNode } from "react"
import { Navigate } from "react-router-dom"
import { useAdminAuthStore } from "../stores/adminAuth"

export default function ProtectedRoute({ children }: { children: ReactNode }) {
  const isAuthenticated = useAdminAuthStore((state) => state.isAuthenticated)
  return isAuthenticated ? <>{children}</> : <Navigate to="/admin/login" replace />
}
