import type { Installation, InstallationInput } from '../types/installation'
import { http } from './client'

const installationPath = (entityId: string) => `/entities/${encodeURIComponent(entityId)}/installation`

/** The entity's schedule, or null when none was set yet. */
export function getInstallation(entityId: string): Promise<Installation | null> {
  return http.get<Installation | null>(installationPath(entityId))
}

export function putInstallation(entityId: string, input: InstallationInput): Promise<Installation> {
  return http.put<Installation>(installationPath(entityId), input)
}

export function deleteInstallation(entityId: string): Promise<void> {
  return http.delete(installationPath(entityId))
}

export function listInstallations(): Promise<Installation[]> {
  return http.get<Installation[]>('/installations')
}
