import { ApiError } from '../../api/client'
import { ConfirmDialog } from '../../components/ConfirmDialog'
import { useToast } from '../../components/toast/context'
import { useI18n } from '../../i18n/context'
import type { Entity } from '../../types/entity'
import { useDeleteEntity } from './hooks'

interface DeleteEntityDialogProps {
  /** The entity to delete; the dialog is open while this is set. */
  entity: Entity | null
  onClose: () => void
  onDeleted: () => void
}

export function DeleteEntityDialog({ entity, onClose, onDeleted }: DeleteEntityDialogProps) {
  const { t, errorText } = useI18n()
  const remove = useDeleteEntity()
  const toast = useToast()

  const confirm = async () => {
    if (!entity) return
    try {
      await remove.mutateAsync(entity.id)
      toast.success(t.deleteEntity.done(entity.name))
      onDeleted()
    } catch (error) {
      if (error instanceof ApiError && error.status === 404) {
        // Already gone (e.g. deleted in another tab); the list was refetched.
        toast.error(t.deleteEntity.gone(entity.name))
        onDeleted()
        return
      }
      // Keep the dialog open so the user can retry.
      toast.error(t.deleteEntity.failed(entity.name, errorText(error)))
    }
  }

  return (
    <ConfirmDialog
      open={entity !== null}
      title={t.deleteEntity.title}
      confirmLabel={t.common.delete}
      pendingLabel={t.common.deleting}
      pending={remove.isPending}
      onConfirm={() => void confirm()}
      onCancel={onClose}
    >
      {entity && <p>{t.deleteEntity.body(entity.name)}</p>}
    </ConfirmDialog>
  )
}
