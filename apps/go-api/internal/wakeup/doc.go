// Package wakeup : la séquence de réveil d'une instance qui s'arrête avec la machine (ADR 0045 D-10).
//
// Un serveur hébergé ne dort jamais et redémarre à chaque déploiement ; un serveur desktop s'arrête
// et se met en veille avec le poste de l'utilisateur. Ce qu'il a manqué se rattrape en trois temps,
// dans cet ordre :
//
//  1. vérifier le refresh token de chaque profil suivi ([TokenCheck]) ;
//  2. si la dernière synchronisation connue est plus vieille que l'intervalle d'auto-sync, en tirer
//     une tout de suite ([CatchUp]) ;
//  3. recommencer à chaque sortie de veille détectée ([SleepDetector]).
//
// La séquence [Sequence] est UNE fonction à deux déclencheurs (le lancement, la sortie de veille).
// Elle ne décide pas du profil qui la lance : l'appelant la branche sous la capacité
// config.DeploymentCapabilities.WakeSequence. Le mode -mode refresh-only réutilise la seule
// vérification des jetons, sans base ni synchronisation.
//
// Les jetons ne sont jamais re-capturés pour « réparer » une authentification (ADR 0023) : un jeton
// vivant se rafraîchit, un jeton mort lève la bannière de reconnexion, un réseau absent ne marque rien.
package wakeup
