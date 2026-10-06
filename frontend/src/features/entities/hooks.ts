import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  createEntity,
  deleteEntity,
  getMeta,
  listEntities,
  updateEntity,
  updateEntityLocation,
} from '../../api/entities'
import type { Entity, EntityFilter, EntityInput, LocationInput } from '../../types/entity'
import { installationKeys } from '../installations/hooks'
import { sensorKeys } from '../sensors/hooks'

export const metaKey = ['meta'] as const

// Mutation onSuccess handlers return the invalidation promise, so mutateAsync
// resolves only after the list has been refetched.
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
    // A type change can show or hide an installation schedule.
    onSuccess: () =>
      Promise.all([
        queryClient.invalidateQueries({ queryKey: entityKeys.all }),
        queryClient.invalidateQueries({ queryKey: installationKeys.all }),
        queryClient.invalidateQueries({ queryKey: sensorKeys.all }),
      ]),
  })
}

export function useDeleteEntity() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: deleteEntity,
    // Refetch on failure too: a 404 means it was already deleted elsewhere.
    // The entity's installation schedule is deleted with it.
    onSettled: () =>
      Promise.all([
        queryClient.invalidateQueries({ queryKey: entityKeys.all }),
        queryClient.invalidateQueries({ queryKey: installationKeys.all }),
        queryClient.invalidateQueries({ queryKey: sensorKeys.all }),
      ]),
  })
}

/**
 * Moves an entity (marker drag) with an optimistic update: the cached lists
 * change immediately and are restored if the request fails, which also moves
 * the marker back to its original position.
 */
export function useUpdateEntityLocation() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ id, location }: { id: string; location: LocationInput }) =>
      updateEntityLocation(id, location),
    onMutate: async ({ id, location }) => {
      // Stop in-flight refetches from overwriting the optimistic value.
      await queryClient.cancelQueries({ queryKey: entityKeys.all })
      const previous = queryClient.getQueriesData<Entity[]>({ queryKey: entityKeys.all })
      queryClient.setQueriesData<Entity[]>({ queryKey: entityKeys.all }, (entities) =>
        entities?.map((e) => (e.id === id ? { ...e, ...location } : e)),
      )
      return { previous }
    },
    onError: (_error, _variables, context) => {
      for (const [key, data] of context?.previous ?? []) {
        queryClient.setQueryData(key, data)
      }
    },
    onSettled: () => queryClient.invalidateQueries({ queryKey: entityKeys.all }),
  })
}
