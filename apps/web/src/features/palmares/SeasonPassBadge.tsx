/**
 * SeasonPassBadge — pastille pleine (fond saturé + texte blanc) des surfaces du
 * pass saisonnier. Reutilise NarrativeBadge(solid) — le meme rendu que les pills
 * de joueur du hub Communaute > Relations. Le choix des tokens (set sombre
 * palette-invariant `narrative-encounter-*`) est justifie dans battlePassBadgeStyle.
 */
import { NarrativeBadge } from '@/components/feedback/NarrativeBadge'
import { tokenVar } from '@/lib/accessibility'

import { seasonPassBadgeToken, type SeasonPassBadgeRole } from './battlePassBadgeStyle'

export function SeasonPassBadge({ role, label }: { role: SeasonPassBadgeRole; label: string }) {
  return <NarrativeBadge solid size="md" label={label} colorVar={tokenVar(seasonPassBadgeToken(role))} />
}
