/**
 * AdminAccessPage — onglet Accès : comptes utilisateurs + codes d'invitation
 * (sections extraites 1:1 de l'ancienne AdminPage).
 */
import { useAppShellStore } from '@/stores/appShellStore'
import { UsersSection } from '../sections/UsersSection'
import { InvitesSection } from '../sections/InvitesSection'

export function AdminAccessPage() {
  const currentUsername = useAppShellStore((s) => s.currentUsername)
  return (
    <div className="space-y-8">
      <UsersSection currentUsername={currentUsername} />
      <InvitesSection />
    </div>
  )
}
