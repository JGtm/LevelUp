/**
 * AdminAccessPage — onglet Accès : comptes utilisateurs (rôles, suppression).
 *
 * Les codes d'invitation ont migré vers la page end-user /groups (invitation
 * "rejoindre un groupe" via login Xbox SSO) ; l'Admin ne garde que la gestion
 * des comptes, qui relève de l'opérateur.
 */
import { useAppShellStore } from '@/stores/appShellStore'
import { UsersSection } from '../sections/UsersSection'

export function AdminAccessPage() {
  const currentUsername = useAppShellStore((s) => s.currentUsername)
  return (
    <div className="space-y-8">
      <UsersSection currentUsername={currentUsername} />
    </div>
  )
}
