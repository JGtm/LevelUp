/**
 * ExplorerMatchesTable.placement — badge « En placement » des colonnes
 * Perf / ΔPerf / Note du tableau Matchs de l'Explorer (V72-32, corrigé V72-34).
 *
 * Contexte : ces 3 colonnes affichaient un simple "-" quand la note manquait,
 * qu'elle manque parce que le classement est en phase de placement OU parce
 * qu'aucune note ne sera JAMAIS calculée pour ce match (absence structurelle).
 * Sans signal de placement, "-" reste — on ne fabrique pas un faux état.
 *
 * DEUX SIGNAUX DISTINCTS (V72-34) — c'est le cœur de la correction :
 *
 *  - `placement_done`/`placement_total` : placement du CLASSEMENT (LUSR/CSR),
 *    consommé par la colonne "Note"/"Rang" (applyCSRPlacements /
 *    applyLUSRPlacements, match_history_placement.go). applyLUSRPlacements
 *    SAUTE les matchs qui ont déjà un LUSR — correct pour la colonne Note.
 *  - `perf_placement_done`/`perf_placement_total` : placement de la CHAÎNE DE
 *    PERFORMANCE, consommé par Perf/ΔPerf (applyPerfPlacements, même fichier).
 *    Posé dès que la chaîne perf du match compte moins de
 *    MinMatchesPerChainForRelative (=10) matchs perf-éligibles.
 *
 * Pourquoi deux signaux plutôt qu'un : les deux phases sont INDÉPENDANTES. Le
 * cas qui a motivé la correction (JGtm, 3 matchs BTB du 24/07, vérifié en
 * direct) a un LUSR établi + Rang "Or II" (chaîne LUSR calibrée de longue date)
 * MAIS seulement 8 matchs BTB perf-éligibles au total → chaîne perf sous le
 * seuil → aucun perf_score possible. Le signal partagé ne couvrait pas ce cas
 * (applyLUSRPlacements saute les matchs avec LUSR) et Perf/ΔPerf affichaient un
 * tiret nu. V72-32 avait diagnostiqué à tort un retard de sync : la cause
 * réelle est structurelle et se calcule. Le placement perf est un compteur DE
 * CHAÎNE (identique sur tous les matchs de la chaîne : « 8/10 »), pas un rang
 * par-match — c'est une progression de calibration.
 *
 * Chaîne perf > seuil sans perf_score = trou structurel ou retard de sync :
 * AUCUN badge (le back ne renseigne pas le signal) → "-" inchangé, on ne
 * masque pas un défaut de fraîcheur derrière un état inventé.
 *
 * Libellé court réutilisé de `common.home.rank_placement` (déjà servi par les
 * cards Accueil + `match-view/i18n.ts` `rankPlacement`, même sentinelle
 * "Placement" côté back) : FR "En placement" / EN "In placement" — cohérence
 * inter-écrans plutôt qu'un 3e libellé légèrement différent pour cette seule
 * table (CLAUDE.md §6, ≤2 copies avant centralisation — ici on RÉUTILISE la
 * copie existante). Tooltip : `explorer.briefing.placement_remaining` (même
 * manifest que la table, déjà utilisé par le bloc "Classement" du briefing
 * Explorer pour le même concept), avec le nombre de matchs restants avant la
 * 1re note = total - done (matchs restants APRÈS ce match).
 */
import { formatMessage, type ManifestLocale } from '@/lib/i18n/format'
import { commonManifest } from '@/lib/i18n/generated/common'
import { explorerManifest } from '@/lib/i18n/generated/explorer'
import type { ExplorerMatchRow } from '@/lib/api/types'

/**
 * true si le match porte le signal de placement du CLASSEMENT (LUSR/CSR) —
 * source de la colonne "Note"/"Rang". Ne préjuge PAS que rating_type est nul :
 * l'appelant vérifie déjà value == null avant d'appeler ce composant.
 */
// eslint-disable-next-line react-refresh/only-export-components -- fonction pure partagée avec le composant PlacementPendingCell (même pattern que SessionMatchesTable.toExplorerRow).
export function hasPlacementSignal(row: ExplorerMatchRow): boolean {
  return row.placement_done != null && row.placement_total != null
}

/**
 * true si le match porte le signal de placement de la CHAÎNE DE PERFORMANCE —
 * source des colonnes Perf/ΔPerf. Indépendant de hasPlacementSignal : un match
 * peut avoir une Note LUSR établie ET une chaîne perf encore en calibration
 * (cas BTB, cf. en-tête de fichier).
 */
// eslint-disable-next-line react-refresh/only-export-components -- fonction pure partagée avec le composant PlacementPendingCell.
export function hasPerfPlacementSignal(row: ExplorerMatchRow): boolean {
  return row.perf_placement_done != null && row.perf_placement_total != null
}

/**
 * Cellule de remplacement du "-" pour Perf / ΔPerf / Note quand la note manque
 * PARCE QUE la phase correspondante est en placement. Compacte (colonnes
 * étroites, `whitespace-nowrap` déjà posé par le <td> parent) : le libellé
 * court est visible, le détail (matchs restants) est dans le `title` natif —
 * même mécanisme que les colonnes Playlist/Mode tronquées de cette table.
 *
 * `variant` choisit la PAIRE DE CHAMPS lue, jamais le rendu (identique dans les
 * deux cas — même concept utilisateur) : "rating" (défaut) = placement du
 * classement pour la colonne Note ; "perf" = placement de la chaîne de
 * performance pour Perf/ΔPerf.
 */
export function PlacementPendingCell({
  row,
  locale,
  variant = 'rating',
}: {
  row: ExplorerMatchRow
  locale: ManifestLocale
  variant?: 'rating' | 'perf'
}) {
  const done = (variant === 'perf' ? row.perf_placement_done : row.placement_done) ?? 0
  const total = (variant === 'perf' ? row.perf_placement_total : row.placement_total) ?? 0
  const remaining = Math.max(0, total - done)
  const label = formatMessage(commonManifest, 'common.home.rank_placement', locale)
  const tooltip = formatMessage(
    explorerManifest,
    'explorer.briefing.placement_remaining',
    locale,
    { n: remaining },
  )
  return (
    <span className="text-2xs italic text-muted-foreground" title={tooltip}>
      {label}
    </span>
  )
}
