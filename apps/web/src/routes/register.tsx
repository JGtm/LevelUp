/**
 * Route /register — inscription utilisateur.
 */
import { createFileRoute } from '@tanstack/react-router'
import { RegisterPage } from '@/features/auth/RegisterPage'

export const Route = createFileRoute('/register')({
  component: RegisterPage,
})
