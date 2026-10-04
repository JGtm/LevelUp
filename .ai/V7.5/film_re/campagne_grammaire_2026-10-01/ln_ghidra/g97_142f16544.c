
undefined8 FUN_142f16544(undefined8 param_1,ulonglong param_2,uint *param_3,longlong param_4)

{
  ulonglong uVar1;
  uint uVar2;
  undefined1 uVar3;
  uint uVar4;
  ulonglong uVar5;
  uint uVar6;
  ulonglong *puVar7;
  int iVar8;
  ulonglong uVar9;
  ulonglong uVar10;
  uint uVar11;
  
  uVar10 = 0;
  uVar11 = 0;
  iVar8 = 0x40 - *(int *)(param_4 + 0x38);
  uVar4 = (uint)((ulonglong)*(longlong *)(param_4 + 0x30) >> 0x20);
  if (iVar8 < 6) {
    puVar7 = *(ulonglong **)(param_4 + 0x40);
    uVar6 = 0;
    if (*(ulonglong **)(param_4 + 0x10) < puVar7 + 1) {
      uVar5 = uVar10;
      uVar9 = uVar10;
      if (puVar7 < *(ulonglong **)(param_4 + 0x10)) {
        do {
          uVar6 = (int)uVar5 + 8;
          uVar5 = (ulonglong)uVar6;
          uVar1 = *puVar7;
          puVar7 = (ulonglong *)((longlong)puVar7 + 1);
          uVar9 = (ulonglong)(byte)uVar1 | uVar9 << 8;
          *(ulonglong **)(param_4 + 0x40) = puVar7;
        } while (puVar7 < *(ulonglong **)(param_4 + 0x10));
        uVar9 = uVar9 << (-(char)uVar6 & 0x3fU);
      }
    }
    else {
      uVar5 = *puVar7;
      uVar6 = 0x40;
      *(ulonglong **)(param_4 + 0x40) = puVar7 + 1;
      uVar9 = uVar5 >> 0x38 | (uVar5 & 0xff000000000000) >> 0x28 | (uVar5 & 0xff0000000000) >> 0x18
              | (uVar5 & 0xff00000000) >> 8 | (uVar5 & 0xff000000) << 8 | (uVar5 & 0xff0000) << 0x18
              | (uVar5 & 0xff00) << 0x28 | uVar5 << 0x38;
    }
    *(int *)(param_4 + 0x28) = *(int *)(param_4 + 0x28) + uVar6;
    *(int *)(param_4 + 0x2c) = *(int *)(param_4 + 0x2c) + 6;
    uVar6 = 6 - iVar8;
    param_2 = uVar9 << ((byte)uVar6 & 0x3f);
    *(ulonglong *)(param_4 + 0x30) = -(ulonglong)(uVar6 < 0x40) & param_2;
    *(uint *)(param_4 + 0x38) = uVar6;
    uVar4 = (uint)(uVar9 >> (-(byte)uVar6 & 0x3f)) | uVar4 >> 0x1a;
  }
  else {
    *(int *)(param_4 + 0x2c) = *(int *)(param_4 + 0x2c) + 6;
    *(longlong *)(param_4 + 0x30) = *(longlong *)(param_4 + 0x30) << 6;
    *(int *)(param_4 + 0x38) = *(int *)(param_4 + 0x38) + 6;
    uVar4 = uVar4 >> 0x1a;
  }
  *param_3 = uVar4;
  *(undefined2 *)(param_3 + 1) = 0;
  *(undefined2 *)(param_3 + 0x41) = 0;
  FUN_1407f0094(param_4,param_2,param_3 + 1,0x80);
  FUN_1407f0094(param_4);
  uVar2 = DAT_143b458e8._4_4_;
  uVar6 = (uint)DAT_143b458e8;
  uVar4 = DAT_143b458e0._4_4_;
  param_3[0xc5] = (uint)DAT_143b458e0;
  param_3[0xc6] = uVar4;
  param_3[199] = uVar6;
  param_3[200] = uVar2;
  uVar2 = DAT_143b458e8._4_4_;
  uVar6 = (uint)DAT_143b458e8;
  uVar4 = DAT_143b458e0._4_4_;
  param_3[0xc1] = (uint)DAT_143b458e0;
  param_3[0xc2] = uVar4;
  param_3[0xc3] = uVar6;
  param_3[0xc4] = uVar2;
  iVar8 = FUN_141102ed0(0x61);
  if (iVar8 != 0) {
    FUN_1406d676c(param_4);
    FUN_1406d676c(param_4);
  }
  uVar4 = FUN_141102ed0(0x61);
  if (1 < uVar4) {
    FUN_1406d676c(param_4);
  }
  uVar4 = FUN_141102ed0(0x61);
  if (2 < uVar4) {
    iVar8 = *(int *)(param_4 + 0x38);
    uVar4 = (uint)((ulonglong)*(longlong *)(param_4 + 0x30) >> 0x20);
    if (0x40 - iVar8 < 0x20) {
      puVar7 = *(ulonglong **)(param_4 + 0x40);
      if (*(ulonglong **)(param_4 + 0x10) < puVar7 + 1) {
        uVar5 = uVar10;
        if (puVar7 < *(ulonglong **)(param_4 + 0x10)) {
          do {
            uVar9 = *puVar7;
            uVar11 = (int)uVar10 + 8;
            uVar10 = (ulonglong)uVar11;
            puVar7 = (ulonglong *)((longlong)puVar7 + 1);
            uVar5 = uVar5 << 8 | (ulonglong)(byte)uVar9;
            *(ulonglong **)(param_4 + 0x40) = puVar7;
          } while (puVar7 < *(ulonglong **)(param_4 + 0x10));
          uVar10 = uVar5 << (-(char)uVar11 & 0x3fU);
        }
      }
      else {
        uVar10 = *puVar7;
        uVar11 = 0x40;
        uVar10 = uVar10 >> 0x38 | (uVar10 & 0xff000000000000) >> 0x28 |
                 (uVar10 & 0xff0000000000) >> 0x18 | (uVar10 & 0xff00000000) >> 8 |
                 (uVar10 & 0xff000000) << 8 | (uVar10 & 0xff0000) << 0x18 |
                 (uVar10 & 0xff00) << 0x28 | uVar10 << 0x38;
        *(ulonglong **)(param_4 + 0x40) = puVar7 + 1;
      }
      *(int *)(param_4 + 0x28) = *(int *)(param_4 + 0x28) + uVar11;
      uVar11 = iVar8 - 0x20;
      *(int *)(param_4 + 0x2c) = *(int *)(param_4 + 0x2c) + 0x20;
      uVar5 = -(ulonglong)(uVar11 < 0x40) & uVar10 << ((byte)uVar11 & 0x3f);
      uVar4 = (uint)(uVar10 >> (-(byte)uVar11 & 0x3f)) | uVar4;
    }
    else {
      *(int *)(param_4 + 0x2c) = *(int *)(param_4 + 0x2c) + 0x20;
      uVar5 = *(longlong *)(param_4 + 0x30) << 0x20;
      uVar11 = iVar8 + 0x20;
    }
    *(ulonglong *)(param_4 + 0x30) = uVar5;
    *(uint *)(param_4 + 0x38) = uVar11;
    param_3[0xcd] = uVar4;
  }
  uVar4 = FUN_141102ed0(0x61);
  if (3 < uVar4) {
    uVar3 = FUN_1406cf008(param_4);
    *(undefined1 *)(param_3 + 0xce) = uVar3;
  }
  return 1;
}

