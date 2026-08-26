/**
 * replayCapture.ts — CE QUI SORT DU REJEU : le nom du fichier, le téléchargement, et l'image
 * du canvas.
 *
 * POURQUOI UN MODULE À PART, et pas trois lignes dans le composant. Les trois opérations
 * ci-dessous sont les seules du chantier de capture qui ne dépendent NI de React, NI de
 * l'état de lecture : nommer, remettre un blob au navigateur, et lire les pixels d'une toile.
 * Les isoler les rend testables sans monter un composant — et le cliquet de taille du canvas
 * (`placementFamily.guard.test.ts`) interdit de toute façon d'y loger la moindre logique.
 *
 * # LE NOM DIT L'INSTANT DU MATCH, PAS L'HEURE DE L'ORDINATEUR
 *
 * Un rejeu se relit à plusieurs, et le fichier voyage : `rejeu-<match>-12m34s.png` se replace
 * tout seul dans la partie, là où un horodatage local ne dit rien à personne d'autre. C'est
 * pour cela que l'horodatage n'est qu'un REPLI — il n'apparaît que quand le match ou son
 * instant manquent, et il vaut alors mieux qu'un nom qui MENTIRAIT sur l'instant capturé
 * (« 0m00s » se lirait « coup d'envoi »).
 *
 * L'HORLOGE DU REPLI EST UN PARAMÈTRE, jamais un `Date.now()` enfoui : une fonction de nommage
 * qui lit l'heure du système n'est pas testable, et ce module existe justement pour l'être.
 */

/** Les caractères qu'un nom de fichier garde ; tout le reste devient un tiret. */
const FILENAME_SAFE = /[^A-Za-z0-9._-]+/g

/**
 * buildCaptureFilename rend le nom du fichier téléchargé (décision 8 du plan).
 *
 * Nominal : `rejeu-<matchId>-<XmYYs>.<ext>`, où `XmYYs` est le temps de MATCH de l'instant
 * capturé (le début de l'enregistrement, pour une vidéo). Repli, quand le match n'est pas
 * identifiable OU que son instant est inconnu : `rejeu-<AAAAMMJJ-HHMMSS>.<ext>`.
 */
export function buildCaptureFilename(
  matchId: string | null,
  matchMs: number | null,
  ext: string,
  now: Date,
): string {
  const id = (matchId ?? '').trim().replace(FILENAME_SAFE, '-')
  if (id === '' || matchMs === null || !Number.isFinite(matchMs)) {
    return `rejeu-${stampOf(now)}.${ext}`
  }
  return `rejeu-${id}-${matchClock(matchMs)}.${ext}`
}

/** `XmYYs` : les minutes sans zéro de tête, les secondes toujours sur deux chiffres. */
function matchClock(ms: number): string {
  const total = Math.max(Math.floor(ms / 1000), 0)
  const sec = total % 60
  return `${Math.floor(total / 60)}m${sec < 10 ? '0' : ''}${sec}s`
}

/** `AAAAMMJJ-HHMMSS` en heure LOCALE : le repli sert à l'utilisateur, pas à une machine. */
function stampOf(now: Date): string {
  const p = (n: number) => String(n).padStart(2, '0')
  const date = `${now.getFullYear()}${p(now.getMonth() + 1)}${p(now.getDate())}`
  return `${date}-${p(now.getHours())}${p(now.getMinutes())}${p(now.getSeconds())}`
}

/**
 * triggerDownload remet un blob au navigateur sous le nom donné.
 *
 * MÊME PATRON QUE LE RESTE DU DÉPÔT (`queries.ts` pour le fond de carte, `lib/api/client.ts`
 * pour l'export CSV) : une URL d'objet, une ancre `download`, et la RÉVOCATION tout de suite
 * après. Sans elle le blob — plusieurs mégaoctets pour une vidéo — resterait retenu jusqu'à
 * la fermeture de l'onglet.
 */
export function triggerDownload(blob: Blob, filename: string): void {
  const url = URL.createObjectURL(blob)
  try {
    const a = document.createElement('a')
    a.href = url
    a.download = filename
    a.rel = 'noopener'
    a.click()
  } finally {
    URL.revokeObjectURL(url)
  }
}

/**
 * captureCanvasImage rend les pixels de la toile en PNG, à sa résolution NATIVE.
 *
 * « Native » veut dire le backing store : le canvas du rejeu est dimensionné en px physiques
 * (largeur × devicePixelRatio) et le contexte porte la mise à l'échelle — `toBlob` sort donc
 * l'image en pleine densité sans qu'on ait rien à recalculer ici.
 *
 * `null` N'EST PAS UNE PANNE À REMONTER : une toile vierge, un contexte perdu ou un navigateur
 * qui refuse rendent `null`, et l'appelant ne télécharge alors rien. Un fichier vide serait
 * pire que pas de fichier.
 */
export function captureCanvasImage(canvas: HTMLCanvasElement): Promise<Blob | null> {
  return new Promise((resolve) => {
    if (typeof canvas.toBlob !== 'function') {
      resolve(null)
      return
    }
    canvas.toBlob((blob) => resolve(blob), 'image/png')
  })
}
