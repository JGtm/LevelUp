
/* WARNING: Globals starting with '_' overlap smaller symbols at the same address */

ulonglong FUN_1405f4478(longlong param_1,undefined8 *param_2,int param_3,uint *param_4)

{
  byte bVar1;
  char cVar2;
  uint uVar3;
  int iVar4;
  int iVar5;
  longlong lVar6;
  undefined4 *puVar7;
  ulonglong uVar8;
  uint uVar9;
  uint uVar10;
  ulonglong uVar11;
  ulonglong uVar12;
  uint *puVar13;
  longlong lVar14;
  uint *puVar15;
  int *piVar16;
  ulonglong uVar17;
  float fVar18;
  float fVar19;
  float fVar20;
  int local_res8 [2];
  undefined8 *local_res10;
  longlong local_d8;
  undefined4 local_c0;
  undefined4 local_bc;
  undefined4 local_b8;
  ulonglong local_b0;
  uint *local_a8;
  longlong local_98;
  undefined1 local_78 [64];
  
  fVar18 = DAT_14498bdb4;
  uVar8 = 0;
  if (*(char *)(param_1 + 0x19) == '\0') {
    uVar11 = 0;
  }
  else {
    uVar11 = uVar8;
    local_res10 = param_2;
    if ((*(char *)(param_1 + 0x2564) != '\0') && (uVar11 = 0, 0 < param_3)) {
      uVar11 = 1;
      *param_4 = *(int *)(param_1 + 8) << 0x1e | *param_4 & 0x3fffffff;
      FUN_140bbd808(param_4,fVar18);
      *param_4 = *param_4 & 0xc17fffff | 0x1001fff;
    }
    fVar18 = DAT_143cd8374;
    piVar16 = (int *)(param_1 + 0x2570);
    lVar14 = param_1 + 0x4a88;
    local_d8 = 0;
    puVar13 = param_4 + uVar11;
    uVar12 = uVar11;
    uVar17 = uVar8;
    do {
      if ((longlong)param_3 <= (longlong)uVar12) break;
      uVar3 = 1 << ((byte)uVar17 & 0x1f);
      uVar10 = uVar3 & *(uint *)(param_1 + 0x1f44);
      uVar9 = uVar3 & *(uint *)(param_1 + 0x2550);
      if (((uVar10 != 0) || (uVar9 != 0)) || ((uVar3 & *(uint *)(param_1 + 0x2558)) != 0)) {
        local_b0 = uVar12 + 1;
        uVar11 = (ulonglong)((int)uVar11 + 1);
        local_a8 = puVar13 + 1;
        if (((((uVar10 != 0) || (fVar19 = DAT_14498bdac, uVar9 != 0)) &&
             ((uVar3 = FUN_14076c748(*param_2), fVar19 = DAT_14498bdb4, 0x20 < uVar3 ||
              ((&DAT_144de4348)[(int)uVar3] == '\0')))) && (*piVar16 != 0)) &&
           ((cVar2 = FUN_1404f1ca4(), cVar2 != '\0' ||
            (cVar2 = FUN_14048ee34(), fVar19 = DAT_14498bdb4, cVar2 == '\0')))) {
          fVar20 = 0.0;
          lVar6 = FUN_140b763e4(uVar17);
          iVar5 = 0;
          if ((lVar6 != 0) && (*(int *)(lVar6 + 0x2e0) != -1)) {
            lVar6 = FUN_140498800(*(int *)(lVar6 + 0x2e0),0x1003);
            local_res8[0] = -1;
            puVar7 = (undefined4 *)FUN_140493720(lVar6,local_78);
            local_c0 = *puVar7;
            local_bc = puVar7[1];
            local_b8 = puVar7[2];
            if (*(int *)(lVar6 + 0x114) != -1) {
              local_res8[0] = *(int *)(lVar6 + 0x114);
            }
            if (local_res8[0] == -1) {
              fVar20 = (float)FUN_141fd95ec(&local_c0,DAT_144989f3c,local_res10);
            }
            else {
              fVar20 = (float)FUN_14076c3b8(1,local_res8);
            }
            iVar5 = DAT_144989f48;
            if ((fVar20 < DAT_144989f40) && (iVar5 = DAT_144989f4c, DAT_144989f44 < fVar20)) {
              iVar5 = (int)((float)DAT_144989f4c -
                           ((fVar20 - DAT_144989f44) / (DAT_144989f40 - DAT_144989f44)) *
                           (float)(DAT_144989f4c - DAT_144989f48));
            }
          }
          iVar4 = FUN_1405f5008(*piVar16);
          fVar19 = DAT_14498bdb4;
          if (iVar4 < DAT_14498bdb0) {
            if (iVar4 < iVar5) {
              fVar19 = fVar20 * _DAT_14498bdc4 + _DAT_14498bdc0;
            }
            else {
              fVar19 = fVar20 * _DAT_14498bdbc + _DAT_14498bdb8;
            }
          }
          bVar1 = *(byte *)(local_d8 + 0x5a10 + param_1);
          FUN_1404f1ca4();
          if ((bVar1 != 0xff) &&
             (local_98 = lVar14 + 0x588,
             *(longlong *)
              (*(longlong *)(lVar14 + 0x590) +
              (*(longlong *)(lVar14 + 0x5a0) + (ulonglong)bVar1 & *(longlong *)(lVar14 + 0x598) - 1U
              ) * 8) != 0)) {
            uVar8 = FUN_1422daca3();
            return uVar8;
          }
        }
        if (fVar19 <= DAT_143cd8370) {
          fVar19 = DAT_143cd8370;
        }
        if (fVar18 <= fVar19) {
          fVar19 = fVar18;
        }
        *puVar13 = ((uint)(longlong)(fVar19 * DAT_143cd9720) & 0x3ff) << 0xd | (uint)uVar17 & 0x1fff
                   | *(int *)(param_1 + 8) << 0x1e;
        uVar12 = local_b0;
        param_2 = local_res10;
        puVar13 = local_a8;
      }
      local_d8 = local_d8 + 1;
      uVar3 = (uint)uVar17 + 1;
      uVar17 = (ulonglong)uVar3;
      piVar16 = piVar16 + 1;
      lVar14 = lVar14 + 0x28;
    } while ((int)uVar3 < 0x20);
    lVar14 = (longlong)(int)uVar11;
    puVar7 = (undefined4 *)(param_1 + 0x1040);
    puVar13 = param_4 + lVar14;
    do {
      if (param_3 <= lVar14) {
        return uVar11;
      }
      uVar3 = (uint)uVar8;
      puVar15 = puVar13;
      if ((*(uint *)(param_1 + 0x2048) >> (uVar3 & 0x1f) & 1) != 0) {
        puVar15 = puVar13 + 1;
        uVar11 = (ulonglong)((int)uVar11 + 1);
        lVar14 = lVar14 + 1;
        iVar5 = FUN_1405f5008(*puVar7);
        fVar18 = (float)FUN_140474bcc((float)iVar5 / DAT_143cd8398);
        fVar18 = ((DAT_14498bdb4 - DAT_143cd837c) - _DAT_14498bdb8) * fVar18 + _DAT_14498bdb8;
        *puVar13 = *(int *)(param_1 + 8) << 0x1e | *puVar13 & 0x3fffffff;
        FUN_140bbd808(puVar13,fVar18);
        *puVar13 = *puVar13 & 0xc0ffe000 | uVar3 & 0x1fff | 0x800000;
      }
      uVar8 = (ulonglong)(uVar3 + 1);
      puVar7 = puVar7 + 1;
      puVar13 = puVar15;
    } while ((int)(uVar3 + 1) < 0x20);
  }
  return uVar11;
}

