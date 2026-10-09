
undefined8 FUN_1410f03b4(undefined8 param_1,undefined8 param_2,longlong param_3,longlong param_4)

{
  int iVar1;
  char cVar2;
  ulonglong uVar3;
  longlong lVar4;
  undefined4 *puVar5;
  ulonglong *puVar6;
  int iVar7;
  uint uVar8;
  ulonglong uVar9;
  uint uVar10;
  bool bVar11;
  undefined4 uVar12;
  undefined8 extraout_XMM0_Qa;
  undefined4 uVar13;
  undefined1 uVar14;
  undefined1 uVar15;
  undefined8 local_18;
  undefined4 local_10;
  
  cVar2 = FUN_1406cf008(param_4);
  *(char *)(param_3 + 0x3d) = cVar2;
  if (cVar2 == '\0') {
    FUN_14080d69c(extraout_XMM0_Qa,param_4,param_3,0xffffffff);
    FUN_14080dec4(param_4,"variant-name",param_3 + 4);
    cVar2 = FUN_1405838f0(param_3);
    if (cVar2 == '\0') {
      *(undefined4 *)(param_3 + 8) = 0xffffffff;
    }
    else {
      puVar5 = (undefined4 *)FUN_14080d61c(&stack0x00000028,param_3,*(undefined4 *)(param_3 + 4));
      *(undefined4 *)(param_3 + 8) = *puVar5;
    }
  }
  uVar12 = FUN_1406d84b4(param_4);
  uVar15 = 1;
  uVar14 = 0;
  uVar13 = 7;
  *(undefined4 *)(param_3 + 0xc) = uVar12;
  uVar12 = FUN_1406d84b4(param_4);
  *(undefined4 *)(param_3 + 0x10) = uVar12;
  FUN_14076dc04(param_4);
  cVar2 = FUN_14076f91c();
  if (cVar2 == '\0') {
    FUN_14076e524(&local_18,param_4,&stack0x00000028,0xc,uVar13,uVar14,uVar15);
  }
  else {
    FUN_1411b259c(&local_18);
  }
  *(undefined8 *)(param_3 + 0x20) = local_18;
  *(undefined4 *)(param_3 + 0x28) = local_10;
  FUN_14076dc04(param_4);
  iVar1 = *(int *)(param_4 + 0x38);
  uVar8 = (uint)((ulonglong)*(longlong *)(param_4 + 0x30) >> 0x20);
  if (0x40 - iVar1 < 9) {
    puVar6 = *(ulonglong **)(param_4 + 0x40);
    uVar9 = 0;
    iVar7 = 0;
    if (*(ulonglong **)(param_4 + 0x10) < puVar6 + 1) {
      if (puVar6 < *(ulonglong **)(param_4 + 0x10)) {
        do {
          uVar3 = *puVar6;
          iVar7 = iVar7 + 8;
          puVar6 = (ulonglong *)((longlong)puVar6 + 1);
          uVar9 = uVar9 << 8 | (ulonglong)(byte)uVar3;
          *(ulonglong **)(param_4 + 0x40) = puVar6;
        } while (puVar6 < *(ulonglong **)(param_4 + 0x10));
        uVar9 = uVar9 << (0x40U - (char)iVar7 & 0x3f);
      }
    }
    else {
      uVar3 = *puVar6;
      iVar7 = 0x40;
      uVar9 = uVar3 >> 0x38 | (uVar3 & 0xff000000000000) >> 0x28 | (uVar3 & 0xff0000000000) >> 0x18
              | (uVar3 & 0xff00000000) >> 8 | (uVar3 & 0xff000000) << 8 | (uVar3 & 0xff0000) << 0x18
              | (uVar3 & 0xff00) << 0x28 | uVar3 << 0x38;
      *(ulonglong **)(param_4 + 0x40) = puVar6 + 1;
    }
    *(int *)(param_4 + 0x28) = *(int *)(param_4 + 0x28) + iVar7;
    uVar10 = iVar1 - 0x37;
    *(int *)(param_4 + 0x2c) = *(int *)(param_4 + 0x2c) + 9;
    uVar3 = -(ulonglong)(uVar10 < 0x40) & uVar9 << ((byte)uVar10 & 0x3f);
    uVar8 = (uint)(uVar9 >> (0x40 - (byte)uVar10 & 0x3f)) | uVar8 >> 0x17;
  }
  else {
    *(int *)(param_4 + 0x2c) = *(int *)(param_4 + 0x2c) + 9;
    uVar3 = *(longlong *)(param_4 + 0x30) << 9;
    uVar10 = iVar1 + 9;
    uVar8 = uVar8 >> 0x17;
  }
  *(ulonglong *)(param_4 + 0x30) = uVar3;
  *(uint *)(param_4 + 0x38) = uVar10;
  *(uint *)(param_3 + 0x38) = uVar8 - 1;
  if (*(uint *)(param_4 + 0x38) < 0x40) {
    *(int *)(param_4 + 0x2c) = *(int *)(param_4 + 0x2c) + 1;
    bVar11 = SUB81((ulonglong)*(longlong *)(param_4 + 0x30) >> 0x3f,0);
    *(longlong *)(param_4 + 0x30) = *(longlong *)(param_4 + 0x30) * 2;
    *(uint *)(param_4 + 0x38) = *(uint *)(param_4 + 0x38) + 1;
  }
  else {
    lVar4 = FUN_1406d6c7c(param_4,1);
    bVar11 = lVar4 != 0;
  }
  *(bool *)(param_3 + 0x3c) = bVar11;
  return 1;
}

