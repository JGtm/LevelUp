
undefined8 FUN_14080bb4c(undefined8 param_1,undefined8 param_2,uint *param_3,longlong param_4)

{
  ulonglong uVar1;
  char cVar2;
  ulonglong uVar3;
  uint uVar4;
  ulonglong *puVar5;
  int iVar6;
  uint *puVar7;
  uint uVar8;
  ulonglong uVar9;
  ulonglong uVar10;
  uint uVar11;
  
  puVar7 = param_3;
  cVar2 = FUN_1404f25f4();
  uVar9 = 0;
  uVar11 = 0;
  if (cVar2 == '\0') {
    *(undefined2 *)(puVar7 + 0x1a) = 0xffff;
  }
  else {
    FUN_14080bd28(puVar7 + 0x1a,param_4);
  }
  iVar6 = 0x40 - *(int *)(param_4 + 0x38);
  uVar8 = (uint)((ulonglong)*(longlong *)(param_4 + 0x30) >> 0x20);
  if (iVar6 < 0xd) {
    puVar5 = *(ulonglong **)(param_4 + 0x40);
    uVar4 = 0;
    if (*(ulonglong **)(param_4 + 0x10) < puVar5 + 1) {
      uVar3 = uVar9;
      uVar10 = uVar9;
      if (puVar5 < *(ulonglong **)(param_4 + 0x10)) {
        do {
          uVar4 = (int)uVar3 + 8;
          uVar3 = (ulonglong)uVar4;
          uVar1 = *puVar5;
          puVar5 = (ulonglong *)((longlong)puVar5 + 1);
          uVar10 = (ulonglong)(byte)uVar1 | uVar10 << 8;
          *(ulonglong **)(param_4 + 0x40) = puVar5;
        } while (puVar5 < *(ulonglong **)(param_4 + 0x10));
        uVar10 = uVar10 << (-(char)uVar4 & 0x3fU);
      }
    }
    else {
      uVar3 = *puVar5;
      uVar4 = 0x40;
      *(ulonglong **)(param_4 + 0x40) = puVar5 + 1;
      uVar10 = uVar3 >> 0x38 | (uVar3 & 0xff000000000000) >> 0x28 | (uVar3 & 0xff0000000000) >> 0x18
               | (uVar3 & 0xff00000000) >> 8 | (uVar3 & 0xff000000) << 8 |
               (uVar3 & 0xff0000) << 0x18 | (uVar3 & 0xff00) << 0x28 | uVar3 << 0x38;
    }
    *(int *)(param_4 + 0x28) = *(int *)(param_4 + 0x28) + uVar4;
    *(int *)(param_4 + 0x2c) = *(int *)(param_4 + 0x2c) + 0xd;
    uVar4 = 0xd - iVar6;
    *(ulonglong *)(param_4 + 0x30) = -(ulonglong)(uVar4 < 0x40) & uVar10 << ((byte)uVar4 & 0x3f);
    *(uint *)(param_4 + 0x38) = uVar4;
    uVar8 = (uint)(uVar10 >> (-(byte)uVar4 & 0x3f)) | uVar8 >> 0x13;
  }
  else {
    *(int *)(param_4 + 0x2c) = *(int *)(param_4 + 0x2c) + 0xd;
    *(longlong *)(param_4 + 0x30) = *(longlong *)(param_4 + 0x30) << 0xd;
    *(int *)(param_4 + 0x38) = *(int *)(param_4 + 0x38) + 0xd;
    uVar8 = uVar8 >> 0x13;
  }
  *param_3 = uVar8;
  iVar6 = *(int *)(param_4 + 0x38);
  uVar8 = (uint)((ulonglong)*(longlong *)(param_4 + 0x30) >> 0x20);
  if (0x40 - iVar6 < 10) {
    puVar5 = *(ulonglong **)(param_4 + 0x40);
    if (*(ulonglong **)(param_4 + 0x10) < puVar5 + 1) {
      uVar3 = uVar9;
      if (puVar5 < *(ulonglong **)(param_4 + 0x10)) {
        do {
          uVar10 = *puVar5;
          uVar11 = (int)uVar9 + 8;
          uVar9 = (ulonglong)uVar11;
          puVar5 = (ulonglong *)((longlong)puVar5 + 1);
          uVar3 = uVar3 << 8 | (ulonglong)(byte)uVar10;
          *(ulonglong **)(param_4 + 0x40) = puVar5;
        } while (puVar5 < *(ulonglong **)(param_4 + 0x10));
        uVar9 = uVar3 << (-(char)uVar11 & 0x3fU);
      }
    }
    else {
      uVar9 = *puVar5;
      uVar11 = 0x40;
      uVar9 = uVar9 >> 0x38 | (uVar9 & 0xff000000000000) >> 0x28 | (uVar9 & 0xff0000000000) >> 0x18
              | (uVar9 & 0xff00000000) >> 8 | (uVar9 & 0xff000000) << 8 | (uVar9 & 0xff0000) << 0x18
              | (uVar9 & 0xff00) << 0x28 | uVar9 << 0x38;
      *(ulonglong **)(param_4 + 0x40) = puVar5 + 1;
    }
    *(int *)(param_4 + 0x28) = *(int *)(param_4 + 0x28) + uVar11;
    uVar11 = iVar6 - 0x36;
    *(int *)(param_4 + 0x2c) = *(int *)(param_4 + 0x2c) + 10;
    uVar3 = -(ulonglong)(uVar11 < 0x40) & uVar9 << ((byte)uVar11 & 0x3f);
    uVar8 = (uint)(uVar9 >> (-(byte)uVar11 & 0x3f)) | uVar8 >> 0x16;
  }
  else {
    *(int *)(param_4 + 0x2c) = *(int *)(param_4 + 0x2c) + 10;
    uVar3 = *(longlong *)(param_4 + 0x30) << 10;
    uVar11 = iVar6 + 10;
    uVar8 = uVar8 >> 0x16;
  }
  *(ulonglong *)(param_4 + 0x30) = uVar3;
  *(uint *)(param_4 + 0x38) = uVar11;
  param_3[1] = uVar8;
  FUN_1406d676c(param_4);
  return 1;
}

