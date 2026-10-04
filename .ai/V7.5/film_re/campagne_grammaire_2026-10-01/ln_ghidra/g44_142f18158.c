
undefined8 FUN_142f18158(undefined8 param_1,undefined8 param_2,undefined1 *param_3,longlong param_4)

{
  int iVar1;
  undefined1 uVar2;
  ulonglong uVar3;
  ulonglong *puVar4;
  int iVar5;
  uint uVar6;
  ulonglong uVar7;
  uint uVar8;
  
  uVar2 = FUN_1406d0f20(param_4);
  *param_3 = uVar2;
  uVar2 = FUN_1406d00ec(param_4);
  param_3[1] = uVar2;
  uVar2 = FUN_1406d00ec(param_4);
  param_3[2] = uVar2;
  uVar2 = 1;
  iVar1 = *(int *)(param_4 + 0x38);
  uVar6 = (uint)((ulonglong)*(longlong *)(param_4 + 0x30) >> 0x20);
  if (0x40 - iVar1 < 3) {
    puVar4 = *(ulonglong **)(param_4 + 0x40);
    uVar7 = 0;
    iVar5 = 0;
    if (*(ulonglong **)(param_4 + 0x10) < puVar4 + 1) {
      if (puVar4 < *(ulonglong **)(param_4 + 0x10)) {
        do {
          uVar3 = *puVar4;
          iVar5 = iVar5 + 8;
          puVar4 = (ulonglong *)((longlong)puVar4 + 1);
          uVar7 = uVar7 << 8 | (ulonglong)(byte)uVar3;
          *(ulonglong **)(param_4 + 0x40) = puVar4;
        } while (puVar4 < *(ulonglong **)(param_4 + 0x10));
        uVar7 = uVar7 << (0x40U - (char)iVar5 & 0x3f);
      }
    }
    else {
      uVar7 = *puVar4;
      iVar5 = 0x40;
      uVar7 = uVar7 >> 0x38 | (uVar7 & 0xff000000000000) >> 0x28 | (uVar7 & 0xff0000000000) >> 0x18
              | (uVar7 & 0xff00000000) >> 8 | (uVar7 & 0xff000000) << 8 | (uVar7 & 0xff0000) << 0x18
              | (uVar7 & 0xff00) << 0x28 | uVar7 << 0x38;
      *(ulonglong **)(param_4 + 0x40) = puVar4 + 1;
    }
    *(int *)(param_4 + 0x28) = *(int *)(param_4 + 0x28) + iVar5;
    uVar8 = iVar1 - 0x3d;
    *(int *)(param_4 + 0x2c) = *(int *)(param_4 + 0x2c) + 3;
    uVar3 = -(ulonglong)(uVar8 < 0x40) & uVar7 << ((byte)uVar8 & 0x3f);
    uVar6 = (uint)(uVar7 >> (0x40 - (byte)uVar8 & 0x3f)) | uVar6 >> 0x1d;
  }
  else {
    *(int *)(param_4 + 0x2c) = *(int *)(param_4 + 0x2c) + 3;
    uVar3 = *(longlong *)(param_4 + 0x30) * 8;
    uVar8 = iVar1 + 3;
    uVar6 = uVar6 >> 0x1d;
  }
  *(ulonglong *)(param_4 + 0x30) = uVar3;
  *(uint *)(param_4 + 0x38) = uVar8;
  *(uint *)(param_3 + 4) = uVar6;
  if ((((param_3[1] != 0xff) && (3 < (byte)param_3[1])) ||
      ((param_3[2] != 0xff && (3 < (byte)param_3[2])))) || (8 < uVar6)) {
    uVar2 = 0;
  }
  return CONCAT71((int7)(uVar3 >> 8),uVar2);
}

