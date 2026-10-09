
undefined1 FUN_142f17d74(undefined8 param_1,undefined8 param_2,ushort *param_3,longlong param_4)

{
  int iVar1;
  ulonglong uVar2;
  undefined1 uVar3;
  char cVar4;
  ushort uVar5;
  undefined4 uVar6;
  ulonglong *puVar7;
  undefined1 uVar8;
  uint uVar9;
  ulonglong uVar10;
  ulonglong uVar11;
  
  iVar1 = *(int *)(param_4 + 0x38);
  uVar10 = 0;
  uVar8 = 1;
  uVar5 = (ushort)((ulonglong)*(longlong *)(param_4 + 0x30) >> 0x30);
  if (0x40 - iVar1 < 1) {
    puVar7 = *(ulonglong **)(param_4 + 0x40);
    uVar9 = 0;
    if (*(ulonglong **)(param_4 + 0x10) < puVar7 + 1) {
      uVar11 = uVar10;
      if (puVar7 < *(ulonglong **)(param_4 + 0x10)) {
        do {
          uVar2 = *puVar7;
          uVar9 = (int)uVar10 + 8;
          uVar10 = (ulonglong)uVar9;
          puVar7 = (ulonglong *)((longlong)puVar7 + 1);
          uVar11 = uVar11 << 8 | (ulonglong)(byte)uVar2;
          *(ulonglong **)(param_4 + 0x40) = puVar7;
        } while (puVar7 < *(ulonglong **)(param_4 + 0x10));
        uVar10 = uVar11 << (-(char)uVar9 & 0x3fU);
      }
    }
    else {
      uVar10 = *puVar7;
      uVar9 = 0x40;
      uVar10 = uVar10 >> 0x38 | (uVar10 & 0xff000000000000) >> 0x28 |
               (uVar10 & 0xff0000000000) >> 0x18 | (uVar10 & 0xff00000000) >> 8 |
               (uVar10 & 0xff000000) << 8 | (uVar10 & 0xff0000) << 0x18 | (uVar10 & 0xff00) << 0x28
               | uVar10 << 0x38;
      *(ulonglong **)(param_4 + 0x40) = puVar7 + 1;
    }
    *(int *)(param_4 + 0x28) = *(int *)(param_4 + 0x28) + uVar9;
    uVar9 = iVar1 - 0x3f;
    *(int *)(param_4 + 0x2c) = *(int *)(param_4 + 0x2c) + 1;
    uVar5 = (ushort)(uVar10 >> (-(byte)uVar9 & 0x3f)) | uVar5 >> 0xf;
    *(ulonglong *)(param_4 + 0x30) = -(ulonglong)(uVar9 < 0x40) & uVar10 << ((byte)uVar9 & 0x3f);
  }
  else {
    *(int *)(param_4 + 0x2c) = *(int *)(param_4 + 0x2c) + 1;
    *(longlong *)(param_4 + 0x30) = *(longlong *)(param_4 + 0x30) * 2;
    uVar9 = iVar1 + 1;
    uVar5 = uVar5 >> 0xf;
  }
  *(uint *)(param_4 + 0x38) = uVar9;
  *param_3 = uVar5;
  uVar5 = FUN_1406d00ec(param_4);
  uVar6 = 0xffffffff;
  param_3[1] = uVar5;
  FUN_14080d69c();
  FUN_14080dec4(param_4,"variant_name",param_3 + 4);
  uVar3 = FUN_1406cf008(param_4);
  *(undefined1 *)(param_3 + 8) = uVar3;
  cVar4 = FUN_1405838f0(param_3 + 2);
  if (cVar4 != '\0') {
    uVar6 = FUN_142e2de9c(param_3 + 2,*(undefined4 *)(param_3 + 4));
  }
  *(undefined4 *)(param_3 + 6) = uVar6;
  if ((((short)*param_3 < 0) || (1 < *param_3)) || (3 < param_3[1])) {
    uVar8 = 0;
  }
  return uVar8;
}

