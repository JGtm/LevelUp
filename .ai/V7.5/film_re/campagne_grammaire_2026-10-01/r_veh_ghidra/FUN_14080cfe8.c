
undefined1 FUN_14080cfe8(int *param_1,longlong param_2,undefined1 param_3)

{
  longlong *plVar1;
  bool bVar2;
  char cVar3;
  byte bVar4;
  short sVar5;
  int iVar6;
  uint uVar7;
  uint uVar8;
  int *piVar9;
  int *piVar10;
  ulonglong uVar11;
  longlong lVar12;
  longlong lVar13;
  undefined1 uVar14;
  int *piVar15;
  undefined4 uVar16;
  int extraout_XMM0_Da;
  undefined1 local_res8 [8];
  longlong local_res10;
  int local_res18 [2];
  int local_res20 [2];
  undefined1 local_a8 [4];
  undefined1 local_a4 [4];
  undefined1 local_a0 [8];
  longlong *local_98;
  undefined1 local_90 [56];
  undefined8 local_58;
  
  local_res18[0] = CONCAT31(local_res18[0]._1_3_,param_3);
  uVar14 = 1;
  local_res10 = param_2;
  uVar16 = FUN_141fd72c0(param_2,param_2,param_1 + 6);
  piVar10 = param_1 + 3;
  FUN_14080d6f0(uVar16,param_2,piVar10);
  cVar3 = FUN_1406cf008(param_2);
  *(char *)((longlong)param_1 + 0x1b) = cVar3;
  piVar15 = param_1 + 4;
  if (cVar3 == '\0') {
    FUN_14080dec4(param_2,"variant-name",piVar15);
  }
  else {
    piVar9 = (int *)FUN_14080d7cc(local_a8,piVar10);
    *piVar15 = *piVar9;
  }
  if (DAT_145121140 == '\x01') {
    local_res18[0] = *piVar10;
    FUN_14074d064(local_res8,local_res18);
    cVar3 = FUN_1404785a0(local_res8);
    if (cVar3 == '\0') {
      local_58 = 0;
      local_res20[0] = *piVar10;
      FUN_142ad4f20(local_a0,local_res20,DAT_144b404f0,0,local_90);
      if (local_98 != (longlong *)0x0) {
        LOCK();
        plVar1 = local_98 + 1;
        lVar12 = *plVar1;
        *(int *)plVar1 = (int)*plVar1 + -1;
        UNLOCK();
        param_2 = local_res10;
        if ((int)lVar12 == 1) {
          (**(code **)*local_98)(local_98);
          LOCK();
          piVar9 = (int *)((longlong)local_98 + 0xc);
          iVar6 = *piVar9;
          *piVar9 = *piVar9 + -1;
          UNLOCK();
          param_2 = local_res10;
          if (iVar6 == 1) {
            (**(code **)(*local_98 + 8))(local_98);
            param_2 = local_res10;
          }
        }
      }
    }
  }
  piVar10 = (int *)FUN_14080d61c(local_a4,piVar10,*piVar15);
  param_1[5] = *piVar10;
  cVar3 = FUN_1406cf008(param_2);
  if (cVar3 == '\0') {
    iVar6 = -1;
  }
  else {
    if (0x40 - *(int *)(param_2 + 0x38) < 0x12) {
      sVar5 = FUN_1406d6c7c(param_2,0x12);
    }
    else {
      sVar5 = (short)(*(ulonglong *)(param_2 + 0x30) >> 0x2e);
      *(int *)(param_2 + 0x2c) = *(int *)(param_2 + 0x2c) + 0x12;
      *(ulonglong *)(param_2 + 0x30) = *(ulonglong *)(param_2 + 0x30) << 0x12;
      *(int *)(param_2 + 0x38) = *(int *)(param_2 + 0x38) + 0x12;
    }
    iVar6 = (int)sVar5;
  }
  *param_1 = iVar6;
  FUN_14080d524(param_1 + 1,&local_res10);
  lVar12 = local_res10;
  piVar10 = (int *)(param_2 + 0x38);
  if (0x40 - *piVar10 < 2) {
    bVar4 = FUN_1406d6c7c(local_res10,2);
  }
  else {
    bVar4 = (byte)((ulonglong)*(longlong *)(param_2 + 0x30) >> 0x3e);
    *(int *)(param_2 + 0x2c) = *(int *)(param_2 + 0x2c) + 2;
    *(longlong *)(param_2 + 0x30) = *(longlong *)(param_2 + 0x30) << 2;
    *piVar10 = *piVar10 + 2;
    lVar12 = param_2;
  }
  *(byte *)(param_1 + 2) = bVar4;
  iVar6 = *piVar10;
  if (0x40 - iVar6 < 5) {
    bVar4 = FUN_1406d6c7c(lVar12,5);
  }
  else {
    bVar4 = (byte)((ulonglong)*(longlong *)(lVar12 + 0x30) >> 0x3b);
    *(int *)(lVar12 + 0x2c) = *(int *)(lVar12 + 0x2c) + 5;
    *(longlong *)(lVar12 + 0x30) = *(longlong *)(lVar12 + 0x30) << 5;
    *piVar10 = iVar6 + 5;
  }
  lVar13 = local_res10;
  *(byte *)((longlong)param_1 + 0x1a) = bVar4 - 1;
  iVar6 = *piVar10;
  if (0x40 - iVar6 < 3) {
    uVar7 = FUN_1406d6c7c(local_res10,3);
  }
  else {
    uVar7 = (uint)((ulonglong)*(longlong *)(lVar12 + 0x30) >> 0x3d);
    *(int *)(lVar12 + 0x2c) = *(int *)(lVar12 + 0x2c) + 3;
    *(longlong *)(lVar12 + 0x30) = *(longlong *)(lVar12 + 0x30) << 3;
    *piVar10 = iVar6 + 3;
    lVar13 = lVar12;
  }
  if (uVar7 < 5) {
    param_1[0xb] = uVar7;
    for (piVar15 = param_1 + 0xd; lVar12 = local_res10,
        piVar15 != param_1 + (longlong)(int)uVar7 * 2 + 0xd; piVar15 = piVar15 + 2) {
      iVar6 = *piVar10;
      if (0x40 - iVar6 < 5) {
        uVar8 = FUN_1406d6c7c(local_res10,5);
        uVar11 = (ulonglong)uVar8;
      }
      else {
        uVar11 = *(ulonglong *)(lVar13 + 0x30) >> 0x3b;
        *(int *)(lVar13 + 0x2c) = *(int *)(lVar13 + 0x2c) + 5;
        *(ulonglong *)(lVar13 + 0x30) = *(ulonglong *)(lVar13 + 0x30) << 5;
        *piVar10 = iVar6 + 5;
        lVar12 = lVar13;
      }
      *(char *)(piVar15 + 1) = (char)uVar11;
      FUN_14080d69c(uVar11,lVar12,piVar15,0xffffffff);
      lVar13 = lVar12;
    }
    bVar2 = true;
  }
  else {
    bVar2 = false;
  }
  FUN_14080d4d0(param_1 + 0x15,lVar13);
  cVar3 = FUN_1406cf008(lVar13);
  *(char *)(param_1 + 7) = cVar3;
  if (cVar3 == '\0') {
    iVar6 = 0;
  }
  else {
    uVar16 = FUN_14080dec4(lVar13);
    FUN_14080d69c(uVar16,lVar13,param_1 + 8,0xffffffff);
    FUN_1406d84b4(lVar13);
    iVar6 = extraout_XMM0_Da;
  }
  param_1[10] = iVar6;
  if (((!bVar2) || (*(int *)(lVar13 + 0x18) * 8 < *(int *)(lVar13 + 0x2c))) ||
     (cVar3 = FUN_1404785a0(param_1 + 5), cVar3 == '\0')) {
    uVar14 = 0;
  }
  return uVar14;
}

