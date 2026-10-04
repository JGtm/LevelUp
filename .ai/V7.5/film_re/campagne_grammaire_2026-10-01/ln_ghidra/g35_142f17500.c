
char FUN_142f17500(undefined8 param_1,undefined8 param_2,undefined1 *param_3,longlong param_4,
                  char param_5)

{
  bool bVar1;
  char cVar2;
  undefined1 uVar3;
  char cVar4;
  byte bVar5;
  char cVar6;
  undefined2 uVar7;
  uint uVar8;
  undefined4 uVar9;
  undefined4 *puVar10;
  ulonglong uVar11;
  byte *pbVar12;
  int iVar13;
  int iVar14;
  undefined4 extraout_XMM0_Da;
  undefined1 local_res18 [8];
  uint in_stack_ffffffffffffff88;
  undefined1 local_58 [8];
  undefined4 local_50;
  undefined4 local_48;
  undefined4 uStack_44;
  uint uStack_3c;
  undefined4 local_38;
  undefined4 uStack_34;
  undefined4 uStack_30;
  uint uStack_2c;
  
  FUN_141fcf670(param_3 + 1,param_4);
  uVar8 = FUN_1406d00ec(param_4);
  *(uint *)(param_3 + 8) = uVar8;
  FUN_14080d69c(uVar8 == 0xffffffff,param_4,param_3 + 0x10);
  FUN_14080dec4(param_4,"variant_name",param_3 + 0x14);
  puVar10 = (undefined4 *)FUN_14080cd58(local_res18,param_3 + 0x10,*(undefined4 *)(param_3 + 0x14));
  *(undefined4 *)(param_3 + 0x18) = *puVar10;
  if (0x40 - *(int *)(param_4 + 0x38) < 1) {
    bVar5 = FUN_1406d6c7c(param_4,1);
  }
  else {
    *(int *)(param_4 + 0x2c) = *(int *)(param_4 + 0x2c) + 1;
    bVar5 = (byte)((ulonglong)*(longlong *)(param_4 + 0x30) >> 0x3f);
    *(longlong *)(param_4 + 0x30) = *(longlong *)(param_4 + 0x30) * 2;
    *(int *)(param_4 + 0x38) = *(int *)(param_4 + 0x38) + 1;
  }
  param_3[0x1d] = bVar5;
  if ((uVar8 == 0xffffffff || uVar8 < 4) && (bVar5 < 2)) {
    bVar1 = true;
  }
  else {
    bVar1 = false;
  }
  uVar3 = FUN_1406cf008(param_4);
  param_3[2] = uVar3;
  if (0x40 - *(int *)(param_4 + 0x38) < 4) {
    uVar8 = FUN_1406d6c7c(param_4,4);
  }
  else {
    *(int *)(param_4 + 0x2c) = *(int *)(param_4 + 0x2c) + 4;
    uVar8 = (uint)((ulonglong)*(longlong *)(param_4 + 0x30) >> 0x3c);
    *(longlong *)(param_4 + 0x30) = *(longlong *)(param_4 + 0x30) << 4;
    *(int *)(param_4 + 0x38) = *(int *)(param_4 + 0x38) + 4;
  }
  cVar2 = param_5;
  iVar13 = 0;
  *(uint *)(param_3 + 0x34) = uVar8;
  if (0 < (int)uVar8) {
    puVar10 = (undefined4 *)(param_3 + 0x40);
    do {
      if (0x40 - *(int *)(param_4 + 0x38) < 2) {
        bVar5 = FUN_1406d6c7c(param_4,2);
      }
      else {
        *(int *)(param_4 + 0x2c) = *(int *)(param_4 + 0x2c) + 2;
        bVar5 = (byte)((ulonglong)*(longlong *)(param_4 + 0x30) >> 0x3e);
        *(longlong *)(param_4 + 0x30) = *(longlong *)(param_4 + 0x30) << 2;
        *(int *)(param_4 + 0x38) = *(int *)(param_4 + 0x38) + 2;
      }
      *(byte *)(puVar10 + -2) = bVar5;
      uVar3 = FUN_1406cf008(param_4);
      *(undefined1 *)((longlong)puVar10 + -7) = uVar3;
      if (cVar2 == '\0') {
        if (0x40 - *(int *)(param_4 + 0x38) < 0x20) {
          uVar9 = FUN_1406d6c7c(param_4,0x20);
        }
        else {
          *(int *)(param_4 + 0x2c) = *(int *)(param_4 + 0x2c) + 0x20;
          uVar9 = (undefined4)((ulonglong)*(longlong *)(param_4 + 0x30) >> 0x20);
          *(longlong *)(param_4 + 0x30) = *(longlong *)(param_4 + 0x30) << 0x20;
          *(int *)(param_4 + 0x38) = *(int *)(param_4 + 0x38) + 0x20;
        }
        *puVar10 = uVar9;
      }
      else {
        FUN_1406d3140(extraout_XMM0_Da,param_4,1,puVar10 + -1);
      }
      iVar13 = iVar13 + 1;
      puVar10 = puVar10 + 3;
    } while (iVar13 < *(int *)(param_3 + 0x34));
  }
  if (0x40 - *(int *)(param_4 + 0x38) < 4) {
    uVar8 = FUN_1406d6c7c(param_4,4);
  }
  else {
    *(int *)(param_4 + 0x2c) = *(int *)(param_4 + 0x2c) + 4;
    uVar8 = (uint)((ulonglong)*(longlong *)(param_4 + 0x30) >> 0x3c);
    *(longlong *)(param_4 + 0x30) = *(longlong *)(param_4 + 0x30) << 4;
    *(int *)(param_4 + 0x38) = *(int *)(param_4 + 0x38) + 4;
  }
  *(uint *)(param_3 + 0xf8) = uVar8;
  uVar8 = FUN_141102ed0(0x23);
  uVar9 = FUN_1407eda24(param_3,1 < uVar8);
  iVar13 = *(int *)(param_3 + 0xf8);
  iVar14 = 0;
  if (0 < iVar13) {
    pbVar12 = param_3 + 0x100;
    do {
      if (0x40 - *(int *)(param_4 + 0x38) < 4) {
        uVar8 = FUN_1406d6c7c(param_4,4);
      }
      else {
        *(int *)(param_4 + 0x2c) = *(int *)(param_4 + 0x2c) + 4;
        uVar8 = (uint)((ulonglong)*(longlong *)(param_4 + 0x30) >> 0x3c);
        *(longlong *)(param_4 + 0x30) = *(longlong *)(param_4 + 0x30) << 4;
        *(int *)(param_4 + 0x38) = *(int *)(param_4 + 0x38) + 4;
      }
      *(uint *)(pbVar12 + -4) = uVar8;
      cVar4 = FUN_1406cf008(param_4);
      if (cVar4 == '\0') {
        *pbVar12 = *pbVar12 & 0xfe;
      }
      else {
        *pbVar12 = *pbVar12 | 1;
        iVar13 = FUN_1406d310c(6);
        if (0x40 - *(int *)(param_4 + 0x38) < iVar13) {
          bVar5 = FUN_1406d6c7c(param_4,iVar13);
        }
        else {
          *(int *)(param_4 + 0x2c) = *(int *)(param_4 + 0x2c) + iVar13;
          bVar5 = (byte)(*(ulonglong *)(param_4 + 0x30) >> (-(byte)iVar13 & 0x3f));
          *(ulonglong *)(param_4 + 0x30) = *(ulonglong *)(param_4 + 0x30) << ((byte)iVar13 & 0x3f);
          *(int *)(param_4 + 0x38) = iVar13 + *(int *)(param_4 + 0x38);
        }
        pbVar12[1] = bVar5;
        if (0x40 - *(int *)(param_4 + 0x38) < 4) {
          uVar8 = FUN_1406d6c7c(param_4,4);
        }
        else {
          *(int *)(param_4 + 0x2c) = *(int *)(param_4 + 0x2c) + 4;
          uVar8 = (uint)((ulonglong)*(longlong *)(param_4 + 0x30) >> 0x3c);
          *(longlong *)(param_4 + 0x30) = *(longlong *)(param_4 + 0x30) << 4;
          *(int *)(param_4 + 0x38) = *(int *)(param_4 + 0x38) + 4;
        }
        *(uint *)(pbVar12 + 0x10) = uVar8;
        if (0x40 - *(int *)(param_4 + 0x38) < 0x10) {
          uVar7 = FUN_1406d6c7c(param_4,0x10);
        }
        else {
          *(int *)(param_4 + 0x2c) = *(int *)(param_4 + 0x2c) + 0x10;
          uVar7 = (undefined2)((ulonglong)*(longlong *)(param_4 + 0x30) >> 0x30);
          *(longlong *)(param_4 + 0x30) = *(longlong *)(param_4 + 0x30) << 0x10;
          *(int *)(param_4 + 0x38) = *(int *)(param_4 + 0x38) + 0x10;
        }
        *(undefined2 *)(pbVar12 + 2) = uVar7;
        FUN_140c1e924(param_4,pbVar12 + 4,param_3[(longlong)*(int *)(pbVar12 + 0x10) * 0xc + 0x38],
                      uVar9);
      }
      iVar13 = *(int *)(param_3 + 0xf8);
      iVar14 = iVar14 + 1;
      pbVar12 = pbVar12 + 0x18;
    } while (iVar14 < iVar13);
  }
  if ((!bVar1) || (cVar4 = '\x01', iVar13 < 0)) {
    cVar4 = '\0';
  }
  if (cVar2 == '\0') {
    cVar6 = FUN_1406cd5b8(param_4,param_3 + 0x2a0,param_3 + 700);
  }
  else {
    cVar6 = FUN_140c9e4d8(param_3 + 0x27c,param_4,1,&param_5);
  }
  if ((cVar4 == '\0') || (cVar4 = '\x01', cVar6 == '\0')) {
    cVar4 = '\0';
  }
  if (0x40 - *(int *)(param_4 + 0x38) < 0x1e) {
    uVar8 = FUN_1406d6c7c(param_4,0x1e);
    uVar11 = (ulonglong)uVar8;
  }
  else {
    *(int *)(param_4 + 0x2c) = *(int *)(param_4 + 0x2c) + 0x1e;
    uVar11 = *(ulonglong *)(param_4 + 0x30) >> 0x22;
    *(ulonglong *)(param_4 + 0x30) = *(ulonglong *)(param_4 + 0x30) << 0x1e;
    *(int *)(param_4 + 0x38) = *(int *)(param_4 + 0x38) + 0x1e;
  }
  FUN_1406d8288(uVar11,param_3 + 0x28,0x1e);
  param_3[3] = 1;
  FUN_1408eff64(param_3 + 0x2c8,param_4,cVar2);
  cVar6 = FUN_1406cf008(param_4);
  if (cVar6 == '\0') {
    *(undefined4 *)(param_3 + 0x2e0) = 0;
  }
  else {
    if (0x40 - *(int *)(param_4 + 0x38) < 6) {
      uVar8 = FUN_1406d6c7c(param_4,6);
      uVar11 = (ulonglong)uVar8;
    }
    else {
      *(int *)(param_4 + 0x2c) = *(int *)(param_4 + 0x2c) + 6;
      uVar11 = *(ulonglong *)(param_4 + 0x30) >> 0x3a;
      *(ulonglong *)(param_4 + 0x30) = *(ulonglong *)(param_4 + 0x30) << 6;
      *(int *)(param_4 + 0x38) = *(int *)(param_4 + 0x38) + 6;
    }
    in_stack_ffffffffffffff88 = in_stack_ffffffffffffff88 & 0xffffff00;
    uVar9 = FUN_1406d8cb0(uVar11,0,DAT_143cd8374,0x40,in_stack_ffffffffffffff88,1);
    *(undefined4 *)(param_3 + 0x2e0) = uVar9;
  }
  if (0x40 - *(int *)(param_4 + 0x38) < 6) {
    uVar8 = FUN_1406d6c7c(param_4,6);
    uVar11 = (ulonglong)uVar8;
  }
  else {
    *(int *)(param_4 + 0x2c) = *(int *)(param_4 + 0x2c) + 6;
    uVar11 = *(ulonglong *)(param_4 + 0x30) >> 0x3a;
    *(ulonglong *)(param_4 + 0x30) = *(ulonglong *)(param_4 + 0x30) << 6;
    *(int *)(param_4 + 0x38) = *(int *)(param_4 + 0x38) + 6;
  }
  uVar9 = FUN_1406d8cb0(uVar11,0,DAT_143cd8374,0x40,in_stack_ffffffffffffff88 & 0xffffff00,1);
  *(undefined4 *)(param_3 + 0x2e4) = uVar9;
  uVar7 = FUN_14080cb98(param_4);
  *(undefined2 *)(param_3 + 0x1e) = uVar7;
  FUN_14080cb50(param_3 + 0x20,param_4);
  if (param_3[0x20] != '\0') {
    FUN_14320c36c(param_3 + 0x2fc,param_4);
    FUN_142a40f18(param_3 + 0x304,param_4);
  }
  if (0x40 - *(int *)(param_4 + 0x38) < 6) {
    uVar8 = FUN_1406d6c7c(param_4,6);
  }
  else {
    *(int *)(param_4 + 0x2c) = *(int *)(param_4 + 0x2c) + 6;
    uVar8 = (uint)((ulonglong)*(longlong *)(param_4 + 0x30) >> 0x3a);
    *(longlong *)(param_4 + 0x30) = *(longlong *)(param_4 + 0x30) << 6;
    *(int *)(param_4 + 0x38) = *(int *)(param_4 + 0x38) + 6;
  }
  *(uint *)(param_3 + 0x30c) = uVar8;
  cVar6 = FUN_1406cf008(param_4);
  param_3[0x310] = cVar6;
  if (cVar6 != '\0') {
    uVar9 = FUN_1406d84b4(param_4);
    *(undefined4 *)(param_3 + 0x314) = uVar9;
  }
  cVar6 = FUN_1406cf008(param_4);
  if (cVar6 == '\0') {
    uStack_2c = uStack_2c & 0xffffff00;
    local_48 = local_38;
    uStack_44 = uStack_34;
    local_50 = uStack_30;
    uStack_3c = uStack_2c;
  }
  else {
    FUN_14076e494(param_4,local_58,0x10,0,cVar2,0);
    uStack_3c = CONCAT31(uStack_3c._1_3_,1);
    local_48 = local_58._0_4_;
    uStack_44 = local_58._4_4_;
  }
  *(undefined4 *)(param_3 + 0x318) = local_48;
  *(undefined4 *)(param_3 + 0x31c) = uStack_44;
  *(undefined4 *)(param_3 + 800) = local_50;
  *(uint *)(param_3 + 0x324) = uStack_3c;
  param_3[4] = 0;
  *param_3 = 0;
  return cVar4;
}

