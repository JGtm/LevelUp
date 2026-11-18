/**
 * FicheTirs — « TIRS ET DÉGÂTS » : la précision (touches sur tirs), les dégâts infligés et subis, puis, par arme, les
 * tirs et les dégâts. Tout est une ESTIMATION et la section le dit (capsule à côté du titre) : les tirs sont ceux que
 * le film écrit, ramenés au bilan de tirs du joueur.
 *
 * La section n'existe que pour un joueur dont la lecture des tirs est suffisante ; sinon la fiche ne l'écrit pas. La
 * précision n'a de ligne que lorsqu'un tir est estimé ET que les touches n'en dépassent pas le nombre : sinon elle
 * n'est pas établie (une touche d'arme à zone n'est pas un tir) et la ligne manque, jamais un pour-cent au-dessus
 * de 100. Une arme sans nom connu n'a pas de ligne. Le rendu ne lit rien.
 */
import { REPLAY_TEXT, type ReplayLocale } from '../../i18n/i18n'
import { formatCount, formatPercent } from '../../model/ficheFormat'
import type { ShotsView } from '../../model/playerInstantView'
import { EstimateCapsule, FicheRow, FicheRows, FicheSection, Note } from './ficheParts'

export function FicheTirs({ shots, locale }: { shots: ShotsView; locale: ReplayLocale }) {
  const t = REPLAY_TEXT[locale].playerInstant
  const count = (n: number) => formatCount(n, locale)
  const { accuracy } = shots
  return (
    <FicheSection title={t.shotsAndDamage} aside={<EstimateCapsule text={t.estimate} hint={t.estimateHint} />}>
      <FicheRows>
        {accuracy && (
          <FicheRow label={t.shotAccuracy}>
            {formatPercent(accuracy.ratio * 100, locale)}
            <Note>
              {count(accuracy.hit)} / {count(accuracy.fired)}
            </Note>
          </FicheRow>
        )}
        <FicheRow label={t.damageDealtTaken}>
          {count(shots.damageDealt)} / {count(shots.damageTaken)}
        </FicheRow>
        {shots.weapons.map((w) => (
          <FicheRow key={w.name} label={w.name}>
            {t.shotsFmt(w.fired)}
            <Note>{t.damageFmt(count(w.damage))}</Note>
          </FicheRow>
        ))}
      </FicheRows>
    </FicheSection>
  )
}
