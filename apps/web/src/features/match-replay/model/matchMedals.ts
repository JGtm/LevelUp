/**
 * matchMedals — LA LECTURE « MÉDAILLES DU MATCH », depuis le document de rejeu.
 *
 * Une seule source : `medals[]` du document (qui a gagné quoi, et quand, sur l'horloge des temps forts
 * du fil des éliminations) et sa table `medalLabels` (libellé et description par langue, image), posée par
 * le serveur à la lecture du document. Le fil du rejeu et la fiche d'un joueur lisent ici, jamais les
 * événements du match : un match dont le document ne porte pas de `medals` n'affiche aucune médaille, sans
 * repli sur une autre source.
 *
 * Une médaille que la table ne connaît pas garde le nom du film (`name`, anglais) et un libellé vide :
 * l'affichage écrit alors ce nom tel quel, il n'invente ni libellé ni visuel.
 */
import { FALLBACK_LOCALE, type Locale } from '@/lib/i18n/locale'
import { pickText } from '@/lib/i18n/pickText'

import type { ReplayDocumentReady } from '../../../lib/replay/replayNormalize'
import type { MedalEvent } from './killFeedLogic'

/**
 * Les médailles du match, dans l'ordre du temps. Une médaille sans joueur ou sans nom (indice absent de la
 * table du décodeur) est écartée : il n'y a rien à en montrer, pas même un texte. Le gamertag n'est pas
 * dans le document : il reste vide, le fil le retrouve par le xuid dans la feuille de score.
 */
export function readMatchMedals(
  doc: ReplayDocumentReady | null | undefined,
  locale: Locale = FALLBACK_LOCALE,
): MedalEvent[] {
  if (!doc) return []
  const out: MedalEvent[] = []
  for (const m of doc.medals) {
    if (!m.xuid || !m.name) continue
    const entry = doc.medalLabels?.[m.name]
    out.push({
      tMs: m.ms,
      xuid: m.xuid,
      gamertag: '',
      name: m.name,
      label: pickText(entry, locale),
      description: pickText(entry?.description, locale),
      imageUrl: entry?.img ?? '',
      sprite: entry?.sprite ?? null,
    })
  }
  return out.sort((a, b) => a.tMs - b.tMs)
}
