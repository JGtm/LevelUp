/**
 * AdminDataPage — onglet Données : « mon warehouse est-il intègre ? » (DC-8).
 * Fusion des anciens onglets Qualité données + Convergence + de la section
 * Invariants (ex-Système) en sections d'une même page. Les composants existants
 * sont composés tels quels (déplacés, pas réécrits — A3.2).
 */
import { useAdminT } from '../useAdminText'
import { AdminDataQualityPage } from '../data-quality/AdminDataQualityPage'
import { AdminConvergencePage } from '../convergence/AdminConvergencePage'
import { InvariantsSection } from '../sections/InvariantsSection'

export function AdminDataPage() {
  const tA = useAdminT()
  return (
    <div className="space-y-10">
      <section className="space-y-4">
        <h2 className="border-b pb-2 text-base font-semibold text-foreground">
          {tA('admin.data.section_quality')}
        </h2>
        <AdminDataQualityPage />
      </section>

      <section className="space-y-4">
        <h2 className="border-b pb-2 text-base font-semibold text-foreground">
          {tA('admin.data.section_convergence')}
        </h2>
        <AdminConvergencePage />
      </section>

      <section className="space-y-4">
        <h2 className="border-b pb-2 text-base font-semibold text-foreground">
          {tA('admin.data.section_invariants')}
        </h2>
        <InvariantsSection />
      </section>
    </div>
  )
}
