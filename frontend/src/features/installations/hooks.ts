import { useQuery } from '@tanstack/react-query'
import { getInstallation, listInstallations } from '../../api/installations'

export const installationKeys = {
  all: ['installations'] as const,
  one: (entityId: string) => [...installationKeys.all, 'one', entityId] as const,
  list: () => [...installationKeys.all, 'list'] as const,
}

/** One entity's schedule (null if none). Pass enabled=false for types without the capability. */
export function useInstallation(entityId: string, enabled: boolean) {
  return useQuery({
    queryKey: installationKeys.one(entityId),
    queryFn: () => getInstallation(entityId),
    enabled,
  })
}

export function useInstallations() {
  return useQuery({ queryKey: installationKeys.list(), queryFn: listInstallations })
}
