import axios from "axios"

const client = axios.create({
  baseURL: "/api/v1",
  timeout: 10000,
})

const isAdminUrl = (url?: string) => !!url && url.startsWith("/admin")

client.interceptors.request.use((config) => {
  if (isAdminUrl(config.url)) {
    const token = localStorage.getItem("admin_token")
    if (token) {
      config.headers = config.headers || {}
      config.headers.Authorization = `Bearer ${token}`
    }
  }
  return config
})

client.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error?.response?.status === 401 && isAdminUrl(error?.config?.url)) {
      localStorage.removeItem("admin_token")
      window.location.href = "/admin/login"
    }
    return Promise.reject(error)
  }
)

export default client
