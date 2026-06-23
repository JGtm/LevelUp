/**
 * Source unique des ids d'onglets Settings + rétro-compat des deep-links.
 *
 * Centralisé ici (plutôt que dans routes/settings.tsx) pour être partagé par la
 * route ET SettingsPage sans créer de cycle d'import.
 */
const SETTINGS_TABS = ['appearance', 'analyse', 'notifications', 'data', 'account'] as const
export type SettingsTab = (typeof SETTINGS_TABS)[number]

/**
 * Anciens ids retombent sur un onglet valide pour ne pas casser les
 * URLs/bookmarks existants : general/accessibility → appearance ; sync/users/lab
 * (migrés vers Admin) et backup (déplacé vers Admin · Système) → appearance.
 */
const TAB_ALIASES: Record<string, SettingsTab> = {
  general: 'appearance',
  accessibility: 'appearance',
  sync: 'appearance',
  users: 'appearance',
  lab: 'appearance',
  backup: 'appearance',
}

export function resolveSettingsTab(raw: string | null | undefined): SettingsTab {
  if (raw && (SETTINGS_TABS as readonly string[]).includes(raw)) return raw as SettingsTab
  return (raw && TAB_ALIASES[raw]) || 'appearance'
}
