import { latLngBounds, type Marker as LeafletMarker } from 'leaflet'
import { useEffect, useRef } from 'react'
import {
  MapContainer,
  Marker,
  TileLayer,
  Tooltip,
  useMap,
  useMapEvents,
  ZoomControl,
} from 'react-leaflet'
import type { Entity } from '../../types/entity'
import { DEFAULT_CENTER, DEFAULT_ZOOM, toCoordinates, WORLD_BOUNDS } from './geo'
import { draftMarkerIcon, markerIcon } from './markerIcon'

interface EntityMapProps {
  entities: Entity[]
  selectedId: string | null
  onSelect: (id: string) => void
  onMapClick: (latitude: number, longitude: number) => void
  /** Location of an entity being created (not saved yet), shown as a draggable pin. */
  draft?: { latitude: number; longitude: number } | null
  onDraftMove?: (latitude: number, longitude: number) => void
}

export function EntityMap({
  entities,
  selectedId,
  onSelect,
  onMapClick,
  draft,
  onDraftMove,
}: EntityMapProps) {
  return (
    <MapContainer
      center={DEFAULT_CENTER}
      zoom={DEFAULT_ZOOM}
      minZoom={2}
      maxBounds={WORLD_BOUNDS}
      maxBoundsViscosity={1}
      zoomControl={false}
      className="h-full w-full"
    >
      <TileLayer
        attribution='&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> contributors'
        url="https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png"
        noWrap
      />
      <ZoomControl position="bottomright" />
      <MapClickHandler onMapClick={onMapClick} />
      <FitToEntitiesOnce entities={entities} />
      {entities.map((entity) => (
        <Marker
          key={entity.id}
          position={[entity.latitude, entity.longitude]}
          icon={markerIcon(entity.status, entity.id === selectedId)}
          zIndexOffset={entity.id === selectedId ? 1000 : 0}
          eventHandlers={{ click: () => onSelect(entity.id) }}
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
          title="New entity location"
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
    map.fitBounds(bounds, { padding: [48, 48], maxZoom: 14 })
  }, [entities, map])

  return null
}
