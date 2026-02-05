import client from "./client"

export const publicApi = {
  getLeaderboard: (params?: { grade?: number; page?: number; size?: number }) =>
    client.get("/leaderboard", { params }),
  searchStudents: (params?: { query?: string; page?: number; size?: number }) =>
    client.get("/students", { params }),
  getStudent: (id: string) => client.get(`/students/${id}`),
  getStudentHistory: (id: string) => client.get(`/students/${id}/history`),
  getContests: () => client.get("/contests"),
  getTiers: () => client.get("/tiers"),
  getStatistics: () => client.get("/statistics"),
  getSiteConfig: () => client.get("/site-config"),
}
