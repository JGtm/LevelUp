/**
 * AdminManagementPage — onglet Gestion : administration (DC-8/A3.4).
 * Regroupe les ex-onglets Accès (comptes utilisateurs) et Titres (registre
 * multi-titres + diagnostic par titre). Composants existants composés tels
 * quels. La valeur opérationnelle résiduelle du Lab supprimé (diagnostic par
 * titre) vit dans la section Titres (DC-9).
 */
import { useAppShellStore } from '@/stores/appShellStore'
import { useAdminT } from '../useAdminText'
import { UsersSection } from '../sections/UsersSection'
import { AdminTitlesPage } from '../titles/AdminTitlesPage'

export function AdminManagementPage() {
  const currentUsername = useAppShellStore((s) => s.currentUsername)
  const tA = useAdminT()
  return (
    <div className="space-y-10">
      <section className="space-y-4">
        <h2 className="border-b pb-2 text-base font-semibold text-foreground">
          {tA('admin.management.section_users')}
        </h2>
        <UsersSection currentUsername={currentUsername} />
      </section>

      <section className="space-y-4">
        <h2 className="border-b pb-2 text-base font-semibold text-foreground">
          {tA('admin.management.section_titles')}
        </h2>
        <AdminTitlesPage />
      </section>
    </div>
  )
}
