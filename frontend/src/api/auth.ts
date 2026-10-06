import type { Credentials, User } from '../types/auth'
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
