import { divIcon, type DivIcon } from 'leaflet'
import { statusColor } from './labels'

const cache = new Map<string, DivIcon>()

const DRAFT_COLOR = '#111827'

/** A pin colored by status. Icons are cached so markers do not re-render needlessly. */
export function markerIcon(status: string, selected: boolean): DivIcon {
  return pinIcon(statusColor(status), selected)
}

/** The pin for an entity that is being created and not saved yet. */
export function draftMarkerIcon(): DivIcon {
  return pinIcon(DRAFT_COLOR, true)
}

function pinIcon(color: string, selected: boolean): DivIcon {
  const key = `${color}|${selected}`
  let icon = cache.get(key)
  if (!icon) {
    const width = selected ? 36 : 28
    const height = Math.round(width * (40 / 28))
    icon = divIcon({
      // Only the palette color is interpolated, never server-provided text.
      html: `<svg width="${width}" height="${height}" viewBox="0 0 28 40" aria-hidden="true">
        <path d="M14 1C6.8 1 1 6.8 1 14c0 9.8 13 25 13 25s13-15.2 13-25C27 6.8 21.2 1 14 1z"
          fill="${color}" stroke="white" stroke-width="2"/>
        <circle cx="14" cy="14" r="5" fill="white"/>
      </svg>`,
      className: selected ? 'entity-marker entity-marker--selected' : 'entity-marker',
      iconSize: [width, height],
      iconAnchor: [width / 2, height],
    })
    cache.set(key, icon)
  }
  return icon
}
