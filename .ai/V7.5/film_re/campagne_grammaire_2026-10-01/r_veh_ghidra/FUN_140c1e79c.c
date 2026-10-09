
void FUN_140c1e79c(longlong param_1,undefined8 param_2,undefined8 param_3,undefined8 *param_4)

{
  int iVar1;
  undefined *puVar2;
  ulonglong uVar3;
  longlong lVar4;
  ulonglong *puVar5;
  int iVar6;
  uint uVar7;
  ulonglong uVar8;
  ulonglong uVar9;
  undefined4 uVar10;
  
  if (*(uint *)(param_1 + 0x38) < 0x40) {
    lVar4 = *(longlong *)(param_1 + 0x30);
    *(int *)(param_1 + 0x2c) = *(int *)(param_1 + 0x2c) + 1;
    *(longlong *)(param_1 + 0x30) = lVar4 * 2;
    *(uint *)(param_1 + 0x38) = *(uint *)(param_1 + 0x38) + 1;
    if (-1 < lVar4) {
LAB_140c1e839:
      iVar1 = *(int *)(param_1 + 0x38);
      uVar8 = *(ulonglong *)(param_1 + 0x30);
      if (0x40 - iVar1 < 0x13) {
        puVar5 = *(ulonglong **)(param_1 + 0x40);
        uVar9 = 0;
        iVar6 = 0;
        if (*(ulonglong **)(param_1 + 0x10) < puVar5 + 1) {
          if (puVar5 < *(ulonglong **)(param_1 + 0x10)) {
            do {
              uVar3 = *puVar5;
              iVar6 = iVar6 + 8;
              puVar5 = (ulonglong *)((longlong)puVar5 + 1);
              uVar9 = uVar9 << 8 | (ulonglong)(byte)uVar3;
              *(ulonglong **)(param_1 + 0x40) = puVar5;
            } while (puVar5 < *(ulonglong **)(param_1 + 0x10));
            uVar9 = uVar9 << (0x40U - (char)iVar6 & 0x3f);
          }
        }
        else {
          uVar9 = *puVar5;
          iVar6 = 0x40;
          uVar9 = uVar9 >> 0x38 | (uVar9 & 0xff000000000000) >> 0x28 |
                  (uVar9 & 0xff0000000000) >> 0x18 | (uVar9 & 0xff00000000) >> 8 |
                  (uVar9 & 0xff000000) << 8 | (uVar9 & 0xff0000) << 0x18 | (uVar9 & 0xff00) << 0x28
                  | uVar9 << 0x38;
          *(ulonglong **)(param_1 + 0x40) = puVar5 + 1;
        }
        *(int *)(param_1 + 0x28) = *(int *)(param_1 + 0x28) + iVar6;
        uVar7 = iVar1 - 0x2d;
        *(int *)(param_1 + 0x2c) = *(int *)(param_1 + 0x2c) + 0x13;
        uVar3 = -(ulonglong)(uVar7 < 0x40) & uVar9 << ((byte)uVar7 & 0x3f);
        uVar8 = uVar9 >> (0x40 - (byte)uVar7 & 0x3f) | uVar8 >> 0x2d;
      }
      else {
        *(int *)(param_1 + 0x2c) = *(int *)(param_1 + 0x2c) + 0x13;
        uVar3 = uVar8 << 0x13;
        uVar7 = iVar1 + 0x13;
        uVar8 = uVar8 >> 0x2d;
      }
      *(ulonglong *)(param_1 + 0x30) = uVar3;
      *(uint *)(param_1 + 0x38) = uVar7;
      FUN_1406d8288(uVar8 & 0xffffffff,param_4,0x13);
      goto LAB_140c1e7f2;
    }
  }
  else {
    lVar4 = FUN_1406d6c7c(param_1);
    if (lVar4 == 0) goto LAB_140c1e839;
  }
  puVar2 = PTR_DAT_14474c2e8;
  *param_4 = *(undefined8 *)PTR_DAT_14474c2e8;
  *(undefined4 *)(param_4 + 1) = *(undefined4 *)(puVar2 + 8);
LAB_140c1e7f2:
  uVar10 = FUN_1406d84b4(param_1);
  FUN_1406d8678(param_4,uVar10,param_3);
  return;
}

