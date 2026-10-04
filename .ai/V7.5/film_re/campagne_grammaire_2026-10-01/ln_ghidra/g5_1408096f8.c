
undefined8 FUN_1408096f8(undefined8 param_1,undefined8 param_2,longlong param_3,longlong param_4)

{
  int iVar1;
  char cVar2;
  undefined1 uVar3;
  ulonglong uVar4;
  longlong lVar5;
  undefined4 *puVar6;
  ulonglong *puVar7;
  int iVar8;
  int iVar9;
  uint uVar10;
  ulonglong uVar11;
  uint uVar12;
  bool bVar13;
  undefined4 uVar14;
  undefined8 extraout_XMM0_Qa;
  undefined8 uVar15;
  undefined8 extraout_XMM0_Qa_00;
  undefined8 extraout_XMM0_Qa_01;
  undefined8 local_18;
  undefined4 local_10;
  
  FUN_140809454(param_4,param_2,param_3 + 0x50);
  cVar2 = FUN_1406cf008(param_4);
  *(char *)(param_3 + 0x53) = cVar2;
  uVar15 = extraout_XMM0_Qa;
  if (cVar2 == '\0') {
    FUN_14080d69c(extraout_XMM0_Qa,param_4,param_3,0xffffffff);
    FUN_14080dec4(param_4,"variant-name",param_3 + 4);
    cVar2 = FUN_1405838f0(param_3);
    if (cVar2 == '\0') {
      *(undefined4 *)(param_3 + 8) = 0xffffffff;
      uVar15 = extraout_XMM0_Qa_00;
    }
    else {
      puVar6 = (undefined4 *)FUN_14080d61c(&stack0x00000028,param_3,*(undefined4 *)(param_3 + 4));
      *(undefined4 *)(param_3 + 8) = *puVar6;
      uVar15 = extraout_XMM0_Qa_01;
    }
  }
  lVar5 = param_3 + 0xc;
  FUN_14080d69c(uVar15,param_4,lVar5,0xffffffff);
  cVar2 = FUN_1405838f0(lVar5);
  if (cVar2 == '\0') {
    uVar14 = 0xffffffff;
  }
  else {
    puVar6 = (undefined4 *)FUN_1407f21b4(&stack0x00000028,lVar5);
    uVar14 = *puVar6;
  }
  *(undefined4 *)(param_3 + 0x10) = uVar14;
  cVar2 = FUN_14076f91c();
  if (cVar2 == '\0') {
    FUN_14076e524(&local_18,param_4,&stack0x00000028,0xf);
  }
  else {
    FUN_1411b259c(&local_18);
  }
  *(undefined8 *)(param_3 + 0x14) = local_18;
  *(undefined4 *)(param_3 + 0x1c) = local_10;
  FUN_14076dc04(param_4);
  uVar14 = FUN_1406d84b4(param_4);
  *(undefined4 *)(param_3 + 0x38) = uVar14;
  uVar3 = FUN_1406cf008(param_4);
  *(undefined1 *)(param_3 + 0x51) = uVar3;
  iVar1 = *(int *)(param_4 + 0x38);
  uVar10 = (uint)((ulonglong)*(longlong *)(param_4 + 0x30) >> 0x20);
  if (0x40 - iVar1 < 9) {
    puVar7 = *(ulonglong **)(param_4 + 0x40);
    uVar11 = 0;
    iVar9 = 0;
    if (*(ulonglong **)(param_4 + 0x10) < puVar7 + 1) {
      iVar8 = 0;
      if (puVar7 < *(ulonglong **)(param_4 + 0x10)) {
        do {
          uVar4 = *puVar7;
          iVar9 = iVar8 + 8;
          puVar7 = (ulonglong *)((longlong)puVar7 + 1);
          uVar11 = uVar11 << 8 | (ulonglong)(byte)uVar4;
          *(ulonglong **)(param_4 + 0x40) = puVar7;
          iVar8 = iVar9;
        } while (puVar7 < *(ulonglong **)(param_4 + 0x10));
        uVar11 = uVar11 << (0x40U - (char)iVar9 & 0x3f);
      }
    }
    else {
      uVar11 = *puVar7;
      uVar11 = uVar11 >> 0x38 | (uVar11 & 0xff000000000000) >> 0x28 |
               (uVar11 & 0xff0000000000) >> 0x18 | (uVar11 & 0xff00000000) >> 8 |
               (uVar11 & 0xff000000) << 8 | (uVar11 & 0xff0000) << 0x18 | (uVar11 & 0xff00) << 0x28
               | uVar11 << 0x38;
      *(ulonglong **)(param_4 + 0x40) = puVar7 + 1;
      iVar9 = 0x40;
    }
    *(int *)(param_4 + 0x28) = *(int *)(param_4 + 0x28) + iVar9;
    uVar12 = iVar1 - 0x37;
    *(int *)(param_4 + 0x2c) = *(int *)(param_4 + 0x2c) + 9;
    uVar4 = -(ulonglong)(uVar12 < 0x40) & uVar11 << ((byte)uVar12 & 0x3f);
    uVar10 = (uint)(uVar11 >> (0x40 - (byte)uVar12 & 0x3f)) | uVar10 >> 0x17;
  }
  else {
    *(int *)(param_4 + 0x2c) = *(int *)(param_4 + 0x2c) + 9;
    uVar4 = *(longlong *)(param_4 + 0x30) << 9;
    uVar12 = iVar1 + 9;
    uVar10 = uVar10 >> 0x17;
  }
  *(ulonglong *)(param_4 + 0x30) = uVar4;
  *(uint *)(param_4 + 0x38) = uVar12;
  *(uint *)(param_3 + 0x40) = uVar10 - 1;
  if (*(uint *)(param_4 + 0x38) < 0x40) {
    *(int *)(param_4 + 0x2c) = *(int *)(param_4 + 0x2c) + 1;
    bVar13 = SUB81((ulonglong)*(longlong *)(param_4 + 0x30) >> 0x3f,0);
    *(longlong *)(param_4 + 0x30) = *(longlong *)(param_4 + 0x30) * 2;
    *(uint *)(param_4 + 0x38) = *(uint *)(param_4 + 0x38) + 1;
  }
  else {
    lVar5 = FUN_1406d6c7c(param_4,1);
    bVar13 = lVar5 != 0;
  }
  *(bool *)(param_3 + 0x52) = bVar13;
  if (bVar13 == false) {
    *(undefined4 *)(param_3 + 0x44) = 0xffffffff;
  }
  else {
    FUN_140809530(param_4,param_3 + 0x44);
  }
  FUN_1406cf008(param_4);
  FUN_14076dc04(param_4);
  FUN_1424cd2fc(param_4);
  return 1;
}

