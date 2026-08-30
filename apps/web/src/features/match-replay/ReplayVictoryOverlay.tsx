/**
 * ReplayVictoryOverlay — L'ÉCRAN DE FIN DE MATCH, quand la lecture atteint la fin déclarée.
 *
 * Le rejeu se terminait sur rien : la lecture s'arrêtait, le terrain restait affiché, et le
 * résultat du match — la seule chose que tout le monde retient — n'était écrit nulle part sur
 * la page. Ce panneau le dit, à l'instant où il se produit.
 *
 * C'EST L'ÉCRAN DU JOUEUR DE LA PAGE, PAS CELUI DU VAINQUEUR (amendement utilisateur du
 * 2026-08-26). Le logo, la couleur et le nom sont ceux de SON équipe, en victoire COMME en
 * défaite — exactement comme l'écran de fin du jeu, qui ne vous affiche pas l'emblème adverse
 * parce qu'il a gagné. Ce que l'issue change, c'est le TITRE, pas l'habillage.
 *
 * LE TITRE VIENT DU BACKEND, ET IL N'EST PAS RÉÉCRIT ICI : `header.outcome_label` est déjà
 * localisé côté serveur (« Victoire » / « Défaite » / « Égalité »), c'est le même mot que la
 * Match View affiche pour ce match. En fabriquer une variante côté front donnerait deux
 * verdicts pour un seul match. Sans ce libellé, pas d'écran : un panneau plein cadre qui
 * n'annonce rien serait pire que pas de panneau.
 *
 * IL EST DÉRIVÉ DE LA POSITION DE LECTURE, PAS D'UN ÉTAT (décision D-B5) : visible tant que la
 * lecture est à la borne de fin ou au-delà, invisible dès qu'on remonte la frise ou qu'on
 * recommence. Il n'a donc AUCUN bouton de fermeture — il n'y a rien à fermer, seulement une
 * position à quitter.
 *
 * IL LAISSE PASSER LES CLICS (`pointer-events-none`), et ce n'est pas un détail de confort : la
 * FRISE DE LECTURE est dans le même conteneur, sous le terrain. Un voile qui capterait les
 * clics enfermerait l'utilisateur dans l'écran de fin — le seul geste qui le fait disparaître
 * serait précisément celui que le voile bloque. Le panneau n'a aucun élément interactif : rien
 * ne se perd à le rendre transparent au pointeur.
 *
 * SANS CADRAGE, PAS D'ÉCRAN (décision D-A3) : `playWindow` à `null` veut dire qu'on ne sait pas
 * où le match finit (artefact ancien, en-tête sans durée jouable). Annoncer le résultat à la
 * fin du FILM afficherait le panneau six secondes trop tard, par-dessus une scène qui
 * n'appartient déjà plus au match.
 *
 * LA COULEUR EST CELLE QUE L'UTILISATEUR A CHOISIE, ET LA DÉCISION D1 REDEVIENT SANS EXCEPTION
 * (retour du 2026-08-28 : « il doit respecter les choix de couleur d'équipe choisie par le
 * user »). L'écran a porté un temps la couleur d'IDENTITÉ officielle (Eagle bleu, Cobra rouge,
 * cascade `teamColorResolver` du scoreboard) : un joueur qui a réglé son camp en vert voyait
 * donc la page entière en vert et son écran de fin en bleu. C'est le token `team-ally` —
 * l'équipe de l'écran est TOUJOURS celle du joueur de la page — surchargeable par les réglages
 * d'accessibilité, comme les pions, les barres et les fiches. Seule la RECETTE de teinte reste
 * empruntée au scoreboard (`teamTintStyles`, 22 % / 55 %) : elle ne dit pas quelle couleur,
 * seulement à quelle dose. LE TEXTE RESTE EN `--foreground` : une couleur de camp peut être
 * très claire, et le contraste ne se négocie pas.
 *
 * LE BLOC NE PORTE QUE LE STATUT (même retour) : le verdict est dans la carte colorée, le nom
 * de l'équipe et le score sont posés SOUS elle, en texte libre sur le voile. Le logo, lui, est
 * TEINTÉ à cette même couleur — c'est une silhouette monochrome, la laisser au bleu du jeu
 * rouvrirait par l'image l'écart que la couleur vient de fermer.
 *
 * L'ÉGALITÉ N'EMPRUNTE RIEN (D-B1). Elle ne désigne personne : ni logo, ni couleur d'équipe,
 * ni nom — les tokens du thème, le verdict du backend, et les deux scores.
 */
import { useMemo, useState } from 'react'
import type { CSSProperties } from 'react'

import { teamTintStyles } from '@/features/match-view/teamColor'
import type { XuidMeta } from '@/features/match-view/xuidMeta'
import { tokenCssVar } from '@/lib/accessibility'
import { resolveTeamLabel } from '@/lib/halo/teamLabel'
import { teamLogoPath } from '@/lib/halo/teamNames'
import { scoreTimelineOf, type ReplayScoreDocument } from '@/lib/replay/scoreTimeline'
import type { MatchScoreboardRow } from '@/lib/api/types'

import { REPLAY_TEXT, type ReplayLocale } from './i18n'
import type { ReplayText } from './i18nContract'
import { OVERLAY_STATUS_BLOCK, OVERLAY_STATUS_NEUTRAL } from './replayOverlayStyles'
import type { ReplayWindowBounds } from './replayWindow'
import { readScoreBanner, type ScoreBannerReading } from './scoreBannerLogic'
import { readVictory, type FinalScoreReading, type VictoryTeam } from './victoryLogic'

interface Props {
  /** Le document du rejeu — le score final s'y lit par le calque, à la borne de fin. */
  doc: ReplayScoreDocument
  scoreboard: readonly MatchScoreboardRow[]
  xuidMeta?: XuidMeta
  /** Verdict du joueur de la page (`header.outcome_code`) — la source de l'issue (D-B2). */
  outcomeCode: number | null | undefined
  /** Le verdict ÉCRIT (`header.outcome_label`), déjà localisé par le backend. */
  outcomeLabel: string | null | undefined
  /** La fenêtre de gameplay : sa borne de fin déclenche l'écran. `null` → pas d'écran. */
  playWindow: ReplayWindowBounds | null
  /** Image de lecture courante, publiée par le canvas toutes les 150 ms. */
  frame: number
  /** Slug du titre — paramètre du chemin d'asset du logo, jamais une branche de comportement. */
  titleSlug: string
  /**
   * Le score final SERVI PAR L'API, quand il ne se déduit pas du film — c'est-à-dire sur un
   * mode à MANCHES, où la lecture du calque à la borne de fin rendrait les points de la
   * dernière manche (« 100 - 43 ») au lieu du résultat (« 2 - 1 »). `null` sur un mode en
   * points : le calque, plus fin, reste la source. Cf. `finalScoreFromHeader`.
   */
  finalScore?: FinalScoreReading | null
  locale: ReplayLocale
}

export function ReplayVictoryOverlay({
  doc,
  scoreboard,
  xuidMeta,
  outcomeCode,
  outcomeLabel,
  playWindow,
  frame,
  titleSlug,
  finalScore,
  locale,
}: Props) {
  const t = REPLAY_TEXT[locale]
  const reading = useMemo(() => readVictory(scoreboard, outcomeCode), [scoreboard, outcomeCode])
  // LE SCORE SE LIT À LA BORNE DE FIN, pas à l'image courante (D-B4) : la lecture peut être
  // allée au-delà (frise tirée au bout), et le calque n'a plus rien à dire après la fin.
  const score = useMemo(
    () =>
      playWindow
        ? readScoreBanner(scoreTimelineOf(doc), scoreboard, xuidMeta, playWindow.endFrame)
        : null,
    [doc, scoreboard, xuidMeta, playWindow],
  )
  if (!playWindow || frame < playWindow.endFrame || !reading || !outcomeLabel) return null
  return (
    <div
      role="status"
      aria-live="polite"
      aria-label={t.victoryPanelLabel}
      className="pointer-events-none absolute inset-0 z-10 flex items-center justify-center overflow-hidden bg-background/70"
    >
      {reading.mine ? (
        <TeamPanel
          team={reading.mine}
          scoreboard={scoreboard}
          titleSlug={titleSlug}
          title={outcomeLabel}
          t={t}
          score={score}
          finalScore={finalScore}
        />
      ) : (
        <NeutralPanel title={outcomeLabel} t={t} score={score} finalScore={finalScore} />
      )}
    </div>
  )
}

interface TeamPanelProps {
  /** L'équipe DU JOUEUR DE LA PAGE — l'habillage, quelle que soit l'issue. */
  team: VictoryTeam
  scoreboard: readonly MatchScoreboardRow[]
  titleSlug: string
  title: string
  t: ReplayText
  score: ScoreBannerReading | null
  finalScore?: FinalScoreReading | null
}

/**
 * Le panneau habillé aux couleurs du camp du joueur : son logo en filigrane derrière, le STATUT
 * dans le bloc coloré, le nom de l'équipe et le score en texte libre dessous.
 *
 * LE BLOC N'EMBARQUE QUE LE STATUT (retour du 2026-08-28) : nom et score sont sortis de la
 * carte. Ils restent lisibles — le voile du panneau (`bg-background/70`) est leur fond.
 *
 * LE FILIGRANE EST DÉCORATIF et il le dit (`aria-hidden`) : le nom de l'équipe est écrit juste
 * en dessous, un lecteur d'écran n'a pas à l'entendre deux fois.
 */
function TeamPanel({ team, scoreboard, titleSlug, title, t, score, finalScore }: TeamPanelProps) {
  const rows = scoreboard.filter((r) => r.team_side === team.teamSide)
  const label = resolveTeamLabel(rows, team.teamSide, t)
  // LA COULEUR DU JOUEUR DE LA PAGE, telle qu'il l'a réglée (D1, cf. l'en-tête) : l'écran est
  // TOUJOURS celui de son camp, donc toujours `team-ally`. Fond et trait par la recette du
  // scoreboard. PAS D'ACCENT LATÉRAL GAUCHE : l'utilisateur l'a fait retirer de ce style.
  const teamColor = tokenCssVar('team-ally')
  const tint = teamTintStyles(teamColor)
  const blockStyle: CSSProperties = {
    background: tint.background,
    border: `2px solid ${tint.border}`,
  }
  return (
    <div className="relative flex max-w-[90%] flex-col items-center justify-center">
      <TeamLogoWatermark src={teamLogoPath(titleSlug, team.teamID)} color={teamColor} />
      <p className={`relative ${OVERLAY_STATUS_BLOCK}`} style={blockStyle}>
        {title}
      </p>
      <p className="relative mt-2 text-sm font-semibold uppercase tracking-wide text-foreground">
        {label}
      </p>
      <FinalScoreLine t={t} score={score} finalScore={finalScore} />
    </div>
  )
}

/**
 * Le filigrane du logo, TEINTÉ à la couleur du camp (retour du 2026-08-28) : les emblèmes
 * publiés sont des silhouettes monochromes aux couleurs officielles du jeu, l'image est donc
 * consommée comme un MASQUE et l'aplat par-dessous porte la couleur réglée par l'utilisateur.
 *
 * LA SONDE DE CHARGEMENT N'EST PAS UN LUXE : un `team_id` sans asset publié répond 404, et un
 * masque qui échoue ne masque plus RIEN — l'aplat s'afficherait en carré de couleur pleine, au
 * milieu de l'écran. Tant que l'image n'est pas chargée, il n'y a pas de filigrane ; le panneau
 * reste entier, il n'a jamais eu besoin du logo pour dire comment le match s'est terminé.
 */
function TeamLogoWatermark({ src, color }: { src: string | null; color: string }) {
  const [loaded, setLoaded] = useState(false)
  if (!src) return null
  const mask: CSSProperties = {
    backgroundColor: color,
    maskImage: `url(${src})`,
    WebkitMaskImage: `url(${src})`,
    maskRepeat: 'no-repeat',
    WebkitMaskRepeat: 'no-repeat',
    maskPosition: 'center',
    WebkitMaskPosition: 'center',
    maskSize: 'contain',
    WebkitMaskSize: 'contain',
  }
  return (
    <>
      {/* La sonde : une image hors flux, jamais peinte, qui dit seulement si l'asset existe. */}
      <img src={src} alt="" aria-hidden hidden onLoad={() => setLoaded(true)} />
      {loaded && (
        <span
          aria-hidden
          className="pointer-events-none absolute h-[min(18rem,60vh)] w-[min(18rem,60vh)] opacity-20"
          style={mask}
        />
      )}
    </>
  )
}

/** Le panneau neutre de l'égalité : les tokens du thème, et rien de l'identité des équipes. */
function NeutralPanel({
  title,
  t,
  score,
  finalScore,
}: {
  title: string
  t: ReplayText
  score: ScoreBannerReading | null
  finalScore?: FinalScoreReading | null
}) {
  return (
    <div className="flex max-w-[90%] flex-col items-center justify-center">
      <p className={OVERLAY_STATUS_NEUTRAL}>{title}</p>
      <FinalScoreLine t={t} score={score} finalScore={finalScore} />
    </div>
  )
}

/**
 * La ligne de score FINAL, absente quand le mode n'en publie pas (D-B4) — un « 0 — 0 » de
 * remplacement se lirait comme une mesure alors que personne n'a compté.
 *
 * L'ORDRE EST CELUI DU BANDEAU juste au-dessus du terrain : allié à gauche, adverse à droite.
 * Les deux surfaces sont visibles ensemble ; les inverser ici (« le vainqueur d'abord ») ferait
 * lire deux scores contradictoires à trois centimètres l'un de l'autre.
 */
function FinalScoreLine({
  t,
  score,
  finalScore,
}: {
  t: ReplayText
  score: ScoreBannerReading | null
  finalScore?: FinalScoreReading | null
}) {
  // LE SCORE SERVI PAR L'API L'EMPORTE QUAND IL EXISTE : sur un mode à manches, le calque du
  // film rendrait ici les points de la DERNIÈRE MANCHE, présentés comme le score du match.
  // C'est la seule ligne du rejeu qui ne se lit pas dans le film, et c'est délibéré — elle
  // annonce un RÉSULTAT, et le résultat est celui que toute l'app affiche par ailleurs.
  const ally = finalScore ? finalScore.ally : score?.ally.score
  const enemy = finalScore ? finalScore.enemy : score?.enemy.score
  if (ally == null || enemy == null) return null
  return (
    <p className="relative mt-2 font-mono text-3xl font-bold tabular-nums text-foreground">
      <span className="sr-only">{t.victoryScoreLabel} : </span>
      <span aria-label={t.scoreBannerAlly}>{ally}</span>
      <span className="px-3 text-muted-foreground">-</span>
      <span aria-label={t.scoreBannerEnemy}>{enemy}</span>
    </p>
  )
}
