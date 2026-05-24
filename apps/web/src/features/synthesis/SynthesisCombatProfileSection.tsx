/**
 * SynthesisCombatProfileSection — profil combat 3 axes (PLAN_COMBAT_PROFILE_WIRING Phase 1).
 *
 * Affiche OC/DR via CombatYieldBar + 3 badges de style quand au moins 15 matchs.
 */
import { CombatYieldBar } from '@/components/ui/combat-yield-bar'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import type { CombatProfileBlock } from '@/lib/api/types'

const STYLE_OFFENSIVE_LABELS: Record<string, string> = {
  precis: 'Offensif précis',
  equilibre: 'Offensif équilibré',
  genereux: 'Offensif généreux',
}

const STYLE_DEFENSIVE_LABELS: Record<string, string> = {
  resistant: 'Défensif résistant',
  solide: 'Défensif solide',
  fragile: 'Défensif fragile',
}

const STYLE_ACTIVITY_LABELS: Record<string, string> = {
  actif: 'Très actif',
  modere: 'Modéré',
  discret: 'Discret',
}

interface SynthesisCombatProfileSectionProps {
  combatProfile: CombatProfileBlock
}

export function SynthesisCombatProfileSection({ combatProfile }: SynthesisCombatProfileSectionProps) {
  const hasStyles =
    combatProfile.style_offensive != null ||
    combatProfile.style_defensive != null ||
    combatProfile.style_activity != null

  return (
    <Card>
      <CardHeader className="pb-2">
        <CardTitle className="text-base">Profil de combat</CardTitle>
        <p className="text-xs text-muted-foreground">{combatProfile.match_count} matchs analysés</p>
      </CardHeader>
      <CardContent className="space-y-3">
        <div className="flex flex-col items-center gap-1">
          <div className="flex items-center gap-3 text-xs text-muted-foreground">
            <span className="w-14 text-right">Offensif</span>
            <CombatYieldBar
              offensiveConversion={combatProfile.avg_oc}
              defensiveResistance={combatProfile.avg_dr}
            />
            <span className="w-14">Défensif</span>
          </div>
          <div className="flex gap-6 text-xs text-muted-foreground">
            <span>OC {(combatProfile.avg_oc * 100).toFixed(0)}%</span>
            <span>DR {Math.round((combatProfile.avg_dr - 1) * 100)}%</span>
          </div>
        </div>

        {hasStyles && (
          <div className="flex flex-wrap gap-2">
            {combatProfile.style_offensive && (
              <Badge variant="outline">
                {STYLE_OFFENSIVE_LABELS[combatProfile.style_offensive] ?? combatProfile.style_offensive}
              </Badge>
            )}
            {combatProfile.style_defensive && (
              <Badge variant="outline">
                {STYLE_DEFENSIVE_LABELS[combatProfile.style_defensive] ?? combatProfile.style_defensive}
              </Badge>
            )}
            {combatProfile.style_activity && (
              <Badge variant="secondary">
                {STYLE_ACTIVITY_LABELS[combatProfile.style_activity] ?? combatProfile.style_activity}
              </Badge>
            )}
          </div>
        )}
      </CardContent>
    </Card>
  )
}
