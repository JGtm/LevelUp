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
 * LA COULEUR EST CELLE DE L'IDENTITÉ D'ÉQUIPE, ET C'EST UNE EXCEPTION ASSUMÉE À LA DÉCISION D1
 * DE CETTE PAGE. Partout ailleurs sur le rejeu, la couleur dit le CAMP (tokens allié / adverse,
 * surchargeables par les réglages d'accessibilité) : un pion bleu et une barre bleue pour la
 * même équipe. Ici, la couleur dit UNE APPARTENANCE — décision utilisateur du 2026-08-26, « la
 * même couleur que les en-têtes du scoreboard de la Match View ». L'écran de fin est le moment
 * où l'équipe cesse d'être « nous » pour redevenir Cobra ou Eagle ; il emprunte donc la cascade
 * d'identité (`teamColorResolver`) et la recette de teinte (`teamTintStyles`) du scoreboard,
 * sans en réécrire une ligne. LE TEXTE, LUI, RESTE EN `--foreground` : une couleur d'identité
 * peut être très claire (le jaune Valor) et le contraste ne s'y négocie pas.
 *
 * L'ÉGALITÉ N'EMPRUNTE RIEN (D-B1). Elle ne désigne personne : ni logo, ni couleur d'équipe,
 * ni nom — les tokens du thème, le verdict du backend, et les deux scores.
 */
import { useMemo } from 'react'
import type { CSSProperties } from 'react'

import { teamColorResolver, teamTintStyles } from '@/features/match-view/teamColor'
import type { XuidMeta } from '@/features/match-view/xuidMeta'
import { resolveTeamLabel } from '@/lib/halo/teamLabel'
import { teamLogoPath } from '@/lib/halo/teamNames'
import { scoreTimelineOf, type ReplayScoreDocument } from '@/lib/replay/scoreTimeline'
import type { MatchScoreboardRow } from '@/lib/api/types'

import { REPLAY_TEXT, type ReplayLocale } from './i18n'
import type { ReplayText } from './i18nContract'
import type { ReplayWindowBounds } from './replayWindow'
import { readScoreBanner, type ScoreBannerReading } from './scoreBannerLogic'
import { readVictory, type VictoryTeam } from './victoryLogic'

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
        />
      ) : (
        <NeutralPanel title={outcomeLabel} t={t} score={score} />
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
}

/**
 * Le panneau habillé aux couleurs d'une équipe : son logo en filigrane derrière, le verdict
 * devant, son nom dessous.
 *
 * LE FILIGRANE EST DÉCORATIF et il le dit (`aria-hidden`) : le nom de l'équipe est écrit juste
 * en dessous, un lecteur d'écran n'a pas à l'entendre deux fois. Un `team_id` sans asset publié
 * répond 404 — l'`onError` retire l'image, et le panneau reste entier : il n'a jamais eu besoin
 * du logo pour dire comment le match s'est terminé.
 */
function TeamPanel({ team, scoreboard, titleSlug, title, t, score }: TeamPanelProps) {
  const rows = scoreboard.filter((r) => r.team_side === team.teamSide)
  const label = resolveTeamLabel(rows, team.teamSide, t)
  const tint = teamTintStyles(teamColorResolver(scoreboard)(team.teamID, team.ally))
  const logo = teamLogoPath(titleSlug, team.teamID)
  const panelStyle: CSSProperties = {
    background: tint.background,
    border: `2px solid ${tint.border}`,
    borderLeft: `6px solid ${tint.accent}`,
  }
  return (
    <div className="relative flex max-w-[90%] items-center justify-center">
      {logo && (
        <img
          src={logo}
          alt=""
          aria-hidden
          className="pointer-events-none absolute h-[min(18rem,60vh)] w-auto max-w-none opacity-20"
          onError={(e) => {
            e.currentTarget.style.display = 'none'
          }}
        />
      )}
      <div
        className="relative rounded-lg px-8 py-5 text-center shadow-lg backdrop-blur-sm"
        style={panelStyle}
      >
        <p className="text-2xl font-bold uppercase tracking-wide text-foreground">{title}</p>
        <p className="mt-1 text-sm font-semibold uppercase tracking-wide text-foreground">
          {label}
        </p>
        <FinalScoreLine t={t} score={score} />
      </div>
    </div>
  )
}

/** Le panneau neutre de l'égalité : les tokens du thème, et rien de l'identité des équipes. */
function NeutralPanel({
  title,
  t,
  score,
}: {
  title: string
  t: ReplayText
  score: ScoreBannerReading | null
}) {
  return (
    <div className="rounded-lg border-2 border-border bg-card px-8 py-5 text-center shadow-lg backdrop-blur-sm">
      <p className="text-2xl font-bold uppercase tracking-wide text-foreground">{title}</p>
      <FinalScoreLine t={t} score={score} />
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
function FinalScoreLine({ t, score }: { t: ReplayText; score: ScoreBannerReading | null }) {
  if (!score) return null
  return (
    <p className="mt-2 font-mono text-3xl font-bold tabular-nums text-foreground">
      <span className="sr-only">{t.victoryScoreLabel} : </span>
      <span aria-label={t.scoreBannerAlly}>{score.ally.score}</span>
      <span className="px-3 text-muted-foreground">-</span>
      <span aria-label={t.scoreBannerEnemy}>{score.enemy.score}</span>
    </p>
  )
}
