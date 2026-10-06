import type { ReactNode } from 'react'
import type { Entity, Meta } from '../../types/entity'
import { formatLabel, statusColor } from '../entities/labels'
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

export function Dashboard({ entities, meta, showUserStats }: DashboardProps) {
  return (
    <div className="mx-auto max-w-5xl space-y-8 px-4 pb-8">
      <EntitySummary entities={entities} meta={meta} />
      {showUserStats && <UserSummary />}
    </div>
  )
}

function EntitySummary({ entities, meta }: Pick<DashboardProps, 'entities' | 'meta'>) {
  if (!entities || !meta) {
    return (
      <Section title="Entities">
        <p className="text-sm text-gray-500">Loading…</p>
      </Section>
    )
  }

  return (
    <Section title="Entities">
      <div className="grid gap-4 md:grid-cols-3">
        <StatTile label="Total entities" value={entities.length} />
        <Card title="By status">
          <CountBars counts={countBy(entities, 'status', meta.statuses)} color={statusColor} />
        </Card>
        <Card title="By type">
          <CountBars counts={countBy(entities, 'type', meta.types)} color={() => NEUTRAL_BAR} />
        </Card>
      </div>
    </Section>
  )
}

function UserSummary() {
  const stats = useAdminStats()

  if (stats.isPending) {
    return (
      <Section title="Users">
        <p className="text-sm text-gray-500">Loading…</p>
      </Section>
    )
  }
  if (stats.isError) {
    return (
      <Section title="Users">
        <div role="alert" className="flex items-center gap-3 text-sm text-red-700">
          <span>{stats.error.message}</span>
          <button type="button" onClick={() => void stats.refetch()} className="font-medium underline">
            Retry
          </button>
        </div>
      </Section>
    )
  }

  const { users, online_window_minutes: window } = stats.data
  const roles = Object.entries(users.by_role).map(([value, count]) => ({ value, count }))
  return (
    <Section
      title="Users"
      aside={`Updated ${new Date(stats.dataUpdatedAt).toLocaleTimeString()} · refreshes every 30 s`}
    >
      <div className="grid gap-4 sm:grid-cols-2 md:grid-cols-4">
        <StatTile label="Registered users" value={users.total} />
        <StatTile
          label={`Online (last ${window} min)`}
          value={users.online}
          hint={`Made a request in the last ${window} minutes.`}
        />
        <StatTile
          label="With an active session"
          value={users.with_active_session}
          hint="Logged in and not logged out or expired yet. Includes closed browsers."
        />
        <Card title="By role">
          <CountBars counts={roles} color={() => NEUTRAL_BAR} />
        </Card>
      </div>
    </Section>
  )
}

function Section({ title, aside, children }: { title: string; aside?: string; children: ReactNode }) {
  return (
    <section aria-label={title}>
      <div className="mb-3 flex flex-wrap items-baseline justify-between gap-x-4 gap-y-1">
        <h2 className="text-base font-semibold text-gray-900">{title}</h2>
        {aside && <p className="text-xs text-gray-500">{aside}</p>}
      </div>
      {children}
    </section>
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
  return (
    <div className="rounded-xl bg-white p-4 shadow-sm ring-1 ring-black/5">
      <p className="text-sm font-medium text-gray-500">{label}</p>
      <p className="mt-1 text-3xl font-semibold text-gray-900">{value.toLocaleString()}</p>
      {hint && <p className="mt-1 text-xs text-gray-500">{hint}</p>}
    </div>
  )
}

/**
 * Horizontal bars, one per value, scaled to the largest count. Each row is
 * labeled in text, so color is never the only way to tell rows apart.
 */
function CountBars({ counts, color }: { counts: Count[]; color: (value: string) => string }) {
  const max = Math.max(1, ...counts.map((c) => c.count))
  return (
    <ul className="space-y-2.5">
      {counts.map(({ value, count }) => (
        <li key={value} title={`${formatLabel(value)}: ${count}`}>
          <div className="mb-1 flex justify-between text-xs">
            <span className="text-gray-700">{formatLabel(value)}</span>
            <span className="font-medium tabular-nums text-gray-900">{count}</span>
          </div>
          <div className="h-2 rounded-full bg-gray-100">
            {count > 0 && (
              <div
                className="h-2 rounded-full"
                style={{ width: `${(count / max) * 100}%`, backgroundColor: color(value) }}
              />
            )}
          </div>
        </li>
      ))}
    </ul>
  )
}
