
/* WARNING: Type propagation algorithm not settling */
/* WARNING: Globals starting with '_' overlap smaller symbols at the same address */

ulonglong FUN_140544ec8(int *param_1)

{
  int *piVar1;
  undefined4 uVar2;
  undefined4 uVar3;
  undefined4 uVar4;
  char cVar5;
  uint uVar6;
  undefined8 uVar7;
  ulonglong uVar8;
  undefined8 *puVar9;
  undefined8 *puVar10;
  char *pcVar11;
  undefined8 *puVar12;
  char *pcVar13;
  longlong lVar14;
  undefined1 uVar15;
  longlong lVar16;
  undefined1 local_res8 [8];
  int *local_res10;
  char local_368 [272];
  char local_258;
  undefined8 local_257 [33];
  undefined8 local_148 [34];
  
  uVar15 = 0;
  local_368[0] = '\x01';
  local_368[1] = '\0';
  piVar1 = param_1 + 0x3a959;
  uVar6 = FUN_1409a477c(param_1,piVar1);
  if (uVar6 < 2) {
    FUN_1408be85c(0);
    FUN_1408bdd4c(0);
    *(undefined4 *)(DAT_144c1cfa8 + 4) = 6;
  }
  cVar5 = FUN_140545350();
  _DAT_144e50320 = 1;
  DAT_144e50324 = 0xffffffff;
  _DAT_144e50328 = 0;
  uRam0000000144e50330 = 0;
  _DAT_144e50338 = 0;
  _DAT_144e50340 = 0;
  DAT_1445a7744 = DAT_1445a7744 | 0x20;
  _DAT_144e50310 = piVar1;
  if (cVar5 != '\0') {
    FUN_140aea8e0(param_1 + 0x3a547,param_1 + 0x3a9a5,1);
  }
  _DAT_144eb5848 = *param_1;
  lVar16 = 2;
  if (*(int *)(DAT_144c1cfa8 + 4) == 5) {
    uVar15 = 1;
LAB_140544fb7:
    if (1 < uVar6) goto LAB_140544fc0;
    *(undefined4 *)(DAT_144c1cfa8 + 4) = 0;
    FUN_1408be85c(param_1);
    cVar5 = '\x01';
  }
  else {
    if (((uVar6 == 0) || (uVar6 == 1)) || (uVar6 != 4)) goto LAB_140544fb7;
    puVar9 = (undefined8 *)
             FUN_1424cf86c(local_148,
                           "s_main_game_globals->game_loaded_status == \'_map_load_status_failed_to_load\'. Map is unavailable."
                          );
    local_258 = '\0';
    lVar14 = 2;
    puVar10 = local_257;
    do {
      uVar7 = puVar9[1];
      *puVar10 = *puVar9;
      puVar10[1] = uVar7;
      uVar7 = puVar9[3];
      puVar10[2] = puVar9[2];
      puVar10[3] = uVar7;
      uVar7 = puVar9[5];
      puVar10[4] = puVar9[4];
      puVar10[5] = uVar7;
      uVar7 = puVar9[7];
      puVar10[6] = puVar9[6];
      puVar10[7] = uVar7;
      uVar7 = puVar9[9];
      puVar10[8] = puVar9[8];
      puVar10[9] = uVar7;
      uVar7 = puVar9[0xb];
      puVar10[10] = puVar9[10];
      puVar10[0xb] = uVar7;
      uVar2 = *(undefined4 *)((longlong)puVar9 + 100);
      uVar3 = *(undefined4 *)(puVar9 + 0xd);
      uVar4 = *(undefined4 *)((longlong)puVar9 + 0x6c);
      *(undefined4 *)(puVar10 + 0xc) = *(undefined4 *)(puVar9 + 0xc);
      *(undefined4 *)((longlong)puVar10 + 100) = uVar2;
      *(undefined4 *)(puVar10 + 0xd) = uVar3;
      *(undefined4 *)((longlong)puVar10 + 0x6c) = uVar4;
      puVar12 = puVar10 + 0x10;
      uVar2 = *(undefined4 *)((longlong)puVar9 + 0x74);
      uVar3 = *(undefined4 *)(puVar9 + 0xf);
      uVar4 = *(undefined4 *)((longlong)puVar9 + 0x7c);
      *(undefined4 *)(puVar10 + 0xe) = *(undefined4 *)(puVar9 + 0xe);
      *(undefined4 *)((longlong)puVar10 + 0x74) = uVar2;
      *(undefined4 *)(puVar10 + 0xf) = uVar3;
      *(undefined4 *)((longlong)puVar10 + 0x7c) = uVar4;
      puVar9 = puVar9 + 0x10;
      lVar14 = lVar14 + -1;
      puVar10 = puVar12;
    } while (lVar14 != 0);
    *(undefined4 *)puVar12 = *(undefined4 *)puVar9;
    FUN_1405450c8(local_368,&local_258);
    cVar5 = local_368[0];
  }
  if (cVar5 == '\0') {
    uVar8 = FUN_1422c810c();
    return uVar8;
  }
LAB_140544fc0:
  FUN_14054521c();
  FUN_1405451e0();
  *(undefined4 *)(DAT_144c1cfa8 + 4) = 1;
  FUN_1424c7f08(local_res8);
  local_res10 = param_1 + 0x3a547;
  FUN_140544be8(&local_258);
  cVar5 = local_258;
  local_368[0] = local_258;
  if (local_258 == '\0') {
    puVar10 = local_257;
    lVar14 = 2;
    pcVar11 = local_368 + 1;
    do {
      uVar7 = puVar10[1];
      *(undefined8 *)pcVar11 = *puVar10;
      *(undefined8 *)(pcVar11 + 8) = uVar7;
      uVar7 = puVar10[3];
      *(undefined8 *)(pcVar11 + 0x10) = puVar10[2];
      *(undefined8 *)(pcVar11 + 0x18) = uVar7;
      uVar7 = puVar10[5];
      *(undefined8 *)(pcVar11 + 0x20) = puVar10[4];
      *(undefined8 *)(pcVar11 + 0x28) = uVar7;
      uVar7 = puVar10[7];
      *(undefined8 *)(pcVar11 + 0x30) = puVar10[6];
      *(undefined8 *)(pcVar11 + 0x38) = uVar7;
      uVar7 = puVar10[9];
      *(undefined8 *)(pcVar11 + 0x40) = puVar10[8];
      *(undefined8 *)(pcVar11 + 0x48) = uVar7;
      uVar7 = puVar10[0xb];
      *(undefined8 *)(pcVar11 + 0x50) = puVar10[10];
      *(undefined8 *)(pcVar11 + 0x58) = uVar7;
      uVar2 = *(undefined4 *)((longlong)puVar10 + 100);
      uVar3 = *(undefined4 *)(puVar10 + 0xd);
      uVar4 = *(undefined4 *)((longlong)puVar10 + 0x6c);
      *(undefined4 *)(pcVar11 + 0x60) = *(undefined4 *)(puVar10 + 0xc);
      *(undefined4 *)(pcVar11 + 100) = uVar2;
      *(undefined4 *)(pcVar11 + 0x68) = uVar3;
      *(undefined4 *)(pcVar11 + 0x6c) = uVar4;
      pcVar13 = pcVar11 + 0x80;
      uVar7 = puVar10[0xf];
      *(undefined8 *)(pcVar11 + 0x70) = puVar10[0xe];
      *(undefined8 *)(pcVar11 + 0x78) = uVar7;
      puVar10 = puVar10 + 0x10;
      lVar14 = lVar14 + -1;
      pcVar11 = pcVar13;
    } while (lVar14 != 0);
    *(undefined4 *)pcVar13 = *(undefined4 *)puVar10;
  }
  else {
    FUN_1405450d8(param_1);
    *(undefined4 *)(DAT_144c1cfa8 + 4) = 2;
    FUN_14051c0f4(DAT_144c1cfa8 + 8,piVar1,0x104);
  }
  if (cVar5 == '\0') {
    FUN_1428c7d04(uVar15);
  }
  uVar7 = FUN_1405f6254();
  lVar14 = DAT_1451a71c8;
  DAT_14494a958 = 0;
  DAT_1451a71c0 = 0;
  DAT_1451a71c8 = 0;
  if (lVar14 != 0) {
    uVar7 = FUN_1404f965c();
  }
  if ((cVar5 == '\0') && (*param_1 == 3)) {
    pcVar11 = local_368 + 1;
    puVar10 = local_148;
    do {
      uVar7 = *(undefined8 *)(pcVar11 + 8);
      *puVar10 = *(undefined8 *)pcVar11;
      puVar10[1] = uVar7;
      uVar7 = *(undefined8 *)(pcVar11 + 0x18);
      puVar10[2] = *(undefined8 *)(pcVar11 + 0x10);
      puVar10[3] = uVar7;
      uVar7 = *(undefined8 *)(pcVar11 + 0x28);
      puVar10[4] = *(undefined8 *)(pcVar11 + 0x20);
      puVar10[5] = uVar7;
      uVar7 = *(undefined8 *)(pcVar11 + 0x38);
      puVar10[6] = *(undefined8 *)(pcVar11 + 0x30);
      puVar10[7] = uVar7;
      uVar7 = *(undefined8 *)(pcVar11 + 0x48);
      puVar10[8] = *(undefined8 *)(pcVar11 + 0x40);
      puVar10[9] = uVar7;
      uVar7 = *(undefined8 *)(pcVar11 + 0x58);
      puVar10[10] = *(undefined8 *)(pcVar11 + 0x50);
      puVar10[0xb] = uVar7;
      uVar2 = *(undefined4 *)(pcVar11 + 100);
      uVar3 = *(undefined4 *)(pcVar11 + 0x68);
      uVar4 = *(undefined4 *)(pcVar11 + 0x6c);
      *(undefined4 *)(puVar10 + 0xc) = *(undefined4 *)(pcVar11 + 0x60);
      *(undefined4 *)((longlong)puVar10 + 100) = uVar2;
      *(undefined4 *)(puVar10 + 0xd) = uVar3;
      *(undefined4 *)((longlong)puVar10 + 0x6c) = uVar4;
      puVar9 = puVar10 + 0x10;
      uVar2 = *(undefined4 *)(pcVar11 + 0x74);
      uVar3 = *(undefined4 *)(pcVar11 + 0x78);
      uVar4 = *(undefined4 *)(pcVar11 + 0x7c);
      *(undefined4 *)(puVar10 + 0xe) = *(undefined4 *)(pcVar11 + 0x70);
      *(undefined4 *)((longlong)puVar10 + 0x74) = uVar2;
      *(undefined4 *)(puVar10 + 0xf) = uVar3;
      *(undefined4 *)((longlong)puVar10 + 0x7c) = uVar4;
      pcVar11 = pcVar11 + 0x80;
      lVar16 = lVar16 + -1;
      puVar10 = puVar9;
    } while (lVar16 != 0);
    *(undefined4 *)puVar9 = *(undefined4 *)pcVar11;
    uVar7 = FUN_1414d1af0("main_game_load_map failed for \'%s\', failure reason: %s",piVar1,
                          local_148);
  }
  return CONCAT71((int7)((ulonglong)uVar7 >> 8),cVar5 == '\0') ^ 1;
}

