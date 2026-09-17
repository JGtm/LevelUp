/**
 * ExplorerBriefingWeapons — bloc « Arme favorite » du bandeau de briefing.
 *
 * L'arme favorite est celle qui a produit le PLUS DE FRAGS sur la sélection — ni la plus
 * tenue, ni la plus tirée. Le backend en sert une ou deux, déjà triées.
 *
 * UNE SEULE FORME D'HABILLAGE : la `BriefingSectionCard` de ses voisines, qu'il soit
 * empilé sous « Par contexte » ou seul dans sa cellule (décision utilisateur au gate
 * visuel du 2026-09-17 — le rendu nu « ne collait pas du tout » avec le reste de la
 * rangée ; l'harmonie l'emporte sur la promesse de hauteur constante). `slots` ne choisit
 * donc plus que le NOMBRE d'armes montrées, une ou deux, toujours avec leur barre.
 *
 * Le bloc est une LISTE `flex flex-col` et jamais une grille à colonnes nommées : le test
 * DP-3 vise la DERNIÈRE grille portant cette classe et tomberait sur celle-ci. Le nom du
 * fichier porte « Briefing » pour rester sous les garde-rails qui filtrent sur ce motif.
 * Couleurs : `fragClassColor` seul (barre teintée par la classe de l'arme).
 */
import { fragClassColor } from '@/lib/accessibility/scales'
import { intlLocale } from '@/lib/formatters'
import type { ExplorerBriefingWeapons, SynthesisWeaponKillEntry } from '@/lib/api/types'
import type { Locale } from '@/lib/i18n/locale'
import type { ExplorerManifestKey } from '@/lib/i18n/generated/explorer'
import { BriefingSectionCard } from './BriefingSectionCard'

type T = (key: ExplorerManifestKey, values?: Record<string, string | number>) => string

/** Ligne d'arme : libellé tronqué, frags alignés à droite, barre fine teintée par classe. */
function WeaponRow({
  weapon,
  maxKills,
  locale,
}: {
  weapon: SynthesisWeaponKillEntry
  maxKills: number
  locale: Locale
}) {
  const pct = Math.round((weapon.kills / maxKills) * 100)
  return (
    <li className="flex flex-col gap-1">
      <div className="flex items-baseline justify-between gap-2">
        <span className="min-w-0 truncate text-xs font-medium text-foreground">{weapon.label}</span>
        <span className="flex-shrink-0 text-xs font-semibold tabular-nums text-foreground">
          {weapon.kills.toLocaleString(intlLocale(locale))}
        </span>
      </div>
      <div className="h-1 w-full overflow-hidden rounded-full bg-muted-foreground/15">
        <div
          className="h-full rounded-full"
          style={{ width: `${pct}%`, backgroundColor: fragClassColor(weapon.class) }}
        />
      </div>
    </li>
  )
}

/**
 * Bloc « Arme favorite ». `slots` vient de `favoriteWeaponSlots` : il choisit combien
 * d'armes tiennent dans la rangée, jamais la présence du bloc — l'omission du module est
 * décidée par le backend (aucun frag rattaché à une arme nommée).
 *
 * Le payload porte aussi `measured_kills` / `scope_kills` : ces champs restent vrais et
 * servis par l'API, mais ne sont PLUS affichés depuis le gate visuel du 2026-09-17 (la
 * note de couverture a quitté l'UI). Ils attendent une éventuelle infobulle.
 */
export function FavoriteWeaponBlock({
  weapons,
  slots,
  t,
  locale,
}: {
  weapons: ExplorerBriefingWeapons
  slots: 1 | 2
  t: T
  locale: Locale
}) {
  const entries = (weapons.entries ?? []).slice(0, slots)
  if (entries.length === 0) return null
  const maxKills = Math.max(1, ...entries.map((w) => w.kills))
  return (
    <BriefingSectionCard className="h-full" title={t('explorer.briefing.weapons_title')}>
      <ul className="flex flex-col gap-2">
        {entries.map((w, i) => (
          <WeaponRow key={`${w.label}-${i}`} weapon={w} maxKills={maxKills} locale={locale} />
        ))}
      </ul>
    </BriefingSectionCard>
  )
}
