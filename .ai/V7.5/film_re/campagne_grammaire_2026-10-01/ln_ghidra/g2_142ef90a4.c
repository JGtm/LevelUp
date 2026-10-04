
undefined8 FUN_142ef90a4(undefined8 param_1,undefined8 param_2,uint *param_3,longlong param_4)

{
  int iVar1;
  ulonglong uVar2;
  ulonglong uVar3;
  char cVar4;
  int iVar5;
  ulonglong uVar6;
  ulonglong *puVar7;
  byte bVar8;
  uint uVar9;
  ulonglong uVar10;
  uint uVar11;
  
  cVar4 = FUN_1406cf008(param_4);
  uVar10 = 0;
  if (cVar4 == '\0') {
    *param_3 = 0xffffffff;
    iVar5 = FUN_1406d310c(3);
    iVar1 = *(int *)(param_4 + 0x38);
    uVar2 = *(ulonglong *)(param_4 + 0x30);
    bVar8 = (byte)iVar5;
    if (0x40 - iVar1 < iVar5) {
      puVar7 = *(ulonglong **)(param_4 + 0x40);
      uVar6 = uVar10 & 0xffffffff;
      uVar11 = (uint)uVar10;
      if (*(ulonglong **)(param_4 + 0x10) < puVar7 + 1) {
        if (puVar7 < *(ulonglong **)(param_4 + 0x10)) {
          do {
            uVar3 = *puVar7;
            uVar11 = (int)uVar6 + 8;
            uVar6 = (ulonglong)uVar11;
            puVar7 = (ulonglong *)((longlong)puVar7 + 1);
            uVar10 = uVar10 << 8 | (ulonglong)(byte)uVar3;
            *(ulonglong **)(param_4 + 0x40) = puVar7;
          } while (puVar7 < *(ulonglong **)(param_4 + 0x10));
          uVar10 = uVar10 << (-(char)uVar11 & 0x3fU);
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
      *(int *)(param_4 + 0x2c) = *(int *)(param_4 + 0x2c) + iVar5;
      uVar11 = iVar1 + -0x40 + iVar5;
      uVar6 = -(ulonglong)(uVar11 < 0x40) & uVar10 << ((byte)uVar11 & 0x3f);
      *(ulonglong *)(param_4 + 0x30) = uVar6;
      *(uint *)(param_4 + 0x38) = uVar11;
      uVar11 = (uint)(uVar2 >> (-bVar8 & 0x3f)) | (uint)(uVar10 >> (-(byte)uVar11 & 0x3f));
    }
    else {
      *(int *)(param_4 + 0x2c) = *(int *)(param_4 + 0x2c) + iVar5;
      *(ulonglong *)(param_4 + 0x30) = uVar2 << (bVar8 & 0x3f);
      uVar6 = (ulonglong)(uint)(iVar1 + iVar5);
      uVar11 = (uint)(uVar2 >> (-bVar8 & 0x3f));
      *(int *)(param_4 + 0x38) = iVar1 + iVar5;
    }
    param_3[1] = uVar11;
  }
  else {
    iVar5 = FUN_1406d310c(0x20);
    iVar1 = *(int *)(param_4 + 0x38);
    uVar11 = (uint)uVar10;
    uVar9 = uVar11 + 0x40;
    uVar2 = *(ulonglong *)(param_4 + 0x30);
    bVar8 = (byte)iVar5;
    cVar4 = (char)uVar9;
    if ((int)(uVar9 - iVar1) < iVar5) {
      puVar7 = *(ulonglong **)(param_4 + 0x40);
      uVar6 = uVar10 & 0xffffffff;
      if (*(ulonglong **)(param_4 + 0x10) < puVar7 + 1) {
        if (puVar7 < *(ulonglong **)(param_4 + 0x10)) {
          do {
            uVar3 = *puVar7;
            uVar11 = (int)uVar6 + 8;
            uVar6 = (ulonglong)uVar11;
            puVar7 = (ulonglong *)((longlong)puVar7 + 1);
            uVar10 = uVar10 << 8 | (ulonglong)(byte)uVar3;
            *(ulonglong **)(param_4 + 0x40) = puVar7;
          } while (puVar7 < *(ulonglong **)(param_4 + 0x10));
          uVar10 = uVar10 << (cVar4 - (char)uVar11 & 0x3fU);
        }
      }
      else {
        uVar10 = *puVar7;
        uVar10 = uVar10 >> 0x38 | (uVar10 & 0xff000000000000) >> 0x28 |
                 (uVar10 & 0xff0000000000) >> 0x18 | (uVar10 & 0xff00000000) >> 8 |
                 (uVar10 & 0xff000000) << 8 | (uVar10 & 0xff0000) << 0x18 |
                 (uVar10 & 0xff00) << 0x28 | uVar10 << 0x38;
        *(ulonglong **)(param_4 + 0x40) = puVar7 + 1;
        uVar11 = uVar9;
      }
      *(int *)(param_4 + 0x28) = *(int *)(param_4 + 0x28) + uVar11;
      *(int *)(param_4 + 0x2c) = *(int *)(param_4 + 0x2c) + iVar5;
      iVar5 = iVar1 + -0x40 + iVar5;
      uVar6 = -(ulonglong)((ulonglong)(longlong)iVar5 < (ulonglong)uVar9) &
              uVar10 << ((byte)iVar5 & 0x3f);
      *(ulonglong *)(param_4 + 0x30) = uVar6;
      *(int *)(param_4 + 0x38) = iVar5;
      uVar11 = (uint)(uVar2 >> (cVar4 - bVar8 & 0x3f)) |
               (uint)(uVar10 >> (cVar4 - (byte)iVar5 & 0x3f));
    }
    else {
      *(int *)(param_4 + 0x2c) = *(int *)(param_4 + 0x2c) + iVar5;
      *(ulonglong *)(param_4 + 0x30) = uVar2 << (bVar8 & 0x3f);
      uVar6 = (ulonglong)(uint)(iVar1 + iVar5);
      uVar11 = (uint)(uVar2 >> (cVar4 - bVar8 & 0x3f));
      *(int *)(param_4 + 0x38) = iVar1 + iVar5;
    }
    *param_3 = uVar11;
  }
  return CONCAT71((int7)(uVar6 >> 8),1);
}

