
undefined8 FUN_142f17178(undefined8 param_1,undefined8 param_2,uint *param_3,longlong param_4)

{
  ulonglong uVar1;
  char cVar2;
  undefined1 uVar3;
  ulonglong uVar4;
  uint uVar5;
  ulonglong *puVar6;
  uint uVar7;
  int iVar8;
  ushort uVar9;
  ulonglong uVar10;
  ulonglong uVar11;
  uint uVar12;
  
  uVar11 = 0;
  uVar7 = 0;
  iVar8 = 0x40 - *(int *)(param_4 + 0x38);
  uVar12 = (uint)((ulonglong)*(longlong *)(param_4 + 0x30) >> 0x20);
  if (iVar8 < 6) {
    puVar6 = *(ulonglong **)(param_4 + 0x40);
    uVar5 = 0;
    if (*(ulonglong **)(param_4 + 0x10) < puVar6 + 1) {
      uVar4 = uVar11;
      uVar10 = uVar11;
      if (puVar6 < *(ulonglong **)(param_4 + 0x10)) {
        do {
          uVar5 = (int)uVar4 + 8;
          uVar4 = (ulonglong)uVar5;
          uVar1 = *puVar6;
          puVar6 = (ulonglong *)((longlong)puVar6 + 1);
          uVar10 = (ulonglong)(byte)uVar1 | uVar10 << 8;
          *(ulonglong **)(param_4 + 0x40) = puVar6;
        } while (puVar6 < *(ulonglong **)(param_4 + 0x10));
        uVar10 = uVar10 << (-(char)uVar5 & 0x3fU);
      }
    }
    else {
      uVar4 = *puVar6;
      uVar5 = 0x40;
      *(ulonglong **)(param_4 + 0x40) = puVar6 + 1;
      uVar10 = uVar4 >> 0x38 | (uVar4 & 0xff000000000000) >> 0x28 | (uVar4 & 0xff0000000000) >> 0x18
               | (uVar4 & 0xff00000000) >> 8 | (uVar4 & 0xff000000) << 8 |
               (uVar4 & 0xff0000) << 0x18 | (uVar4 & 0xff00) << 0x28 | uVar4 << 0x38;
    }
    *(int *)(param_4 + 0x28) = *(int *)(param_4 + 0x28) + uVar5;
    *(int *)(param_4 + 0x2c) = *(int *)(param_4 + 0x2c) + 6;
    uVar5 = 6 - iVar8;
    *(ulonglong *)(param_4 + 0x30) = -(ulonglong)(uVar5 < 0x40) & uVar10 << ((byte)uVar5 & 0x3f);
    *(uint *)(param_4 + 0x38) = uVar5;
    uVar12 = (uint)(uVar10 >> (-(byte)uVar5 & 0x3f)) | uVar12 >> 0x1a;
  }
  else {
    *(int *)(param_4 + 0x2c) = *(int *)(param_4 + 0x2c) + 6;
    *(longlong *)(param_4 + 0x30) = *(longlong *)(param_4 + 0x30) << 6;
    *(int *)(param_4 + 0x38) = *(int *)(param_4 + 0x38) + 6;
    uVar12 = uVar12 >> 0x1a;
  }
  *param_3 = uVar12;
  cVar2 = FUN_1406cf008(param_4);
  if (cVar2 == '\0') {
    uVar9 = 0xffff;
  }
  else {
    iVar8 = *(int *)(param_4 + 0x38);
    uVar9 = (ushort)((ulonglong)*(longlong *)(param_4 + 0x30) >> 0x30);
    if (0x40 - iVar8 < 7) {
      puVar6 = *(ulonglong **)(param_4 + 0x40);
      if (*(ulonglong **)(param_4 + 0x10) < puVar6 + 1) {
        uVar4 = uVar11;
        if (puVar6 < *(ulonglong **)(param_4 + 0x10)) {
          do {
            uVar10 = *puVar6;
            uVar7 = (int)uVar11 + 8;
            uVar11 = (ulonglong)uVar7;
            puVar6 = (ulonglong *)((longlong)puVar6 + 1);
            uVar4 = uVar4 << 8 | (ulonglong)(byte)uVar10;
            *(ulonglong **)(param_4 + 0x40) = puVar6;
          } while (puVar6 < *(ulonglong **)(param_4 + 0x10));
          uVar11 = uVar4 << (-(char)uVar7 & 0x3fU);
        }
      }
      else {
        uVar11 = *puVar6;
        uVar7 = 0x40;
        uVar11 = uVar11 >> 0x38 | (uVar11 & 0xff000000000000) >> 0x28 |
                 (uVar11 & 0xff0000000000) >> 0x18 | (uVar11 & 0xff00000000) >> 8 |
                 (uVar11 & 0xff000000) << 8 | (uVar11 & 0xff0000) << 0x18 |
                 (uVar11 & 0xff00) << 0x28 | uVar11 << 0x38;
        *(ulonglong **)(param_4 + 0x40) = puVar6 + 1;
      }
      *(int *)(param_4 + 0x28) = *(int *)(param_4 + 0x28) + uVar7;
      uVar12 = iVar8 - 0x39;
      *(int *)(param_4 + 0x2c) = *(int *)(param_4 + 0x2c) + 7;
      uVar4 = -(ulonglong)(uVar12 < 0x40) & uVar11 << ((byte)uVar12 & 0x3f);
      uVar9 = (ushort)(uVar11 >> (-(byte)uVar12 & 0x3f)) | uVar9 >> 9;
    }
    else {
      *(int *)(param_4 + 0x2c) = *(int *)(param_4 + 0x2c) + 7;
      uVar4 = *(longlong *)(param_4 + 0x30) << 7;
      uVar12 = iVar8 + 7;
      uVar9 = uVar9 >> 9;
    }
    *(ulonglong *)(param_4 + 0x30) = uVar4;
    *(uint *)(param_4 + 0x38) = uVar12;
  }
  *(ushort *)(param_3 + 2) = uVar9;
  uVar12 = FUN_1407f2058(param_4);
  param_3[1] = uVar12;
  uVar3 = FUN_1406cf008(param_4);
  *(undefined1 *)((longlong)param_3 + 10) = uVar3;
  return 1;
}

