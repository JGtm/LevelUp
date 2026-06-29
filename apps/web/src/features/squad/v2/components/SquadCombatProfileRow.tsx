/**
 * SquadCombatProfileRow — grille de profils combat par joueur (PLAN_COMBAT_PROFILE_WIRING Phase 2).
 *
 * Affiche un CombatYieldBar + badges de style par joueur du squad.
 * Masqué si aucun joueur n'a de profil combat calculé (< 15 matchs avec données dégâts).
 */
import { CombatYieldBar } from '@/components/ui/combat-yield-bar'
import { Badge } from '@/components/ui/badge'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import type { PlayerScoreCard, KPIStats } from '../types'
import { offensiveLabel, defensiveLabel, activityLabel } from '@/features/_shared/combatProfileLabels'
import { useAppShellStore } from '@/stores/appShellStore'
import type { ManifestLocale } from '@/lib/i18n/format'

export interface SquadCombatProfileRowProps {
  playerCards: PlayerScoreCard[]
  kpisByXuid: Record<string, KPIStats>
}

export function SquadCombatProfileRow({ playerCards, kpisByXuid }: SquadCombatProfileRowProps) {
  const locale = useAppShellStore((s) => s.locale) as ManifestLocale
  const playersWithProfile = playerCards.filter(
    (p) => kpisByXuid[p.xuid]?.combat_profile != null,
  )

  if (playersWithProfile.length === 0) return null

  return (
    <Card>
      <CardHeader className="pb-2">
        <CardTitle className="text-base">Profil de combat</CardTitle>
      </CardHeader>
      <CardContent>
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {playersWithProfile.map((player) => {
            const profile = kpisByXuid[player.xuid]?.combat_profile
            if (!profile) return null
            return (
              <div key={player.xuid} className="flex flex-col gap-2 rounded-lg border p-3">
                <span className="text-xs font-medium">{player.gamertag}</span>
                <div className="flex items-center gap-2 text-xs text-muted-foreground">
                  <span className="w-12 text-right">Off.</span>
                  <CombatYieldBar
                    offensiveConversion={profile.avg_oc}
                    defensiveResistance={profile.avg_dr}
                  />
                  <span className="w-12">Déf.</span>
                </div>
                <div className="flex flex-wrap gap-1">
                  {profile.style_offensive && (
                    <Badge variant="outline" className="text-xs">
                      {offensiveLabel(profile.style_offensive, locale)}
                    </Badge>
                  )}
                  {profile.style_defensive && (
                    <Badge variant="outline" className="text-xs">
                      {defensiveLabel(profile.style_defensive, locale)}
                    </Badge>
                  )}
                  {profile.style_activity && (
                    <Badge variant="secondary" className="text-xs">
                      {activityLabel(profile.style_activity, locale)}
                    </Badge>
                  )}
                </div>
              </div>
            )
          })}
        </div>
      </CardContent>
    </Card>
  )
}
