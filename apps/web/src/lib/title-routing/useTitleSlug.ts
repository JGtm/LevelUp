import { useAppShellStore } from '@/stores/appShellStore'

/**
 * useTitleSlug — slug du titre actif, pour renseigner les `params` des liens
 * title-scoped `/{-$lang}/t/$titleSlug/…` (D-6, D-10).
 *
 * Contexte : TanStack EXIGE `titleSlug` dans `params` de chaque Link/navigate/
 * redirect vers une route title-scoped (le segment est un param requis) ; la forme
 * fonctionnelle `params={(prev) => ({ ...prev })}` NE suffit PAS (hors `from`, tous
 * les params sont typés optionnels, donc `titleSlug` n'est pas garanti). Ce hook est
 * la forme centralisée UNIQUE (règle CLAUDE.md n°6) : les liens joueur/titre lisent
 * ce hook plutôt que de dupliquer `titleSlug: currentTitleSlug`.
 *
 * Source = store (`currentTitleSlug`), TOUJOURS une string valide (défaut inclus),
 * y compris pour les composants de shell rendus HORS sous-arbre title-scoped
 * (ex. cloche de notifications sur `/settings`). Le layout `t/$titleSlug` garantit
 * la convergence store↔segment quand le sous-arbre joueur est rendu → cette valeur
 * égale le segment d'URL sur toutes les pages title-scoped.
 *
 * Point d'évolution unique pour la Phase 2 item 2g (préservation du segment `lang`
 * courant) et pour la centralisation `buildPlayerDestination` (item 2c).
 */
export function useTitleSlug(): string {
  return useAppShellStore((s) => s.currentTitleSlug)
}
