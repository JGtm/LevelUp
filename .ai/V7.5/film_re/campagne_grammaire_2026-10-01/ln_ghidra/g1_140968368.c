
undefined8 FUN_140968368(undefined8 param_1,undefined8 param_2,uint *param_3,longlong param_4)

{
  int iVar1;
  uint uVar2;
  ulonglong uVar3;
  longlong lVar4;
  ulonglong *puVar5;
  int iVar6;
  ulonglong uVar7;
  uint uVar8;
  bool bVar9;
  undefined8 local_18;
  uint local_10;
  
  iVar1 = *(int *)(param_4 + 0x38);
  uVar2 = (uint)((ulonglong)*(longlong *)(param_4 + 0x30) >> 0x20);
  if (0x40 - iVar1 < 5) {
    puVar5 = *(ulonglong **)(param_4 + 0x40);
    uVar7 = 0;
    iVar6 = 0;
    if (*(ulonglong **)(param_4 + 0x10) < puVar5 + 1) {
      if (puVar5 < *(ulonglong **)(param_4 + 0x10)) {
        do {
          uVar3 = *puVar5;
          iVar6 = iVar6 + 8;
          puVar5 = (ulonglong *)((longlong)puVar5 + 1);
          uVar7 = uVar7 << 8 | (ulonglong)(byte)uVar3;
          *(ulonglong **)(param_4 + 0x40) = puVar5;
        } while (puVar5 < *(ulonglong **)(param_4 + 0x10));
        uVar7 = uVar7 << (0x40U - (char)iVar6 & 0x3f);
      }
    }
    else {
      uVar7 = *puVar5;
      iVar6 = 0x40;
      uVar7 = uVar7 >> 0x38 | (uVar7 & 0xff000000000000) >> 0x28 | (uVar7 & 0xff0000000000) >> 0x18
              | (uVar7 & 0xff00000000) >> 8 | (uVar7 & 0xff000000) << 8 | (uVar7 & 0xff0000) << 0x18
              | (uVar7 & 0xff00) << 0x28 | uVar7 << 0x38;
      *(ulonglong **)(param_4 + 0x40) = puVar5 + 1;
    }
    *(int *)(param_4 + 0x28) = *(int *)(param_4 + 0x28) + iVar6;
    uVar8 = iVar1 - 0x3b;
    *(int *)(param_4 + 0x2c) = *(int *)(param_4 + 0x2c) + 5;
    uVar3 = -(ulonglong)(uVar8 < 0x40) & uVar7 << ((byte)uVar8 & 0x3f);
    uVar2 = (uint)(uVar7 >> (0x40 - (byte)uVar8 & 0x3f)) | uVar2 >> 0x1b;
  }
  else {
    *(int *)(param_4 + 0x2c) = *(int *)(param_4 + 0x2c) + 5;
    uVar3 = *(longlong *)(param_4 + 0x30) << 5;
    uVar8 = iVar1 + 5;
    uVar2 = uVar2 >> 0x1b;
  }
  *(ulonglong *)(param_4 + 0x30) = uVar3;
  *(uint *)(param_4 + 0x38) = uVar8;
  *param_3 = uVar2;
  uVar2 = FUN_1409684dc(param_4);
  param_3[1] = uVar2;
  uVar2 = FUN_1424d0f48(param_4);
  param_3[2] = uVar2;
  if (*(uint *)(param_4 + 0x38) < 0x40) {
    *(int *)(param_4 + 0x2c) = *(int *)(param_4 + 0x2c) + 1;
    bVar9 = SUB81((ulonglong)*(longlong *)(param_4 + 0x30) >> 0x3f,0);
    *(longlong *)(param_4 + 0x30) = *(longlong *)(param_4 + 0x30) * 2;
    *(uint *)(param_4 + 0x38) = *(uint *)(param_4 + 0x38) + 1;
  }
  else {
    lVar4 = FUN_1406d6c7c(param_4,1);
    bVar9 = lVar4 != 0;
  }
  *(bool *)(param_3 + 6) = bVar9;
  if (bVar9 != false) {
    FUN_14076dc04(param_4);
    param_3[3] = (uint)local_18;
    param_3[4] = (uint)((ulonglong)local_18 >> 0x20);
    param_3[5] = local_10;
  }
  return 1;
}

