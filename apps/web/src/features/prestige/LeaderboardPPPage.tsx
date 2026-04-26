/**
 * LeaderboardPPPage — page Leaderboard PP dans Communauté.
 *
 * Auto-peuplée depuis l'escouade et les Relations DB (pas de gestion d'amis manuelle).
 * Affichage décomposé brut/bonus/total selon Axe 5 du plan conceptuel.
 *
 * Phase 5 : structure complète avec composant LeaderboardPP. La récupération
 * du leaderboard cross-amis dépend du wiring backend (PRESTIGE_ENABLED + sources
 * amis dérivées de squad_member + Relations DB).
 */
import { useState } from 'react'
import { useAppShellStore } from '@/stores/appShellStore'
import { LeaderboardPP, type LeaderboardEntry } from './components/LeaderboardPP'

export function LeaderboardPPPage() {
  const currentPlayer = useAppShellStore((s) => s.currentPlayer)
  const [period, setPeriod] = useState<'week' | 'month' | 'all'>('all')

  // Phase 5 minimale : pas de fetch backend tant que le leaderboard cross-amis
  // n'est pas exposé. Le composant LeaderboardPP gère l'état vide proprement.
  const entries: LeaderboardEntry[] = []

  return (
    <div className="space-y-4 p-4">
      <header className="space-y-1">
        <h1 className="text-2xl font-bold">Leaderboard PP</h1>
        <p className="text-sm text-muted-foreground">
          Classement Prestige entre amis dérivés de ton escouade et tes relations.
        </p>
      </header>

      {currentPlayer ? (
        <LeaderboardPP entries={entries} period={period} onPeriodChange={setPeriod} />
      ) : (
        <div className="rounded-lg border border-dashed border-border p-6 text-center text-sm text-muted-foreground">
          Sélectionne un joueur pour voir le leaderboard.
        </div>
      )}
    </div>
  )
}
