/**
 * useReplayEngineSound — LA COUTURE React des moteurs de véhicules, consommée par
 * `useReplaySound` (qui possède le lecteur Web Audio et le battement).
 *
 * Extraite dans son propre fichier pour la même raison que les autres sous-hooks du son :
 * `useReplaySound.ts` est au plafond de taille du dépôt, et ce câblage-ci a une frontière
 * nette — il ne connaît ni la piste d'événements, ni les préférences ; seulement le plan
 * moteur (`vehicleEngineSound.ts`) et son lecteur (`vehicleEnginePlayer.ts`).
 *
 * TOUT EST LU DANS DES RÉFÉRENCES par des rappels stables : `sync` bat à chaque pas
 * d'animation et ne doit jamais recréer la boucle du canvas (même règle que `tick`).
 */
import { useCallback, useEffect, useMemo, useRef } from 'react'

import { soundUrlOf } from './replayAudioMix'
import type { ReplayAudioPlayer } from './replayAudio'
import { frameToMs } from '../replayLogic'
import type { ReplayDocumentReady } from '../replayNormalize'
import { VehicleEnginePlayer } from './vehicleEnginePlayer'
import {
  planVehicleEngines,
  VEHICLE_ENGINE_BUS_GAIN,
  type EnginePlan,
} from './vehicleEngineSound'
import { engineStemsOf } from './vehicleEngineMix'

export interface ReplayEngineSound {
  /** Les stems à PRÉCHARGER avec la piste : un moteur demandé à T0 sonnerait en retard. */
  stems: string[]
  /** Crée le lecteur moteur sur le lecteur principal. Idempotent (rappelé à chaque geste). */
  attach: (player: ReplayAudioPlayer) => void
  /** Fait suivre les moteurs à l'instant du rejeu — à appeler quand la piste est audible. */
  sync: (ms: number) => void
  /** Éteint tout (pause, coupure, vitesse muette, démontage) : rampe courte, jamais un clic. */
  stop: () => void
  /** Le plan, pour l'export hors temps réel (lu à l'appel, jamais au rendu). */
  plansForExport: () => readonly EnginePlan[]
}

export function useReplayEngineSound(doc: ReplayDocumentReady): ReplayEngineSound {
  const plans = useMemo(() => planVehicleEngines(doc.vehicles, frameToMs(1, doc)), [doc])
  const stems = useMemo(() => engineStemsOf(plans), [plans])

  const engineRef = useRef<VehicleEnginePlayer | null>(null)
  const plansRef = useRef(plans)
  useEffect(() => {
    plansRef.current = plans
    // Un document qui change sous un lecteur vivant : le plan se remplace, ce qui sonnait
    // s'éteint (setPlans arrête tout), et la lecture suivante repart du bon film.
    engineRef.current?.setPlans(plans)
  }, [plans])

  // Le lecteur moteur meurt AVEC le composant, comme le lecteur principal juste au-dessus.
  useEffect(() => () => {
    engineRef.current?.dispose()
    engineRef.current = null
  }, [])

  const attach = useCallback((player: ReplayAudioPlayer) => {
    if (engineRef.current) return
    const bus = player.engineBus(VEHICLE_ENGINE_BUS_GAIN)
    const engine = new VehicleEnginePlayer({
      ctx: bus.ctx,
      out: bus.out,
      // Le cache du lecteur principal est indexé par URL ; le plan parle en stems.
      bufferOf: (stem) => bus.bufferOf(soundUrlOf(stem)),
    })
    engine.setPlans(plansRef.current)
    engineRef.current = engine
  }, [])

  const sync = useCallback((ms: number) => engineRef.current?.sync(ms), [])
  const stop = useCallback(() => engineRef.current?.stopAll(), [])
  const plansForExport = useCallback(() => plansRef.current, [])

  return { stems, attach, sync, stop, plansForExport }
}
