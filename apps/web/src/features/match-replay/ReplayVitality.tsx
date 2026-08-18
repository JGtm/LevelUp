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
      <span className="font-mono text-[9.5px] text-muted-foreground">
        {t.respawnUnknown}
      </span>
    )
  }
  const remainMs = frameToMs(state.respawnFrame - frame, doc)
  // La barre montre l'AVANCEMENT DEPUIS LA MORT : la mort est datée par la fin de la vie
  // précédente, le retour par le départ de la suivante — deux lectures, aucune constante.
  // Quand la mort n'est pas datée (sinceDeath < 0), le compte s'affiche sans barre plutôt
  // qu'avec un avancement faux.
  const span = state.sinceDeath >= 0 ? state.respawnFrame - (frame - state.sinceDeath) : 0
  const progress = span > 0 ? Math.max(0, Math.min(1, state.sinceDeath / span)) : null
  return (
    <span className="inline-flex items-center gap-1.5 font-mono text-[9.5px] text-muted-foreground">
      {progress !== null && (
        <span
          className="inline-block h-1 w-9 overflow-hidden rounded-sm bg-muted"
          role="progressbar"
          aria-label={t.respawnBarLabel}
        >
          <span
            className="block h-full rounded-sm opacity-80"
            style={{ width: `${(progress * 100).toFixed(1)}%`, background: tokenCssVar('success') }}
          />
        </span>
      )}
      {t.respawnIn} <b className="tabular-nums">{formatSeconds(remainMs)}</b>
    </span>
  )
}
