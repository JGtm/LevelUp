/**
 * ReplayAssistMark — LA MARQUE D'ASSISTANCE du fil des morts, et le stem de sa vignette.
 *
 * EXTRAIT DE `ReplayKillFeed.tsx` LE 2026-08-18 (lot R2-V), qui portait deja une dette de
 * taille (518 lignes) et gagnait la mise a plat de ses lignes. Ce bloc n'a aucun lien avec la
 * ligne qui le porte — il ignore tueur, victime, medailles et couleurs d'equipe — c'est donc
 * lui qui sort, comme `MedalBadges` avant lui (2026-08-16).
 */
import { WeaponIcon } from '@/components/ui/WeaponIcon'
import { tokenCssVar } from '@/lib/accessibility/semantic-tokens'
import { staticAssetURL } from '@/lib/staticAssets'
import { useTitleSlug } from '@/lib/title-routing/useTitleSlug'

/** Cote du pictogramme quasi carre de l'atlas kill feed (40x40 mesure) : cf. ReplayKillFeed. */
const PICTOGRAM_PX = 11

/**
 * Stem (sans dossier ni extension) de la vignette d'assistance du KILL FEED DU JEU —
 * atlas killfeed, index 62. Exporté pour le garde-rail `assistMarkIcon.guard.test.ts`
 * (même patron que `GRENADE_ICON_STEMS`) : le fichier référencé doit exister sur disque
 * ET rester ce que `jeu/index.json` déclare à cet index.
 */
export const ASSIST_ICON_STEM = 'killfeed-62'

/**
 * AssistMark — LA MARQUE D'ASSISTANCE : la vignette D'ASSISTANCE DU JEU (`jeu/killfeed-62`),
 * en masque teint — la MÊME technique que l'icône d'arme du fil (`WeaponIcon`, masque +
 * `currentColor` porté par le parent). L'ENCRE EST LE TOKEN `bonus` (option 2a du handoff
 * 2026-08-27), plus la couleur d'équipe de l'assistant : son nom la dit déjà sur la même
 * ligne, et une marque à encre CONSTANTE se reconnaît d'une ligne à l'autre.
 *
 * PROVENANCE, CORRECTION DU LOT R1. R1 avait conclu qu'aucune icône d'assistance n'existait
 * dans l'atlas kill feed et dessinait un glyphe SVG neutre à la place : cette conclusion
 * lisait l'ABSENCE de `nom_jeu` sur l'entrée 62 de `jeu/index.json` comme une absence
 * d'icône, alors que l'entrée EXISTE (`source_tag 0302cad3`, celui de l'atlas killfeed) —
 * elle n'a simplement jamais reçu de nom automatique : ni le tag `weap` (l'assistance n'est
 * l'icône d'aucune arme), ni le nommage par hachage du kill feed (`bitd`, cf.
 * cmd/weapon-icons-build/killfeed.go — "assist" y est dans le vocabulaire curaté, mais son
 * hachage n'a jamais été rejoué contre le binaire du jeu depuis). C'est L'UTILISATEUR qui a
 * désigné `killfeed-62.png` à l'œil le 2026-08-17, cohérent avec l'identification
 * indépendante de `.ai/V7.5/icones/NOMMAGE_GATE_2026-08-09.tsv` (killfeed 62 → « Assist »).
 *
 * TAILLE : le pictogramme est quasi carré (40×40 mesuré) — le gabarit bandeau des icônes
 * d'arme (`ICON_W`×`ICON_H`) l'écraserait à la hauteur d'une lettre, d'où `PICTOGRAM_PX`,
 * le même gabarit que le pictogramme de TYPE DE MORT (même atlas, même famille de forme).
 */
export function AssistMark({ label }: { label: string }) {
  const titleSlug = useTitleSlug()
  return (
    <span title={label} className="inline-flex shrink-0 items-center">
      <WeaponIcon
        imageUrl={staticAssetURL('weapon', `jeu/${ASSIST_ICON_STEM}`, '.png', titleSlug)}
        tinted
        label={label}
        width={PICTOGRAM_PX}
        height={PICTOGRAM_PX}
        style={{ color: tokenCssVar('bonus') }}
      />
    </span>
  )
}
