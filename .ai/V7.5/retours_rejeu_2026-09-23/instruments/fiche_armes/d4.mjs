import fs from 'fs';
const id = process.argv[2];
const doc = JSON.parse(fs.readFileSync(`C:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/replays/halo_infinite/${id}.json`, 'utf8'));
const o = {}; for (const g of doc.groundWeapons) { o[g.origin] = (o[g.origin]||0)+1; }
console.log('groundWeapons origins', o);
for (const g of doc.groundWeapons.slice(0,12)) console.log(JSON.stringify(g));
console.log('cov.groundWeapons', JSON.stringify(doc.coverage.groundWeapons));
console.log('cov.groundWeaponItems', JSON.stringify(doc.coverage.groundWeaponItems));
console.log('weaponPads', JSON.stringify(doc.weaponPads).slice(0,600));
console.log('weaponTiers', JSON.stringify(doc.weaponTiers));
