/**
 * AdminDetectionsPage — onglet Détections : « que dois-je traiter ? » (DC-8).
 * Triage des détections persistées avec cycle de vie (open/acked/muted/resolved).
 * Le panneau vit dans monitoring/ (livré en A2 sous l'onglet Logs) — cette page
 * lui donne son onglet dédié.
 */
import { DetectionsPanel } from '../monitoring/DetectionsPanel'

export function AdminDetectionsPage() {
  return (
    <div className="space-y-8">
      <DetectionsPanel />
    </div>
  )
}
