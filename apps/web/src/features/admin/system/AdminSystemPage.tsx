/**
 * AdminSystemPage — onglet Système : contention DB (B-swap), santé des tokens
 * auth et intégrité des données (sections extraites 1:1 de l'ancienne
 * AdminPage).
 */
import { DBContentionSection } from '../sections/DBContentionSection'
import { TokenHealthSection } from '../sections/TokenHealthSection'
import { InvariantsSection } from '../sections/InvariantsSection'
import { PersistPhasesSection } from './PersistPhasesSection'

export function AdminSystemPage() {
  return (
    <div className="space-y-8">
      <DBContentionSection />
      <PersistPhasesSection />
      <TokenHealthSection />
      <InvariantsSection />
    </div>
  )
}
