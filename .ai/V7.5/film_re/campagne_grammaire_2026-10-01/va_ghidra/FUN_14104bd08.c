undefined8 FUN_14104bd08(undefined8 param_1,undefined8 param_2,undefined4 *param_3,longlong param_4)
{
  char cVar1;
  undefined4 uVar2;
  undefined4 uVar3;
  longlong lVar4;
  ulonglong uVar5;
  ulonglong *puVar6;
  int iVar7;
  int iVar8;
  uint uVar9;
  ulonglong uVar10;
  uint uVar11;
  bool bVar12;
  undefined4 local_res18 [4];
  uVar2 = FUN_1407f2058(param_4);
  lVar4 = FUN_14049746c(uVar2);
  uVar3 = 0xffffffff;
  uVar2 = uVar3;
  if (lVar4 != 0) {
    FUN_140e958c4(local_res18);
    uVar2 = local_res18[0];
  }
  *param_3 = uVar2;
  uVar2 = FUN_1407f2058(param_4);
  lVar4 = FUN_14049746c(uVar2);
  if (lVar4 != 0) {
    FUN_140e958c4(local_res18);
    uVar3 = local_res18[0];
  }
  param_3[1] = uVar3;
  iVar8 = 0x40 - *(int *)(param_4 + 0x38);
  uVar9 = (uint)((ulonglong)*(longlong *)(param_4 + 0x30) >> 0x20);
  if (iVar8 < 0x20) {
    puVar6 = *(ulonglong **)(param_4 + 0x40);
    uVar10 = 0;
    iVar7 = 0;
    if (*(ulonglong **)(param_4 + 0x10) < puVar6 + 1) {
      if (puVar6 < *(ulonglong **)(param_4 + 0x10)) {
        do {
          iVar7 = iVar7 + 8;
          uVar5 = *puVar6;
          puVar6 = (ulonglong *)((longlong)puVar6 + 1);
          uVar10 = (ulonglong)(byte)uVar5 | uVar10 << 8;
          *(ulonglong **)(param_4 + 0x40) = puVar6;
        } while (puVar6 < *(ulonglong **)(param_4 + 0x10));
        uVar10 = uVar10 << (-(char)iVar7 & 0x3fU);
      }
    }
    else {
      uVar10 = *puVar6;
      iVar7 = 0x40;
      uVar10 = uVar10 >> 0x38 | (uVar10 & 0xff000000000000) >> 0x28 |
               (uVar10 & 0xff0000000000) >> 0x18 | (uVar10 & 0xff00000000) >> 8 |
               (uVar10 & 0xff000000) << 8 | (uVar10 & 0xff0000) << 0x18 | (uVar10 & 0xff00) << 0x28
               | uVar10 << 0x38;
      *(ulonglong **)(param_4 + 0x40) = puVar6 + 1;
    }
    *(int *)(param_4 + 0x28) = *(int *)(param_4 + 0x28) + iVar7;
    *(int *)(param_4 + 0x2c) = *(int *)(param_4 + 0x2c) + 0x20;
    uVar11 = 0x20 - iVar8;
    *(ulonglong *)(param_4 + 0x30) = -(ulonglong)(uVar11 < 0x40) & uVar10 << ((byte)uVar11 & 0x3f);
    *(uint *)(param_4 + 0x38) = uVar11;
    uVar9 = (uint)(uVar10 >> (-(byte)uVar11 & 0x3f)) | uVar9;
  }
  else {
    *(int *)(param_4 + 0x2c) = *(int *)(param_4 + 0x2c) + 0x20;
    *(longlong *)(param_4 + 0x30) = *(longlong *)(param_4 + 0x30) << 0x20;
    *(int *)(param_4 + 0x38) = *(int *)(param_4 + 0x38) + 0x20;
  }
  param_3[2] = uVar9;
  if (*(uint *)(param_4 + 0x38) < 0x40) {
    *(int *)(param_4 + 0x2c) = *(int *)(param_4 + 0x2c) + 1;
    bVar12 = SUB81((ulonglong)*(longlong *)(param_4 + 0x30) >> 0x3f,0);
    *(longlong *)(param_4 + 0x30) = *(longlong *)(param_4 + 0x30) * 2;
    *(uint *)(param_4 + 0x38) = *(uint *)(param_4 + 0x38) + 1;
  }
  else {
    lVar4 = FUN_1406d6c7c(param_4);
    bVar12 = lVar4 != 0;
  }
  *(bool *)(param_3 + 3) = bVar12;
  uVar2 = FUN_1407f2058(param_4);
  lVar4 = FUN_14049746c(uVar2);
  uVar2 = 0xffffffff;
  if (lVar4 != 0) {
    FUN_140e958c4(local_res18);
    uVar2 = local_res18[0];
  }
  param_3[4] = uVar2;
  iVar8 = *(int *)(param_4 + 0x38);
  uVar9 = (uint)((ulonglong)*(longlong *)(param_4 + 0x30) >> 0x20);
  if (0x40 - iVar8 < 0x20) {
    puVar6 = *(ulonglong **)(param_4 + 0x40);
    uVar10 = 0;
    iVar7 = 0;
    if (*(ulonglong **)(param_4 + 0x10) < puVar6 + 1) {
      if (puVar6 < *(ulonglong **)(param_4 + 0x10)) {
        do {
          uVar5 = *puVar6;
          iVar7 = iVar7 + 8;
          puVar6 = (ulonglong *)((longlong)puVar6 + 1);
          uVar10 = uVar10 << 8 | (ulonglong)(byte)uVar5;
          *(ulonglong **)(param_4 + 0x40) = puVar6;
        } while (puVar6 < *(ulonglong **)(param_4 + 0x10));
        uVar10 = uVar10 << (-(char)iVar7 & 0x3fU);
      }
    }
    else {
      uVar10 = *puVar6;
      iVar7 = 0x40;
      uVar10 = uVar10 >> 0x38 | (uVar10 & 0xff000000000000) >> 0x28 |
               (uVar10 & 0xff0000000000) >> 0x18 | (uVar10 & 0xff00000000) >> 8 |
               (uVar10 & 0xff000000) << 8 | (uVar10 & 0xff0000) << 0x18 | (uVar10 & 0xff00) << 0x28
               | uVar10 << 0x38;
      *(ulonglong **)(param_4 + 0x40) = puVar6 + 1;
    }
    *(int *)(param_4 + 0x28) = *(int *)(param_4 + 0x28) + iVar7;
    uVar11 = iVar8 - 0x20;
    *(int *)(param_4 + 0x2c) = *(int *)(param_4 + 0x2c) + 0x20;
    uVar5 = -(ulonglong)(uVar11 < 0x40) & uVar10 << ((byte)uVar11 & 0x3f);
    uVar9 = (uint)(uVar10 >> (-(byte)uVar11 & 0x3f)) | uVar9;
  }
  else {
    *(int *)(param_4 + 0x2c) = *(int *)(param_4 + 0x2c) + 0x20;
    uVar5 = *(longlong *)(param_4 + 0x30) << 0x20;
    uVar11 = iVar8 + 0x20;
  }
  *(ulonglong *)(param_4 + 0x30) = uVar5;
  *(uint *)(param_4 + 0x38) = uVar11;
  param_3[5] = uVar9;
  if (((DAT_1451789b8 != '\0') && (cVar1 = FUN_1406aed00(), cVar1 != '\0')) &&
     (DAT_145121140 != '\x01')) {
    lVar4 = FUN_1404f1614();
    lVar4 = FUN_1406aed80(lVar4 + 0x28);
    if (*(char *)(lVar4 + 0x238) != '\0') goto LAB_142489ad8;
  }
  if (DAT_145178a48 == '\0') {
    return 1;
  }
  cVar1 = FUN_1406aed00();
  if (cVar1 == '\0') {
    return 1;
  }
  lVar4 = FUN_1404f1614();
  lVar4 = FUN_1406aed80(lVar4 + 0x28);
  if (*(char *)(lVar4 + 0x240) == '\0') {
    return 1;
  }
LAB_142489ad8:
  FUN_1431eb378(param_3 + 6,param_4);
  return 1;
}
