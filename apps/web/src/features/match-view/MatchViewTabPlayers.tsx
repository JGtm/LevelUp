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
        {/* Némésis + Souffre-douleur pleine largeur, puis Antagonistes | Assistances */}
        <div className="flex flex-col gap-4">
          <MatchNemesisCards
            nemesis={nemesis}
            scoreboard={scoreboard}
            meXUID={meXUID}
            t={t}
          />
          <DuelsRow
            killerVictim={killerVictim}
            assistPairs={assistPairs}
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

/**
 * DuelsRow — ANTAGONISTES et ASSISTANCES CÔTE À CÔTE, et l'un seul quand l'autre n'a rien
 * à dire.
 *
 * Les deux graphes sont des MIROIRS (qui m'a tué / qui a aidé à tuer) : les empiler les
 * faisait lire l'un après l'autre alors qu'ils se comparent (demande utilisateur du
 * 2026-09-03). Sur une rangée, chacun tient une demi-largeur à partir de `lg` et reprend
 * toute la largeur en dessous.
 *
 * LE CAS DE LA CELLULE FANTÔME, ET IL EST TRAITÉ EXPLICITEMENT. `MatchAssistChart` ne rend
 * RIEN quand le bloc d'assistances est absent — le match n'a aucune ligne de film, ce qui
 * est le cas de la quasi-totalité des matchs. Dans une grille à deux colonnes, son absence
 * laisserait les antagonistes sur une demi-largeur avec un vide à droite : un trou qui se
 * lit « il manque quelque chose ». On ne pose donc PAS la grille dans ce cas — le graphe
 * restant reprend la pleine largeur, exactement comme avant la rangée.
 *
 * La condition est celle de la porte 1 de `MatchAssistChart` (bloc absent = rien) ; les deux
 * sont couvertes par leurs tests respectifs.
 */
function DuelsRow({
  killerVictim,
  assistPairs,
  scoreboard,
  meXUID,
  t,
}: {
  killerVictim: MatchKillerVictimPair[]
  assistPairs: MatchAssistPairs | undefined
  scoreboard: MatchScoreboardRow[]
  meXUID: string | null
  t: MatchViewText
}) {
  const antagonistes = (
    <MatchAntagonistChart pairs={killerVictim} scoreboard={scoreboard} meXUID={meXUID} t={t} />
  )
  if (assistPairs == null) return antagonistes
  return (
    <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
      {antagonistes}
      <MatchAssistChart block={assistPairs} scoreboard={scoreboard} meXUID={meXUID} t={t} />
    </div>
  )
}
