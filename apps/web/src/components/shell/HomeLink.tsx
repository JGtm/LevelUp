/**
 * HomeLink — le lien vers l'accueil de la marque : la page d'accueil du joueur actif, sinon la racine.
 * Source unique des trois barres de navigation (haut de page, barre latérale dépliée, rail replié) :
 * le garde-rail `homeLink.guard.test.ts` interdit d'en réécrire la cible ou le libellé ailleurs.
 */
import type { ReactNode } from 'react'
import { Link } from '@tanstack/react-router'
import { formatMessage } from '@/lib/i18n/format'
import { commonManifest } from '@/lib/i18n/generated/common'
import type { Locale } from '@/lib/i18n/locale'

interface HomeLinkProps {
  locale: Locale
  titleSlug: string
  /** Joueur actif ; vide : le lien mène à la racine. */
  playerSlug: string
  className: string
  children: ReactNode
}

export function HomeLink({ locale, titleSlug, playerSlug, className, children }: HomeLinkProps) {
  const ariaLabel = formatMessage(commonManifest, 'common.shell.logo_aria', locale)
  if (playerSlug) {
    return (
      <Link
        to="/{-$lang}/t/$titleSlug/players/$playerSlug/home"
        params={{ titleSlug, playerSlug }}
        className={className}
        aria-label={ariaLabel}
      >
        {children}
      </Link>
    )
  }
  return (
    <Link to="/{-$lang}" className={className} aria-label={ariaLabel}>
      {children}
    </Link>
  )
}
