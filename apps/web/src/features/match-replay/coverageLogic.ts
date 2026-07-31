/**
 * coverageLogic.ts — ce que le rejeu a rattaché SUR CE QUI EXISTAIT, et s'il est publiable.
 *
 * POURQUOI CE FICHIER EXISTE. Le rejeu publiait un nombre de tirs sans jamais dire combien le
 * film en portait. Un lecteur ne pouvait donc pas distinguer « la partie a compté 475 tirs » de
 * « on en a rattaché 475 sur 519 ». C'est le défaut que l'artefact corrige côté serveur ; ce
 * module le rend lisible côté écran.
 *
 * LE DÉNOMINATEUR N'EST PAS LE NOMBRE DE TIRS DE LA PARTIE. Le film ne sérialise que les tirs
 * qui infligent un dégât : 519 records pour 2 228 tirs effectués et 595 touchés sur le match
 * témoin. Présenter 475/519 comme « 91 % des tirs du match » serait faux — c'est 91 % de ce que
 * le film porte.
 *
 * LOGIQUE PURE : aucun React, aucun canvas. Testable sans rendu.
 */
import type { ReplayCoverage, ReplayLayerCoverage } from '@/lib/api/types'

import type { ReplayDocumentReady } from './replayNormalize'

/** Verdict d'un calque, tel que le serveur le publie. */
export const VERDICT_NOMINAL = 'nominal'

/** Un calque prêt à l'affichage : son rapport, son taux, son verdict. */
export interface LayerSummary {
  /** Clé du calque ('shots' | 'grenades'), pour l'i18n. */
  key: string
  attached: number
  available: number
  /** Taux dans [0,1] ; 0 quand rien n'est disponible. */
  ratio: number
  /** Verdict publié par le serveur ; undefined si l'artefact est antérieur. */
  verdict?: string
  /** Vrai quand le verdict est nominal — l'écran peut alors peindre franchement. */
  nominal: boolean
  /** Rejets par cause, dans l'ordre où ils orientent vers un chantier différent. */
  rejects: { cause: string; n: number }[]
}

/**
 * ratioOf rend le taux de rattachement d'un calque. Sans disponible, le taux vaut 0 — jamais 1 :
 * « rien à rattacher » n'est pas « tout rattaché ».
 */
export function ratioOf(c: ReplayLayerCoverage): number {
  if (!c || c.available <= 0) return 0
  return c.attached / c.available
}

/**
 * isBalanced vérifie l'invariant du serveur : rattachés + rejetés = disponibles, exactement.
 * Une somme fausse signale une fuite — un chemin de rejet non compté. L'écran ne doit pas la
 * masquer : c'est le symptôme que le décodeur perd des événements sans le savoir.
 */
export function isBalanced(c: ReplayLayerCoverage): boolean {
  if (!c) return true
  return c.attached + c.noSlot + c.ambiguous + c.outOfWindow + c.unpublished === c.available
}

/** summarize prépare un calque pour l'affichage. */
export function summarize(
  key: string,
  c: ReplayLayerCoverage | undefined,
  verdict: string | undefined,
): LayerSummary | null {
  if (!c) return null
  return {
    key,
    attached: c.attached,
    available: c.available,
    ratio: ratioOf(c),
    verdict,
    nominal: verdict === VERDICT_NOMINAL && isBalanced(c),
    // L'ordre n'est pas cosmétique : chaque cause désigne un chantier distinct — le pont
    // (slot introuvable), le découpage des vies (ambigu), le décodage des positions (hors
    // fenêtre), la publication des traces (sans trajectoire).
    rejects: [
      { cause: 'noSlot', n: c.noSlot },
      { cause: 'ambiguous', n: c.ambiguous },
      { cause: 'outOfWindow', n: c.outOfWindow },
      { cause: 'unpublished', n: c.unpublished },
    ].filter((r) => r.n > 0),
  }
}

/** summarizeAll prépare tous les calques d'un document. */
export function summarizeAll(doc: ReplayDocumentReady | undefined): LayerSummary[] {
  const cov: ReplayCoverage | undefined = doc?.coverage
  if (!cov) return []
  return [
    summarize('shots', cov.shots, cov.verdict?.shots),
    summarize('grenades', cov.grenades, cov.verdict?.grenades),
  ].filter((s): s is LayerSummary => s !== null)
}

/**
 * bridgeIsRead dit si le pont trace → joueur repose ENTIÈREMENT sur une lecture.
 *
 * C'est la question qui décide si l'écran peut peindre franchement. Le pont a deux maillons,
 * tous deux lus : le fil des morts nomme chaque vie, et les cinq bits qui précèdent un xuid
 * donnent son index de joueur. Si une autre source venait à l'alimenter, `fromReading`
 * décrocherait de `slots` et l'écran doit le dire plutôt que de l'ignorer.
 */
export function bridgeIsRead(doc: ReplayDocumentReady | undefined): boolean {
  const b = doc?.coverage?.bridge
  if (!b || b.slots === 0) return false
  return (
    b.fromReading === b.slots && b.indexDisagreements === 0 && b.slotCollisions === 0
  )
}
