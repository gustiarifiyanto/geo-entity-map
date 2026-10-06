import type { ReactNode } from 'react'
import { useI18n } from '../../i18n/context'
import { CAP_GEOFENCE, CAP_INSTALLATION } from '../../schemas/capabilities'
import type { Entity, Meta } from '../../types/entity'
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
}

const hasAny = (meta: Meta | undefined, capability: string) =>
  Object.values(meta?.capabilities ?? {}).some((caps) => caps.includes(capability))

export function Dashboard({ entities, meta, showUserStats }: DashboardProps) {
  return (
    <div className="mx-auto max-w-5xl space-y-8 px-4 pb-8">
      <EntitySummary entities={entities} meta={meta} />
      {hasAny(meta, CAP_INSTALLATION) && <InstallationOverview entities={entities} />}
      {hasAny(meta, CAP_GEOFENCE) && <ZoneOverview entities={entities} />}
      {showUserStats && <UserSummary />}
    </div>
  )
}

function EntitySummary({ entities, meta }: Pick<DashboardProps, 'entities' | 'meta'>) {
  const { t } = useI18n()
  if (!entities || !meta) return <Loading title={t.dashboard.entities} />

  return (
    <Section title={t.dashboard.entities}>
      <div className="grid gap-4 md:grid-cols-3">
        <StatTile label={t.dashboard.totalEntities} value={entities.length} />
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
        />
        <Card title={t.dashboard.byRole}>
          <CountBars counts={roles} color={() => NEUTRAL_BAR} />
        </Card>
      </div>
    </Section>
  )
}

/** Installation schedules of every entity type that has them: counts per status and the overdue list. */
function InstallationOverview({ entities }: { entities: Entity[] | undefined }) {
  const { t } = useI18n()
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
        <Card title={t.dashboard.mostOverdue}>
          <TopList
            items={overdue.map((i) => ({
              id: i.entity_id,
              name: names.get(i.entity_id),
              detail: t.installation.late(t.common.days(i.days_late)),
            }))}
            empty={t.dashboard.nothingOverdue}
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
  // The API lists entities outside their zone first, farthest past the edge first.
  const outside = zones.data.filter((z) => !z.inside)

  return (
    <Section title={title}>
      <div className="grid gap-4 md:grid-cols-3">
        <StatTile
          label={t.dashboard.outsideZone}
          value={outside.length}
          hint={t.dashboard.outsideZoneHint(zones.data.length)}
        />
        <div className="md:col-span-2">
          <Card title={t.dashboard.farthestOutside}>
            <TopList
              items={outside.map((z) => ({
                id: z.entity_id,
                name: names.get(z.entity_id),
                detail: t.dashboard.outsideBy(formatDistance(z.distance_m - z.radius_m, locale)),
              }))}
              empty={t.dashboard.allInside}
            />
          </Card>
        </div>
      </div>
    </Section>
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

function StatTile({ label, value, hint }: { label: string; value: number; hint?: string }) {
  const { locale } = useI18n()
  return (
    <div className="rounded-xl bg-white p-4 shadow-sm ring-1 ring-black/5">
      <p className="text-sm font-medium text-gray-500">{label}</p>
      <p className="mt-1 text-3xl font-semibold text-gray-900">{value.toLocaleString(locale)}</p>
      {hint && <p className="mt-1 text-xs text-gray-500">{hint}</p>}
    </div>
  )
}

/** The first five items of an already sorted list, with a red detail per item. */
function TopList({ items, empty }: { items: { id: string; name?: string; detail: string }[]; empty: string }) {
  const { t } = useI18n()
  if (items.length === 0) return <p className="text-xs text-gray-500">{empty}</p>
  return (
    <ul className="space-y-2 text-xs">
      {items.slice(0, 5).map((item) => (
        <li key={item.id} className="flex items-baseline justify-between gap-3">
          <span className="min-w-0 truncate text-gray-700" title={item.name}>
            {item.name ?? t.dashboard.unknownEntity}
          </span>
          <span className="shrink-0 font-medium text-red-600">{item.detail}</span>
        </li>
      ))}
      {items.length > 5 && <li className="text-gray-400">{t.dashboard.andMore(items.length - 5)}</li>}
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
