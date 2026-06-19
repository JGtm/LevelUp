/**
 * Route /groups — gestion end-user des groupes/familles.
 */
import { createFileRoute } from '@tanstack/react-router'
import { GroupsPage } from '@/features/groups/GroupsPage'

export const Route = createFileRoute('/groups')({
  component: GroupsPage,
})
