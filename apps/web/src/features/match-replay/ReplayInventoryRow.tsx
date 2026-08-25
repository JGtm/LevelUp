/**
 * ReplayInventoryRow — grenades portées, capacité d'armure, munitions d'une fiche joueur.
 * Extrait de ReplayTeams.tsx (seuil de taille du dépôt), comme ReplayWeaponsRow avant lui :
 * la ligne d'inventaire et ses cellules de munitions forment une unité.
 *
 * TOUT CE QUI N'EST PAS LU S'AFFICHE COMME LACUNE, jamais comme une valeur par défaut :
 * - une capacité hors table reçoit un GLYPHE NEUTRE (vignette vide en pointillés) et garde
 *   son NUMÉRO en infobulle — la table est partielle (4 index observés pour 11 capacités),
 *   et la combler se lirait comme une certitude ;
 * - un compteur d'utilisations n'est jamais affiché : il n'est pas localisé dans le film
 *   (36 006 positions testées, aucune ne reproduit le relevé) ;
 * - un emplacement dont le film n'écrit RIEN est PLEIN (flux différentiel : le plein est la
 *   valeur par défaut, jamais transmise) — rendu par un pictogramme discret, comme l'état
 *   « armes rangées » (décision produit 4 du plan parité : ces deux états MESURÉS se
 *   montrent sans jargon de flux, un dessin + un tooltip simple chacun).
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
import {
  grenadesCarried,
  type InventoryEmptyState,
  inventoryAt,
  inventoryEmptyHint,
  selectedGrenade,
} from './inventoryReading'
import { formatSeconds, frameToMs, freshness, READING_FADE } from './replayLogic'
import type { ReplayDocumentReady } from './replayNormalize'
import { abilityAt } from './rosterLogic'

/** Boîte d'une vignette de HUD (grenade, capacité) : la hauteur de la ligne. */
const HUD_ICON_PX = 16

export function ReplayInventoryRow({
  doc,
  slot,
  equipped,
  frame,
  readingFull,
  locale,
  compact = false,
}: {
  doc: ReplayDocumentReady
  slot: number
  equipped: EquippedReading | null
  frame: number
  readingFull: number
  locale: ReplayLocale
  /**
   * Fiche COMPACTE (B2/R2-7) : seule l'arme EN MAIN garde ses munitions. C'est la seule
   * information que la variante compacte perd, et elle est choisie — les munitions d'une
   * arme rangée ne se lisent que pour préparer une permutation, ce qui n'est pas ce qu'on
   * regarde dans un rejeu. Sans sélecteur lu, aucune arme n'est « en main » : la ligne perd
   * alors ses munitions entières plutôt que d'en désigner une au hasard.
   */
  compact?: boolean
}) {
  const t = REPLAY_TEXT[locale]
  // La vignette de grenade est une IMAGE VERSIONNÉE, choisie à l'encre du thème (planche du
  // 16/08) : le titre dit OÙ chercher, le thème dit LAQUELLE des deux encres. Les deux se
  // lisent ici plutôt que de descendre en props depuis la page — ce sont des préférences
  // d'application, pas des données du match.
  const titleSlug = useTitleSlug()
  const theme = useSettingsDraftStore((s) => s.localUiPrefs.theme)
  const read = inventoryAt(doc, slot, frame)
  // LA CAPACITÉ A SA PROPRE LECTURE, et donc son propre âge : elle arrive surtout par le
  // canal i48 des paquets delta, qui ne tombe pas sur les images-clés de l'inventaire. Elle
  // peut donc exister SANS inventaire lu — d'où la sortie anticipée qui les regarde tous
  // les deux, et non le seul inventaire.
  const abilityRead = abilityAt(doc, slot, frame)
  const ability = abilityText(doc, abilityRead?.rank, t, locale)
  if (!read && !ability) return null
  const state = read?.state
  const grenades = state ? grenadesCarried(state, doc.grenadeLabels, locale) : []
  const selected = state ? selectedGrenade(state) : null
  const ammo = state?.am ?? []
  // UNE LECTURE VIDE NE FAIT PLUS DISPARAÎTRE LA LIGNE (schéma 19, lot du 2026-08-25). Elle en
  // faisait disparaître 17,4 % pendant ~20 s : `inventoryAt` retenait la lecture vide comme la
  // plus récente, et ce `return null` emportait la fiche entière alors qu'une lecture pleine la
  // précédait. La ligne survit désormais dès qu'il y a quelque chose à dire — un équipement, ou
  // l'état vide lui-même. Elle ne disparaît que quand il n'y a RIEN, ce qui reste vrai.
  const empty = read?.empty
  if (grenades.length === 0 && !ability && ammo.length === 0 && !empty) return null

  // LA LIGNE DES MUNITIONS SUIT L'ORDRE DES ARMES AU-DESSUS (l'arme dégainée d'abord) : les
  // deux lignes partagent la même lecture d'ordre, sinon chaque cellule se rattacherait à la
  // mauvaise arme. Le numéro d'emplacement lève l'ambiguïté quand l'ordre bascule.
  const ammoOrder = (equipped?.order ?? ammo.map((_, i) => i))
    .filter((i) => i < ammo.length)
    .filter((i) => !compact || i === equipped?.drawn)
  // Âge NÉGATIF = lecture d'une image-clé À VENIR (début de vie, cf. inventoryAt) :
  // l'infobulle le dit — l'estompage porte déjà sur la valeur absolue. La capacité, elle,
  // porte le sien (elle vient d'un autre canal) : la ligne n'estompe que ce qu'elle décrit.
  const ageMs = read ? formatSeconds(frameToMs(Math.abs(read.age), doc)) : ''
  const ageTitle = !read
    ? undefined
    : read.age < 0
      ? `${t.inventoryAhead} ${ageMs}`
      : `${t.inventoryAge} ${ageMs}`

  return (
    <div
      className="flex flex-wrap items-center gap-x-2 gap-y-0.5 font-mono text-[9.5px] text-muted-foreground"
      style={{ opacity: read ? freshness(read.age, readingFull, READING_FADE) : 1 }}
      title={ageTitle}
    >
      {empty && read && (
        <InventoryEmptyMark
          empty={empty}
          label={empty.kind === 'dead' ? t.inventoryDeadLabel : t.inventoryEmptyLabel}
          hint={inventoryEmptyHint(t, read, empty, doc)}
        />
      )}
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
               La table des capacités est partielle PAR NATURE — elle est propre à la palette
               du match — et le rang lu reste la seule chose vraie : il vit dans l'infobulle. */
            <AbilityUnknownMark label={ability.text} />
          )}
        </span>
      )}
      {ammoOrder.map((i) => (
        <AmmoCell
          key={i}
          index={i}
          ammo={ammo[i]}
          drawn={equipped?.drawn === i}
          fullLabel={t.ammoFullLabel}
          drawnHint={t.ammoDrawnHint}
          gaugeLabel={t.gaugeLabel}
        />
      ))}
      {ammo.length > 0 && equipped && equipped.drawn === null && (
        equipped.holstered ? (
          <HolsterMark label={t.holsteredLabel} />
        ) : (
          <span className="border-b border-dashed border-border opacity-80">
            {t.drawnUnknown}
          </span>
        )
      )}
    </div>
  )
}

/**
 * GrenadeChip — UN type de grenade porté : sa vignette, son compteur, et la marque du type
 * ÉQUIPÉ quand c'est lui qui partira au prochain lancer.
 *
 * LA VIGNETTE PORTE L'IDENTITÉ, LE NOM PASSE EN INFOBULLE : à cette taille le dessin est
 * plus lisible que le mot (POC), et la planche du 16/08 l'a tranché — « pas de texte sauf
 * pour le compteur ». Seul le compteur reste écrit. Sans AUCUNE image (type hors catalogue
 * ET artefact sans vignette), le libellé revient : mieux vaut un mot qu'un trou.
 *
 * LA MARQUE DU TYPE ÉQUIPÉ dit aussi D'OÙ ON LE SAIT (lu dans le film, ou déduit du fait
 * qu'un seul type est porté) : deux certitudes différentes, deux infobulles différentes.
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
      className={`inline-flex items-center gap-0.5 ${selected ? 'rounded-sm px-0.5 font-semibold' : ''}`}
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
 * plan parité) : « armes rangées » (sélecteur D=2) et « munitions pleines » (emplacement
 * jamais écrit — flux différentiel, le plein est la valeur par défaut). Un dessin discret,
 * UNE infobulle simple : l'explication de flux vit dans le code, pas à l'écran. Encre en
 * currentColor : la couleur vient du texte environnant, aucun littéral.
 */
function StateMark({ label, children }: { label: string; children: ReactNode }) {
  return (
    <span role="img" aria-label={label} title={label} className="inline-flex items-center opacity-70">
      {children}
    </span>
  )
}

/**
 * InventoryEmptyMark — la lecture qui couvre cette image ne rend RIEN, et la ligne le DIT au
 * lieu de disparaître.
 *
 * LES DEUX ÉTATS NE SE CONFONDENT PAS, et c'est tout l'intérêt du marqueur publié par
 * l'artefact : « Mort » est CORROBORÉ par le fil des éliminations (88,3 % des lectures vides
 * mesurées, contre 1,1 % des lectures pleines soumises à la même fenêtre) ; « Inventaire
 * indisponible » dit qu'on ne sait pas. Écrire « Mort » sur le second serait affirmer à l'écran
 * ce qu'aucune pièce n'établit.
 *
 * L'ÉTAT MORT PORTE UN MOT, PAS SEULEMENT UN DESSIN — contrairement aux pictogrammes muets de
 * `StateMark` (armes rangées, chargeur plein). Ces deux-là décorent une lecture PRÉSENTE ;
 * celui-ci explique pourquoi l'équipement affiché a vingt secondes d'âge, ce qu'un dessin seul
 * ne dit pas. L'encre vient du token sémantique, jamais d'une valeur.
 */
function InventoryEmptyMark({
  empty,
  label,
  hint,
}: {
  empty: InventoryEmptyState
  label: string
  hint: string
}) {
  return (
    <span
      className={
        empty.kind === 'dead'
          ? 'inline-flex items-center rounded-sm px-0.5 font-semibold uppercase'
          : 'inline-flex items-center border-b border-dashed border-border opacity-80'
      }
      style={
        empty.kind === 'dead'
          ? {
              color: tokenCssVar('destructive'),
              boxShadow: `0 0 0 1px ${tokenCssVar('destructive')}`,
            }
          : undefined
      }
      title={hint}
    >
      {label}
    </span>
  )
}

/** HolsterMark — armes rangées : une flèche qui descend dans un étui. */
function HolsterMark({ label }: { label: string }) {
  return (
    <StateMark label={label}>
      <svg
        width="10"
        height="10"
        viewBox="0 0 12 12"
        fill="none"
        stroke="currentColor"
        strokeWidth="1.5"
        strokeLinecap="round"
        strokeLinejoin="round"
        aria-hidden="true"
      >
        <path d="M6 1.5v5" />
        <path d="M3.8 4.5 6 6.7l2.2-2.2" />
        <path d="M2 8v2.5h8V8" />
      </svg>
    </StateMark>
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
 * ici, on ne sait pas quoi » — là où un « ? » aurait été un caractère (refusé le 16/08) et
 * où l'icône d'une capacité voisine aurait été un mensonge. Même gabarit que les vignettes
 * de la ligne (16 px), pour que la rangée ne se déforme pas quand un rang manque à la table.
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
 * une. Un index hors table garde son NUMÉRO dans le texte servi (« capacité non identifiée
 * (rang 3) ») : ni nom ni vignette empruntés à un voisin. Renvoie null quand rien n'a été lu
 * — l'absence de capacité et une capacité non identifiée sont deux états différents.
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
 * AmmoCell — une cellule par EMPLACEMENT du record, numérotée.
 *
 * L'EMPLACEMENT k EST L'ARME k, mais seulement quand la lecture du bloc est UNIQUE : 197
 * appariements sur 198 concordent dans ce cas. Sur une lecture à plusieurs candidats, la
 * correspondance se brouille — c'est ce bruit qui avait fait conclure, à tort, que le
 * rattachement échouait.
 *
 * LE NUMÉRO RESTE AFFICHÉ parce qu'il coûte deux caractères et qu'il lève l'ambiguïté sans
 * rien affirmer. Seule la cellule DÉGAINÉE porte une infobulle (le rattachement au
 * sélecteur) ; la méthode d'appariement emplacement↔arme vit dans le code, pas à l'écran.
 */
function AmmoCell({
  index,
  ammo,
  drawn,
  fullLabel,
  drawnHint,
  gaugeLabel,
}: {
  index: number
  ammo: { mag?: number; res?: number; gauge?: number }
  /** L'emplacement est DÉGAINÉ selon le sélecteur : encre plus franche, index accentué. */
  drawn: boolean
  fullLabel: string
  drawnHint: string
  gaugeLabel: string
}) {
  return (
    <span
      className={`inline-flex items-center gap-1 tabular-nums ${drawn ? 'text-foreground' : ''}`}
      title={drawn ? drawnHint : undefined}
    >
      <i
        className={`not-italic text-[8px] ${drawn ? '' : 'opacity-45'}`}
        style={drawn ? { color: tokenCssVar('warning') } : undefined}
      >
        {index}
      </i>
      {ammo.mag !== undefined ? (
        <span>
          {ammo.mag}
          {ammo.res !== undefined && <span className="opacity-60">/{ammo.res}</span>}
        </span>
      ) : ammo.gauge !== undefined ? (
        // LA BARRE MONTRE CE QUI RESTE, ET LA DONNÉE DIT CE QUI A ÉTÉ CONSOMMÉ : d'où le
        // complément. L'afficher tel quel donnait une barre INVERSÉE — un marteau à 10 %
        // consommé s'affichait presque vide alors qu'il lui reste 90 %.
        //
        // Deux témoins fondent cette lecture (cf. ReplayAmmoSlot) : à la première image-clé du
        // match, quand personne n'a tiré, les armes à charge n'émettent AUCUN champ ; et dans
        // une même vie la valeur ne redescend jamais (6 hausses, 0 baisse).
        //
        // PAS DE COMPTEUR « n / N » : le quantum est propre à l'arme et le film ne dit pas
        // combien de charges font un plein.
        <span
          className="inline-block h-1 w-5 overflow-hidden rounded-sm bg-muted align-middle"
          aria-label={gaugeLabel}
        >
          <span
            className="block h-full rounded-sm"
            style={{
              width: `${Math.max(0, 1 - ammo.gauge) * 100}%`,
              background: tokenCssVar('warning'),
            }}
          />
        </span>
      ) : (
        // Emplacement jamais écrit = PLEIN (décision produit 4) : pictogramme, pas de texte.
        <AmmoFullMark label={fullLabel} />
      )}
    </span>
  )
}
