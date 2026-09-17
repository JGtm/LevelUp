//go:build research

// Package reapparition — INSTRUMENT DU LOT 3.7 (RECHERCHE SEULE) : ou le film ecrit la
// reapparition des OBJECTIFS et des VEHICULES.
//
// # POURQUOI IL EXISTE
//
// Le lot 3.7 doit repondre sur PIECES, pas par hypothese : quel composant de quel archetype
// porte (a) le retour d'un drapeau / la remise a zero d'une balle, d'un crane ou d'une bombe,
// (b) le delai de reapparition d'un vehicule. La reponse se cherche d'abord CHEZ L'ECRIVAIN —
// la fonction du jeu qui SERIALISE le composant — et cette recherche se faisait jusqu'ici a la
// main dans Ghidra.
//
// # CE QU'IL FAIT, ET POURQUOI IL N'A PAS BESOIN DE GHIDRA
//
// La chaine de `.ai/V7.5/film_re/NOTE_3_6_METHODE_DESCRIPTEURS_2026-09-16.md` est MECANIQUE :
//
//  1. la chaine de caracteres du nom du composant, en `.rdata` ;
//  2. l'accesseur de nom (« thunk ») : `LEA RAX,[rip+disp] ; RET`, 8 octets, seule reference
//     a la chaine ;
//  3. le descripteur : la seule reference au thunk est un SLOT de table de pointeurs en
//     `.rdata`, et ce slot est `descripteur + 0x18` ;
//  4. l'ecrivain : `descripteur + 0x40`, soit `slot du nom + 0x28`.
//
// Ces quatre pas sont des balayages d'octets sur le PE. Ce paquet les execute en Go, lit la
// SIGNATURE de famille du descripteur pour dire si la forme est celle attendue, borne la
// fonction par le repertoire d'exceptions (`.pdata` : `RUNTIME_FUNCTION`, bornes EXACTES) et
// releve dans ces bornes les `ADD dword ptr [reg + 0x2c], N` — le compteur de bits du flux,
// donc la suite ORDONNEE des largeurs (meme note, § 4) — ainsi que les cibles des `CALL rel32`.
//
// LA CALIBRATION EST DANS L'INSTRUMENT, PAS A COTE. `Calibrations` porte les quatre
// concordances de la note plus deux relevees au lot 3.6 (`ti=12 i11`, `ti=12 i12`) : si l'une
// d'elles ne retombe pas sur son adresse connue, la passe ECHOUE et aucune adresse neuve n'est
// publiee. Un releve sans calibration verte ne vaut rien.
//
// # CE QU'IL NE FAIT PAS
//
// Il ne desassemble pas. La suite des `ADD [reg+0x2c], N` est une LECTURE PARTIELLE : elle
// donne les largeurs litterales, jamais les conditions, ni les appels dequantifies (dont la
// largeur est un argument), ni les boucles. Elle DESIGNE la grammaire, elle ne la remplace pas ;
// la lecture complete reste `objdump -d --start-address=... --stop-address=...` sur les bornes
// que cet instrument publie, ou Ghidra.
//
// REGIME : hors ligne, LECTURE SEULE sur `HaloInfinite.exe`, aucune ecriture, aucun film ouvert.
// Garde d'environnement `REAP_EXE` (chemin de l'executable) — sans elle, rien ne tourne.
package reapparition
