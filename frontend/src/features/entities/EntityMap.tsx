import { latLngBounds, type Marker as LeafletMarker } from 'leaflet'
import { useEffect, useRef } from 'react'
import {
  Circle,
  MapContainer,
  Marker,
  TileLayer,
  Tooltip,
  useMap,
  useMapEvents,
  ZoomControl,
} from 'react-leaflet'
import { useI18n } from '../../i18n/context'
import type { Entity } from '../../types/entity'
import type { Geofence } from '../../types/geofence'
import type { ZonePreview } from '../geofences/zoneEditor'
import { DEFAULT_CENTER, DEFAULT_ZOOM, toCoordinates, WORLD_BOUNDS } from './geo'
import { draftMarkerIcon, markerIcon } from './markerIcon'

// Dashed, light circles: zones are context, the pins are the data.
const ZONE_INSIDE = { color: '#6b7280', weight: 2, dashArray: '6 6', fillColor: '#6b7280', fillOpacity: 0.05 }
const ZONE_OUTSIDE = { color: '#dc2626', weight: 2, dashArray: '6 6', fillColor: '#dc2626', fillOpacity: 0.08 }
const ZONE_PREVIEW = { color: '#2563eb', weight: 2, dashArray: '4 4', fillColor: '#2563eb', fillOpacity: 0.08 }

interface EntityMapProps {
  entities: Entity[]
  selectedId: string | null
  onSelect: (id: string) => void
  onMapClick: (latitude: number, longitude: number) => void
  /**
   * Whether the selected marker can be dragged to a new location. Only the
   * selected marker is ever draggable, so panning the map cannot move an
   * entity by accident.
   */
  markersDraggable: boolean
  onMove: (id: string, latitude: number, longitude: number) => void
  /** Location of an entity being created (not saved yet), shown as a draggable pin. */
  draft?: { latitude: number; longitude: number } | null
  onDraftMove?: (latitude: number, longitude: number) => void
  /** Operating zones to draw (red when the entity is outside). */
  zones?: Geofence[]
  /** A zone being edited, drawn instead of that entity's stored zone. */
  zonePreview?: ZonePreview | null
  /** True while the next click picks a zone center (crosshair cursor). */
  picking?: boolean
}

export function EntityMap({
  entities,
  selectedId,
  onSelect,
  onMapClick,
  markersDraggable,
  onMove,
  draft,
  onDraftMove,
  zones = [],
  zonePreview = null,
  picking = false,
}: EntityMapProps) {
  const { t } = useI18n()
  return (
    <MapContainer
      center={DEFAULT_CENTER}
      zoom={DEFAULT_ZOOM}
      minZoom={2}
      maxBounds={WORLD_BOUNDS}
      maxBoundsViscosity={1}
      zoomControl={false}
      className={`h-full w-full ${picking ? 'picking-zone' : ''}`}
    >
      <TileLayer
        attribution='&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> contributors'
        url="https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png"
        noWrap
      />
      <ZoomControl position="bottomright" />
      {zones
        .filter((z) => !zonePreview || z.entity_id !== zonePreview.entityId)
        .map((z) => (
          <Circle
            key={z.entity_id}
            center={[z.center_latitude, z.center_longitude]}
            radius={z.radius_m}
            interactive={false}
            pathOptions={z.inside ? ZONE_INSIDE : ZONE_OUTSIDE}
          />
        ))}
      {zonePreview && (
        <Circle
          center={[zonePreview.center_latitude, zonePreview.center_longitude]}
          radius={zonePreview.radius_m}
          interactive={false}
          pathOptions={ZONE_PREVIEW}
        />
      )}
      <MapClickHandler onMapClick={onMapClick} />
      <FitToEntitiesOnce entities={entities} />
      {entities.map((entity) => (
        <Marker
          key={entity.id}
          position={[entity.latitude, entity.longitude]}
          icon={markerIcon(entity.status, entity.id === selectedId)}
          zIndexOffset={entity.id === selectedId ? 1000 : 0}
          draggable={markersDraggable && entity.id === selectedId}
          eventHandlers={{
            click: () => onSelect(entity.id),
            dragend: (event) => {
              const { latitude, longitude } = toCoordinates((event.target as LeafletMarker).getLatLng())
              onMove(entity.id, latitude, longitude)
            },
          }}
          keyboard
          title={entity.name}
        >
          <Tooltip direction="top" offset={[0, -36]}>
            {entity.name}
          </Tooltip>
        </Marker>
      ))}
      {draft && (
        <Marker
          position={[draft.latitude, draft.longitude]}
          icon={draftMarkerIcon()}
          zIndexOffset={2000}
          draggable
          eventHandlers={{
            dragend: (event) => {
              const { latitude, longitude } = toCoordinates((event.target as LeafletMarker).getLatLng())
              onDraftMove?.(latitude, longitude)
            },
          }}
          title={t.map.newLocation}
        />
      )}
    </MapContainer>
  )
}

function MapClickHandler({ onMapClick }: Pick<EntityMapProps, 'onMapClick'>) {
  useMapEvents({
    click: (event) => {
      const { latitude, longitude } = toCoordinates(event.latlng)
      onMapClick(latitude, longitude)
    },
  })
  return null
}

/** Zooms to show all entities the first time they load. */
function FitToEntitiesOnce({ entities }: { entities: Entity[] }) {
  const map = useMap()
  const fitted = useRef(false)

  useEffect(() => {
    if (fitted.current || entities.length === 0) return
    fitted.current = true
    const bounds = latLngBounds(entities.map((e): [number, number] => [e.latitude, e.longitude]))
    // Extra top padding keeps pins clear of the translucent app bar and the legend card.
    map.fitBounds(bounds, { paddingTopLeft: [48, 140], paddingBottomRight: [48, 48], maxZoom: 14 })
  }, [entities, map])

  return null
}
