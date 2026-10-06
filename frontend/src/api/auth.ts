import type { Credentials, DemoAccount, User } from '../types/auth'
import { http } from './client'

export function getMe(): Promise<User> {
  return http.get<User>('/auth/me')
}

export function login(credentials: Credentials): Promise<User> {
  return http.post<User>('/auth/login', credentials)
}

export function register(credentials: Credentials): Promise<User> {
  return http.post<User>('/auth/register', credentials)
}

export function logout(): Promise<void> {
  return http.post<void>('/auth/logout', undefined)
}

/** Demo logins for the login page; empty unless the server runs in demo mode. */
export function getDemoAccounts(): Promise<DemoAccount[]> {
  return http.get<DemoAccount[]>('/auth/demo-accounts')
}
