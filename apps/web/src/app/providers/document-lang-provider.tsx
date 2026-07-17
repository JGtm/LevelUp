import { type ReactNode, useLayoutEffect } from 'react'
import { useAppShellStore } from '@/stores/appShellStore'
import { intlLocale } from '@/lib/formatters/intlLocale'

/**
 * DocumentLangProvider — reflète la locale applicative (`'fr' | 'en'`) sur
 * l'attribut `lang` de `<html>` en BCP-47 (`fr-FR` / `en-US`), via la source
 * unique `intlLocale` (CLAUDE.md n°6 — aucun ternaire `en ? en-US : fr-FR`
 * dupliqué).
 *
 * Pourquoi : le shell HTML fige un `lang` statique alors que l'app démarre en FR
 * et bascule à l'exécution. Un `<html lang>` incohérent casse l'a11y (lecteurs
 * d'écran) et le format des contrôles natifs qui suivent la locale du document —
 * notamment `<input type="date">` (Firefox : `jj/mm/aaaa` vs `mm/dd/yyyy`).
 *
 * Limite connue (vérifiée empiriquement le 2026-07-17 sur ce poste) : Chromium
 * IGNORE `lang` pour le format d'affichage d'un `<input type="date">` — il suit
 * la locale du navigateur / de l'OS. Ce binding corrige donc l'a11y et Firefox,
 * mais PAS le format date affiché sous Chrome.
 */
export function DocumentLangProvider({ children }: { children: ReactNode }) {
  const locale = useAppShellStore((s) => s.locale)

  useLayoutEffect(() => {
    document.documentElement.lang = intlLocale(locale)
  }, [locale])

  return <>{children}</>
}
