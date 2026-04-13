/**
 * Store du wizard de configuration initiale (Setup Flow).
 *
 * Gère l'état local de la navigation dans le wizard :
 * - Mode sélectionné (refresh_token vs device_code)
 * - Attempt ID du Device Code Flow en cours
 * - Job ID du smoke test en cours
 *
 * Le store ne reçoit pas le statut réel du setup (SetupStatusResponse) —
 * celui-ci est géré par TanStack Query avec la clé ['setup-status'].
 *
 * Cycle de vie recommandé :
 *   useQuery(['setup-status']) → reading next_blocking_step
 *   → useSetupFlowStore.setSelectedMode()
 *   → POST /auth/device-flow/start → setCurrentAttemptId()
 *   → poll ['device-flow', attemptId] until status = "provisioned"
 *   → POST /setup/players
 *   → POST /setup/smoke-test → setCurrentJobId()
 *   → poll ['job', jobId] until status = "succeeded"
 *   → invalidate ['setup-status']
 */

import { create } from 'zustand'

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

export type SetupAuthMode = 'refresh_token' | 'device_code'

interface SetupFlowState {
  /** Mode d'authentification choisi par l'utilisateur */
  selectedMode: SetupAuthMode | null

  /** attempt_id du Device Code Flow courant (null si aucun en cours) */
  currentAttemptId: string | null

  /** Code affiché à l'utilisateur (ex: "ABCD-1234") — depuis DeviceFlowStartResponse */
  deviceFlowUserCode: string | null

  /** URL de vérification (https://microsoft.com/devicelogin) */
  deviceFlowVerificationUri: string | null

  /** Gamertag résolu après succès du Device Code Flow */
  resolvedGamertag: string | null

  /** XUID résolu après succès du Device Code Flow */
  resolvedXuid: string | null

  /** job_id du smoke test courant (null si aucun en cours) */
  currentJobId: string | null

  // --- Actions ---
  setSelectedMode: (mode: SetupAuthMode) => void
  setCurrentAttemptId: (id: string | null) => void
  setDeviceFlowCodes: (userCode: string, verificationUri: string) => void
  setResolvedIdentity: (gamertag: string, xuid: string | null) => void
  setCurrentJobId: (id: string | null) => void
  reset: () => void
}

// ---------------------------------------------------------------------------
// Store initial
// ---------------------------------------------------------------------------

const INITIAL_STATE = {
  selectedMode: null as SetupAuthMode | null,
  currentAttemptId: null as string | null,
  deviceFlowUserCode: null as string | null,
  deviceFlowVerificationUri: null as string | null,
  resolvedGamertag: null as string | null,
  resolvedXuid: null as string | null,
  currentJobId: null as string | null,
}

// ---------------------------------------------------------------------------
// Zustand store
// ---------------------------------------------------------------------------

export const useSetupFlowStore = create<SetupFlowState>((set) => ({
  ...INITIAL_STATE,

  setSelectedMode: (mode) => set({ selectedMode: mode }),

  setCurrentAttemptId: (id) => set({ currentAttemptId: id }),

  setDeviceFlowCodes: (userCode, verificationUri) =>
    set({ deviceFlowUserCode: userCode, deviceFlowVerificationUri: verificationUri }),

  setResolvedIdentity: (gamertag, xuid) =>
    set({ resolvedGamertag: gamertag, resolvedXuid: xuid }),

  setCurrentJobId: (id) => set({ currentJobId: id }),

  reset: () => set(INITIAL_STATE),
}))
