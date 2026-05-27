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

const STYLE_OFFENSIVE_LABELS: Record<string, string> = {
  precis: 'Offensif précis',
  equilibre: 'Équilibré',
  genereux: 'Offensif généreux',
}

const STYLE_DEFENSIVE_LABELS: Record<string, string> = {
  resistant: 'Défensif résistant',
  solide: 'Solide',
  fragile: 'Fragile',
}

const STYLE_ACTIVITY_LABELS: Record<string, string> = {
  actif: 'Engagement actif',
  modere: 'Engagement modéré',
  discret: 'Engagement discret',
}

export interface SquadCombatProfileRowProps {
  playerCards: PlayerScoreCard[]
  kpisByXuid: Record<string, KPIStats>
}

export function SquadCombatProfileRow({ playerCards, kpisByXuid }: SquadCombatProfileRowProps) {
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
                      {STYLE_OFFENSIVE_LABELS[profile.style_offensive] ?? profile.style_offensive}
                    </Badge>
                  )}
                  {profile.style_defensive && (
                    <Badge variant="outline" className="text-xs">
                      {STYLE_DEFENSIVE_LABELS[profile.style_defensive] ?? profile.style_defensive}
                    </Badge>
                  )}
                  {profile.style_activity && (
                    <Badge variant="secondary" className="text-xs">
                      {STYLE_ACTIVITY_LABELS[profile.style_activity] ?? profile.style_activity}
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
