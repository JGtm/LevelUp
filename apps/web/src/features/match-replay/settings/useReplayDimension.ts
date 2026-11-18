/**
 * useReplayDimension — L'EMPLACEMENT DU COMMUTATEUR 2D / 3D du rejeu : si le serveur offre la vue 3D, et
 * laquelle des deux vues est choisie.
 *
 * L'EMPLACEMENT N'EXISTE À L'ÉCRAN QUE PAR LA CAPACITÉ : `replay3dCapabilityOf` est le SEUL endroit qui la
 * lit, et elle vaut `false` tant que le serveur ne la publie pas. Ni drapeau ni constante « à activer » : le
 * segment (`ReplayDimensionSwitch`) ne se rend pas sans elle.
 *
 * HYPOTHÈSE À CONFIRMER PAR LE LOT 3D : le nom du champ de `capabilities` (`can_view_replay_3d`). Le contrat de
 * l'API ne le porte pas aujourd'hui ; la lecture est donc défensive (un booléen vrai, et rien d'autre) et ne
 * s'appuie sur aucun type généré. Le jour où le serveur publie la capacité sous son vrai nom, seule la
 * constante change, ou la lecture passe par `useDeploymentCapability`.
 *
 * Ce hook ne porte AUCUNE logique de vue 3D : il tient le choix (`value`, 2D au départ) que le segment montre
 * et que la vue 3D lira pour poser son contenu à la place de la toile.
 */
import { useState } from 'react'

import { useAppShellStore } from '@/stores/appShellStore'

/** Les deux vues du rejeu. */
export type ReplayDimension = '2d' | '3d'

/** Le champ de `capabilities` (`/bootstrap`) qui offre la vue 3D : un nom supposé, pas encore servi. */
export const REPLAY_3D_CAPABILITY_KEY = 'can_view_replay_3d'

/** La vue 3D est-elle offerte ? Vrai seulement pour un booléen vrai ; absent, autre type ou pas de bootstrap : faux. */
export function replay3dCapabilityOf(capabilities: object | null | undefined): boolean {
  if (!capabilities) return false
  return (capabilities as Record<string, unknown>)[REPLAY_3D_CAPABILITY_KEY] === true
}

/** Ce que le segment reçoit : si l'emplacement existe, la vue choisie, et le geste qui la change. */
export interface ReplayDimensionState {
  available: boolean
  value: ReplayDimension
  onChange: (value: ReplayDimension) => void
}

export function useReplayDimension(): ReplayDimensionState {
  const available = useAppShellStore((s) => replay3dCapabilityOf(s.capabilities))
  const [value, onChange] = useState<ReplayDimension>('2d')
  return { available, value, onChange }
}
