/**
 * linkDouble — un double de `<Link>` (TanStack Router) pour les tests de composants rendus sans
 * routeur : une ancre à l'URL RÉELLE (`resolveRoutePath`), la recherche exposée en `data-search`.
 *
 * Usage : `vi.mock('@tanstack/react-router', async (orig) => ({ ...(await orig()),
 * Link: (await import('@/test/linkDouble')).LinkDouble }))`.
 */
import type { ReactNode } from 'react'

import { resolveRoutePath } from './routeLinkMock'

interface LinkDoubleProps {
  to: string
  params?: Record<string, string | undefined>
  search?: Record<string, unknown>
  children?: ReactNode
  [attribut: string]: unknown
}

export function LinkDouble({ to, params, search, children, ...rest }: LinkDoubleProps) {
  return (
    <a href={resolveRoutePath(to, params)} data-search={search ? JSON.stringify(search) : undefined} {...rest}>
      {children}
    </a>
  )
}
