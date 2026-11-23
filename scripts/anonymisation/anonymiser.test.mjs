// anonymiser.test.mjs - tests de l'outil d'anonymisation. Lancer : node --test scripts/anonymisation/anonymiser.test.mjs
//
// Tous les noms ci-dessous sont INVENTES (aucun pseudo ni chemin reel).

import assert from 'node:assert/strict';
import { execFileSync } from 'node:child_process';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import test from 'node:test';
import { lireTableTexte } from './table.mjs';
import { compiler, remplacerTexte, verifierIdempotence } from './remplacer.mjs';
import { decoder, globVersRegex, lireListeChemins } from './fichiers.mjs';
import { executer } from './anonymiser.mjs';

const TABLE = [
  '# table d\'exemple',
  'pseudo\tZorbaTheBlue\tjoueur_001\tvariantes',
  'pseudo\tMarmot\tjoueur_002\tambigu',
  'pseudo\tDuo Tampon\tjoueur_003\tvariantes',
  'chemin\tC:\\Users\\Exemple\tposte',
  'chemin\tC:\\Users\\EXEMPLESHORT~1\tposte',
  'texte\tmachine-exemple\t<machine>\tmot',
].join('\n');

function moteurDExemple() {
  const entrees = lireTableTexte(TABLE);
  return compiler(entrees);
}
const appliquer = (t, f = 'doc.md') => remplacerTexte(t, moteurDExemple(), f).texte;

test('table : refuse un type inconnu, une ligne courte, un doublon, un drapeau inconnu', () => {
  assert.throws(() => lireTableTexte('inconnu\tabcd\tx'), /type inconnu/);
  assert.throws(() => lireTableTexte('pseudo\tabcd'), /attendu/);
  assert.throws(() => lireTableTexte('pseudo\tabcd\tj1\npseudo\tabcd\tj2'), /deja defini/);
  assert.throws(() => lireTableTexte('pseudo\tabcd\tj1\tbizarre'), /drapeau inconnu/);
  assert.throws(() => lireTableTexte('pseudo\tab\tj1'), /moins de 3/);
});

test('pseudo : mot entier et sensible a la casse', () => {
  assert.equal(appliquer('ZorbaTheBlue a gagne'), 'joueur_001 a gagne');
  assert.equal(appliquer('xZorbaTheBlue ZorbaTheBlue2 zorbatheblue2'), 'xZorbaTheBlue ZorbaTheBlue2 zorbatheblue2');
  assert.equal(appliquer('(ZorbaTheBlue), "ZorbaTheBlue".'), '(joueur_001), "joueur_001".');
  assert.equal(appliquer('Zorbathebluex'), 'Zorbathebluex');
});

test('pseudo : variantes de casse et d\'URL', () => {
  assert.equal(appliquer('/players/zorbatheblue/x'), '/players/joueur_001/x');
  assert.equal(appliquer('ZORBATHEBLUE'), 'JOUEUR_001');
  assert.equal(appliquer('Duo Tampon'), 'joueur_003');
  assert.equal(appliquer('/players/Duo%20Tampon/ Duo_Tampon Duo+Tampon'), '/players/joueur_003/ joueur_003 joueur_003');
});

test('pseudo ambigu : remplace en contexte, ecarte ailleurs', () => {
  assert.equal(appliquer('le marmot dort et Marmot aussi'), 'le marmot dort et Marmot aussi');
  assert.equal(appliquer('joueur Marmot a gagne'), 'joueur joueur_002 a gagne');
  assert.equal(appliquer('{"gamertag": "Marmot"}'), '{"gamertag": "joueur_002"}');
  assert.equal(appliquer('GET /players/Marmot/stats'), 'GET /players/joueur_002/stats');
  assert.equal(appliquer('Marmot a éliminé ZorbaTheBlue'), 'joueur_002 a éliminé joueur_001');
  assert.equal(appliquer('Le Marmot est un animal'), 'Le Marmot est un animal');
});

test('pseudo ambigu : cellule de tableau sous une colonne de pseudo', () => {
  const md = '| rang | gamertag | score |\n|---|---|---|\n| 1 | Marmot | 12 |\n| 2 | Autre | 9 |\n';
  assert.equal(appliquer(md), md.replace('Marmot', 'joueur_002'));
  const tsv = 'rang\tgamertag\tnote\n1\tMarmot\tMarmot\n';
  assert.equal(appliquer(tsv, 'x.tsv'), 'rang\tgamertag\tnote\n1\tjoueur_002\tMarmot\n');
  const horsColonne = '| rang | note |\n|---|---|\n| 1 | Marmot |\n';
  assert.equal(appliquer(horsColonne), horsColonne);
});

test('rapport : les ecartes sont notes, jamais remplaces', () => {
  const r = remplacerTexte('un Marmot\njoueur Marmot\n', moteurDExemple(), 'a.md');
  assert.deepEqual(r.occurrences.map((o) => [o.ligne, o.statut]), [[1, 'ecarte'], [2, 'remplace']]);
  assert.match(r.occurrences[0].extrait, /un Marmot/);
});

test('chemin : toutes les graphies de separateur et de lecteur', () => {
  const attendu = 'poste';
  for (const forme of [
    'C:\\Users\\Exemple',
    'c:\\users\\exemple',
    'C:/Users/Exemple',
    '/c/Users/Exemple',
    '/mnt/c/Users/Exemple',
    'C:\\\\Users\\\\Exemple',
    'C:\\Users\\EXEMPLESHORT~1',
  ]) {
    assert.equal(appliquer(forme), attendu, forme);
  }
  assert.equal(appliquer('\\Users\\Exemple'), '\\poste');
  assert.equal(appliquer('/Users/Exemple/x'), '/poste/x');
  assert.equal(appliquer('C:\\Users\\Exemple\\Downloads\\x.txt'), 'poste\\Downloads\\x.txt');
  assert.equal(appliquer('"C:/Users/Exemple/a b"'), '"poste/a b"');
});

test('chemin : s\'arrete sur une limite de mot', () => {
  assert.equal(appliquer('C:\\Users\\ExempleBis\\x'), 'C:\\Users\\ExempleBis\\x');
  assert.equal(appliquer('C:\\Users\\Autre\\x'), 'C:\\Users\\Autre\\x');
});

test('texte : limites de mot avec le drapeau mot', () => {
  assert.equal(appliquer('ssh machine-exemple'), 'ssh <machine>');
  assert.equal(appliquer('machine-exemple2'), 'machine-exemple2');
});

test('fins de ligne conservees', () => {
  assert.equal(appliquer('a ZorbaTheBlue\r\nb\r\nZorbaTheBlue\n'), 'a joueur_001\r\nb\r\njoueur_001\n');
  assert.equal(appliquer('x\rZorbaTheBlue'), 'x\rjoueur_001');
});

test('idempotence : un second passage ne change rien', () => {
  const texte = 'ZorbaTheBlue, joueur Marmot, Duo Tampon, C:\\Users\\Exemple\\x, machine-exemple';
  const une = appliquer(texte);
  assert.equal(appliquer(une), une);
});

test('table non idempotente : refusee', () => {
  const entrees = lireTableTexte('pseudo\tabcd\tjoueur_abcd_x\npseudo\tjoueur_abcd_x\tz9z9z9');
  assert.throws(() => verifierIdempotence(entrees, compiler(entrees)));
  const doublon = lireTableTexte('pseudo\tabcd\tj_1\npseudo\tefgh\tj_1');
  assert.throws(() => verifierIdempotence(doublon, compiler(doublon)), /meme remplacement/);
  const poste = lireTableTexte('pseudo\tposte\tj_1\nchemin\tC:\\Users\\Exemple\t<poste>');
  assert.throws(() => verifierIdempotence(poste, compiler(poste)), /non idempotent/);
  const bonne = lireTableTexte(TABLE);
  verifierIdempotence(bonne, compiler(bonne));
});

test('decodage : binaire, UTF-16, non UTF-8 ignores ; BOM reconnu', () => {
  assert.equal(decoder(Buffer.from([0x41, 0x00, 0x42])).ignore, 'binaire');
  assert.equal(decoder(Buffer.from([0xff, 0xfe, 0x41, 0x00])).ignore, 'utf16');
  assert.equal(decoder(Buffer.from([0xe9, 0x41])).ignore, 'non-utf8');
  const d = decoder(Buffer.from([0xef, 0xbb, 0xbf, 0x41]));
  assert.equal(d.texte, 'A');
  assert.equal(d.bom, true);
});

test('globs et liste de chemins', () => {
  assert.ok(globVersRegex('**/*_gen.go').test('a/b/x_gen.go'));
  assert.ok(globVersRegex('**/*_gen.go').test('x_gen.go'));
  assert.ok(!globVersRegex('.ai/*.md').test('.ai/a/b.md'));
  assert.ok(globVersRegex('.ai/**').test('.ai/a/b.md'));
  assert.deepEqual(lireListeChemins('# c\n+docs/**\n-docs/adr/x.md\n'), { inclure: ['docs/**'], exclure: ['docs/adr/x.md'] });
  assert.throws(() => lireListeChemins('docs/**'));
});

// --- integration : un depot git jetable ---------------------------------------------------------

function depotJetable() {
  const base = fs.mkdtempSync(path.join(os.tmpdir(), 'anon-test-'));
  const depot = path.join(base, 'depot');
  fs.mkdirSync(depot);
  const git = (...a) => execFileSync('git', a, { cwd: depot, stdio: 'pipe' });
  git('init', '-q');
  git('config', 'user.email', 't@example.invalid');
  git('config', 'user.name', 't');
  const ecrire = (rel, contenu) => {
    fs.mkdirSync(path.dirname(path.join(depot, rel)), { recursive: true });
    fs.writeFileSync(path.join(depot, rel), contenu);
  };
  ecrire('docs/a.md', 'joueur ZorbaTheBlue\r\nchemin C:\\Users\\Exemple\\x\r\n');
  ecrire('docs/bom.md', Buffer.concat([Buffer.from([0xef, 0xbb, 0xbf]), Buffer.from('ZorbaTheBlue\n')]));
  ecrire('docs/latin1.txt', Buffer.from([0x5a, 0x6f, 0x72, 0x62, 0x61, 0x54, 0x68, 0x65, 0x42, 0x6c, 0x75, 0x65, 0x20, 0xe9]));
  ecrire('docs/image.png', Buffer.from([0x89, 0x50, 0x00, 0x5a, 0x6f, 0x72, 0x62, 0x61, 0x54, 0x68, 0x65, 0x42, 0x6c, 0x75, 0x65]));
  ecrire('apps/web/src/routeTree.gen.ts', 'ZorbaTheBlue\n');
  ecrire('apps/go-api/go.sum', 'ZorbaTheBlue\n');
  ecrire('apps/go-api/x_gen.go', 'ZorbaTheBlue\n');
  ecrire('apps/go-api/x.go', 'var s = "ZorbaTheBlue"\n');
  ecrire('docs/ZorbaTheBlue-notes.md', 'rien\n');
  git('add', '.');
  git('commit', '-q', '-m', 'init');
  const table = path.join(base, 'table.tsv');
  fs.writeFileSync(table, TABLE);
  return { base, depot, table, rapport: path.join(base, 'rapport') };
}

const lire = (d, rel) => fs.readFileSync(path.join(d, rel));

test('integration : la simulation ne touche rien et ecrit le rapport hors depot', () => {
  const { depot, table, rapport } = depotJetable();
  const avant = lire(depot, 'docs/a.md');
  executer(['--table', table, '--racine', depot, '--rapport', rapport, '--simulation']);
  assert.deepEqual(lire(depot, 'docs/a.md'), avant);
  assert.ok(fs.existsSync(path.join(rapport, 'fichiers.tsv')));
  const noms = fs.readFileSync(path.join(rapport, 'noms_de_fichiers.tsv'), 'utf8');
  assert.match(noms, /docs\/ZorbaTheBlue-notes\.md/);
  const ignores = fs.readFileSync(path.join(rapport, 'ignores.tsv'), 'utf8');
  assert.match(ignores, /latin1\.txt\tnon-utf8/);
});

test('integration : appliquer reecrit les textes, saute binaires et generes, puis ne change plus rien', () => {
  const { depot, table, rapport } = depotJetable();
  executer(['--table', table, '--racine', depot, '--rapport', rapport, '--appliquer']);
  assert.equal(lire(depot, 'docs/a.md').toString(), 'joueur joueur_001\r\nchemin poste\\x\r\n');
  assert.deepEqual(lire(depot, 'docs/bom.md'), Buffer.concat([Buffer.from([0xef, 0xbb, 0xbf]), Buffer.from('joueur_001\n')]));
  assert.equal(lire(depot, 'apps/go-api/x.go').toString(), 'var s = "joueur_001"\n');
  for (const intact of ['apps/web/src/routeTree.gen.ts', 'apps/go-api/go.sum', 'apps/go-api/x_gen.go']) {
    assert.equal(lire(depot, intact).toString(), 'ZorbaTheBlue\n', intact);
  }
  assert.equal(lire(depot, 'docs/image.png')[3], 0x5a);
  assert.equal(lire(depot, 'docs/latin1.txt')[0], 0x5a);
  const apres = execFileSync('git', ['status', '--porcelain'], { cwd: depot }).toString();
  executer(['--table', table, '--racine', depot, '--appliquer']);
  assert.equal(execFileSync('git', ['status', '--porcelain'], { cwd: depot }).toString(), apres);
  assert.equal(lire(depot, 'docs/a.md').toString(), 'joueur joueur_001\r\nchemin poste\\x\r\n');
});

test('integration : inclusions et exclusions de chemins', () => {
  const { depot, table } = depotJetable();
  executer(['--table', table, '--racine', depot, '--appliquer', '--inclure', 'docs/**', '--exclure', 'docs/bom.md']);
  assert.equal(lire(depot, 'docs/a.md').toString(), 'joueur joueur_001\r\nchemin poste\\x\r\n');
  assert.equal(lire(depot, 'apps/go-api/x.go').toString(), 'var s = "ZorbaTheBlue"\n');
  assert.equal(lire(depot, 'docs/bom.md').subarray(3).toString(), 'ZorbaTheBlue\n');
});

test('integration : refuse une table ou un rapport sous la racine du depot', () => {
  const { depot, table } = depotJetable();
  const dedans = path.join(depot, 'table.tsv');
  fs.copyFileSync(table, dedans);
  assert.throws(() => executer(['--table', dedans, '--racine', depot, '--appliquer']), /hors du depot/);
  assert.throws(() => executer(['--table', table, '--racine', depot, '--rapport', path.join(depot, 'r')]), /hors du depot/);
  assert.throws(() => executer(['--table', table, '--racine', depot, '--simulation', '--appliquer']), /s'excluent/);
});

// `w` ecrit un chemin Windows a partir de sa forme en `/` (evite les doubles barres inverses).
const w = (s) => s.replaceAll('/', String.fromCharCode(92));

test('profil : seul le nom change, le chemin reste un chemin absolu', () => {
  const m = compiler(lireTableTexte('profil\tExemple\tutilisateur\nprofil\tEXEMPLESHORT~1\tutilisateur'));
  const f = (t) => remplacerTexte(t, m).texte;
  assert.equal(f(w('C:/Users/Exemple/Videos')), w('C:/Users/utilisateur/Videos'));
  assert.equal(f('C:/Users/exemple/x /home/Exemple/y'), 'C:/Users/utilisateur/x /home/utilisateur/y');
  assert.equal(f(w('C:/Users/EXEMPLESHORT~1/x')), w('C:/Users/utilisateur/x'));
  assert.equal(f(w('Exemple est un mot, C:/Users/ExempleBis')), w('Exemple est un mot, C:/Users/ExempleBis'));
  assert.equal(f(f(w('C:/Users/Exemple/x'))), w('C:/Users/utilisateur/x'));
});

test('genre de fichier et drapeaux docs/code', async () => {
  const { genreDeFichier } = await import('./fichiers.mjs');
  assert.equal(genreDeFichier('.ai/PLAN.md'), 'docs');
  assert.equal(genreDeFichier('.ai/archive/x/sortie.tsv'), 'docs');
  assert.equal(genreDeFichier('.ai/archive/x/instrument.mjs'), 'code');
  assert.equal(genreDeFichier('.ai/H5/echantillon.json'), 'docs');
  assert.equal(genreDeFichier('docs/adr/0001.md'), 'docs');
  assert.equal(genreDeFichier('NOTICE'), 'docs');
  assert.equal(genreDeFichier('apps/go-api/x_test.go'), 'code');
  assert.equal(genreDeFichier('apps/web/src/a.json'), 'code');
  assert.equal(genreDeFichier('config/x.toml'), 'code');
});

test('integration : <poste> dans les documents, nom neutre dans le code', () => {
  const { depot, base } = depotJetable();
  fs.writeFileSync(path.join(depot, 'docs/p.md'), w('C:/Users/Exemple/x\n'));
  fs.writeFileSync(path.join(depot, 'apps/go-api/p.go'), w('p := `C:/Users/Exemple/x`\n'));
  execFileSync('git', ['add', '.'], { cwd: depot });
  const table = path.join(base, 'table2.tsv');
  fs.writeFileSync(table, `chemin\t${w('C:/Users/Exemple')}\t<poste>\tdocs\nprofil\tExemple\tutilisateur\tcode\n`);
  executer(['--table', table, '--racine', depot, '--appliquer']);
  assert.equal(lire(depot, 'docs/p.md').toString(), '<poste>' + w('/x\n'));
  assert.equal(lire(depot, 'apps/go-api/p.go').toString(), w('p := `C:/Users/utilisateur/x`\n'));
});
