/**
 * ReplayInventoryRow — munitions de l'arme en main, grenades portées, capacité d'armure.
 *
 * LA RANGÉE EST UNE GRILLE À CELLULES FIXES depuis le 2026-08-24 (même règle que la rangée
 * d'armes, demande utilisateur) : une cellule de MUNITIONS, une boîte de GRENADES, une
 * cellule de CAPACITÉ — toujours rendues, toujours aux mêmes largeurs. Une donnée absente
 * laisse sa cellule vide plutôt que de décaler les colonnes des fiches voisines.
 *
 * TOUT CE QUI N'EST PAS LU S'AFFICHE COMME LACUNE, jamais comme une valeur par défaut :
 * - une capacité hors table reçoit un GLYPHE NEUTRE (vignette vide en pointillés) et garde
 *   son NUMÉRO en infobulle ;
 * - un emplacement dont le film n'écrit RIEN est PLEIN (flux différentiel : le plein est la
 *   valeur par défaut, jamais transmise) ;
 * - les ARMES À CHARGE (familles plasma / mêlée / énergie : épée, marteau, pistolet à
 *   plasma, Pulse Carbine, Ravager, Sentinel Beam...) n'ont PAS de chargeur : leurs champs
 *   mag/res sont des ancrages parasites du film (mesure 2026-08-24 : 46 cellules « 1/0 »
 *   sur le témoin, toutes sur ces familles). Elles affichent leur charge en POURCENTAGE
 *   écrit (« 87% », demande utilisateur du 2026-08-24 — pas une jauge) : 100% quand rien
 *   n'a été consommé — jamais un « 1/0 ».
 *
 * SEULE L'ARME EN MAIN garde ses munitions (règle de la fiche compacte, devenue LA fiche) :
 * les munitions d'une arme rangée ne se lisent que pour préparer une permutation, ce qui
 * n'est pas ce qu'on regarde dans un rejeu.
 *
 * L'ensemble pâlit avec l'âge de la lecture, comme les armes portées et pour la même raison.
 */
import type { ReactNode } from 'react'

import { WeaponIcon } from '@/components/ui/WeaponIcon'
import { tokenCssVar } from '@/lib/accessibility/semantic-tokens'
import { useTitleSlug } from '@/lib/title-routing/useTitleSlug'
import { useSettingsDraftStore } from '@/stores/settingsDraftStore'

import { catalogText, type CatalogLabel } from './catalogLabel'
import type { EquippedReading } from './equippedLogic'
import { grenadeIconOf, type GrenadeIconRef } from './grenadeIcon'
import { REPLAY_TEXT, type ReplayLocale } from './i18n'
import { formatSeconds, frameToMs, freshness, READING_FADE } from './replayLogic'
import type { ReplayDocumentReady } from './replayNormalize'
import { familyOf } from './shotEffects'
import { abilityAt, grenadesCarried, inventoryAt, selectedGrenade } from './rosterLogic'

/** Boîte d'une vignette de HUD (grenade, capacité) : la hauteur de la ligne. */
const HUD_ICON_PX = 16
/** Largeurs FIXES des cellules — la grille des fiches en dépend (cf. en-tête). */
const AMMO_CELL_W = 34
const GRENADES_BOX_W = 62

/** Les familles d'arme À CHARGE : pas de chargeur, une jauge (cf. en-tête). */
const CHARGE_FX = new Set(['plasma', 'melee', 'light'])

export function ReplayInventoryRow({
  doc,
  slot,
  equipped,
  frame,
  readingFull,
  locale,
}: {
  doc: ReplayDocumentReady
  slot: number
  equipped: EquippedReading | null
  frame: number
  readingFull: number
  locale: ReplayLocale
}) {
  const t = REPLAY_TEXT[locale]
  // La vignette de grenade est une IMAGE VERSIONNÉE, choisie à l'encre du thème (planche du
  // 16/08) : le titre dit OÙ chercher, le thème dit LAQUELLE des deux encres.
  const titleSlug = useTitleSlug()
  const theme = useSettingsDraftStore((s) => s.localUiPrefs.theme)
  const read = inventoryAt(doc, slot, frame)
  // LA CAPACITÉ A SA PROPRE LECTURE, et donc son propre âge : elle arrive surtout par le
  // canal i48 des paquets delta, qui ne tombe pas sur les images-clés de l'inventaire.
  const abilityRead = abilityAt(doc, slot, frame)
  const ability = abilityText(doc, abilityRead?.rank, t, locale)
  const state = read?.state
  const grenades = state ? grenadesCarried(state, doc.grenadeLabels, locale) : []
  const selected = state ? selectedGrenade(state) : null
  const ammo = state?.am ?? []

  // Âge NÉGATIF = lecture d'une image-clé À VENIR (début de vie, cf. inventoryAt) :
  // l'infobulle le dit — l'estompage porte déjà sur la valeur absolue.
  const ageMs = read ? formatSeconds(frameToMs(Math.abs(read.age), doc)) : ''
  const ageTitle = !read
    ? undefined
    : read.age < 0
      ? `${t.inventoryAhead} ${ageMs}`
      : `${t.inventoryAge} ${ageMs}`

  // La cellule de munitions décrit l'ARME EN MAIN : l'identifiant de famille dit si c'est
  // une arme à charge (jauge) ou à chargeur (compte).
  const drawnId = equipped && equipped.drawn !== null ? equipped.weapons[0]?.id : undefined

  return (
    <div
      className="flex items-center gap-1 font-mono text-[9.5px] text-muted-foreground"
      style={{ opacity: read ? freshness(read.age, readingFull, READING_FADE) : 1 }}
      title={ageTitle}
    >
      <span className="inline-flex shrink-0 items-center" style={{ width: AMMO_CELL_W }}>
        {equipped && ammo.length > 0 && (
          equipped.drawn !== null ? (
            <AmmoCell
              ammo={ammo[equipped.drawn] ?? {}}
              charge={CHARGE_FX.has(familyOf(drawnId ? doc.weaponLabels?.[drawnId]?.fx : undefined))}
              fullLabel={t.ammoFullLabel}
              drawnHint={t.ammoDrawnHint}
              gaugeLabel={t.gaugeLabel}
            />
          ) : !equipped.holstered ? (
            // Sélecteur non lu : la cellule dit la lacune — armes rangées (D=2), elle,
            // n'affiche RIEN : aucune arme en main, aucune munition à décrire.
            <span className="border-b border-dashed border-border opacity-80">
              {t.drawnUnknown}
            </span>
          ) : null
        )}
      </span>
      <span
        className="inline-flex shrink-0 items-center gap-1 overflow-hidden"
        style={{ width: GRENADES_BOX_W }}
        title={grenades.map((g) => `${g.name} ×${g.count}`).join(' · ') || undefined}
      >
        {grenades.map((g) => (
          <GrenadeChip
            key={g.rank}
            carried={g}
            icon={grenadeIconOf(doc.grenadeLabels?.[g.rank], titleSlug, theme)}
            selected={
              typeof selected === 'object' && selected !== null && g.rank === selected.rank
                ? selected
                : null
            }
            t={t}
          />
        ))}
        {selected === 'indeterminate' && (
          <span className="border-b border-dashed border-border opacity-80">
            {t.grenadeSelUnknown}
          </span>
        )}
      </span>
      <span className="inline-flex shrink-0 items-center" style={{ width: HUD_ICON_PX }}>
        {ability && abilityRead && (
          <span
            className="inline-flex items-center"
            // ÂGE PROPRE À LA CAPACITÉ : sa lecture ne tombe pas sur les images-clés de
            // l'inventaire, l'estompage de la ligne ne la décrit donc pas.
            style={{ opacity: freshness(abilityRead.age, readingFull, READING_FADE) }}
            title={abilityAgeTitle(t, abilityRead.age, doc, ability.text)}
          >
            {ability.img ? (
              <WeaponIcon
                imageUrl={ability.img}
                tinted={ability.tinted}
                label={ability.text}
                width={HUD_ICON_PX}
                height={HUD_ICON_PX}
              />
            ) : ability.known ? (
              ability.text
            ) : (
              /* RANG NON RÉSOLU : un GLYPHE, pas un mot ni un caractère (planche du 16/08).
                 Le rang lu reste la seule chose vraie : il vit dans l'infobulle. */
              <AbilityUnknownMark label={ability.text} />
            )}
          </span>
        )}
      </span>
    </div>
  )
}

/**
 * GrenadeChip — UN type de grenade porté : sa vignette, son compteur, et la marque du type
 * ÉQUIPÉ quand c'est lui qui partira au prochain lancer.
 *
 * LA VIGNETTE PORTE L'IDENTITÉ, LE NOM PASSE EN INFOBULLE (planche du 16/08 : « pas de
 * texte sauf pour le compteur »). Sans AUCUNE image, le libellé revient : mieux vaut un mot
 * qu'un trou.
 */
function GrenadeChip({
  carried,
  icon,
  selected,
  t,
}: {
  carried: { rank: number; name: string; count: number }
  icon: GrenadeIconRef | null
  /** Non nul = c'est CE type qui est équipé ; `read` dit si la lecture ou la déduction l'établit. */
  selected: { rank: number; read: boolean } | null
  t: (typeof REPLAY_TEXT)[ReplayLocale]
}) {
  return (
    <span
      className={`inline-flex shrink-0 items-center gap-0.5 ${selected ? 'rounded-sm px-0.5 font-semibold' : ''}`}
      style={
        selected
          ? {
              color: tokenCssVar('warning'),
              boxShadow: `0 0 0 1px ${tokenCssVar('warning')}`,
              background: `color-mix(in srgb, ${tokenCssVar('warning')} 13%, transparent)`,
            }
          : undefined
      }
      title={
        selected
          ? `${carried.name} — ${selected.read ? t.grenadeSelectedRead : t.grenadeSelected}`
          : carried.name
      }
    >
      {icon ? (
        <WeaponIcon
          imageUrl={icon.url}
          tinted={icon.tinted}
          label={carried.name}
          width={HUD_ICON_PX}
          height={HUD_ICON_PX}
        />
      ) : (
        carried.name
      )}
      <span className="tabular-nums">×{carried.count}</span>
    </span>
  )
}

/**
 * StateMark — le socle des pictogrammes d'état MESURÉ rendus muets (décision produit 4 du
 * plan parité) : « munitions pleines » (emplacement jamais écrit — flux différentiel, le
 * plein est la valeur par défaut). Un dessin discret, UNE infobulle simple. Encre en
 * currentColor : la couleur vient du texte environnant, aucun littéral.
 */
function StateMark({ label, children }: { label: string; children: ReactNode }) {
  return (
    <span role="img" aria-label={label} title={label} className="inline-flex items-center opacity-70">
      {children}
    </span>
  )
}

/** AmmoFullMark — munitions pleines : un chargeur rempli. */
function AmmoFullMark({ label }: { label: string }) {
  return (
    <StateMark label={label}>
      <svg width="10" height="10" viewBox="0 0 12 12" aria-hidden="true">
        <rect x="4" y="1.5" width="4" height="9" rx="1" fill="currentColor" />
      </svg>
    </StateMark>
  )
}

/**
 * AbilityUnknownMark — capacité LUE mais NON IDENTIFIÉE : l'emplacement de la vignette,
 * vide, en pointillés. Le dessin dit exactement ce qu'on sait — « il y avait quelque chose
 * ici, on ne sait pas quoi ». Même gabarit que les vignettes de la ligne (16 px).
 */
function AbilityUnknownMark({ label }: { label: string }) {
  return (
    <StateMark label={label}>
      <svg
        width={HUD_ICON_PX}
        height={HUD_ICON_PX}
        viewBox="0 0 16 16"
        fill="none"
        stroke="currentColor"
        strokeWidth="1.2"
        strokeDasharray="2.4 1.8"
        aria-hidden="true"
      >
        <rect x="2.4" y="2.4" width="11.2" height="11.2" rx="2.4" />
      </svg>
    </StateMark>
  )
}

/**
 * abilityText nomme la capacité — et porte sa vignette de HUD quand le document en sert
 * une. Un index hors table garde son NUMÉRO dans le texte servi. Renvoie null quand rien
 * n'a été lu — l'absence de capacité et une capacité non identifiée sont deux états
 * différents.
 */
function abilityText(
  doc: ReplayDocumentReady,
  rank: number | undefined,
  t: (typeof REPLAY_TEXT)[ReplayLocale],
  locale: ReplayLocale,
): { text: string; known: boolean; img?: string; tinted?: boolean } | null {
  if (rank === undefined) return null
  const lbl: CatalogLabel | undefined = doc.abilityLabels?.[String(rank)]
  const name = catalogText(lbl, locale)
  if (name) return { text: name, known: true, img: lbl?.img, tinted: lbl?.tinted }
  return { text: t.abilityUnidentified(rank), known: false }
}

/**
 * abilityAgeTitle — l'infobulle de la capacité : son nom quand il est connu, et TOUJOURS
 * l'âge de sa lecture. Un âge négatif est une lecture À VENIR (début de vie), dit comme tel.
 */
function abilityAgeTitle(
  t: (typeof REPLAY_TEXT)[ReplayLocale],
  age: number,
  doc: ReplayDocumentReady,
  name: string | null,
): string {
  const ms = formatSeconds(frameToMs(Math.abs(age), doc))
  const when = age < 0 ? `${t.abilityAhead} ${ms}` : `${t.abilityAge} ${ms}`
  return name ? `${t.abilityLabel} — ${name} · ${when}` : when
}

/**
 * AmmoCell — les munitions de l'ARME EN MAIN.
 *
 * DEUX NATURES, DEUX RENDUS. Une arme À CHARGE (`charge`) montre son POURCENTAGE écrit
 * (« 87% », demande utilisateur du 2026-08-24) : la donnée du film compte le CONSOMMÉ, le
 * nombre dit le RESTANT (complément) ; rien de consommé = « 100% » — et ses champs
 * mag/res, quand le film en émet, sont des ancrages parasites qu'on n'affiche JAMAIS
 * (cf. l'en-tête du fichier). Une arme à CHARGEUR montre son compte `mag/res` ; un
 * emplacement jamais écrit est PLEIN (pictogramme).
 *
 * PAS DE COMPTEUR « n / N » sur une charge : le quantum est propre à l'arme et le film ne
 * dit pas combien de charges font un plein.
 */
function AmmoCell({
  ammo,
  charge,
  fullLabel,
  drawnHint,
  gaugeLabel,
}: {
  ammo: { mag?: number; res?: number; gauge?: number }
  charge: boolean
  fullLabel: string
  drawnHint: string
  gaugeLabel: string
}) {
  const chargePct = (consumed: number) => (
    <span title={gaugeLabel}>{Math.round(Math.max(0, 1 - consumed) * 100)}%</span>
  )
  return (
    <span className="inline-flex items-center gap-1 tabular-nums text-foreground" title={drawnHint}>
      {charge ? (
        chargePct(ammo.gauge ?? 0)
      ) : ammo.mag !== undefined ? (
        <span>
          {ammo.mag}
          {ammo.res !== undefined && <span className="opacity-60">/{ammo.res}</span>}
        </span>
      ) : ammo.gauge !== undefined ? (
        chargePct(ammo.gauge)
      ) : (
        <AmmoFullMark label={fullLabel} />
      )}
    </span>
  )
}
