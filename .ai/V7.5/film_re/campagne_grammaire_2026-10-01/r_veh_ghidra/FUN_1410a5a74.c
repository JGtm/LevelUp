
undefined4
FUN_1410a5a74(undefined8 param_1,undefined8 param_2,longlong param_3,longlong param_4,
             undefined1 param_5)

{
  undefined4 uVar1;
  char cVar2;
  char cVar3;
  longlong lVar4;
  ulonglong uVar5;
  ulonglong *puVar6;
  int iVar7;
  int iVar8;
  int iVar9;
  uint uVar10;
  uint uVar11;
  undefined1 uVar12;
  ulonglong uVar13;
  bool bVar14;
  undefined4 local_res18 [2];
  
  cVar2 = FUN_1406cf008(param_4);
  uVar12 = 1;
  if (cVar2 != '\0') {
    iVar9 = 0x40 - *(int *)(param_4 + 0x38);
    if (iVar9 < 8) {
      puVar6 = *(ulonglong **)(param_4 + 0x40);
      uVar13 = 0;
      iVar7 = 0;
      if (*(ulonglong **)(param_4 + 0x10) < puVar6 + 1) {
        iVar8 = 0;
        if (puVar6 < *(ulonglong **)(param_4 + 0x10)) {
          do {
            iVar7 = iVar8 + 8;
            uVar5 = *puVar6;
            puVar6 = (ulonglong *)((longlong)puVar6 + 1);
            uVar13 = (ulonglong)(byte)uVar5 | uVar13 << 8;
            *(ulonglong **)(param_4 + 0x40) = puVar6;
            iVar8 = iVar7;
          } while (puVar6 < *(ulonglong **)(param_4 + 0x10));
          uVar13 = uVar13 << (0x40U - (char)iVar7 & 0x3f);
        }
      }
      else {
        uVar13 = *puVar6;
        uVar13 = uVar13 >> 0x38 | (uVar13 & 0xff000000000000) >> 0x28 |
                 (uVar13 & 0xff0000000000) >> 0x18 | (uVar13 & 0xff00000000) >> 8 |
                 (uVar13 & 0xff000000) << 8 | (uVar13 & 0xff0000) << 0x18 |
                 (uVar13 & 0xff00) << 0x28 | uVar13 << 0x38;
        *(ulonglong **)(param_4 + 0x40) = puVar6 + 1;
        iVar7 = 0x40;
      }
      *(int *)(param_4 + 0x28) = *(int *)(param_4 + 0x28) + iVar7;
      *(int *)(param_4 + 0x2c) = *(int *)(param_4 + 0x2c) + 8;
      uVar10 = 8 - iVar9;
      *(ulonglong *)(param_4 + 0x30) = -(ulonglong)(uVar10 < 0x40) & uVar13 << ((byte)uVar10 & 0x3f)
      ;
      *(uint *)(param_4 + 0x38) = uVar10;
    }
    else {
      *(int *)(param_4 + 0x2c) = *(int *)(param_4 + 0x2c) + 8;
      *(longlong *)(param_4 + 0x30) = *(longlong *)(param_4 + 0x30) << 8;
      *(int *)(param_4 + 0x38) = *(int *)(param_4 + 0x38) + 8;
    }
  }
  cVar2 = FUN_14080cfe8(param_3,param_4);
  if (*(uint *)(param_4 + 0x38) < 0x40) {
    *(int *)(param_4 + 0x2c) = *(int *)(param_4 + 0x2c) + 1;
    bVar14 = SUB81((ulonglong)*(longlong *)(param_4 + 0x30) >> 0x3f,0);
    *(longlong *)(param_4 + 0x30) = *(longlong *)(param_4 + 0x30) * 2;
    *(uint *)(param_4 + 0x38) = *(uint *)(param_4 + 0x38) + 1;
  }
  else {
    lVar4 = FUN_1406d6c7c(param_4,1);
    bVar14 = lVar4 != 0;
  }
  *(bool *)(param_3 + 0x60) = bVar14;
  if (bVar14 != false) {
    FUN_14076e494(param_4,param_3 + 100,0x10,0,param_5,0);
    FUN_140c1e79c(param_4);
  }
  FUN_14076dc04(param_4);
  *(undefined4 *)(param_3 + 0x94) = 0;
  *(undefined4 *)(param_3 + 0x98) = 0;
  *(undefined4 *)(param_3 + 0xa8) = 0xffffffff;
  *(undefined1 *)(param_3 + 0xac) = 0;
  cVar3 = FUN_1406cf008(param_4);
  *(char *)(param_3 + 0xac) = cVar3;
  if (cVar3 == '\0') {
    FUN_14080d69c();
  }
  else {
    iVar9 = *(int *)(param_4 + 0x38);
    uVar10 = (uint)((ulonglong)*(longlong *)(param_4 + 0x30) >> 0x20);
    if (0x40 - iVar9 < 2) {
      puVar6 = *(ulonglong **)(param_4 + 0x40);
      uVar13 = 0;
      iVar7 = 0;
      if (*(ulonglong **)(param_4 + 0x10) < puVar6 + 1) {
        iVar8 = 0;
        if (puVar6 < *(ulonglong **)(param_4 + 0x10)) {
          do {
            uVar5 = *puVar6;
            iVar7 = iVar8 + 8;
            puVar6 = (ulonglong *)((longlong)puVar6 + 1);
            uVar13 = uVar13 << 8 | (ulonglong)(byte)uVar5;
            *(ulonglong **)(param_4 + 0x40) = puVar6;
            iVar8 = iVar7;
          } while (puVar6 < *(ulonglong **)(param_4 + 0x10));
          uVar13 = uVar13 << (0x40U - (char)iVar7 & 0x3f);
        }
      }
      else {
        uVar13 = *puVar6;
        uVar13 = uVar13 >> 0x38 | (uVar13 & 0xff000000000000) >> 0x28 |
                 (uVar13 & 0xff0000000000) >> 0x18 | (uVar13 & 0xff00000000) >> 8 |
                 (uVar13 & 0xff000000) << 8 | (uVar13 & 0xff0000) << 0x18 |
                 (uVar13 & 0xff00) << 0x28 | uVar13 << 0x38;
        *(ulonglong **)(param_4 + 0x40) = puVar6 + 1;
        iVar7 = 0x40;
      }
      *(int *)(param_4 + 0x28) = *(int *)(param_4 + 0x28) + iVar7;
      uVar11 = iVar9 - 0x3e;
      *(int *)(param_4 + 0x2c) = *(int *)(param_4 + 0x2c) + 2;
      uVar5 = -(ulonglong)(uVar11 < 0x40) & uVar13 << ((byte)uVar11 & 0x3f);
      uVar10 = (uint)(uVar13 >> (0x40 - (byte)uVar11 & 0x3f)) | uVar10 >> 0x1e;
    }
    else {
      *(int *)(param_4 + 0x2c) = *(int *)(param_4 + 0x2c) + 2;
      uVar5 = *(longlong *)(param_4 + 0x30) * 4;
      uVar11 = iVar9 + 2;
      uVar10 = uVar10 >> 0x1e;
    }
    *(ulonglong *)(param_4 + 0x30) = uVar5;
    *(uint *)(param_4 + 0x38) = uVar11;
    if (uVar10 != 0) {
      uVar13 = (ulonglong)uVar10;
      do {
        local_res18[0] = 0xffffffff;
        cVar3 = FUN_1406cf008(param_4);
        if (cVar3 == '\0') {
          local_res18[0] = 0xffffffff;
        }
        else {
          FUN_14080d6f0();
        }
        uVar1 = local_res18[0];
        cVar3 = FUN_1405838f0(local_res18);
        if (cVar3 != '\0') {
          iVar9 = *(int *)(param_3 + 0x94);
          *(int *)(param_3 + 0x94) = iVar9 + 1;
          *(undefined4 *)(param_3 + 0x9c + (longlong)iVar9 * 4) = uVar1;
        }
        uVar13 = uVar13 - 1;
      } while (uVar13 != 0);
    }
  }
  iVar9 = *(int *)(param_4 + 0x18) * 8;
  if ((iVar9 < *(int *)(param_4 + 0x2c)) || (cVar2 == '\0')) {
    uVar12 = 0;
  }
  return CONCAT31((int3)((uint)iVar9 >> 8),uVar12);
}

