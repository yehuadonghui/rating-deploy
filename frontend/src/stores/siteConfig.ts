import { create } from "zustand"
import { publicApi } from "../api/public"

type SiteConfig = {
  site_name: string
  site_logo: string
  primary_color: string
  footer_text: string
  chart_show_area: boolean
  chart_smooth: boolean
}

const defaultConfig: SiteConfig = {
  site_name: "Rating System",
  site_logo: "",
  primary_color: "#1677ff",
  footer_text: "",
  chart_show_area: true,
  chart_smooth: true,
}

type SiteConfigStore = {
  config: SiteConfig
  loaded: boolean
  loading: boolean
  loadConfig: () => Promise<void>
  updateLocal: (patch: Partial<SiteConfig>) => void
}

export const useSiteConfigStore = create<SiteConfigStore>((set, get) => ({
  config: defaultConfig,
  loaded: false,
  loading: false,
  loadConfig: async () => {
    if (get().loaded || get().loading) return
    set({ loading: true })
    try {
      const { data } = await publicApi.getSiteConfig()
      set({ config: { ...defaultConfig, ...data }, loaded: true })
    } catch {
      set({ loaded: true })
    } finally {
      set({ loading: false })
    }
  },
  updateLocal: (patch) => set((state) => ({ config: { ...state.config, ...patch } })),
}))
