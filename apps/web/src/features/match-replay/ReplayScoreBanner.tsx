/**
 * ReplayScoreBanner — LE SCORE DU MATCH AU-DESSUS DU TERRAIN.
 *
 * `[ barre alliée, son score écrit dedans ] — [ horloge ] — [ barre adverse ]`, demande
 * utilisateur du 2026-08-20. Le rejeu montrait le score en tête des colonnes de fiches
 * (`ReplayTeamHeader`), à droite du terrain : lisible quand on lit les fiches, pas quand on
 * regarde jouer. Le bandeau le remet là où va le regard pendant la lecture.
 *
 * IL NE DÉCIDE RIEN. Tout ce qui se calcule — les deux camps, lequel est allié, les scores
 * au frame lu, les fractions de barre, la manche — vient de `readScoreBanner`
 * (`scoreBannerLogic.ts`), qui est pur et testé. Ce fichier place et colore, rien d'autre.
 * Quand la lecture est `null`, il ne rend RIEN : ni cadre vide, ni « 0 — 0 » de
 * remplacement (FFA, mode sans compteur, côté allié inconnu — cf. l'en-tête du module).
 *
 * LES DEUX COULEURS SONT CELLES DE TOUTE LA PAGE (décision D1) : `team-ally` / `team-enemy`,
 * les tokens que les réglages d'accessibilité peuvent surcharger. Un pion bleu sur la carte
 * et une barre rouge pour la même équipe seraient une page cassée.
 *
 * LE NOMBRE NE SE TEINT PAS, LA BARRE SI. L'aplat de remplissage est la couleur d'équipe
 * PLEINE depuis le 2026-08-24 (retour utilisateur sur la version à 42 % : « c'est pas la
 * couleur de l'équipe mais une couleur terne ») — la même teinte que les pions sur la
 * carte, sans mélange. Le score est écrit en `text-foreground` par-dessus : les tokens
 * d'équipe sont des teintes moyennes, l'encre du thème y reste lisible des deux côtés de
 * la course — sur la piste nue quand le camp n'a pas marqué, sur l'aplat quand il avance.
 * Le liseré plein, lui, tient l'identité du camp même à barre vide.
 *
 * LA BARRE SE REMPLIT DEPUIS L'HORLOGE VERS L'EXTÉRIEUR (précision utilisateur du
 * 2026-08-24, après un premier sens livré à l'envers) : celle de gauche part du bord de
 * l'horloge et grandit vers la gauche, celle de droite vers la droite. Le liseré franc au
 * bord EXTÉRIEUR devient une ligne d'arrivée : la barre le rejoint quand la cible est
 * atteinte. LE SCORE EST ANCRÉ CÔTÉ HORLOGE (même précision utilisateur) : les deux nombres
 * encadrent l'horloge, là où part le remplissage — ancrés, ils ne glissent pas sous l'œil.
 */
import type { CSSProperties } from 'react'
import { useMemo } from 'react'

import { tokenCssVar } from '@/lib/accessibility/semantic-tokens'
import type { XuidMeta } from '@/features/match-view/xuidMeta'
import { scoreTimelineOf, type ReplayScoreDocument } from '@/lib/replay/scoreTimeline'
import type { MatchScoreboardRow } from '@/lib/api/types'

import { formatClock } from './replayLogic'
import { readScoreBanner, type ScoreBannerSide } from './scoreBannerLogic'
import { REPLAY_TEXT, type ReplayLocale } from './i18n'

interface Props {
  /**
   * Le document du rejeu. Le calque de score en est tiré ICI, par `scoreTimelineOf` — le
   * seul accès qui porte la garde d'horloge : un film dont l'origine n'est pas résolue
   * tiquerait jusqu'à 50 s trop tôt, et un score faux se lit comme un score juste.
   */
  doc: ReplayScoreDocument
  scoreboard: ReadonlyArray<Pick<MatchScoreboardRow, 'xuid' | 'team_side'>>
  xuidMeta?: XuidMeta
  /** Image de lecture courante — le score est lu À CE frame, pas à la fin du match. */
  frame: number
  /** Position de lecture en millisecondes, déjà calculée par la page pour le fil. */
  nowMs: number
  locale: ReplayLocale
}

export function ReplayScoreBanner({ doc, scoreboard, xuidMeta, frame, nowMs, locale }: Props) {
  const t = REPLAY_TEXT[locale]
  const reading = useMemo(
    () => readScoreBanner(scoreTimelineOf(doc), scoreboard, xuidMeta, frame),
    [doc, scoreboard, xuidMeta, frame],
  )
  if (!reading) return null
  return (
    <div
      role="group"
      aria-label={t.scoreBannerLabel}
      className="mb-2 flex items-stretch gap-2"
    >
      <ScoreBar side={reading.ally} label={t.scoreBannerAlly} token="team-ally" anchor="left" />
      <div className="flex shrink-0 flex-col items-center justify-center px-1">
        <span
          className="font-mono text-sm font-bold leading-none tabular-nums"
          aria-label={t.scoreBannerClock}
        >
          {formatClock(nowMs)}
        </span>
        {/* LA MANCHE, seulement quand le mode en a plusieurs : sur un mode à manche unique
            elle ne dirait rien que le total ne dise déjà. Discrète — c'est un repère. */}
        {reading.round && (
          <span
            className="mt-0.5 text-[9px] font-normal leading-none tabular-nums text-muted-foreground"
            title={t.roundOfCountFmt(reading.round.index, reading.round.count)}
          >
            {t.roundNumberFmt(reading.round.index)}
          </span>
        )}
      </div>
      <ScoreBar side={reading.enemy} label={t.scoreBannerEnemy} token="team-enemy" anchor="right" />
    </div>
  )
}

interface BarProps {
  side: ScoreBannerSide
  label: string
  token: 'team-ally' | 'team-enemy'
  /**
   * Bord EXTÉRIEUR de la barre : celui du liseré et du nombre. Le remplissage, lui, part
   * du bord opposé (côté horloge) et grandit VERS ce bord — les camps se font face.
   */
  anchor: 'left' | 'right'
}

/**
 * Une barre de camp : la piste, l'aplat de remplissage, le score écrit dedans.
 *
 * `role="progressbar"` avec `aria-valuetext` FORCÉ AU SCORE : sans lui, un lecteur d'écran
 * annoncerait un pourcentage de la cible déduite (le score final du vainqueur — cf.
 * `scoreBannerLogic`), qui n'est pas une donnée publiée. Le pourcentage décrit la barre,
 * le `valuetext` dit la mesure.
 */
function ScoreBar({ side, label, token, anchor }: BarProps) {
  const accent = tokenCssVar(token)
  const pct = Math.round(side.fill * 100)
  // L'aplat est ancré au bord INTÉRIEUR (côté horloge) et grandit vers l'extérieur : pour
  // la barre de gauche il colle à droite, pour celle de droite il colle à gauche.
  const fillStyle: CSSProperties = {
    width: `${pct}%`,
    background: accent,
    ...(anchor === 'left' ? { right: 0 } : { left: 0 }),
  }
  const trackStyle: CSSProperties =
    anchor === 'left' ? { borderLeft: `3px solid ${accent}` } : { borderRight: `3px solid ${accent}` }
  return (
    <div
      role="progressbar"
      aria-label={label}
      aria-valuemin={0}
      aria-valuemax={100}
      aria-valuenow={pct}
      aria-valuetext={String(side.score)}
      className="relative h-7 min-w-0 flex-1 overflow-hidden rounded-sm bg-muted"
      style={trackStyle}
    >
      <span className="absolute inset-y-0 block" style={fillStyle} />
      {/* Le nombre CÔTÉ HORLOGE : bord droit de la barre de gauche, bord gauche de celle
          de droite — l'inverse de `anchor`, qui nomme le bord extérieur. */}
      <span
        className={`absolute inset-y-0 flex items-center px-2 font-mono text-[15px] font-bold tabular-nums text-foreground ${
          anchor === 'left' ? 'right-0' : 'left-0'
        }`}
      >
        {side.score}
      </span>
    </div>
  )
}
