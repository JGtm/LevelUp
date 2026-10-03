/**
 * AdminSystemPage — onglet Système : bas niveau (DC-8). Contention DB (B-swap),
 * perf des phases persist, sons du rejeu 2D, tail de logs bruts (ex-onglet Logs)
 * et backup. TokenHealthSection a migré vers Sync, InvariantsSection vers Données
 * (A3.3 / A3.2 — chaque information n'apparaît plus qu'à UN endroit).
 */
import { DBContentionSection } from '../sections/DBContentionSection'
import { PersistPhasesSection } from './PersistPhasesSection'
import { AdminBackupSection } from './AdminBackupSection'
import { AdminReplaySoundSection } from './AdminReplaySoundSection'
import { ResourcesSection } from './ResourcesSection'
import { AdminLogsPage } from '../logs/AdminLogsPage'

export function AdminSystemPage() {
  return (
    <div className="space-y-8">
      <ResourcesSection />
      <DBContentionSection />
      <PersistPhasesSection />
      <AdminReplaySoundSection />
      <AdminBackupSection />
      <AdminLogsPage />
    </div>
  )
}
