
undefined1 FUN_140516fa0(longlong param_1)

{
  longlong lVar1;
  bool bVar2;
  char cVar3;
  undefined1 uVar4;
  char cVar5;
  undefined4 uVar6;
  int iVar7;
  uint uVar8;
  byte *pbVar9;
  uint *puVar10;
  uint uVar11;
  uint uVar12;
  uint uVar13;
  undefined1 uVar14;
  uint uVar15;
  undefined1 uVar16;
  undefined1 local_res8;
  uint local_res18 [2];
  undefined1 local_res20 [8];
  char local_5e8 [4];
  int local_5e4;
  int local_5e0;
  int local_5dc;
  int local_5d8 [4];
  undefined1 local_5c8 [8];
  undefined1 *local_5c0;
  undefined1 *local_5b8;
  int local_5b0;
  undefined8 local_5ac;
  undefined1 local_5a4;
  undefined8 local_5a0;
  undefined8 local_598;
  undefined4 local_590;
  undefined1 *local_588;
  undefined4 local_580;
  undefined1 local_4c8 [1160];
  
  local_5e4 = 0;
  iVar7 = *(int *)(param_1 + 0x3044);
  bVar2 = false;
  local_res8 = 0;
  if (iVar7 == 3) {
    if (*(char *)(param_1 + 0x3348) == '\0') goto LAB_140516fed;
    FUN_142fd2158();
    iVar7 = *(int *)(param_1 + 0x3044);
  }
  if ((iVar7 == 4) &&
     (iVar7 = FUN_1405f5008(*(undefined4 *)(param_1 + 0x3358)),
     *(int *)(*(longlong *)(param_1 + 0xab8) + 0xc) < iVar7)) {
    uVar4 = FUN_1422c3bef();
    return uVar4;
  }
LAB_140516fed:
  if (3 < *(int *)(param_1 + 0x3044)) {
    uVar15 = 0xffffffff;
    do {
      if (uVar15 == 0xffffffff) {
        uVar12 = 1;
        uVar13 = 0;
        uVar11 = 0;
LAB_140517025:
        if (uVar15 == 0xffffffff) {
          uVar12 = uVar11;
        }
        if (((int)uVar12 < 0) || (*(int *)(param_1 + 0x2fe8) <= (int)uVar12)) {
          uVar15 = 0x80000000;
          goto LAB_140517295;
        }
        pbVar9 = (byte *)(((longlong)(int)uVar12 + 0x2ff) * 0x10 + param_1);
        uVar15 = uVar13;
      }
      else {
        if (-1 < (int)uVar15) {
          uVar11 = uVar15 & 0x7fffffff;
          uVar12 = uVar11 + 1;
          uVar13 = uVar12 & 0x7fffffff;
          goto LAB_140517025;
        }
        if (uVar15 != 0xffffffff) break;
LAB_140517295:
        lVar1 = *(longlong *)(param_1 + 0x3030);
        if ((lVar1 == 0) || (*(longlong *)(lVar1 + 0x20) == 0)) break;
        pbVar9 = (byte *)(lVar1 + 0x18);
      }
      if (uVar15 == 0xffffffff) break;
      if ((*pbVar9 & 0x10) != 0) {
        if (uVar15 == 0xffffffff) break;
        local_res18[0] = 0;
        cVar3 = (**(code **)(**(longlong **)(pbVar9 + 8) + 0x10))
                          (*(longlong **)(pbVar9 + 8),local_res18);
        if (cVar3 != '\0') {
          uVar4 = FUN_1422c3c31();
          return uVar4;
        }
      }
    } while( true );
  }
  do {
    if (*(int *)(param_1 + 0x3044) < 4) goto LAB_14051726c;
    uVar16 = 0;
    uVar4 = 0;
    uVar14 = 0;
    if (((*(longlong *)(param_1 + 0x3030) == 0) ||
        ((*(byte *)(*(longlong *)(param_1 + 0x3030) + 0x28) & 4) == 0)) ||
       (cVar3 = FUN_1406cb0a4(), cVar3 != '\0')) {
      uVar15 = 1;
    }
    else {
      uVar15 = 0;
    }
    uVar12 = 0xffffffff;
    while( true ) {
      if (uVar12 != 0xffffffff) break;
      uVar11 = 1;
      uVar8 = 0;
      uVar13 = 0;
LAB_140517114:
      if (uVar12 == 0xffffffff) {
        uVar11 = uVar13;
      }
      if (((int)uVar11 < 0) || (*(int *)(param_1 + 0x2fe8) <= (int)uVar11)) {
        uVar12 = 0x80000000;
LAB_1405171ba:
        lVar1 = *(longlong *)(param_1 + 0x3030);
        if ((lVar1 == 0) || (*(longlong *)(lVar1 + 0x20) == 0)) goto LAB_140517158;
        puVar10 = (uint *)(lVar1 + 0x18);
      }
      else {
        puVar10 = (uint *)(((longlong)(int)uVar11 + 0x2ff) * 0x10 + param_1);
        uVar12 = uVar8;
      }
      if (uVar12 == 0xffffffff) goto LAB_14051715b;
      if ((*puVar10 & uVar15) == uVar15) {
LAB_140517166:
        uVar11 = *puVar10;
        local_res18[0] = local_res18[0] & 0xffffff00;
        cVar3 = (**(code **)(**(longlong **)(puVar10 + 2) + 0x18))();
        if (cVar3 != '\0') {
          if ((uVar11 & 0x20) == 0) {
            uVar16 = 1;
          }
          else {
            uVar4 = 1;
          }
          if ((char)local_res18[0] != '\0') {
            uVar14 = 1;
          }
        }
      }
    }
    if (-1 < (int)uVar12) {
      uVar13 = uVar12 & 0x7fffffff;
      uVar11 = uVar13 + 1;
      uVar8 = uVar11 & 0x7fffffff;
      goto LAB_140517114;
    }
    if (uVar12 == 0xffffffff) goto LAB_1405171ba;
LAB_140517158:
    puVar10 = (uint *)0x0;
    uVar12 = 0xffffffff;
LAB_14051715b:
    if (uVar12 != 0xffffffff) goto LAB_140517166;
    uVar6 = FUN_1405179f8(param_1);
    cVar5 = FUN_1405185b0(local_5e8,uVar6,uVar16,uVar4,uVar14,local_5e8,local_res20,&local_5e0);
    cVar3 = local_5e8[0];
    local_5e4 = local_5e4 + 1;
    if ((cVar5 != '\0') && (1 < local_5e4)) {
      cVar5 = '\0';
      bVar2 = true;
    }
    uVar4 = local_res8;
    if (cVar5 == '\0') break;
    local_5c0 = local_4c8;
    local_5b8 = local_4c8 + local_5e0;
    local_588 = local_4c8;
    local_5b0 = local_5e0;
    local_5ac = 1;
    local_5a0 = 0;
    local_580 = 0;
    local_5a4 = 0;
    local_598 = 0;
    local_590 = 0;
    FUN_1405167fc(param_1,local_5c8,local_5e8[0],local_res20[0],&local_5dc,local_5d8);
    uVar6 = FUN_1405f50b8();
    *(undefined4 *)(param_1 + 0x3360) = uVar6;
    if (cVar3 != '\0') {
      uVar6 = FUN_1405f50b8();
      *(undefined4 *)(param_1 + 0x3364) = uVar6;
    }
    if (0 < local_5dc) {
      *(longlong *)(param_1 + 0x3388) = *(longlong *)(param_1 + 0x3388) + 1;
      *(longlong *)(param_1 + 0x3390) = *(longlong *)(param_1 + 0x3390) + (longlong)local_5dc;
      *(longlong *)(param_1 + 0x33a0) = *(longlong *)(param_1 + 0x33a0) + (longlong)local_5dc;
      *(longlong *)(param_1 + 0x3398) = *(longlong *)(param_1 + 0x3398) + 1;
      *(int *)(param_1 + 0x33ac) = *(int *)(param_1 + 0x33ac) + 1;
      *(int *)(param_1 + 0x33b0) = *(int *)(param_1 + 0x33b0) + local_5dc;
      *(int *)(param_1 + 0x3380) = local_5dc;
      if (cVar3 != '\0') {
        *(longlong *)(param_1 + 0x3478) = *(longlong *)(param_1 + 0x3478) + 1;
        *(longlong *)(param_1 + 0x3480) = *(longlong *)(param_1 + 0x3480) + (longlong)local_5d8[0];
        *(longlong *)(param_1 + 0x3490) = *(longlong *)(param_1 + 0x3490) + (longlong)local_5d8[0];
        *(longlong *)(param_1 + 0x3488) = *(longlong *)(param_1 + 0x3488) + 1;
        *(int *)(param_1 + 0x349c) = *(int *)(param_1 + 0x349c) + 1;
        *(int *)(param_1 + 0x34a0) = *(int *)(param_1 + 0x34a0) + local_5d8[0];
        *(int *)(param_1 + 0x3470) = local_5d8[0];
      }
    }
    local_res8 = 1;
    uVar4 = 1;
  } while (!bVar2);
  local_res8 = uVar4;
  if (3 < *(int *)(param_1 + 0x3044)) {
    FUN_140515780(param_1);
  }
LAB_14051726c:
  if (*(char *)(param_1 + 0x3378) != '\0') {
    if (2 < *(int *)(param_1 + 0x3044)) {
      FUN_142fd2248(param_1,6,0x2a0);
    }
    *(undefined1 *)(param_1 + 0x3378) = 0;
  }
  return local_res8;
}

