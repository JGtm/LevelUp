import { useAppShellStore } from '@/stores/appShellStore'
import {
  deploymentCapabilityOf,
  type DeploymentCapabilityKey,
} from '@/lib/capabilities/deploymentCapabilities'

/**
 * Une capacité du profil de déploiement, lue dans le store hydraté depuis /bootstrap
 * (ADR 0044). Hors composant (garde de route, effet) : `deploymentCapabilityOf` sur
 * `useAppShellStore.getState().capabilities`.
 */
export function useDeploymentCapability(key: DeploymentCapabilityKey): boolean {
  return useAppShellStore((s) => deploymentCapabilityOf(s.capabilities, key))
}
