/**
 * ReplayVitality — CE QU'UNE FICHE DIT DE L'ÉTAT DU JOUEUR : ses deux jauges quand il vit, son
 * encadré « Éliminé » quand il est mort.
 *
 * EXTRAIT DE `ReplayTeams.tsx` LE 2026-08-18 (lot R3, item R3.7) : la fiche a gagné sa
 * variante COMPACTE, et le fichier franchissait le seuil de 500 lignes du dépôt. La découpe
 * tombe sur une frontière nette — ces deux composants sont les seuls à ne RIEN savoir de la
 * mise en page de la fiche : ils reçoivent une lecture, ils rendent une barre. Aucune règle
 * n'a changé au passage.
 */
import type { CSSProperties } from 'react'

import { tokenCssVar } from '@/lib/accessibility/semantic-tokens'

import { REPLAY_TEXT, type ReplayLocale } from '../i18n/i18n'
import { formatSeconds, frameToMs, freshness, READING_FADE } from '../model/replayLogic'
import type { ReplayDocumentReady } from '../model/replayNormalize'
import type { PlayerState } from '../model/rosterLogic'

/**
 * La PISTE des deux jauges : l'encre du thème à 10 % (option 2a du handoff 2026-08-27) —
 * visible sur le fond de la tuile dans les deux thèmes, jamais un gris écrit en dur.
 */
const TRACK_INK_PCT = 10

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
 *
 * DEUX BARRES PLEINES, DEUX HAUTEURS (option 2a) : le bouclier 5 px au-dessus de la santé
 * 3 px — la hiérarchie du jeu (le bouclier encaisse d'abord) dite par l'épaisseur, plus par
 * des segments. `heightPx` vient de la fiche, qui apparie hauteur et token.
 */
export function VitalityBar({
  reading,
  fade,
  name,
  token,
  heightPx,
}: {
  reading: { value: number; age: number } | null
  fade: number
  name: string
  token: 'info' | 'success'
  heightPx: number
}) {
  if (!reading) return null
  const fresh = freshness(reading.age, fade, READING_FADE)
  return (
    <div
      className="overflow-hidden rounded-[1px]"
      style={{
        height: heightPx,
        opacity: fresh,
        background: `color-mix(in srgb, var(--foreground) ${TRACK_INK_PCT}%, transparent)`,
      }}
      title={name}
      aria-label={name}
    >
      <div
        className="h-full rounded-[1px]"
        style={{
          width: `${Math.max(0, Math.min(1, reading.value)) * 100}%`,
          background: tokenCssVar(token),
        }}
      />
    </div>
  )
}

/**
 * EliminatedBox — ce que la fiche d'un joueur mort a de plus utile à dire, dans l'ENCADRÉ
 * qui remplace les rangées vitalité + inventaire (option 2a du handoff 2026-08-27, ex-
 * `RespawnRow`) : « ÉLIMINÉ » à gauche à l'encre d'alerte, le décompte à droite en gros
 * chiffres tabulaires. Fond `destructive` très dilué, hachures diagonales de la même encre.
 *
 * LE RETOUR EST LU, PAS DÉDUIT D'UNE CONSTANTE : c'est l'image de départ de la vie suivante du
 * même joueur. Mesure publiée sur le film de référence : 90 épisodes de mort, 82 avec un retour
 * lisible, médiane 8,0 s, 66 sur 82 exactement à 7,9-8,0 s. Les 8 sans retour affichent une
 * LACUNE — « Hors film / ne revient plus » (goneLabel/goneValue) — jamais un délai deviné,
 * ce serait remplacer une mesure absente par une moyenne.
 *
 * L'ENCADRÉ REMPLIT LA ZONE FIXE DE LA FICHE (`h-full`) : la hauteur totale reste identique
 * en vie et en mort (règle du 2026-08-24 — une fiche qui change de hauteur fait sauter toute
 * la colonne à chaque mort). C'est la fiche qui possède cette hauteur, pas l'encadré.
 */
export function EliminatedBox({
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
  const rouge = tokenCssVar('destructive')
  const fond: CSSProperties = {
    backgroundColor: `color-mix(in srgb, ${rouge} 7%, transparent)`,
    backgroundImage:
      `repeating-linear-gradient(135deg, color-mix(in srgb, ${rouge} 10%, transparent) 0 4px, ` +
      'transparent 4px 9px)',
  }
  return (
    <div
      className="flex h-full items-center justify-between overflow-hidden rounded-sm px-2"
      style={fond}
    >
      <span
        className="shrink-0 text-[9.5px] font-bold uppercase tracking-[.18em]"
        style={{ color: rouge }}
      >
        {state.respawnFrame < 0 ? t.goneLabel : t.eliminatedLabel}
      </span>
      {state.respawnFrame < 0 ? (
        // SANS VIE SUIVANTE, LE JOUEUR NE REVIENT PLUS DANS LE FILM — et c'est TOUT ce que la
        // donnée dit (mort de fin de partie, départ en cours de match, vie jamais nommée). Le
        // « Réapparition ? » d'avant affirmait une attente qui n'existait pas : un partant
        // restait « en attente de respawn » jusqu'au bout (retour user 2026-09-02).
        <span className="min-w-0 truncate font-mono text-[13px] font-bold text-foreground">
          {t.goneValue}
        </span>
      ) : (
        <span
          className="shrink-0 font-mono text-[15px] font-bold tabular-nums text-foreground"
          title={t.respawnIn}
        >
          {formatSeconds(frameToMs(state.respawnFrame - frame, doc))}
        </span>
      )}
    </div>
  )
}
