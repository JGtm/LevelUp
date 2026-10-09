/**
 * Route /groups — page « Amis et groupes » : amis du joueur actif + groupes.
 */
import { createFileRoute } from '@tanstack/react-router'
import { GroupsPage } from '@/features/groups/GroupsPage'

export const Route = createFileRoute('/groups')({
  component: GroupsPage,
})
