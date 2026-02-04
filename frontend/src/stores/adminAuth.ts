import { create } from "zustand"
import { adminApi } from "../api/admin"

type AdminAuthStore = {
  token: string | null
  isAuthenticated: boolean
  loading: boolean
  login: (password: string) => Promise<boolean>
  logout: () => void
  checkAuth: () => void
}

export const useAdminAuthStore = create<AdminAuthStore>((set) => ({
  token: localStorage.getItem("admin_token"),
  isAuthenticated: !!localStorage.getItem("admin_token"),
  loading: false,
  login: async (password) => {
    set({ loading: true })
    try {
      const { data } = await adminApi.login(password)
      localStorage.setItem("admin_token", data.token)
      set({ token: data.token, isAuthenticated: true, loading: false })
      return true
    } catch {
      set({ loading: false })
      return false
    }
  },
  logout: () => {
    localStorage.removeItem("admin_token")
    set({ token: null, isAuthenticated: false })
  },
  checkAuth: () => {
    const token = localStorage.getItem("admin_token")
    set({ token, isAuthenticated: !!token })
  },
}))
