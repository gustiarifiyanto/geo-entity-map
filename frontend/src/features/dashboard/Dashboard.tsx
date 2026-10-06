import type { ReactNode } from 'react'
import type { Entity, Meta } from '../../types/entity'
import { formatLabel, statusColor } from '../entities/labels'
import { CAP_INSTALLATION } from '../../schemas/capabilities'
import { useInstallations } from '../installations/hooks'
import { daysText, installationStatusColor, installationStatusLabel, installationStatusOrder } from '../installations/status'
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
      {Object.values(meta?.capabilities ?? {}).some((caps) => caps.includes(CAP_INSTALLATION)) && (
        <InstallationOverview entities={entities} />
      )}
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
        <StatTile label="Registered users" value={users.total} hint="Role user only. Admins are listed under By role." />
        <StatTile
          label={`Online (last ${window} min)`}
          value={users.online}
          hint={`Users who made a request in the last ${window} minutes.`}
        />
        <StatTile
          label="With an active session"
          value={users.with_active_session}
          hint="Users logged in and not logged out or expired yet. Includes closed browsers."
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
    <section aria-label={title} className="animate-fade-in-up">
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
function CountBars({
  counts,
  color,
  label = formatLabel,
}: {
  counts: Count[]
  color: (value: string) => string
  label?: (value: string) => string
}) {
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

/** Installation schedules of every entity type that has them: counts per status and the overdue list. */
function InstallationOverview({ entities }: { entities: Entity[] | undefined }) {
  const installations = useInstallations()

  if (installations.isPending || !entities) {
    return (
      <Section title="Installations">
        <p className="text-sm text-gray-500">Loading…</p>
      </Section>
    )
  }
  if (installations.isError) {
    return (
      <Section title="Installations">
        <div role="alert" className="flex items-center gap-3 text-sm text-red-700">
          <span>{installations.error.message}</span>
          <button type="button" onClick={() => void installations.refetch()} className="font-medium underline">
            Retry
          </button>
        </div>
      </Section>
    )
  }

  const list = installations.data
  if (list.length === 0) {
    return (
      <Section title="Installations">
        <p className="text-sm text-gray-500">No installation schedules yet. Add one when creating or editing an entity that supports it.</p>
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
    <Section title="Installations">
      <div className="grid gap-4 md:grid-cols-3">
        <StatTile
          label="Overdue installations"
          value={overdue.length}
          hint={`Of ${list.length} tracked. Past the target date and not completed.`}
        />
        <Card title="By status">
          <CountBars counts={byStatus} color={installationStatusColor} label={installationStatusLabel} />
        </Card>
        <Card title="Most overdue">
          {overdue.length === 0 ? (
            <p className="text-xs text-gray-500">Nothing is overdue.</p>
          ) : (
            <ul className="space-y-2 text-xs">
              {overdue.slice(0, 5).map((i) => (
                <li key={i.entity_id} className="flex items-baseline justify-between gap-3">
                  <span className="min-w-0 truncate text-gray-700" title={names.get(i.entity_id)}>
                    {names.get(i.entity_id) ?? 'Unknown entity'}
                  </span>
                  <span className="shrink-0 font-medium text-red-600">{daysText(i.days_late)} late</span>
                </li>
              ))}
              {overdue.length > 5 && <li className="text-gray-400">and {overdue.length - 5} more</li>}
            </ul>
          )}
        </Card>
      </div>
    </Section>
  )
}
