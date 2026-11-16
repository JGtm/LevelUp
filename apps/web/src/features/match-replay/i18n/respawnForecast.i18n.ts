/**
 * respawnForecast.i18n.ts — LES TEXTES DE LA PROBABILITÉ DE RÉAPPARITION du rejeu : l'interrupteur du
 * groupe « Vies » du panneau de la vue, et ce que le calque écrit sur la carte (puce du compte à
 * rebours, étiquettes de zone, étiquette d'apparition, légende).
 *
 * « PROBABILITÉ », JAMAIS « PRÉVISION » : le calque ne prédit rien du match affiché (le modèle ne le
 * connaît pas, il a appris sur les autres), il dit où un joueur mort avait le plus de chances de
 * réapparaître. Un sous-dictionnaire à part, comme `lives.i18n.ts` : `fr.ts` et `en.ts` sont au plafond
 * de taille. Servis par `REPLAY_TEXT[locale].respawnForecast`, parité tenue par `satisfies`.
 */
export interface RespawnForecastText {
  /** Libellé de l'interrupteur du panneau et titre de la légende. */
  title: string
  /** Ce que le calque montre, en une phrase (infobulle de l'interrupteur). */
  hint: string
  /** Puce posée sur l'anneau du compte à rebours : le joueur et le temps restant. */
  chipFmt: (name: string, seconds: number) => string
  /** « 2e choix » : le rang du point réel parmi les zones. */
  rankFmt: (rank: number) => string
  /** Une part de moins d'un pour cent. */
  lessThanOnePercent: string
  /** Légende, compte à rebours : « X réapparaît dans N s ». */
  legendCountdownFmt: (name: string, seconds: number) => string
  /** Légende et étiquette d'apparition, point réel dans une zone nommée : « X apparu en Zone ». */
  appearedInFmt: (name: string, zone: string) => string
  /** Point réel hors de toute zone nommée : « X apparu hors zone nommée ». */
  appearedOutsideFmt: (name: string) => string
  /** Second texte de l'étiquette d'apparition : le rang et la probabilité de la zone, ou son absence des zones probables. */
  appearedRankFmt: (rank: string, share: string) => string
  appearedUnlikely: string
  /** Légende : ce que la teinte dit, et la fréquence de mise à jour. */
  legendScale: string
  legendFooter: string
}

export const FR_RESPAWN_FORECAST = {
  title: 'Probabilité de réapparition',
  hint:
    "Pendant le compte à rebours d'un joueur mort, les zones où il a le plus de chances de réapparaître, selon la position de chacun et mises à jour chaque seconde ; puis le point réel. Suit Seul et Masquer",
  chipFmt: (name, seconds) => `${name} · ${seconds} s`,
  rankFmt: (rank) => (rank === 1 ? '1er choix' : `${rank}e choix`),
  lessThanOnePercent: '< 1 %',
  legendCountdownFmt: (name, seconds) => `${name} réapparaît dans ${seconds} s`,
  appearedInFmt: (name, zone) => `${name} apparu en ${zone}`,
  appearedOutsideFmt: (name) => `${name} apparu hors zone nommée`,
  appearedRankFmt: (rank, share) => `${rank} · ${share}`,
  appearedUnlikely: 'hors des zones probables',
  legendScale: 'Teinte : probabilité de la zone',
  legendFooter: 'Mise à jour chaque seconde',
} satisfies RespawnForecastText

/** « 1st », « 2nd », « 3rd », « 4th », « 11th »… */
function englishOrdinal(n: number): string {
  const tens = n % 100
  if (tens >= 11 && tens <= 13) return `${n}th`
  switch (n % 10) {
    case 1:
      return `${n}st`
    case 2:
      return `${n}nd`
    case 3:
      return `${n}rd`
    default:
      return `${n}th`
  }
}

export const EN_RESPAWN_FORECAST = {
  title: 'Respawn probability',
  hint:
    "During a dead player's countdown, the zones where they were most likely to respawn, given everyone's position and updated every second; then the actual spot. Follows Solo and Hide",
  chipFmt: (name, seconds) => `${name} · ${seconds} s`,
  rankFmt: (rank) => `${englishOrdinal(rank)} choice`,
  lessThanOnePercent: '< 1%',
  legendCountdownFmt: (name, seconds) => `${name} respawns in ${seconds} s`,
  appearedInFmt: (name, zone) => `${name} respawned in ${zone}`,
  appearedOutsideFmt: (name) => `${name} respawned outside any named zone`,
  appearedRankFmt: (rank, share) => `${rank} · ${share}`,
  appearedUnlikely: 'outside the likely zones',
  legendScale: 'Tint: zone probability',
  legendFooter: 'Updated every second',
} satisfies RespawnForecastText
