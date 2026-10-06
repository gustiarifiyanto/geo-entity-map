import { createContext, useContext, type ReactNode } from 'react'
import { useI18n } from '../../i18n/context'
import { timeAgo } from '../../i18n/format'
import { CAP_GEOFENCE, CAP_INSTALLATION } from '../../schemas/capabilities'
import type { Entity, Meta } from '../../types/entity'
import type { ActiveUser } from '../../types/stats'
import { statusColor } from '../entities/labels'
import { formatDistance } from '../geofences/distance'
import { useGeofences } from '../geofences/hooks'
import { useInstallations } from '../installations/hooks'
import { installationStatusColor, installationStatusOrder } from '../installations/status'
import { useAdminStats } from './hooks'
import { countBy, type Count } from './summary'

/** One neutral hue for single-series bars (by type, by role). */
const NEUTRAL_BAR = '#475569'

interface DashboardProps {
  entities: Entity[] | undefined
  meta: Meta | undefined
  /** Admins also see user statistics. */
  showUserStats: boolean
  /** Switches to the map, flies to the entity and opens its details. */
  onOpenEntity: (id: string) => void
}

const hasAny = (meta: Meta | undefined, capability: string) =>
  Object.values(meta?.capabilities ?? {}).some((caps) => caps.includes(capability))

export function Dashboard({ entities, meta, showUserStats, onOpenEntity }: DashboardProps) {
  return (
    <OpenEntityContext.Provider value={onOpenEntity}>
      <div className="mx-auto max-w-5xl space-y-8 px-4 pb-8">
        <EntitySummary entities={entities} meta={meta} />
        {hasAny(meta, CAP_INSTALLATION) && <InstallationOverview entities={entities} />}
        {hasAny(meta, CAP_GEOFENCE) && <ZoneOverview entities={entities} />}
        {showUserStats && <UserSummary />}
      </div>
    </OpenEntityContext.Provider>
  )
}

function EntitySummary({ entities, meta }: Pick<DashboardProps, 'entities' | 'meta'>) {
  const { t, value } = useI18n()
  if (!entities || !meta) return <Loading title={t.dashboard.entities} />

  return (
    <Section title={t.dashboard.entities}>
      <div className="grid gap-4 md:grid-cols-3">
        <StatTile label={t.dashboard.totalEntities} value={entities.length}>
          <EntityLinks
            items={[...entities]
              .sort((a, b) => a.name.localeCompare(b.name))
              .map((e) => ({ id: e.id, name: e.name, color: statusColor(e.status), detail: value(e.type) }))}
          />
        </StatTile>
        <Card title={t.dashboard.byStatus}>
          <CountBars counts={countBy(entities, 'status', meta.statuses)} color={statusColor} />
        </Card>
        <Card title={t.dashboard.byType}>
          <CountBars counts={countBy(entities, 'type', meta.types)} color={() => NEUTRAL_BAR} />
        </Card>
      </div>
    </Section>
  )
}

function UserSummary() {
  const { t, locale } = useI18n()
  const stats = useAdminStats()

  if (stats.isPending) return <Loading title={t.dashboard.users} />
  if (stats.isError) {
    return <Failed title={t.dashboard.users} error={stats.error} onRetry={() => void stats.refetch()} />
  }

  const { users, online_window_minutes: window } = stats.data
  const roles = Object.entries(users.by_role).map(([value, count]) => ({ value, count }))
  return (
    <Section title={t.dashboard.users} aside={t.dashboard.updated(new Date(stats.dataUpdatedAt).toLocaleTimeString(locale))}>
      <div className="grid gap-4 sm:grid-cols-2 md:grid-cols-4">
        <StatTile label={t.dashboard.registered} value={users.total} hint={t.dashboard.registeredHint} />
        <StatTile label={t.dashboard.online(window)} value={users.online} hint={t.dashboard.onlineHint(window)} />
        <StatTile
          label={t.dashboard.activeSession}
          value={users.with_active_session}
          hint={t.dashboard.activeSessionHint}
        >
          <ActiveUserList users={users.active_users} />
        </StatTile>
        <Card title={t.dashboard.byRole}>
          <CountBars counts={roles} color={() => NEUTRAL_BAR} />
        </Card>
      </div>
    </Section>
  )
}

/** Installation schedules of every entity type that has them: counts per status and the overdue list. */
function InstallationOverview({ entities }: { entities: Entity[] | undefined }) {
  const { t, value } = useI18n()
  const installations = useInstallations()
  const title = t.dashboard.installations

  if (installations.isPending || !entities) return <Loading title={title} />
  if (installations.isError) {
    return <Failed title={title} error={installations.error} onRetry={() => void installations.refetch()} />
  }

  const list = installations.data
  if (list.length === 0) {
    return (
      <Section title={title}>
        <p className="text-sm text-gray-500">{t.dashboard.noInstallations}</p>
      </Section>
    )
  }

  const counts = new Map<string, number>()
  for (const i of list) counts.set(i.status, (counts.get(i.status) ?? 0) + 1)
  const byStatus = [...counts]
    .map(([value, count]) => ({ value, count }))
    .sort((a, b) => installationStatusOrder(a.value) - installationStatusOrder(b.value))
  const names = new Map(entities.map((e) => [e.id, e.name]))
  // The API lists overdue first, most days late first.
  const overdue = list.filter((i) => i.status === 'overdue')

  return (
    <Section title={title}>
      <div className="grid gap-4 md:grid-cols-3">
        <StatTile label={t.dashboard.overdue} value={overdue.length} hint={t.dashboard.overdueHint(list.length)} />
        <Card title={t.dashboard.byStatus}>
          <CountBars counts={byStatus} color={installationStatusColor} />
        </Card>
        <Card title={t.dashboard.tracked}>
          {/* Every entity with a schedule; the API lists the most overdue first. */}
          <EntityLinks
            items={list.map((i) => ({
              id: i.entity_id,
              name: names.get(i.entity_id),
              color: installationStatusColor(i.status),
              detail:
                i.days_late > 0
                  ? `${value(i.status)} · ${t.installation.late(t.common.days(i.days_late))}`
                  : value(i.status),
              danger: i.days_late > 0,
            }))}
          />
        </Card>
      </div>
    </Section>
  )
}

/** Operating zones: how many entities are outside, and which (farthest first). */
function ZoneOverview({ entities }: { entities: Entity[] | undefined }) {
  const { t, locale } = useI18n()
  const zones = useGeofences()
  const title = t.dashboard.zones

  if (zones.isPending || !entities) return <Loading title={title} />
  if (zones.isError) {
    return <Failed title={title} error={zones.error} onRetry={() => void zones.refetch()} />
  }
  if (zones.data.length === 0) {
    return (
      <Section title={title}>
        <p className="text-sm text-gray-500">{t.dashboard.noZones}</p>
      </Section>
    )
  }

  const names = new Map(entities.map((e) => [e.id, e.name]))
  const outside = zones.data.filter((z) => !z.inside)
  const byStatus = [
    { value: 'outside', count: outside.length },
    { value: 'inside', count: zones.data.length - outside.length },
  ]

  return (
    <Section title={title}>
      <div className="grid gap-4 md:grid-cols-3">
        <StatTile
          label={t.dashboard.outsideZone}
          value={outside.length}
          hint={t.dashboard.outsideZoneHint(zones.data.length)}
        />
        <Card title={t.dashboard.byStatus}>
          <CountBars counts={byStatus} color={zoneStatusColor} />
        </Card>
        <Card title={t.dashboard.withZone}>
          {/* Every vehicle with a zone; the API lists those outside first, farthest first. */}
          <EntityLinks
            items={zones.data.map((z) => ({
              id: z.entity_id,
              name: names.get(z.entity_id),
              color: zoneStatusColor(z.inside ? 'inside' : 'outside'),
              detail: z.inside
                ? t.zone.inside(formatDistance(z.distance_m, locale))
                : t.zone.outside(formatDistance(z.distance_m - z.radius_m, locale)),
              danger: !z.inside,
            }))}
          />
        </Card>
      </div>
    </Section>
  )
}

/** Same colors as the zone circles on the map: gray inside, red outside. */
const zoneStatusColor = (status: string) => (status === 'outside' ? '#dc2626' : '#6b7280')

/** Accounts with an active session; not links, since users are not on the map. */
function ActiveUserList({ users }: { users: ActiveUser[] }) {
  const i18n = useI18n()
  const { t } = i18n
  if (users.length === 0) return <p className="text-xs text-gray-500">{t.dashboard.noActiveUsers}</p>
  return (
    <ul className="-mx-2 max-h-48 space-y-0.5 overflow-y-auto text-xs">
      {users.map((u) => (
        <li key={u.id} className="flex items-center gap-2 rounded-md px-2 py-1.5">
          <span className={`h-2 w-2 shrink-0 rounded-full ${u.online ? 'bg-green-600' : 'bg-gray-300'}`} />
          <span className="min-w-0 flex-1 truncate text-gray-800" title={u.email}>
            {u.email}
          </span>
          <span className={`shrink-0 ${u.online ? 'font-medium text-green-700' : 'text-gray-500'}`}>
            {u.online
              ? t.dashboard.onlineNow
              : u.last_seen_at
                ? timeAgo(u.last_seen_at, i18n)
                : t.dashboard.neverSeen}
          </span>
        </li>
      ))}
    </ul>
  )
}

function Section({ title, aside, children }: { title: string; aside?: string; children: ReactNode }) {
  return (
    <section aria-label={title} className="animate-fade-in-up">
      <div className="mb-3 flex flex-wrap items-baseline justify-between gap-x-4 gap-y-1">
        <h2 className="text-base font-semibold text-gray-900">{title}</h2>
        {aside && <p className="text-xs text-gray-500">{aside}</p>}
      </div>
      {children}
    </section>
  )
}

function Loading({ title }: { title: string }) {
  const { t } = useI18n()
  return (
    <Section title={title}>
      <p className="text-sm text-gray-500">{t.common.loading}</p>
    </Section>
  )
}

function Failed({ title, error, onRetry }: { title: string; error: unknown; onRetry: () => void }) {
  const { t, errorText } = useI18n()
  return (
    <Section title={title}>
      <div role="alert" className="flex items-center gap-3 text-sm text-red-700">
        <span>{errorText(error)}</span>
        <button type="button" onClick={onRetry} className="font-medium underline">
          {t.common.retry}
        </button>
      </div>
    </Section>
  )
}

function Card({ title, children }: { title: string; children: ReactNode }) {
  return (
    <div className="rounded-xl bg-white p-4 shadow-sm ring-1 ring-black/5">
      <h3 className="mb-3 text-sm font-medium text-gray-500">{title}</h3>
      {children}
    </div>
  )
}

function StatTile({
  label,
  value,
  hint,
  children,
}: {
  label: string
  value: number
  hint?: string
  /** Extra content under the number, e.g. the list of entities behind it. */
  children?: ReactNode
}) {
  const { locale } = useI18n()
  return (
    <div className="rounded-xl bg-white p-4 shadow-sm ring-1 ring-black/5">
      <p className="text-sm font-medium text-gray-500">{label}</p>
      <p className="mt-1 text-3xl font-semibold text-gray-900">{value.toLocaleString(locale)}</p>
      {hint && <p className="mt-1 text-xs text-gray-500">{hint}</p>}
      {children && <div className="mt-3 border-t border-gray-100 pt-3">{children}</div>}
    </div>
  )
}

/** Opens an entity on the map; provided by Dashboard so every list can link to the map. */
const OpenEntityContext = createContext<(id: string) => void>(() => {})

interface EntityLink {
  id: string
  /** Undefined when the entity is not in the loaded list (e.g. just deleted). */
  name?: string
  /** Dot color: the entity's status, installation status, … */
  color: string
  detail: string
  /** Red detail text (late, outside the zone). */
  danger?: boolean
}

/**
 * Entities as rows that open the entity on the map. Long lists scroll inside
 * the card; the dot is backed by text, so color is never the only signal.
 */
function EntityLinks({ items, empty }: { items: EntityLink[]; empty?: string }) {
  const { t } = useI18n()
  const openEntity = useContext(OpenEntityContext)
  if (items.length === 0) return <p className="text-xs text-gray-500">{empty}</p>
  return (
    <ul className="-mx-2 max-h-48 space-y-0.5 overflow-y-auto text-xs">
      {items.map((item) => {
        const name = item.name ?? t.dashboard.unknownEntity
        return (
          <li key={item.id}>
            <button
              type="button"
              onClick={() => openEntity(item.id)}
              disabled={item.name === undefined}
              title={t.dashboard.openOnMap(name)}
              className="group flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-left transition-colors hover:bg-gray-100 disabled:cursor-default disabled:hover:bg-transparent"
            >
              <span className="h-2 w-2 shrink-0 rounded-full" style={{ backgroundColor: item.color }} />
              <span className="min-w-0 flex-1 truncate text-gray-800 group-hover:underline">{name}</span>
              <span className={`shrink-0 ${item.danger ? 'font-medium text-red-600' : 'text-gray-500'}`}>
                {item.detail}
              </span>
              <svg viewBox="0 0 20 20" className="h-3.5 w-3.5 shrink-0 text-gray-300 group-hover:text-gray-600" fill="currentColor" aria-hidden="true">
                <path
                  fillRule="evenodd"
                  d="M8.22 5.22a.75.75 0 0 1 1.06 0l4.25 4.25a.75.75 0 0 1 0 1.06l-4.25 4.25a.75.75 0 0 1-1.06-1.06L11.94 10 8.22 6.28a.75.75 0 0 1 0-1.06Z"
                  clipRule="evenodd"
                />
              </svg>
            </button>
          </li>
        )
      })}
    </ul>
  )
}

/**
 * Horizontal bars, one per value, scaled to the largest count. Each row is
 * labeled in text, so color is never the only way to tell rows apart.
 */
function CountBars({ counts, color }: { counts: Count[]; color: (value: string) => string }) {
  const { value: label } = useI18n()
  const max = Math.max(1, ...counts.map((c) => c.count))
  return (
    <ul className="space-y-2.5">
      {counts.map(({ value, count }) => (
        <li key={value} title={`${label(value)}: ${count}`}>
          <div className="mb-1 flex justify-between text-xs">
            <span className="text-gray-700">{label(value)}</span>
            <span className="font-medium tabular-nums text-gray-900">{count}</span>
          </div>
          <div className="h-2 rounded-full bg-gray-100">
            {count > 0 && (
              <div
                className="h-2 animate-grow-x rounded-full transition-[width] duration-500 ease-out"
                style={{ width: `${(count / max) * 100}%`, backgroundColor: color(value) }}
              />
            )}
          </div>
        </li>
      ))}
    </ul>
  )
}
