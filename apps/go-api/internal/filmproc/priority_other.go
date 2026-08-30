//go:build !windows

package filmproc

// priority_other.go — LA PRIORITE CPU BASSE DE L'ENFANT, hors Windows.
//
// LE NICE SE POSE SUR L'ENFANT, PAS SUR LE PARENT, et c'est toute la subtilite : `os/exec` ne
// permet pas d'appeler `setpriority` entre le `fork` et l'`exec`. On passe donc par le champ
// `SysProcAttr` qui existe partout — mais aucun Unix n'y expose de priorite. La voie portable
// et sans dependance est l'HERITAGE : un enfant herite du `nice` de son parent.
//
// CE QUE CE FICHIER FAIT DONC, ET CE QU'IL NE FAIT PAS. Il ne fait RIEN, deliberement : sur les
// plateformes non-Windows, ces boucles ne tournent qu'en CI (Linux, machine dediee, personne
// devant l'ecran) ou sur le VPS de calcul — deux contextes ou la priorite basse ne protege
// aucun usage interactif. Le poste de TRAVAIL de l'utilisateur, celui que le sinistre du
// 2026-08-26 a fait suffoquer, est sous Windows.
//
// Le jour ou une boucle de mesure tournerait sur un poste de travail Unix, c'est ici qu'il
// faudrait poser un `syscall.Setpriority` sur le parent avant le lancement — en assumant qu'il
// s'applique aussi a lui.

import "os/exec"

// lowPriority : sans effet hors Windows (cf. l'en-tete — c'est une decision, pas un oubli).
func lowPriority(_ *exec.Cmd) {}
