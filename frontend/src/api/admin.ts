import type { ContestRewardType, PointCategory } from "../types"
import client from "./client"

export const adminApi = {
  login: (password: string) => client.post("/admin/login", { password }),
  getConfig: () => client.get("/admin/config"),
  updateConfig: (data: Record<string, number | string>) => client.put("/admin/config", data),
  uploadContest: (formData: FormData, onProgress?: (p: number) => void) =>
    client.post("/admin/contests", formData, {
      headers: { "Content-Type": "multipart/form-data" },
      onUploadProgress: (evt) => {
        if (!evt.total || !onProgress) return
        onProgress(evt.loaded / evt.total)
      },
    }),
  updateContest: (id: number, data: { name?: string; weight?: number; reward_type?: ContestRewardType }) =>
    client.put(`/admin/contests/${id}`, data),
  deleteContest: (id: number) => client.delete(`/admin/contests/${id}`),
  updateTiers: (tiers: Array<Record<string, number | string>>) => client.put("/admin/tiers", tiers),
  triggerReplay: () => client.post("/admin/replay/apply"),
  getReplayStatus: () => client.get("/admin/replay/status"),
  previewReplay: (data: Record<string, number>) => client.post("/admin/replay/preview", data),
  listPointRecords: (params?: {
    student_id?: string
    contest_id?: number
    category?: PointCategory
    source?: "auto" | "manual"
    page?: number
    size?: number
  }) => client.get("/admin/points", { params }),
  updateSiteConfig: (data: Record<string, number | string | boolean>) =>
    client.put("/admin/site-config", data),
  listStudents: (params?: { query?: string; page?: number; size?: number }) =>
    client.get("/admin/students", { params }),
  createStudent: (data: {
    student_id: string
    name: string
    email?: string
    class?: string
    grade?: number
  }) => client.post("/admin/students", data),
  updateStudent: (
    id: string,
    data: { name?: string; email?: string; class?: string; grade?: number }
  ) => client.put(`/admin/students/${id}`, data),
  createStudentPointRecord: (
    id: string,
    data: { category: PointCategory; points: number; title: string; description?: string }
  ) => client.post(`/admin/students/${id}/points`, data),
  deleteStudent: (id: string) => client.delete(`/admin/students/${id}`),
  deleteAllStudents: () => client.delete("/admin/students"),
}
