/**
 * PlanMaskDef — le `<mask>` SVG d'une couche de zones rognee au plan (cf. `planMask.ts`). A poser dans les
 * `<defs>` de la couche ; la couche porte ensuite `mask={planMaskUrl(id)}`. Le masque est de type alpha :
 * l'image plan decoupe, sa teinte ne compte pas.
 */
import { type CadreDuPlan } from './planMask'

export function PlanMaskDef({ id, href, cadre }: { id: string; href: string; cadre: CadreDuPlan }) {
  return (
    <mask
      id={id}
      maskUnits="userSpaceOnUse"
      x={cadre.x}
      y={cadre.y}
      width={cadre.width}
      height={cadre.height}
      style={{ maskType: 'alpha' }}
    >
      <image
        href={href}
        x={cadre.x}
        y={cadre.y}
        width={cadre.width}
        height={cadre.height}
        preserveAspectRatio="none"
      />
    </mask>
  )
}
