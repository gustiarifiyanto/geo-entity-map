import { useState } from 'react'
import { ConfirmDialog } from '../../components/ConfirmDialog'
import { useToast } from '../../components/toast/context'
import type { DeviceKey, SensorConfig } from '../../types/sensor'
import { useCreateDeviceKey } from './hooks'

interface DeviceKeyPanelProps {
  entityId: string
  sensor: SensorConfig
}

/**
 * Creates the API key a real device uses to send readings. Unlike the rest of
 * the form this acts immediately: the key is shown exactly once, here, and
 * only kept in this component's state until the form closes.
 */
export function DeviceKeyPanel({ entityId, sensor }: DeviceKeyPanelProps) {
  const create = useCreateDeviceKey(entityId)
  const toast = useToast()
  const [key, setKey] = useState<DeviceKey | null>(null)
  const [confirming, setConfirming] = useState(false)
  const hasKey = sensor.has_api_key || key !== null

  const generate = async () => {
    try {
      setKey(await create.mutateAsync())
    } catch (error) {
      toast.error(`Could not create the key: ${error instanceof Error ? error.message : 'unknown error'}`)
    } finally {
      setConfirming(false)
    }
  }

  const endpoint = `${window.location.origin}/api/devices/${entityId}/readings`

  return (
    <fieldset className="rounded-lg border border-gray-200 p-3">
      <legend className="px-1 text-sm font-medium text-gray-700">Device API key</legend>
      <div className="flex items-center justify-between gap-3 text-xs">
        <span className="text-gray-600">
          {hasKey
            ? `Active since ${new Date((key?.key_created_at ?? sensor.key_created_at) as string).toLocaleString()}`
            : 'No key yet. The device needs one to send readings.'}
        </span>
        <button
          type="button"
          onClick={() => (hasKey ? setConfirming(true) : void generate())}
          disabled={create.isPending}
          className="shrink-0 rounded-lg border border-gray-300 px-3 py-1.5 font-medium text-gray-700 transition-colors hover:bg-gray-50 disabled:opacity-50"
        >
          {create.isPending ? 'Generating…' : hasKey ? 'Regenerate' : 'Generate API key'}
        </button>
      </div>

      {key && (
        <div className="mt-3 animate-fade-in space-y-2 rounded-lg bg-amber-50 p-3 text-xs ring-1 ring-amber-200">
          <p className="font-medium text-amber-900">Copy this key now. It will not be shown again.</p>
          <div className="flex items-center gap-2">
            <code className="min-w-0 flex-1 truncate rounded bg-white px-2 py-1 font-mono text-gray-900 ring-1 ring-amber-200" title={key.api_key}>
              {key.api_key}
            </code>
            <CopyButton text={key.api_key} label="Copy key" />
          </div>
          <p className="text-amber-900">The device sends each reading like this:</p>
          <div className="flex items-start gap-2">
            <pre className="min-w-0 flex-1 overflow-x-auto rounded bg-white p-2 font-mono text-[11px] text-gray-800 ring-1 ring-amber-200">
              {`curl -X POST ${endpoint} \\\n  -H "Authorization: Bearer ${key.api_key}" \\\n  -H "Content-Type: application/json" \\\n  -d '{"value": 27.5}'`}
            </pre>
          </div>
        </div>
      )}

      <ConfirmDialog
        open={confirming}
        title="Regenerate the device key?"
        confirmLabel="Regenerate"
        pendingLabel="Generating…"
        pending={create.isPending}
        onConfirm={() => void generate()}
        onCancel={() => setConfirming(false)}
      >
        <p>The current key stops working immediately. The device sends nothing until it is given the new key.</p>
      </ConfirmDialog>
    </fieldset>
  )
}

function CopyButton({ text, label }: { text: string; label: string }) {
  const [copied, setCopied] = useState(false)
  return (
    <button
      type="button"
      onClick={() => {
        void navigator.clipboard.writeText(text).then(
          () => {
            setCopied(true)
            setTimeout(() => setCopied(false), 2000)
          },
          () => setCopied(false),
        )
      }}
      className="shrink-0 rounded-md bg-gray-900 px-2 py-1 font-medium text-white transition-colors hover:bg-gray-700"
    >
      {copied ? 'Copied' : label}
    </button>
  )
}
