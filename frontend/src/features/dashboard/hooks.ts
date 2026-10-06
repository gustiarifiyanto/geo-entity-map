import { useQuery } from '@tanstack/react-query'
import { getAdminStats } from '../../api/admin'

/** Refreshed while the dashboard is open and the browser tab is visible. */
export const ADMIN_STATS_REFRESH_MS = 30_000

/** User counts for admins. Only mount this for admins: other roles get 403. */
export function useAdminStats() {
  return useQuery({
    queryKey: ['admin', 'stats'],
    queryFn: getAdminStats,
    refetchInterval: ADMIN_STATS_REFRESH_MS,
    refetchIntervalInBackground: false,
  })
}
