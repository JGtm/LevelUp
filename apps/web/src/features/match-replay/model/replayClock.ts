/**
 * replayClock — L'HORLOGE DE LA PAGE DE REJEU : UN SEUL VERDICT, UNE SEULE POLITIQUE.
 *
 * CE QU'IL REMPLACE (registre 2026-09-05, P0-5). La même question — « à quel instant du
 * match tombe l'image zéro du film ? » — recevait CINQ écritures et TROIS réponses :
 * `replayWindow` rendait `null`, le fil des éliminations mesurait le décalage par
 * appariement, et les médias, la présence et les sièges posaient `0`. Sur un artefact sans
 * origine publiée (5 des 106 artefacts locaux au 2026-09-05), la piste Médias et la piste
 * Kills se retrouvaient donc sur la MÊME frise à deux décalages différents : une capture et
 * le frag qu'elle montre pouvaient s'éloigner de l'origine réelle du film, mesurée de 3,6 s
 * à 50,8 s selon le match.
 *
 * LA POLITIQUE, EN UNE PHRASE : l'origine est celle que l'ARTEFACT PUBLIE ; sans elle, la
 * page n'a pas d'horloge et AUCUNE surface ne place quoi que ce soit sur l'axe du film.
 *
 * POURQUOI PAS `0` EN REPLI. C'est le producteur lui-même qui l'écrit, en capitales, dans
 * `internal/analysis/replay/origin.go` : « ZERO N'EST PAS UNE ORIGINE NEUTRE, C'EST UN
 * REPLI ». Il REFUSE de publier une origine qu'il ne peut pas établir (chunk illisible,
 * témoin contradictoire) plutôt que d'en inventer une ; recopier `0` côté client revient à
 * défaire ce refus et à poser chaque marque à un instant faux — qui se lit comme juste.
 * `replayWindow` avait déjà tranché ainsi (« SANS DONNÉE, PAS DE CADRAGE ») ; les quatre
 * autres surfaces s'alignent.
 *
 * L'EXCEPTION EST NOMMÉE, ET ELLE EST MESURÉE : le fil des éliminations. Lui seul dispose
 * d'une seconde source pour l'origine — l'appariement des kills aux fins de vie de leurs
 * victimes — et cette mesure est étayée (médiane +3 678 ms sur 000d5950, +10 589 ms sur
 * 64e8adfa, +39 856 ms sur e94163af, à 20-70 ms de l'origine publiée quand les deux
 * existent, cf. l'en-tête de `killFeedLogic.ts`). Il garde donc son repli mesuré quand
 * l'horloge n'est pas établie, et rien d'autre ne mesure : une seconde surface qui
 * s'inventerait une origine réintroduirait exactement le défaut ci-dessus.
 *
 * CE QUI ÉTABLIT L'HORLOGE NE DÉPEND QUE DE L'ARTEFACT. L'en-tête du match n'apporte que le
 * countdown (`t0_ms`) : il n'établit ni ne détruit l'horloge, et deux appelants qui n'ont
 * pas le même en-tête obtiennent donc le MÊME verdict. C'est ce qui permet à cette fonction
 * d'être appelée là où chaque surface se trouve, sans que la page ait à faire circuler un
 * objet — un seul verdict par construction, pas par discipline.
 *
 * LES CONVERSIONS, ELLES, VIVENT DANS `lib/replay/matchClock` : ce module-ci ne fait que
 * poser le verdict de la page de rejeu par-dessus.
 */
import { matchClock, type MatchClock, type MatchClockHeader } from '@/lib/replay/matchClock'

import type { ReplayDocumentReady } from '../replayNormalize'

/** Ce que l'horloge lit de l'en-tête du match : le countdown, et lui seul. */
export type ReplayClockHeader = MatchClockHeader

/**
 * L'horloge du rejeu, ÉTABLIE. C'est `MatchClock` tel quel : le rejeu n'ajoute aucune
 * conversion, il ajoute un VERDICT — l'objet existe, ou il n'existe pas.
 */
export type ReplayClock = MatchClock

/**
 * replayClock rend l'horloge de la page, ou `null` quand elle n'est pas établie : artefact
 * sans origine publiée, sans échelle temporelle, ou sans deux images à mettre bout à bout.
 *
 * TROIS PORTES, ET TROIS MESURES. L'origine manque sur 5 des 106 artefacts locaux (les cinq
 * portent `coverage.originResolved: false` — le producteur a refusé de les dater).
 * L'échelle temporelle, elle, est publiée par les 106, toujours à 100 ms : l'exiger ne
 * retire donc rien, et supprime en revanche les deux replis divergents qui traînaient
 * (60 images/s dans `replayLogic`, 100 ms dans `seatLogic`). Un film d'une seule image n'a
 * pas d'axe.
 */
export function replayClock(
  doc: ReplayDocumentReady | null | undefined,
  header?: ReplayClockHeader | null,
): ReplayClock | null {
  return matchClock(doc, header)
}
