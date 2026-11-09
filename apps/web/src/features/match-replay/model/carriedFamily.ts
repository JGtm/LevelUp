/**
 * carriedFamily.ts — LA FAMILLE D'UN OBJET PORTÉ, lue là où le document la met.
 *
 * Depuis le schéma 107, le serveur publie la famille de chaque objet porté (`carriedEquipment[].family`) : il la
 * lit sur l'identité de l'objet (le tag du jeu que son record de création écrit), quelle que soit la façon dont
 * le joueur l'a obtenu — ramassé, équipement de réapparition ou de départ de match. Le rang `r` est celui de la
 * palette du film pour cette famille, ou -1 quand elle n'en nomme aucun : il n'est plus une identité.
 *
 * UN ARTEFACT ANTÉRIEUR n'a pas de `family` : sa famille est celle de son rang dans la table des capacités du
 * document (`abilityLabels[r].family`), comme avant. Une seule fonction fait ce choix, pour que les lecteurs
 * (objets engendrés, sons d'utilisation, compte des usages) ne le recopient pas.
 *
 * PUR : aucun React.
 */
import type { ReplayDocumentReady } from '../../../lib/replay/replayNormalize'

type CarriedObject = ReplayDocumentReady['carriedEquipment'][number]

/** carriedFamilyOf — la famille de l'objet porté `c`, ou undefined quand le document ne la dit pas. */
export function carriedFamilyOf(
  labels: ReplayDocumentReady['abilityLabels'],
  c: Pick<CarriedObject, 'r' | 'family'>,
): string | undefined {
  return c.family ?? labels?.[String(c.r)]?.family
}

/**
 * carriedIsAbilityRank — l'objet porté est-il la capacité de rang `rank` (celle que la vignette montre) ? Même
 * rang de palette, ou même famille quand les deux la disent.
 */
export function carriedIsAbilityRank(
  labels: ReplayDocumentReady['abilityLabels'],
  c: Pick<CarriedObject, 'r' | 'family'>,
  rank: number,
): boolean {
  if (c.r === rank) return true
  const family = c.family
  return family !== undefined && family === labels?.[String(rank)]?.family
}
