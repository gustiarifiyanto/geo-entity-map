import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  createEntity,
  deleteEntity,
  getMeta,
  listEntities,
  updateEntity,
} from '../../api/entities'
import type { EntityFilter, EntityInput } from '../../types/entity'

export const metaKey = ['meta'] as const

export const entityKeys = {
  all: ['entities'] as const,
  list: (filter: EntityFilter) => [...entityKeys.all, 'list', filter] as const,
}

/** Allowed entity types and statuses; they only change when the backend is redeployed. */
export function useMeta() {
  return useQuery({ queryKey: metaKey, queryFn: getMeta, staleTime: Infinity })
}

export function useEntities(filter: EntityFilter = {}) {
  return useQuery({
    queryKey: entityKeys.list(filter),
    queryFn: () => listEntities(filter),
  })
}

export function useCreateEntity() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: createEntity,
    onSuccess: () => queryClient.invalidateQueries({ queryKey: entityKeys.all }),
  })
}

export function useUpdateEntity() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ id, input }: { id: string; input: EntityInput }) => updateEntity(id, input),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: entityKeys.all }),
  })
}

export function useDeleteEntity() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: deleteEntity,
    onSuccess: () => queryClient.invalidateQueries({ queryKey: entityKeys.all }),
  })
}
