
undefined1 FUN_142f18284(undefined8 param_1,undefined8 param_2,ushort *param_3,longlong param_4)

{
  int iVar1;
  ulonglong uVar2;
  char cVar3;
  ushort uVar4;
  undefined4 uVar5;
  ulonglong *puVar6;
  undefined1 uVar7;
  uint uVar8;
  ulonglong uVar9;
  ulonglong uVar10;
  
  iVar1 = *(int *)(param_4 + 0x38);
  uVar9 = 0;
  uVar7 = 1;
  uVar4 = (ushort)((ulonglong)*(longlong *)(param_4 + 0x30) >> 0x30);
  if (0x40 - iVar1 < 1) {
    puVar6 = *(ulonglong **)(param_4 + 0x40);
    uVar8 = 0;
    if (*(ulonglong **)(param_4 + 0x10) < puVar6 + 1) {
      uVar10 = uVar9;
      if (puVar6 < *(ulonglong **)(param_4 + 0x10)) {
        do {
          uVar2 = *puVar6;
          uVar8 = (int)uVar9 + 8;
          uVar9 = (ulonglong)uVar8;
          puVar6 = (ulonglong *)((longlong)puVar6 + 1);
          uVar10 = uVar10 << 8 | (ulonglong)(byte)uVar2;
          *(ulonglong **)(param_4 + 0x40) = puVar6;
        } while (puVar6 < *(ulonglong **)(param_4 + 0x10));
        uVar9 = uVar10 << (-(char)uVar8 & 0x3fU);
      }
    }
    else {
      uVar9 = *puVar6;
      uVar8 = 0x40;
      uVar9 = uVar9 >> 0x38 | (uVar9 & 0xff000000000000) >> 0x28 | (uVar9 & 0xff0000000000) >> 0x18
              | (uVar9 & 0xff00000000) >> 8 | (uVar9 & 0xff000000) << 8 | (uVar9 & 0xff0000) << 0x18
              | (uVar9 & 0xff00) << 0x28 | uVar9 << 0x38;
      *(ulonglong **)(param_4 + 0x40) = puVar6 + 1;
    }
    *(int *)(param_4 + 0x28) = *(int *)(param_4 + 0x28) + uVar8;
    uVar8 = iVar1 - 0x3f;
    *(int *)(param_4 + 0x2c) = *(int *)(param_4 + 0x2c) + 1;
    uVar4 = (ushort)(uVar9 >> (-(byte)uVar8 & 0x3f)) | uVar4 >> 0xf;
    *(ulonglong *)(param_4 + 0x30) = -(ulonglong)(uVar8 < 0x40) & uVar9 << ((byte)uVar8 & 0x3f);
  }
  else {
    *(int *)(param_4 + 0x2c) = *(int *)(param_4 + 0x2c) + 1;
    *(longlong *)(param_4 + 0x30) = *(longlong *)(param_4 + 0x30) * 2;
    uVar8 = iVar1 + 1;
    uVar4 = uVar4 >> 0xf;
  }
  *(uint *)(param_4 + 0x38) = uVar8;
  *param_3 = uVar4;
  uVar4 = FUN_1406d00ec(param_4);
  uVar5 = 0xffffffff;
  param_3[1] = uVar4;
  FUN_14080d69c();
  FUN_14080dec4(param_4,"variant_name",param_3 + 4);
  cVar3 = FUN_1405838f0(param_3 + 2);
  if (cVar3 != '\0') {
    uVar5 = FUN_142e2de9c(param_3 + 2,*(undefined4 *)(param_3 + 4));
  }
  *(undefined4 *)(param_3 + 6) = uVar5;
  if ((((short)*param_3 < 0) || (1 < *param_3)) || (3 < param_3[1])) {
    uVar7 = 0;
  }
  return uVar7;
}

