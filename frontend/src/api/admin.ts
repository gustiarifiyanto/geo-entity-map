import type { AdminStats } from '../types/stats'
import { http } from './client'

export function getAdminStats(): Promise<AdminStats> {
  return http.get<AdminStats>('/admin/stats')
}
