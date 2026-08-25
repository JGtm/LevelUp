/**
 * ReplaySettingsToggle — LA LIGNE DE BASCULE du tiroir de reglages, et la marque « (i) » de
 * reserve de mesure qui l'accompagne parfois.
 *
 * EXTRAIT DE `ReplaySettingsDrawer.tsx` LE 2026-08-18 (lot R2-V) : le tiroir a franchi le seuil
 * de taille du depot (CLAUDE.md n°5) en recevant la bascule de la trainee et la portee de la
 * carte de chaleur. La decoupe tombe sur une frontiere nette — ce fichier ne connait AUCUN
 * reglage du rejeu, il ne sait que dessiner un interrupteur ; le tiroir, lui, ne sait plus
 * comment un interrupteur se dessine.
 *
 * Il est employe une dizaine de fois par le tiroir (calques, effets, chaleur, portee, son) :
 * un seul rendu plutot que dix copies presque identiques (CLAUDE.md n°6).
 */
import { Button } from '@/components/ui/button'

/**
 * InfoMark — la marque « (i) » d'une RÉSERVE DE MESURE, en SVG et jamais en caractère : le
 * « i cerclé » typographique dépend de la police installée et se rend en carré vide sur les
 * machines qui ne l'ont pas. Elle ne prend pas le focus et n'ouvre rien — tout ce qu'elle a
 * à dire tient dans son infobulle, et le lecteur d'écran lit la même phrase.
 *
 * EXPORTÉE depuis le 2026-08-24 : la marque se pose sur le TITRE de la section du tiroir
 * (demande utilisateur), plus à côté d'une bascule — le tiroir la compose lui-même.
 */
export function InfoMark({ text }: { text: string }) {
  return (
    <span
      role="img"
      aria-label={text}
      title={text}
      className="inline-flex shrink-0 items-center text-muted-foreground"
    >
      <svg
        width="12"
        height="12"
        viewBox="0 0 12 12"
        fill="none"
        stroke="currentColor"
        strokeWidth="1.1"
        strokeLinecap="round"
        aria-hidden="true"
      >
        <circle cx="6" cy="6" r="4.6" />
        <path d="M6 5.4v3" />
        <path d="M6 3.6v.2" />
      </svg>
    </span>
  )
}

/**
 * Une ligne de bascule : même gabarit pour les calques, les effets et les catégories de son —
 * une dizaine d'usages dans le tiroir, un seul rendu plutôt que dix copies presque identiques
 * (CLAUDE.md règle 6, « à la 3e copie, centraliser »).
 */
export function SettingsToggle({
  label, pressed, onToggle, hint,
}: {
  label: string
  pressed: boolean
  onToggle: () => void
  hint?: string
}) {
  return (
    <Button
      type="button"
      variant={pressed ? 'default' : 'ghost'}
      size="sm"
      onClick={onToggle}
      className="h-7 justify-start px-2 text-xs"
      title={hint}
      aria-pressed={pressed}
    >
      {label}
    </Button>
  )
}
