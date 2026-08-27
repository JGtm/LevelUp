/**
 * useReplayFx — LES EFFETS PRÉCALCULÉS DU FILM, en monde, une fois pour ce document.
 *
 * ONZIÈME EXTRACTION IMPOSÉE PAR LE CLIQUET DE TAILLE (`placementFamily.guard.test.ts`) : le
 * canvas était PILE à son plafond et le lot C y fait entrer la fin de partie sonore. Ces cinq
 * mémos partagent une seule et même nature — ils ne dépendent que du FILM (et, pour les morts,
 * du fil déjà résolu), ils ne lisent ni thème, ni palette, ni cadrage, et ils ne dessinent
 * rien : ce sont des LISTES D'ÉVÉNEMENTS placées en coordonnées monde, que la boucle se
 * contente ensuite de projeter. Le canvas garde le DESSIN.
 *
 * POURQUOI PRÉCALCULER, ET C'EST LE PATRON DU POC : pendant la lecture, seul le passage monde
 * -> pixels reste à faire. Relire les vies à chaque image pour retrouver le regard d'un tireur
 * ou la position d'une mort coûterait le même travail soixante fois par seconde.
 *
 * LES NOMS SORTENT INCHANGÉS, exactement comme à la neuvième extraction (`useReplayView`) :
 * pas une ligne du tracé ne bouge, et le diff du canvas se lit comme un déplacement.
 */
import { useMemo } from 'react'

import type { KillEvent } from '@/features/match-view/_momentum'

import { buildFireMarks } from './fireMark'
import { buildGrappleFx } from './grappleLayer'
import { buildGrenadeRestFx } from './grenadeFx'
import { buildKillFx } from './killFx'
import type { ReplayDocumentReady } from './replayNormalize'
import { buildShotFx } from './shotFx'

export function useReplayFx(
  doc: ReplayDocumentReady,
  kills: KillEvent[] | undefined,
  t0Ms: number | undefined,
  aimHold: number,
) {
  // Les tirs : famille, teinte et REGARD du tireur résolus une fois au chargement (mesure : la
  // couverture d'orientation passe de 18,6 % à 100 % sur le film témoin en relisant le regard
  // plutôt que le champ de l'événement).
  const shotFx = useMemo(() => buildShotFx(doc, aimHold), [doc, aimHold])
  // Le « ! » dans le point du tireur (demande du 2026-08-24) : mêmes tirs, même fenêtre que
  // l'éclair de bouche — deux effets du même événement (cf. fireMark.ts).
  const fireMarks = useMemo(() => buildFireMarks(doc), [doc])
  // Les effets de mort, positions relues une fois (patron POC).
  const killFx = useMemo(() => buildKillFx(doc, kills ?? [], t0Ms ?? 0), [doc, kills, t0Ms])
  // Fins de vol de grenade : le lien lancer -> projectile est dans l'artefact (v3).
  const grenadeRestFx = useMemo(() => buildGrenadeRestFx(doc), [doc])
  // Les tractions de grappin, jointes une fois aux points de leur vie (schéma 8).
  const grappleFx = useMemo(() => buildGrappleFx(doc), [doc])

  return { shotFx, fireMarks, killFx, grenadeRestFx, grappleFx }
}
