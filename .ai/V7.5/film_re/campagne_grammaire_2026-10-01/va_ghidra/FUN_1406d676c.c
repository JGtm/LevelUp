
void FUN_1406d676c(longlong param_1,undefined8 param_2,ulonglong *param_3,uint param_4)

{
  ulonglong uVar1;
  byte bVar2;
  byte bVar3;
  uint uVar4;
  ulonglong uVar5;
  ulonglong *puVar6;
  int iVar7;
  int iVar8;
  ulonglong uVar9;
  longlong lVar10;
  longlong lVar11;
  
  for (; 0x3f < param_4; param_4 = param_4 - 0x40) {
    uVar4 = *(uint *)(param_1 + 0x38);
    uVar5 = *(ulonglong *)(param_1 + 0x30);
    if ((int)(0x40 - uVar4) < 0x40) {
      puVar6 = *(ulonglong **)(param_1 + 0x40);
      uVar9 = 0;
      iVar8 = 0;
      if (*(ulonglong **)(param_1 + 0x10) < puVar6 + 1) {
        iVar7 = 0;
        if (puVar6 < *(ulonglong **)(param_1 + 0x10)) {
          do {
            uVar1 = *puVar6;
            iVar8 = iVar7 + 8;
            puVar6 = (ulonglong *)((longlong)puVar6 + 1);
            uVar9 = uVar9 << 8 | (ulonglong)(byte)uVar1;
            *(ulonglong **)(param_1 + 0x40) = puVar6;
            iVar7 = iVar8;
          } while (puVar6 < *(ulonglong **)(param_1 + 0x10));
          uVar9 = uVar9 << (-(char)iVar8 & 0x3fU);
        }
      }
      else {
        uVar9 = *puVar6;
        iVar8 = 0x40;
        uVar9 = uVar9 >> 0x38 | (uVar9 & 0xff000000000000) >> 0x28 |
                (uVar9 & 0xff0000000000) >> 0x18 | (uVar9 & 0xff00000000) >> 8 |
                (uVar9 & 0xff000000) << 8 | (uVar9 & 0xff0000) << 0x18 | (uVar9 & 0xff00) << 0x28 |
                uVar9 << 0x38;
        *(ulonglong **)(param_1 + 0x40) = puVar6 + 1;
      }
      *(int *)(param_1 + 0x28) = *(int *)(param_1 + 0x28) + iVar8;
      *(int *)(param_1 + 0x2c) = *(int *)(param_1 + 0x2c) + 0x40;
      uVar5 = uVar9 >> (-(byte)uVar4 & 0x3f) | uVar5;
      uVar9 = -(ulonglong)(uVar4 < 0x40) & uVar9 << ((byte)uVar4 & 0x3f);
    }
    else {
      *(int *)(param_1 + 0x2c) = *(int *)(param_1 + 0x2c) + 0x40;
      *(uint *)(param_1 + 0x38) = uVar4 + 0x40;
      uVar9 = 0;
    }
    *(ulonglong *)(param_1 + 0x30) = uVar9;
    *param_3 = uVar5 >> 0x38 | (uVar5 & 0xff000000000000) >> 0x28 | (uVar5 & 0xff0000000000) >> 0x18
               | (uVar5 & 0xff00000000) >> 8 | (uVar5 & 0xff000000) << 8 |
               (uVar5 & 0xff0000) << 0x18 | (uVar5 & 0xff00) << 0x28 | uVar5 << 0x38;
    param_3 = param_3 + 1;
  }
  if (param_4 != 0) {
    iVar8 = *(int *)(param_1 + 0x38);
    uVar5 = *(ulonglong *)(param_1 + 0x30);
    bVar2 = (byte)param_4;
    if (0x40 - iVar8 < (int)param_4) {
      puVar6 = *(ulonglong **)(param_1 + 0x40);
      uVar9 = 0;
      iVar7 = 0;
      if (*(ulonglong **)(param_1 + 0x10) < puVar6 + 1) {
        if (puVar6 < *(ulonglong **)(param_1 + 0x10)) {
          do {
            uVar1 = *puVar6;
            iVar7 = iVar7 + 8;
            puVar6 = (ulonglong *)((longlong)puVar6 + 1);
            uVar9 = uVar9 << 8 | (ulonglong)(byte)uVar1;
            *(ulonglong **)(param_1 + 0x40) = puVar6;
          } while (puVar6 < *(ulonglong **)(param_1 + 0x10));
          uVar9 = uVar9 << (-(char)iVar7 & 0x3fU);
        }
      }
      else {
        uVar9 = *puVar6;
        iVar7 = 0x40;
        uVar9 = uVar9 >> 0x38 | (uVar9 & 0xff000000000000) >> 0x28 |
                (uVar9 & 0xff0000000000) >> 0x18 | (uVar9 & 0xff00000000) >> 8 |
                (uVar9 & 0xff000000) << 8 | (uVar9 & 0xff0000) << 0x18 | (uVar9 & 0xff00) << 0x28 |
                uVar9 << 0x38;
        *(ulonglong **)(param_1 + 0x40) = puVar6 + 1;
      }
      *(int *)(param_1 + 0x28) = *(int *)(param_1 + 0x28) + iVar7;
      *(int *)(param_1 + 0x2c) = *(int *)(param_1 + 0x2c) + param_4;
      uVar4 = iVar8 + -0x40 + param_4;
      *(uint *)(param_1 + 0x38) = uVar4;
      *(ulonglong *)(param_1 + 0x30) = -(ulonglong)(uVar4 < 0x40) & uVar9 << ((byte)uVar4 & 0x3f);
      bVar3 = 0x40 - bVar2;
      uVar9 = uVar9 >> (-(byte)uVar4 & 0x3f) | uVar5 >> (bVar3 & 0x3f);
    }
    else {
      *(int *)(param_1 + 0x2c) = *(int *)(param_1 + 0x2c) + param_4;
      bVar3 = 0x40 - bVar2;
      uVar9 = uVar5 >> (bVar3 & 0x3f);
      *(ulonglong *)(param_1 + 0x30) = -(ulonglong)(param_4 < 0x40) & uVar5 << (bVar2 & 0x3f);
      *(uint *)(param_1 + 0x38) = iVar8 + param_4;
    }
    lVar10 = uVar9 << (bVar3 & 0x3f);
    if (7 < (int)param_4) {
      uVar5 = (ulonglong)(param_4 >> 3);
      lVar11 = lVar10;
      do {
        lVar10 = lVar11 << 8;
        *(char *)param_3 = (char)((ulonglong)lVar11 >> 0x38);
        param_3 = (ulonglong *)((longlong)param_3 + 1);
        uVar5 = uVar5 - 1;
        lVar11 = lVar10;
      } while (uVar5 != 0);
      if (param_4 + (param_4 >> 3) * -8 == 0) {
        return;
      }
    }
    *(char *)param_3 = (char)((ulonglong)lVar10 >> 0x38);
  }
  return;
}

