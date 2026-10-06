import { useMutation, useQuery, useQueryClient, type QueryClient } from '@tanstack/react-query'
import { getMe, login, logout, register } from '../../api/auth'
import { ApiError } from '../../api/client'
import type { User } from '../../types/auth'

export const meKey = ['auth', 'me'] as const

/**
 * The logged-in user, or null when logged out. A 401 is the normal
 * "logged out" answer here, not an error.
 */
export function useMe() {
  return useQuery({
    queryKey: meKey,
    queryFn: async (): Promise<User | null> => {
      try {
        return await getMe()
      } catch (error) {
        if (error instanceof ApiError && error.status === 401) return null
        throw error
      }
    },
    // Changes only through login/logout below, or a 401 from another request.
    staleTime: Infinity,
  })
}

/** Switches the app to the logged-out state, which shows the login screen. */
export function markLoggedOut(queryClient: QueryClient) {
  queryClient.setQueryData<User | null>(meKey, null)
}

/**
 * Stores the user who just logged in. Data cached for a previous account is
 * dropped first; this runs while the login screen is shown, so no mounted
 * query refetches it with the old session.
 */
function markLoggedIn(queryClient: QueryClient, user: User) {
  queryClient.removeQueries({ predicate: (query) => query.queryKey[0] !== meKey[0] })
  queryClient.setQueryData(meKey, user)
}

/** True for a 401 that means the session is missing or expired (not a wrong password). */
export function isSessionExpired(error: unknown): boolean {
  return error instanceof ApiError && error.status === 401 && error.code === 'unauthorized'
}

export function useLogin() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: login,
    onSuccess: (user) => markLoggedIn(queryClient, user),
  })
}

export function useRegister() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: register,
    onSuccess: (user) => markLoggedIn(queryClient, user),
  })
}

export function useLogout() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: logout,
    onSuccess: () => markLoggedOut(queryClient),
  })
}
