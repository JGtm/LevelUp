/**
 * exportOverlayPanels.ts — QUEL PANNEAU À QUELLE IMAGE, pour l'export.
 *
 * # CE QUE CE MODULE TRANSPOSE, ET CE QU'IL NE RECALCULE PAS
 *
 * `ReplayVictoryOverlay` et `ReplayRoundBreakOverlay` décident, à chaque rendu, s'ils ont
 * quelque chose à montrer. Leur condition est courte et elle est la MÊME pour l'export — mais
 * elle vit dans des composants React, que la boucle d'export ne monte pas. Ce module la
 * transpose, une fois, en fonction pure.
 *
 * IL NE RECALCULE AUCUNE DONNÉE : le verdict vient de `readVictory`, le score de
 * `readScoreBanner`, les bascules de manche de `roundTransitions` / `activeRoundTransition` —
 * exactement les modules que les deux composants appellent. Un panneau exporté ne peut donc
 * pas dire autre chose que le panneau affiché.
 *
 * # DEUXIÈME COPIE DE LA CONDITION, ET C'EST LA DERNIÈRE
 *
 * La règle CLAUDE.md n° 6 tolère deux copies et impose la centralisation à la troisième. Nous
 * y sommes : le DOM, et ici. Une TROISIÈME surface qui voudrait ces panneaux (une vignette
 * serveur, un partage) devra d'abord faire descendre la condition dans un module que les
 * composants consommeront aussi — pas en recopier une de plus.
 *
 * # L'ORDRE DES DEUX PANNEAUX N'EST PAS ARBITRAIRE
 *
 * La fin de match l'emporte sur la fin de manche. Sur un mode à manches, la DERNIÈRE bascule
 * tombe au même instant que la fin du match : afficher « Manche 3 terminée » par-dessus le
 * verdict final ferait passer la conclusion du match pour une transition.
 */
import type { XuidMeta } from '@/features/match-view/xuidMeta'
import { resolveTeamLabel } from '@/lib/halo/teamLabel'
import type { MatchScoreboardRow } from '@/lib/api/types'
import { scoreTimelineOf, type ReplayScoreDocument } from '@/lib/replay/scoreTimeline'

import { ROUND_BREAK_WINDOW_MS } from '../ui/ReplayRoundBreakOverlay'
import { REPLAY_TEXT, type ReplayLocale } from '../i18n/i18n'
import type { OverlayInk, OverlayPanel, OverlayStatusStyle } from './overlayPaint'
import { neutralStatusStyle } from './overlayPaint'
import { msToFrames } from '../../../lib/replay/replayLogic'
import type { ReplayDocumentReady } from '../../../lib/replay/replayNormalize'
import type { ReplayWindowBounds } from '../model/replayWindow'
import { activeRoundTransition, roundTransitions } from '../model/roundsLogic'
import { readScoreBanner } from '../model/scoreBannerLogic'
import { readVictory, type FinalScoreReading } from '../model/victoryLogic'

/**
 * exportFinalScore choisit ce que le panneau de fin écrit : le score SERVI PAR L'API quand il
 * existe (mode à manches), sinon la lecture du calque à la borne de fin. Extrait pour que la
 * règle soit la MÊME ligne de code que celle du DOM (`FinalScoreLine`) — deux panneaux qui
 * annoncent le même match ne peuvent pas décider séparément.
 */
export function exportFinalScore(
  fromAPI: FinalScoreReading | null | undefined,
  fromFilm: { ally: { score: number }; enemy: { score: number } } | null,
): { ally: number; enemy: number } | null {
  if (fromAPI) return fromAPI
  return fromFilm ? { ally: fromFilm.ally.score, enemy: fromFilm.enemy.score } : null
}

/** Le verdict du match, tel que la page le reçoit du backend (jamais réécrit côté front). */
export interface ExportOutcome {
  code: number | null | undefined
  /** Déjà localisé par le serveur. Sans lui, pas d'écran de fin — comme dans le DOM. */
  label: string | null | undefined
  /**
   * Le score final SERVI PAR L'API sur un mode à MANCHES (« 2 - 1 »), là où le calque du film
   * rendrait les points de la dernière manche. `null` sur un mode en points : le calque reste
   * la source. MÊME valeur que celle de l'écran affiché — l'export ne doit jamais raconter
   * autre chose que la page. Cf. `finalScoreFromHeader`.
   */
  finalScore?: FinalScoreReading | null
}

/** Tout ce dont la construction d'un panneau a besoin, résolu une fois par export. */
export interface OverlayPanelDeps {
  doc: ReplayDocumentReady & ReplayScoreDocument
  scoreboard: readonly MatchScoreboardRow[]
  xuidMeta?: XuidMeta
  playWindow: ReplayWindowBounds | null
  outcome: ExportOutcome | null
  locale: ReplayLocale
  ink: OverlayInk
  /**
   * La teinte du camp du joueur de la page. TOUJOURS fournie : l'égalité est déjà traitée par
   * l'absence de `victory.mine`, qui bascule seule sur le style neutre. Un `null` ici serait un
   * second chemin vers la même chose, qu'aucun appelant ne peut produire.
   */
  teamStyle: OverlayStatusStyle
  /** Le filigrane DÉJÀ teinté, ou `null` tant qu'il n'est pas chargé. */
  logo: CanvasImageSource | null
}

/**
 * Ce qui ne change pas d'une image à l'autre, calculé UNE FOIS avant la boucle.
 *
 * Sans cette séparation, un export de dix-huit mille images relirait dix-huit mille fois la
 * frise de score et les bascules de manche pour en tirer les mêmes valeurs.
 */
export interface OverlayPanelSource {
  panelAt: (frame: number) => OverlayPanel | null
}

/**
 * buildOverlayPanelSource prépare les lectures constantes et rend le sélecteur par image.
 *
 * `null` en sortie de `panelAt` est le cas NOMINAL : la très grande majorité des images d'un
 * match n'a aucune surimpression à porter.
 */
export function buildOverlayPanelSource(deps: OverlayPanelDeps): OverlayPanelSource {
  const t = REPLAY_TEXT[deps.locale]
  const timeline = scoreTimelineOf(deps.doc)
  const transitions = roundTransitions(timeline)
  const breakFrames = Math.max(1, Math.round(msToFrames(ROUND_BREAK_WINDOW_MS, deps.doc)))
  const victory = readVictory(deps.scoreboard, deps.outcome?.code)
  // LE SCORE SE LIT À LA BORNE DE FIN, pas à l'image courante (décision D-B4 du DOM) : la
  // lecture peut être allée au-delà, et le panneau n'a plus rien à dire après la fin.
  const finalScore = deps.playWindow
    ? readScoreBanner(timeline, deps.scoreboard, deps.xuidMeta, deps.playWindow.endFrame)
    : null
  const neutral = neutralStatusStyle(deps.ink)

  const victoryPanel = (): OverlayPanel | null => {
    const label = deps.outcome?.label
    if (!label || !victory) return null
    const mine = victory.mine
    const rows = mine ? deps.scoreboard.filter((r) => r.team_side === mine.teamSide) : []
    return {
      status: label,
      // L'ÉGALITÉ N'EMPRUNTE RIEN (décision D-B1) : ni camp, ni logo, ni nom.
      statusStyle: mine ? deps.teamStyle : neutral,
      label: mine ? resolveTeamLabel(rows, mine.teamSide, t) : null,
      score: exportFinalScore(deps.outcome?.finalScore, finalScore),
      logo: mine ? deps.logo : null,
      veil: true,
    }
  }

  const panelAt = (frame: number): OverlayPanel | null => {
    // LA FIN DE MATCH L'EMPORTE (cf. l'en-tête) : elle est testée en premier. MAIS elle ne
    // CONFISQUE pas l'image quand elle n'a rien à dire — sans libellé d'issue, le DOM affiche
    // quand même le message inter-manche, puisque ses deux surimpressions sont montées
    // indépendamment. Un `return` sec ici faisait diverger l'export de la page.
    if (deps.playWindow && frame >= deps.playWindow.endFrame) {
      const fin = victoryPanel()
      if (fin) return fin
    }
    const active = activeRoundTransition(transitions, frame, breakFrames)
    if (!active) return null
    return {
      status: t.roundOverFmt(active.endedIndex),
      statusStyle: neutral,
      // Pas de voile : une manche qui s'achève ne masque pas le terrain (parité DOM).
      veil: false,
    }
  }

  return { panelAt }
}
