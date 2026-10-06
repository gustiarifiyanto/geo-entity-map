import { useQuery } from '@tanstack/react-query'
import { listPhotos } from '../../api/photos'

export const photoKeys = {
  list: (entityId: string) => ['photos', entityId] as const,
}

export function usePhotos(entityId: string) {
  return useQuery({ queryKey: photoKeys.list(entityId), queryFn: () => listPhotos(entityId) })
}
