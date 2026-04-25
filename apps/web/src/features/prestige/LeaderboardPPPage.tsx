/**
 * LeaderboardPPPage — page Leaderboard PP dans Communauté.
 *
 * Auto-peuplée depuis l'escouade et les Relations DB (pas de gestion d'amis manuelle).
 * Affichage décomposé brut/bonus/total selon Axe 5 du plan conceptuel.
 *
 * Phase 5 minimale : placeholder fonctionnel. La récupération du leaderboard
 * cross-amis dépend du wiring backend (PRESTIGE_ENABLED + sources amis dérivées).
 */
import { useAppShellStore } from '@/stores/appShellStore'

export function LeaderboardPPPage() {
  const currentPlayer = useAppShellStore((s) => s.currentPlayer)

  return (
    <div className="space-y-4 p-4">
      <header className="space-y-1">
        <h1 className="text-2xl font-bold">Leaderboard PP</h1>
        <p className="text-sm text-muted-foreground">
          Classement Prestige entre amis dérivés de ton escouade et tes relations.
        </p>
      </header>

      <div className="rounded-lg border border-dashed border-border p-8 text-center">
        <p className="text-sm text-muted-foreground">
          {currentPlayer
            ? "Le module Prestige n'est pas encore activé sur ce serveur. Le leaderboard apparaîtra quand PRESTIGE_ENABLED=true."
            : 'Sélectionne un joueur pour voir le leaderboard.'}
        </p>
      </div>
    </div>
  )
}
