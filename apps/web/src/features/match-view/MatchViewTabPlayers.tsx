/**
 * MatchViewTabPlayers — contenu de l'onglet « Joueurs » de la page match.
 *
 * Extrait de MatchViewPage.tsx au passage à 3 onglets (Général / Chronologie /
 * Joueurs, 2026-08-24) : les trois sous-sections de l'ancien onglet Détails
 * (Duels & confrontations, Tableau des scores, Historique des rencontres) sont
 * DÉPLACÉES telles quelles, sous-titres compris.
 */
import type {
  MatchAssistPairs,
  MatchCitationSnippet,
  MatchEncounterRow,
  MatchHighlightEvent,
  MatchKillerVictimPair,
  MatchNemesisRow,
  MatchRosterRow,
  MatchScoreboardRow,
  MatchViewHeader,
  MatchViewRank,
} from '@/lib/api/types'
import type { Locale } from '@/lib/i18n/locale'
import { DetailSection } from './DetailSection'
import { MatchAntagonistChart } from './MatchAntagonistChart'
import { MatchAssistChart } from './MatchAssistChart'
import { MatchEncountersTable } from './MatchEncountersTable'
import { MatchFragDiffChart } from './MatchFragDiffChart'
import { MatchNemesisCards } from './MatchNemesisCards'
import { MatchScoreboard } from './MatchScoreboard'
import type { MatchViewText } from './i18n'

interface Props {
  header: MatchViewHeader
  rank: MatchViewRank
  scoreboard: MatchScoreboardRow[]
  roster: MatchRosterRow[]
  nemesis: MatchNemesisRow[]
  killerVictim: MatchKillerVictimPair[]
  /** Bloc « assistances » — absent quand le match n'a aucune ligne de film. */
  assistPairs: MatchAssistPairs | undefined
  highlightEvents: MatchHighlightEvent[]
  citations: MatchCitationSnippet[]
  encounters: MatchEncounterRow[]
  meXUID: string | null
  friendGamertags: readonly string[]
  locale: Locale
  t: MatchViewText
}

export function MatchViewTabPlayers({
  header,
  rank,
  scoreboard,
  roster,
  nemesis,
  killerVictim,
  assistPairs,
  highlightEvents,
  citations,
  encounters,
  meXUID,
  friendGamertags,
  locale,
  t,
}: Props) {
  return (
    <>
      {/* §2 — Duels & confrontations (face-à-face) */}
      <DetailSection title={t.sectionDuels}>
        {/* Némésis + Souffre-douleur | Antagonistes */}
        <div className="flex flex-col gap-4">
          <MatchNemesisCards
            nemesis={nemesis}
            scoreboard={scoreboard}
            meXUID={meXUID}
            t={t}
          />
          <MatchAntagonistChart
            pairs={killerVictim}
            scoreboard={scoreboard}
            meXUID={meXUID}
            t={t}
          />
          {/* Assistances (assistant → tueur assisté) — sous les antagonistes, dont il
              est le miroir. Ne rend rien quand le match n'a aucune ligne de film. */}
          <MatchAssistChart
            block={assistPairs}
            scoreboard={scoreboard}
            meXUID={meXUID}
            t={t}
          />
        </div>

        {/* Frags différentiel cumulé — descendu ici (après Antagonistes) */}
        <MatchFragDiffChart
          events={highlightEvents}
          scoreboard={scoreboard}
          roster={roster}
          pairs={killerVictim}
          meXUID={meXUID}
          t={t}
          friendGamertags={friendGamertags}
        />
      </DetailSection>

      {/* §3 — Tableau des scores (table sortie de son bloc) */}
      <DetailSection title={t.scoreboardTitle}>
        <MatchScoreboard
          rows={scoreboard}
          killerVictim={killerVictim}
          citations={citations}
          header={header}
          rank={rank}
          t={t}
        />
      </DetailSection>

      {/* §4 — Historique des rencontres (table sortie de son bloc) */}
      <DetailSection title={t.sectionEncounters}>
        <MatchEncountersTable
          rows={encounters}
          locale={locale}
          hideCardWrapper
        />
      </DetailSection>
    </>
  )
}
