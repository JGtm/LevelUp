/**
 * ExplorerMatchesTable.placement — badge « En placement » partagé par les
 * colonnes Perf / ΔPerf / Note du tableau Matchs de l'Explorer (V72-32).
 *
 * Contexte : ces 3 colonnes affichaient un simple "-" quand la note manquait,
 * qu'elle manque parce que le joueur est encore en phase de placement (LUSR ou
 * CSR — moins de sync.LUSRPlacementThreshold/MinMatchesPerChainForRelative=10
 * matchs dans la chaîne du match, cf. match_history_placement.go +
 * internal/sync/performance.go côté back) OU parce qu'aucune note ne sera
 * JAMAIS calculée pour ce type de match (absence structurelle — pas de cas
 * connu à ce jour, tout match reçoit une chaîne via GetPerformanceChain, mais
 * on ne suppose rien : sans signal de placement, "-" reste, sans fabriquer un
 * faux état).
 *
 * IMPORTANT — vérifié en direct sur JGtm (V72-32, 2026-07-25) : le "-" restant
 * n'est PAS toujours du placement. Ses 3 derniers matchs BTB (24/07) montrent
 * Note=LUSR + Rang=Or II (chaîne "btb" établie depuis 2023, donc HORS
 * placement) mais Perf/ΔPerf="-" quand même — sur les mêmes matchs, en même
 * temps, un coéquipier (Chocoboflor) a bien son Perf calculé. Cause probable :
 * `batchComputePerformanceScores` n'a pas tourné (ou a échoué silencieusement)
 * sur les derniers matchs de CE joueur spécifiquement lors de son dernier sync
 * — un problème de fraîcheur des données côté back, PAS un état à exposer ici.
 * Ne pas élargir ce badge pour couvrir ce cas : on ne sait pas distinguer ce
 * lag transitoire d'une absence permanente, donc "-" est la seule réponse
 * honnête (cf. règle "pas de faux état"). Investiguer côté sync/backfill perf,
 * pas côté affichage.
 *
 * Signal réutilisé : `placement_done`/`placement_total` — DÉJÀ calculés côté
 * service (`applyMatchPlacements`, match_history_placement.go) et DÉJÀ utilisés
 * par la colonne "Rang" pour afficher "X/Y" à la place du palier. Pour BTB
 * (Big Team Battle, le cas rapporté), la chaîne de performance ET la chaîne
 * LUSR sont IDENTIQUES (GetPerformanceChain retombe sur GetLUSRChain pour tout
 * match non classé/non Firefight) et les deux seuils valent 10 — le signal de
 * placement de la colonne Rang est donc EXACT pour Perf/ΔPerf, pas une
 * approximation. Pour les matchs classés (CSR), c'est une bonne approximation
 * (même sémantique "en cours de calibration") ; pour Firefight (PvE), aucun
 * signal de placement n'existe aujourd'hui → "-" inchangé (cf. skill_config.go).
 *
 * Libellé court réutilisé de `common.home.rank_placement` (déjà servi par les
 * cards Accueil + `match-view/i18n.ts` `rankPlacement`, même sentinelle
 * "Placement" côté back) : FR "En placement" / EN "In placement" — cohérence
 * inter-écrans plutôt qu'un 3e libellé légèrement différent pour cette seule
 * table (CLAUDE.md §6, ≤2 copies avant centralisation — ici on RÉUTILISE la
 * copie existante). Tooltip : `explorer.briefing.placement_remaining` (même
 * manifest que la table, déjà utilisé par le bloc "Classement" du briefing
 * Explorer pour le même concept), avec le nombre de matchs restants avant la
 * 1re note = placement_total - placement_done (matchs restants APRÈS ce match).
 */
import { formatMessage, type ManifestLocale } from '@/lib/i18n/format'
import { commonManifest } from '@/lib/i18n/generated/common'
import { explorerManifest } from '@/lib/i18n/generated/explorer'
import type { ExplorerMatchRow } from '@/lib/api/types'

/**
 * true si le match porte un signal de placement (LUSR ou CSR) — même source
 * que la colonne "Rang". Ne préjuge PAS que perf_score/rating_type sont nuls :
 * l'appelant vérifie déjà value == null avant d'appeler ce composant.
 */
// eslint-disable-next-line react-refresh/only-export-components -- fonction pure partagée avec le composant PlacementPendingCell (même pattern que SessionMatchesTable.toExplorerRow).
export function hasPlacementSignal(row: ExplorerMatchRow): boolean {
  return row.placement_done != null && row.placement_total != null
}

/**
 * Cellule de remplacement du "-" pour Perf / ΔPerf / Note quand la note
 * manque PARCE QUE le classement est en placement. Compacte (colonnes
 * étroites, `whitespace-nowrap` déjà posé par le <td> parent) : le libellé
 * court est visible, le détail (matchs restants) est dans le `title` natif —
 * même mécanisme que les colonnes Playlist/Mode tronquées de cette table.
 */
export function PlacementPendingCell({
  row,
  locale,
}: {
  row: ExplorerMatchRow
  locale: ManifestLocale
}) {
  const done = row.placement_done ?? 0
  const total = row.placement_total ?? 0
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
