// Shared types mirroring the backend auth contract.

export interface User {
  id: string
  email: string
  /** Roles are defined by the backend; the frontend only needs to recognize admin. */
  role: string
  created_at: string
}

/** The role allowed to create, edit, move and delete entities. */
export const ADMIN_ROLE = 'admin'

/** Body for POST /api/auth/login and POST /api/auth/register. */
export interface Credentials {
  email: string
  password: string
}
