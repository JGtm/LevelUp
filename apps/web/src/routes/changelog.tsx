/**
 * Route /changelog — journal des modifications.
 */
import { createFileRoute } from '@tanstack/react-router'
import { ChangelogPage } from '@/features/changelog/ChangelogPage'

export const Route = createFileRoute('/changelog')({
  component: ChangelogPage,
})
