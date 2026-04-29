/**
 * SetupPage helpers partagés entre les Step* sous-composants.
 *
 * P8.4 (revue 2026-04-29) : extrait de SetupPage.tsx.
 */
import type { ApiError } from '@/lib/api/client'

export function getApiErrorMessage(error: unknown, fallback: string): string {
  if (
    error &&
    typeof error === 'object' &&
    'message' in error &&
    typeof (error as ApiError).message === 'string'
  ) {
    return (error as ApiError).message
  }
  return fallback
}
