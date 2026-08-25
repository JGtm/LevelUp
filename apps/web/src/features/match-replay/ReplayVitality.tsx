/**
 * ReplayVitality — CE QU'UNE FICHE DIT DE L'ÉTAT DU JOUEUR : ses deux jauges quand il vit, son
 * retour quand il est mort.
 *
 * EXTRAIT DE `ReplayTeams.tsx` LE 2026-08-18 (lot R3, item R3.7) : la fiche a gagné sa
 * variante COMPACTE, et le fichier franchissait le seuil de 500 lignes du dépôt. La découpe
 * tombe sur une frontière nette — ces deux composants sont les seuls à ne RIEN savoir de la
 * mise en page de la fiche : ils reçoivent une lecture, ils rendent une barre. Aucune règle
 * n'a changé au passage.
 */
import { tokenCssVar } from '@/lib/accessibility/semantic-tokens'

import { REPLAY_TEXT, type ReplayLocale } from './i18n'
import { formatSeconds, frameToMs, freshness, READING_FADE } from './replayLogic'
import type { ReplayDocumentReady } from './replayNormalize'
import type { PlayerState } from './rosterLogic'

/**
 * VitalityBar — bouclier ou santé, lus dans le MÊME enregistrement que la position.
 *
 * LA BARRE EST TOUJOURS PLEINE AU DÉPART D'UNE VIE : on apparaît vie et bouclier pleins
 * (règle du jeu), et le flux différentiel ne retransmet que ce qui change — l'absence de
 * mesure depuis le spawn veut dire « plein », pas « inconnu » (décision utilisateur
 * 2026-08-12, doctrine du POC). UNE PISTE VIDE reste une MESURE : bouclier brisé, vie
 * entamée. Reading null = le document ne porte pas ce champ (titre sans décodage film) :
 * la ligne n'existe pas — on n'invente pas une jauge pour une donnée qui n'existe nulle
 * part dans le document.
 */
export function VitalityBar({
  reading,
  fade,
  name,
  token,
}: {
  reading: { value: number; age: number } | null
  fade: number
  name: string
  token: 'info' | 'success'
}) {
  if (!reading) return null
  const fresh = freshness(reading.age, fade, READING_FADE)
  return (
    <div
      className="h-1 overflow-hidden rounded-sm bg-muted"
      style={{ opacity: fresh }}
      title={name}
      aria-label={name}
    >
      <div
        className="h-full rounded-sm"
        style={{
          width: `${Math.max(0, Math.min(1, reading.value)) * 100}%`,
          background: tokenCssVar(token),
        }}
      />
    </div>
  )
}

/**
 * RespawnRow — ce que la fiche d'un joueur mort a de plus utile à dire.
 *
 * LE RETOUR EST LU, PAS DÉDUIT D'UNE CONSTANTE : c'est l'image de départ de la vie suivante du
 * même joueur. Mesure publiée sur le film de référence : 90 épisodes de mort, 82 avec un retour
 * lisible, médiane 8,0 s, 66 sur 82 exactement à 7,9-8,0 s. Les 8 sans retour affichent une
 * LACUNE — jamais un délai deviné, ce serait remplacer une mesure absente par une moyenne.
 *
 * LE COMPTE SEUL, ET CENTRÉ (demande utilisateur du 2026-08-25 : « centre le compteur de
 * réapparition et virer la jauge »). La barre d'avancement depuis la mort est SUPPRIMÉE : elle
 * disait la même chose que le compte à rebours, en moins précis — le compte donne les secondes,
 * la barre n'en donnait que la fraction, sur 9 px de large. Ce qu'on perd est nul : les deux se
 * dérivaient des deux MÊMES lectures (fin de la vie précédente, départ de la suivante).
 * Le centrage vaut pour les DEUX états de la rangée, le compte et la lacune : c'est la même
 * cellule, elle ne doit pas se décaler selon ce qu'elle porte.
 */
export function RespawnRow({
  state,
  doc,
  frame,
  locale,
}: {
  state: PlayerState
  doc: ReplayDocumentReady
  frame: number
  locale: ReplayLocale
}) {
  const t = REPLAY_TEXT[locale]
  if (state.respawnFrame < 0) {
    // « retour ? » sans infobulle de méthode : la justification (fin de partie sans vie
    // suivante) vit dans le commentaire de PlayerState.respawnFrame, pas à l'écran.
    return (
      <span className="block text-center font-mono text-[9.5px] text-muted-foreground">
        {t.respawnUnknown}
      </span>
    )
  }
  const remainMs = frameToMs(state.respawnFrame - frame, doc)
  return (
    <span className="flex items-center justify-center gap-1 font-mono text-[9.5px] text-muted-foreground">
      {t.respawnIn} <b className="tabular-nums">{formatSeconds(remainMs)}</b>
    </span>
  )
}
