/**
 * Route /players/$playerSlug/home — Accueil Mission Control.
 */
import { createFileRoute } from '@tanstack/react-router'
import { HomePage } from '@/features/home/HomePage'

export const Route = createFileRoute('/players/$playerSlug/home')({
  component: HomePage,
})
