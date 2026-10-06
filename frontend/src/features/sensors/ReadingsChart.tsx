import { useId, useMemo, useState, type PointerEvent } from 'react'
import type { MetricSpec } from '../../types/entity'
import type { Reading } from '../../types/sensor'

const WIDTH = 320
const HEIGHT = 96
const PAD = { top: 8, right: 8, bottom: 18, left: 34 }
const LINE = '#2563eb'

interface ReadingsChartProps {
  readings: Reading[]
  metric: MetricSpec
}

/**
 * Single-series line chart of recent readings (SVG, no library): a 2px line
 * over a light wash, a recessive axis, and a hover crosshair with the value.
 * One series, so the title names it instead of a legend.
 */
export function ReadingsChart({ readings, metric }: ReadingsChartProps) {
  const gradientId = useId()
  const [hover, setHover] = useState<number | null>(null)

  const points = useMemo(() => {
    const times = readings.map((r) => new Date(r.recorded_at).getTime())
    const values = readings.map((r) => r.value)
    const tMin = Math.min(...times)
    const tMax = Math.max(...times)
    // A little headroom so a flat line does not sit on the edge.
    const vLo = Math.min(...values)
    const vHi = Math.max(...values)
    const pad = Math.max((vHi - vLo) * 0.15, 0.5)
    const lo = vLo - pad
    const hi = vHi + pad
    const x = (t: number) => PAD.left + ((t - tMin) / Math.max(tMax - tMin, 1)) * (WIDTH - PAD.left - PAD.right)
    const y = (v: number) => PAD.top + (1 - (v - lo) / (hi - lo)) * (HEIGHT - PAD.top - PAD.bottom)
    return {
      list: readings.map((r, i) => ({ x: x(times[i]), y: y(r.value), reading: r })),
      lo: vLo,
      hi: vHi,
      yLo: y(vLo),
      yHi: y(vHi),
      tMin,
      tMax,
    }
  }, [readings])

  if (readings.length < 2) {
    return <p className="text-xs text-gray-400">Not enough data for a chart yet.</p>
  }

  const line = points.list.map((p, i) => `${i === 0 ? 'M' : 'L'}${p.x.toFixed(1)},${p.y.toFixed(1)}`).join(' ')
  const baseY = HEIGHT - PAD.bottom
  const area = `${line} L${points.list.at(-1)!.x.toFixed(1)},${baseY} L${points.list[0].x.toFixed(1)},${baseY} Z`
  const active = hover !== null ? points.list[hover] : null
  const time = (ms: number) => new Date(ms).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
  const fmt = (v: number) => `${v.toLocaleString(undefined, { maximumFractionDigits: 1 })} ${metric.unit}`

  const onMove = (event: PointerEvent<SVGSVGElement>) => {
    const box = event.currentTarget.getBoundingClientRect()
    const px = ((event.clientX - box.left) / box.width) * WIDTH
    let nearest = 0
    for (let i = 1; i < points.list.length; i++) {
      if (Math.abs(points.list[i].x - px) < Math.abs(points.list[nearest].x - px)) nearest = i
    }
    setHover(nearest)
  }

  return (
    <figure className="relative">
      <svg
        viewBox={`0 0 ${WIDTH} ${HEIGHT}`}
        className="h-auto w-full touch-none select-none"
        role="img"
        aria-label={`${metric.label} over the last 24 hours, from ${fmt(points.lo)} to ${fmt(points.hi)}`}
        onPointerMove={onMove}
        onPointerLeave={() => setHover(null)}
      >
        <defs>
          <linearGradient id={gradientId} x1="0" x2="0" y1="0" y2="1">
            <stop offset="0%" stopColor={LINE} stopOpacity="0.15" />
            <stop offset="100%" stopColor={LINE} stopOpacity="0" />
          </linearGradient>
        </defs>
        {/* Recessive axis: min/max guides and the time span. */}
        {[points.yHi, points.yLo].map((gy) => (
          <line key={gy} x1={PAD.left} x2={WIDTH - PAD.right} y1={gy} y2={gy} stroke="#e5e7eb" strokeWidth="1" />
        ))}
        <text x={PAD.left - 4} y={points.yHi + 3} textAnchor="end" className="fill-gray-400 text-[9px]">
          {points.hi.toLocaleString(undefined, { maximumFractionDigits: 1 })}
        </text>
        <text x={PAD.left - 4} y={points.yLo + 3} textAnchor="end" className="fill-gray-400 text-[9px]">
          {points.lo.toLocaleString(undefined, { maximumFractionDigits: 1 })}
        </text>
        <text x={PAD.left} y={HEIGHT - 4} className="fill-gray-400 text-[9px]">
          {time(points.tMin)}
        </text>
        <text x={WIDTH - PAD.right} y={HEIGHT - 4} textAnchor="end" className="fill-gray-400 text-[9px]">
          {time(points.tMax)}
        </text>

        <path d={area} fill={`url(#${gradientId})`} />
        <path d={line} fill="none" stroke={LINE} strokeWidth="2" strokeLinejoin="round" strokeLinecap="round" />

        {active && (
          <g>
            <line x1={active.x} x2={active.x} y1={PAD.top} y2={baseY} stroke="#9ca3af" strokeWidth="1" />
            <circle cx={active.x} cy={active.y} r="4" fill={LINE} stroke="white" strokeWidth="2" />
          </g>
        )}
      </svg>
      {active && (
        <figcaption
          className="pointer-events-none absolute top-0 rounded bg-gray-900 px-2 py-1 text-[11px] whitespace-nowrap text-white shadow"
          style={{
            left: `${(active.x / WIDTH) * 100}%`,
            transform: `translateX(${active.x > WIDTH / 2 ? 'calc(-100% - 6px)' : '6px'})`,
          }}
        >
          {fmt(active.reading.value)} · {time(new Date(active.reading.recorded_at).getTime())}
        </figcaption>
      )}
    </figure>
  )
}
