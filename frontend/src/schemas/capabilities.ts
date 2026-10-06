import type { Meta } from '../types/entity'

// Capability names the frontend reacts to. Which types have them comes only
// from GET /api/meta, so no entity type is hardcoded here.
export const CAP_INSTALLATION = 'installation'

export function hasCapability(meta: Meta | undefined, type: string, capability: string): boolean {
  return meta?.capabilities[type]?.includes(capability) ?? false
}
