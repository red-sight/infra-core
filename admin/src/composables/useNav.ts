// Bridges the design's nav(screen, params) callback onto vue-router. Params are
// carried as route query so deep links (?open=org_x, ?create=1) work.
import { useRouter } from 'vue-router'
import { SCREEN_PATHS } from '@/components/shell/nav'

export function useNav() {
  const router = useRouter()
  return (screen: string, params?: Record<string, string>) => {
    const path = SCREEN_PATHS[screen] ?? '/'
    router.push({ path, query: params })
    window.scrollTo({ top: 0 })
  }
}
