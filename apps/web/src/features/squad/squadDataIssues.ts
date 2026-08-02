/**
 * squadDataIssues — traduction des dégradations de chargement remontées par
 * l'API Escouade (`data_issues`) en messages affichables FR/EN.
 *
 * Le backend n'envoie que des CODES stables (domain.DataIssue) : les libellés
 * vivent ici, dans le dictionnaire de la feature. Un code inconnu (backend plus
 * récent que le front) dégrade en message générique — jamais de ligne muette,
 * puisque le but même de ce canal est que l'utilisateur voie que ses chiffres
 * sont partiels.
 */
import type { DataIssue } from '@/lib/api/types'
import type { SquadText } from './i18n'

export function formatDataIssues(issues: DataIssue[] | undefined, t: SquadText): string[] {
  const td = t.dataIssues
  return (issues ?? []).map((issue) => {
    switch (issue.code) {
      case 'teammate_matches':
        return td.teammateMatches(issue.detail ?? '')
      case 'heatmap_teammate':
        return td.heatmapTeammate(issue.detail ?? '')
      case 'main_team_participants':
        return td.mainTeamParticipants
      case 'map_stats':
        return td.mapStats
      default:
        return td.unknown(issue.code)
    }
  })
}
