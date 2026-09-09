/**
 * intensityTooltipText — texte explicatif UNIQUE du profil « Intensité ».
 *
 * Il en existait TROIS copies (Sessions, Séries temporelles, Escouade), déjà divergentes
 * de formulation : « tes frags » ici, « les frags » là, « les frags du joueur » ailleurs.
 * Même concept, même graphe, trois textes à maintenir — le patron de
 * `FdaGapTooltipText` existait déjà pour exactement ce cas (CLAUDE.md n°6).
 *
 * Une FONCTION et non un composant, contrairement à `FdaGapTooltipText` :
 * `SquadIntensityProfileChart` reçoit son aide en `string` et la passe à `InfoTooltip`
 * par prop. Une chaîne convient aux trois appelants, un nœud React non.
 *
 * `withTeam` ajoute la phrase de la courbe agrégée d'équipe — la seule différence
 * légitime entre les trois surfaces, portée par sa propre clé.
 *
 * Garde-rail : `intensityTooltipText.guard.test.ts` interdit le retour des littéraux.
 */
import { formatMessage, type ManifestLocale } from '@/lib/i18n/format'
import { commonManifest } from '@/lib/i18n/generated/common'

export function intensityTooltipText(
  locale: ManifestLocale,
  opts?: { withTeam?: boolean },
): string {
  const base = formatMessage(commonManifest, 'common.charts.intensity_tooltip', locale)
  if (!opts?.withTeam) return base
  return `${base} ${formatMessage(commonManifest, 'common.charts.intensity_tooltip_team', locale)}`
}
