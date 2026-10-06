// Mirrors GET /api/admin/stats. Roles are not hardcoded: by_role lists every
// role the backend knows.

export interface UserStats {
  total: number
  by_role: Record<string, number>
  with_active_session: number
  online: number
  /** Who is behind with_active_session (role user only), most recently seen first. */
  active_users: ActiveUser[]
}

export interface ActiveUser {
  id: string
  email: string
  /** Null for sessions from before activity was recorded. */
  last_seen_at: string | null
  online: boolean
}

export interface AdminStats {
  users: UserStats
  /** How recently a user must have been active to count as online. */
  online_window_minutes: number
}
