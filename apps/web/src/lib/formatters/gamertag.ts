/**
 * formatGamertag — résolveur défensif côté frontend pour les IDs de bots Halo.
 *
 * Le backend résout normalement les bots via la vue SQL `v_gamertag_lookup`
 * (apps/go-api/internal/migration/steps_shared.go), qui transforme `bid(N.0)`
 * en `343 Bot N`. Ce helper sert de DERNIÈRE LIGNE DE DÉFENSE quand cette
 * résolution n'a pas eu lieu côté backend (migration silencieuse, alias
 * orphelin, query qui n'a pas joint la vue, etc.).
 *
 * Il n'est PAS censé remplacer la résolution backend — il garantit juste que
 * l'utilisateur ne verra jamais "bid(5.0)" affiché tel quel dans l'UI.
 */
const BOT_ID_RE = /^bid\((\d+)(?:\.\d+)?\)?$/i

export function formatGamertag(gamertag: string | null | undefined): string {
  if (gamertag == null) return '—'
  const trimmed = gamertag.trim()
  if (trimmed === '') return '—'

  const m = trimmed.match(BOT_ID_RE)
  if (m) {
    return `343 Bot ${m[1]}`
  }
  return trimmed
}

export function isBotGamertag(gamertag: string | null | undefined): boolean {
  if (gamertag == null) return false
  return BOT_ID_RE.test(gamertag.trim())
}
