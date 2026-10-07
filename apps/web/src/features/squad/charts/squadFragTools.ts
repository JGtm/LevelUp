/**
 * squadFragTools — « Outils de destruction » de l'Escouade : les lignes `weapon_tools` du
 * serveur (décision D8 du plan PLAN_EMPRISE_ET_CARTES_DEPLACEES_2026-09-26), nommées dans la
 * langue de l'interface et mises à la forme du graphe à barres groupées.
 *
 * Le serveur a déjà tout décidé : une ligne par clé d'arme (grenades par type quand le film
 * les type, sinon une ligne « Grenade » au total de la feuille), la
 * mêlée de la feuille de match, les objets explosifs et la chute d'après la catégorie de
 * source du film, le reliquat « Non attribué » en dernier. Ce module NE regroupe RIEN et ne
 * plafonne RIEN (plus de « Autres armes ») : il nomme.
 *
 *   - une arme porte son nom de registre, FR d'abord (`label`) ou EN d'abord (`label_en`)
 *     selon la locale ;
 *   - les autres natures sont nommées ici, par clé (`kind`) — le serveur n'écrit aucun
 *     libellé (ratchet `no_french_label_literal_test.go`).
 */
import type { SquadWeaponToolLine, SquadWeaponTools } from '@/lib/api/types'
import { formatMessage } from '@/lib/i18n/format'
import { fragsManifest } from '@/lib/i18n/generated/frags'
import type { Locale } from '@/lib/i18n/locale'
import type { SquadBarRow, SquadBarRows } from './squadWeaponKillsChart'

/** Nature du reliquat (contrat Go `domain.SquadToolKindUnattributed`). */
const UNATTRIBUTED_KIND = 'unattributed'

/** Libellés des natures sans nom de registre (i18n de l'appelant). */
export interface SquadToolKindLabels {
  melee: string
  /** Grenades de la feuille, en une ligne, quand le film ne les type pas (titre sans film). */
  grenade: string
  assassination: string
  ground_pound: string
  shoulder_bash: string
  explosive_object: string
  environment: string
  unattributed: string
}

/**
 * Les libellés des natures, une seule source pour l'Escouade, Sessions et la Vue match (garde-rail
 * `squadFragTools.labels.guard.test.ts`) : le manifeste `frags` pour la mêlée, les grenades, les
 * mécaniques et le reliquat ; les deux catégories de source du film viennent des textes de l'appelant.
 */
export function toolKindLabels(locale: Locale, source: { explosiveObject: string; environment: string }): SquadToolKindLabels {
  const frag = (k: string) => formatMessage(fragsManifest, k as never, locale)
  return {
    melee: frag('frags.class.melee'),
    grenade: frag('frags.class.grenade'),
    assassination: frag('frags.role.assassination'),
    ground_pound: frag('frags.role.ground_pound'),
    shoulder_bash: frag('frags.role.shoulder_bash'),
    explosive_object: source.explosiveObject,
    environment: source.environment,
    unattributed: frag('frags.class.unattributed'),
  }
}

/** Nom affiché d'une ligne. Nature inconnue du web → sa clé brute (jamais une ligne vide). */
export function toolLineLabel(line: SquadWeaponToolLine, locale: Locale, labels: SquadToolKindLabels): string {
  if (line.kind === 'weapon') {
    const fr = line.label ?? ''
    const en = line.label_en ?? ''
    return (locale === 'en' ? en || fr : fr || en) || (line.weapon_key ?? '')
  }
  return labels[line.kind as keyof SquadToolKindLabels] ?? line.kind
}

/**
 * Lignes du graphe. Le serveur les rend du plus gros total au plus petit (« Non attribué »
 * en dernier) ; le graphe pose la PREMIÈRE catégorie EN BAS, d'où l'ordre renversé : le
 * BR75 de la soirée se lit en haut, le reliquat tout en bas.
 */
export function buildSquadToolRows(
  tools: SquadWeaponTools | null | undefined,
  /** `top` (vue compacte de Sessions) : les `top` premiers outils du serveur, « Non attribué » exclu. */
  opts: { locale: Locale; labels: SquadToolKindLabels; top?: number },
): SquadBarRows | null {
  const players = tools?.players ?? []
  const all = tools?.lines ?? []
  const lines = opts.top == null ? all : all.filter((l) => l.kind !== UNATTRIBUTED_KIND).slice(0, opts.top)
  if (players.length === 0 || lines.length === 0) return null
  const rows: SquadBarRow[] = lines.map((l) => ({
    key: l.kind === 'weapon' ? `weapon:${l.weapon_key ?? l.label ?? ''}` : l.kind,
    label: toolLineLabel(l, opts.locale, opts.labels),
    cls: l.class,
    killsByPlayer: l.kills_by_player,
    total: l.total_squad,
  }))
  return { players, rows: rows.reverse() }
}
