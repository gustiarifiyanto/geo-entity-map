import { createContext, useContext } from 'react'

/** A zone being edited in a form, drawn on the map before it is saved. */
export interface ZonePreview {
  /** The entity whose stored zone this replaces on the map; null for a new entity. */
  entityId: string | null
  center_latitude: number
  center_longitude: number
  radius_m: number
}

/**
 * Lets the entity form talk to the map, which lives elsewhere in the tree:
 * "use the next map click as the zone center" and "draw this zone preview".
 */
export interface ZoneEditor {
  picking: boolean
  /** The next map click calls onPick instead of its usual action. */
  startPicking: (onPick: (latitude: number, longitude: number) => void) => void
  cancelPicking: () => void
  setPreview: (preview: ZonePreview | null) => void
}

export const ZoneEditorContext = createContext<ZoneEditor | null>(null)

export function useZoneEditor(): ZoneEditor {
  const editor = useContext(ZoneEditorContext)
  if (!editor) throw new Error('useZoneEditor must be used within ZoneEditorContext')
  return editor
}
