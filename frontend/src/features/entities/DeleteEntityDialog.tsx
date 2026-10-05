import { ApiError } from '../../api/client'
import { ConfirmDialog } from '../../components/ConfirmDialog'
import { useToast } from '../../components/toast/context'
import type { Entity } from '../../types/entity'
import { useDeleteEntity } from './hooks'

interface DeleteEntityDialogProps {
  /** The entity to delete; the dialog is open while this is set. */
  entity: Entity | null
  onClose: () => void
  onDeleted: () => void
}

export function DeleteEntityDialog({ entity, onClose, onDeleted }: DeleteEntityDialogProps) {
  const remove = useDeleteEntity()
  const toast = useToast()

  const confirm = async () => {
    if (!entity) return
    try {
      await remove.mutateAsync(entity.id)
      toast.success(`"${entity.name}" deleted.`)
      onDeleted()
    } catch (error) {
      if (error instanceof ApiError && error.status === 404) {
        // Already gone (e.g. deleted in another tab); the list was refetched.
        toast.error(`"${entity.name}" no longer exists.`)
        onDeleted()
        return
      }
      // Keep the dialog open so the user can retry.
      toast.error(`Could not delete "${entity.name}": ${error instanceof Error ? error.message : 'unknown error'}`)
    }
  }

  return (
    <ConfirmDialog
      open={entity !== null}
      title="Delete entity?"
      confirmLabel="Delete"
      pendingLabel="Deleting…"
      pending={remove.isPending}
      onConfirm={() => void confirm()}
      onCancel={onClose}
    >
      {entity && (
        <p>
          <span className="font-medium text-gray-900">{entity.name}</span> will be permanently deleted.
          This cannot be undone.
        </p>
      )}
    </ConfirmDialog>
  )
}
