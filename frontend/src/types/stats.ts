// Mirrors GET /api/admin/stats. Roles are not hardcoded: by_role lists every
// role the backend knows.

export interface UserStats {
  total: number
  by_role: Record<string, number>
  with_active_session: number
  online: number
}

export interface AdminStats {
  users: UserStats
  /** How recently a user must have been active to count as online. */
  online_window_minutes: number
}
