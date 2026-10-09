
undefined8 FUN_140ff8d70(undefined8 param_1,undefined8 param_2,byte *param_3,longlong param_4)

{
  byte *pbVar1;
  longlong lVar2;
  bool bVar3;
  char cVar4;
  int iVar5;
  undefined4 uVar6;
  undefined4 *puVar7;
  ulonglong uVar8;
  byte bVar9;
  byte bVar10;
  uint uVar11;
  ulonglong *puVar12;
  int iVar13;
  int iVar14;
  undefined1 uVar15;
  int iVar16;
  ushort uVar17;
  ulonglong uVar18;
  uint uVar19;
  uint local_res18 [2];
  
  uVar15 = 1;
  bVar3 = true;
  iVar5 = FUN_141102ed0(0x28);
  uVar11 = *(uint *)(param_4 + 0x38);
  iVar14 = 0x40;
  lVar2 = *(longlong *)(param_4 + 0x30);
  iVar16 = 0x40 - uVar11;
  bVar10 = (byte)((ulonglong)lVar2 >> 0x38);
  if (iVar5 < 2) {
    if (1 < iVar16) {
      *(int *)(param_4 + 0x2c) = *(int *)(param_4 + 0x2c) + 2;
      *(longlong *)(param_4 + 0x30) = lVar2 * 4;
      *(uint *)(param_4 + 0x38) = uVar11 + 2;
      bVar10 = bVar10 >> 6;
      goto LAB_140ff8de6;
    }
    puVar12 = *(ulonglong **)(param_4 + 0x40);
    uVar18 = 0;
    iVar5 = 0;
    if (*(ulonglong **)(param_4 + 0x10) < puVar12 + 1) {
      iVar13 = 0;
      if (puVar12 < *(ulonglong **)(param_4 + 0x10)) {
        do {
          iVar5 = iVar13 + 8;
          uVar8 = *puVar12;
          puVar12 = (ulonglong *)((longlong)puVar12 + 1);
          uVar18 = (ulonglong)(byte)uVar8 | uVar18 << 8;
          *(ulonglong **)(param_4 + 0x40) = puVar12;
          iVar13 = iVar5;
        } while (puVar12 < *(ulonglong **)(param_4 + 0x10));
        uVar18 = uVar18 << (0x40U - (char)iVar5 & 0x3f);
      }
    }
    else {
      uVar18 = *puVar12;
      uVar18 = uVar18 >> 0x38 | (uVar18 & 0xff000000000000) >> 0x28 |
               (uVar18 & 0xff0000000000) >> 0x18 | (uVar18 & 0xff00000000) >> 8 |
               (uVar18 & 0xff000000) << 8 | (uVar18 & 0xff0000) << 0x18 | (uVar18 & 0xff00) << 0x28
               | uVar18 << 0x38;
      *(ulonglong **)(param_4 + 0x40) = puVar12 + 1;
      iVar5 = iVar14;
    }
    *(int *)(param_4 + 0x28) = *(int *)(param_4 + 0x28) + iVar5;
    *(int *)(param_4 + 0x2c) = *(int *)(param_4 + 0x2c) + 2;
    uVar11 = 2 - iVar16;
    uVar8 = uVar18 << ((byte)uVar11 & 0x3f);
    bVar10 = bVar10 >> 6;
  }
  else {
    if (2 < iVar16) {
      *(int *)(param_4 + 0x2c) = *(int *)(param_4 + 0x2c) + 3;
      *(longlong *)(param_4 + 0x30) = lVar2 * 8;
      *(uint *)(param_4 + 0x38) = uVar11 + 3;
      bVar10 = bVar10 >> 5;
      goto LAB_140ff8de6;
    }
    puVar12 = *(ulonglong **)(param_4 + 0x40);
    uVar18 = 0;
    iVar5 = 0;
    if (*(ulonglong **)(param_4 + 0x10) < puVar12 + 1) {
      if (puVar12 < *(ulonglong **)(param_4 + 0x10)) {
        do {
          iVar5 = iVar5 + 8;
          uVar8 = *puVar12;
          puVar12 = (ulonglong *)((longlong)puVar12 + 1);
          uVar18 = (ulonglong)(byte)uVar8 | uVar18 << 8;
          *(ulonglong **)(param_4 + 0x40) = puVar12;
        } while (puVar12 < *(ulonglong **)(param_4 + 0x10));
        uVar18 = uVar18 << (0x40U - (char)iVar5 & 0x3f);
      }
    }
    else {
      uVar18 = *puVar12;
      uVar18 = uVar18 >> 0x38 | (uVar18 & 0xff000000000000) >> 0x28 |
               (uVar18 & 0xff0000000000) >> 0x18 | (uVar18 & 0xff00000000) >> 8 |
               (uVar18 & 0xff000000) << 8 | (uVar18 & 0xff0000) << 0x18 | (uVar18 & 0xff00) << 0x28
               | uVar18 << 0x38;
      *(ulonglong **)(param_4 + 0x40) = puVar12 + 1;
      iVar5 = iVar14;
    }
    *(int *)(param_4 + 0x28) = *(int *)(param_4 + 0x28) + iVar5;
    *(int *)(param_4 + 0x2c) = *(int *)(param_4 + 0x2c) + 3;
    uVar11 = 3 - iVar16;
    uVar8 = uVar18 << ((byte)uVar11 & 0x3f);
    bVar10 = bVar10 >> 5;
  }
  *(ulonglong *)(param_4 + 0x30) = -(ulonglong)(uVar11 < 0x40) & uVar8;
  *(uint *)(param_4 + 0x38) = uVar11;
  bVar9 = 0x40 - (char)uVar11;
  uVar11 = (uint)bVar9;
  bVar10 = (byte)(uVar18 >> (bVar9 & 0x3f)) | bVar10;
LAB_140ff8de6:
  *param_3 = bVar10;
  if (bVar10 == 1) {
    uVar11 = *(uint *)(param_4 + 0x38);
    bVar10 = (byte)((ulonglong)*(longlong *)(param_4 + 0x30) >> 0x38);
    if ((int)(0x40 - uVar11) < 2) {
      puVar12 = *(ulonglong **)(param_4 + 0x40);
      uVar18 = 0;
      iVar5 = 0;
      if (*(ulonglong **)(param_4 + 0x10) < puVar12 + 1) {
        if (puVar12 < *(ulonglong **)(param_4 + 0x10)) {
          do {
            iVar5 = iVar5 + 8;
            uVar8 = *puVar12;
            puVar12 = (ulonglong *)((longlong)puVar12 + 1);
            uVar18 = (ulonglong)(byte)uVar8 | uVar18 << 8;
            *(ulonglong **)(param_4 + 0x40) = puVar12;
          } while (puVar12 < *(ulonglong **)(param_4 + 0x10));
          uVar18 = uVar18 << (0x40U - (char)iVar5 & 0x3f);
        }
      }
      else {
        uVar18 = *puVar12;
        uVar18 = uVar18 >> 0x38 | (uVar18 & 0xff000000000000) >> 0x28 |
                 (uVar18 & 0xff0000000000) >> 0x18 | (uVar18 & 0xff00000000) >> 8 |
                 (uVar18 & 0xff000000) << 8 | (uVar18 & 0xff0000) << 0x18 |
                 (uVar18 & 0xff00) << 0x28 | uVar18 << 0x38;
        *(ulonglong **)(param_4 + 0x40) = puVar12 + 1;
        iVar5 = iVar14;
      }
      *(int *)(param_4 + 0x28) = *(int *)(param_4 + 0x28) + iVar5;
      *(int *)(param_4 + 0x2c) = *(int *)(param_4 + 0x2c) + 2;
      uVar11 = 2 - (0x40 - uVar11);
      *(ulonglong *)(param_4 + 0x30) = -(ulonglong)(uVar11 < 0x40) & uVar18 << ((byte)uVar11 & 0x3f)
      ;
      *(uint *)(param_4 + 0x38) = uVar11;
      bVar9 = 0x40 - (byte)uVar11;
      uVar11 = (uint)bVar9;
      bVar10 = (byte)(uVar18 >> (bVar9 & 0x3f)) | bVar10 >> 6;
    }
    else {
      *(int *)(param_4 + 0x2c) = *(int *)(param_4 + 0x2c) + 2;
      *(longlong *)(param_4 + 0x30) = *(longlong *)(param_4 + 0x30) * 4;
      *(uint *)(param_4 + 0x38) = uVar11 + 2;
      bVar10 = bVar10 >> 6;
    }
    param_3[1] = bVar10;
    bVar3 = bVar10 < 4;
  }
  else {
    param_3[1] = 0;
  }
  iVar5 = *(int *)(param_4 + 0x38);
  uVar17 = (ushort)((ulonglong)*(longlong *)(param_4 + 0x30) >> 0x30);
  if (0x40 - iVar5 < 2) {
    puVar12 = *(ulonglong **)(param_4 + 0x40);
    uVar18 = 0;
    iVar16 = 0;
    if (*(ulonglong **)(param_4 + 0x10) < puVar12 + 1) {
      iVar14 = iVar16;
      if (puVar12 < *(ulonglong **)(param_4 + 0x10)) {
        do {
          uVar8 = *puVar12;
          iVar16 = iVar16 + 8;
          puVar12 = (ulonglong *)((longlong)puVar12 + 1);
          uVar18 = uVar18 << 8 | (ulonglong)(byte)uVar8;
          *(ulonglong **)(param_4 + 0x40) = puVar12;
        } while (puVar12 < *(ulonglong **)(param_4 + 0x10));
        uVar18 = uVar18 << (0x40U - (char)iVar16 & 0x3f);
        iVar14 = iVar16;
      }
    }
    else {
      uVar18 = *puVar12;
      uVar18 = uVar18 >> 0x38 | (uVar18 & 0xff000000000000) >> 0x28 |
               (uVar18 & 0xff0000000000) >> 0x18 | (uVar18 & 0xff00000000) >> 8 |
               (uVar18 & 0xff000000) << 8 | (uVar18 & 0xff0000) << 0x18 | (uVar18 & 0xff00) << 0x28
               | uVar18 << 0x38;
      *(ulonglong **)(param_4 + 0x40) = puVar12 + 1;
    }
    *(int *)(param_4 + 0x28) = *(int *)(param_4 + 0x28) + iVar14;
    uVar19 = iVar5 - 0x3e;
    *(int *)(param_4 + 0x2c) = *(int *)(param_4 + 0x2c) + 2;
    bVar10 = 0x40 - (byte)uVar19;
    uVar11 = (uint)bVar10;
    uVar8 = -(ulonglong)(uVar19 < 0x40) & uVar18 << ((byte)uVar19 & 0x3f);
    uVar17 = (ushort)(uVar18 >> (bVar10 & 0x3f)) | uVar17 >> 0xe;
  }
  else {
    *(int *)(param_4 + 0x2c) = *(int *)(param_4 + 0x2c) + 2;
    uVar8 = *(longlong *)(param_4 + 0x30) * 4;
    uVar19 = iVar5 + 2;
    uVar17 = uVar17 >> 0xe;
  }
  *(ulonglong *)(param_4 + 0x30) = uVar8;
  pbVar1 = param_3 + 4;
  *(uint *)(param_4 + 0x38) = uVar19;
  *(ushort *)(param_3 + 2) = uVar17;
  FUN_14080d69c(uVar11,param_4,pbVar1,0xffffffff);
  FUN_14080dec4(param_4,"variant-name",param_3 + 8);
  cVar4 = FUN_1405838f0(pbVar1);
  if (cVar4 == '\0') {
    param_3[0xc] = 0xff;
    param_3[0xd] = 0xff;
    param_3[0xe] = 0xff;
    param_3[0xf] = 0xff;
  }
  else {
    puVar7 = (undefined4 *)FUN_14080d61c(local_res18,pbVar1,*(undefined4 *)(param_3 + 8));
    *(undefined4 *)(param_3 + 0xc) = *puVar7;
  }
  uVar6 = FUN_1407f2058(param_4);
  FUN_140495860(local_res18,uVar6);
  uVar18 = (ulonglong)local_res18[0];
  *(uint *)(param_3 + 0x10) = local_res18[0];
  if (((!bVar3) ||
      (uVar18 = (ulonglong)CONCAT31((int3)(local_res18[0] >> 8),*param_3 - 1),
      3 < (byte)(*param_3 - 1))) || (2 < *(ushort *)(param_3 + 2))) {
    uVar15 = 0;
  }
  return CONCAT71((int7)(uVar18 >> 8),uVar15);
}

