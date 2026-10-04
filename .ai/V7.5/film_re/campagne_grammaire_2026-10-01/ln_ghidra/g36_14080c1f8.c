
char FUN_14080c1f8(undefined8 param_1,undefined8 param_2,byte *param_3,longlong param_4,char param_5
                  )

{
  byte bVar1;
  char cVar2;
  char cVar3;
  undefined2 uVar4;
  uint uVar5;
  int iVar6;
  undefined4 *puVar7;
  ulonglong uVar8;
  undefined8 uVar9;
  int iVar10;
  byte *pbVar11;
  uint *puVar12;
  char cVar13;
  bool bVar14;
  undefined4 uVar15;
  undefined4 extraout_XMM0_Da;
  undefined4 uVar16;
  char local_res20 [8];
  uint in_stack_ffffffffffffff38;
  undefined1 local_a4 [4];
  undefined1 local_a0 [8];
  ulonglong local_98;
  undefined8 local_90;
  undefined4 local_88;
  undefined4 uStack_74;
  undefined8 local_68;
  undefined8 uStack_60;
  undefined4 local_58;
  undefined4 uStack_54;
  undefined4 uStack_50;
  uint uStack_4c;
  
  memset(param_3,0,0x328);
  uStack_4c = uStack_4c & 0xffffff00;
  param_3[0x284] = 0xff;
  param_3[0x285] = 0xff;
  param_3[0x286] = 0xff;
  param_3[0x287] = 0xff;
  param_3[0x280] = 0xff;
  param_3[0x281] = 0xff;
  param_3[0x282] = 0xff;
  param_3[0x283] = 0xff;
  *(undefined8 *)(param_3 + 0x288) = DAT_143cef530;
  *(undefined4 *)(param_3 + 0x290) = DAT_143cef538;
  param_3[0x294] = 0;
  param_3[0x298] = 0;
  param_3[0x299] = 0;
  param_3[0x29a] = 0;
  param_3[0x29b] = 0;
  param_3[0x29c] = 0;
  param_3[0x29d] = 0;
  param_3[0x29e] = 0;
  param_3[0x29f] = 0;
  param_3[0x2a0] = 0;
  param_3[0x2a1] = 0;
  param_3[0x2a2] = 0;
  param_3[0x2a3] = 0;
  param_3[0x2a4] = 0;
  param_3[0x2a5] = 0;
  param_3[0x2a6] = 0;
  param_3[0x2a7] = 0;
  param_3[0x2a8] = 0xff;
  param_3[0x2a9] = 0xff;
  param_3[0x2aa] = 0xff;
  param_3[0x2ab] = 0xff;
  param_3[0x2a4] = 0xff;
  param_3[0x2a5] = 0xff;
  param_3[0x2a6] = 0xff;
  param_3[0x2a7] = 0xff;
  *(undefined8 *)(param_3 + 0x2ac) = DAT_143cef530;
  *(undefined4 *)(param_3 + 0x2b4) = DAT_143cef538;
  param_3[0x2b8] = 0;
  param_3[0x2c8] = 0;
  param_3[0x2d0] = 0;
  param_3[0x2d4] = 0xff;
  param_3[0x2d5] = 0xff;
  param_3[0x2d6] = 0xff;
  param_3[0x2d7] = 0xff;
  param_3[0x2d8] = 0xff;
  param_3[0x2d9] = 0xff;
  param_3[0x2da] = 0xff;
  param_3[0x2db] = 0xff;
  param_3[0x2cc] = 0xff;
  param_3[0x2cd] = 0xff;
  param_3[0x2ce] = 0xff;
  param_3[0x2cf] = 0xff;
  param_3[0x2f8] = 0;
  param_3[0x2f9] = 0;
  param_3[0x2fa] = 0x80;
  param_3[0x2fb] = 0xbf;
  *(undefined4 *)(param_3 + 0x318) = local_58;
  *(undefined4 *)(param_3 + 0x31c) = uStack_54;
  *(undefined4 *)(param_3 + 800) = uStack_50;
  *(uint *)(param_3 + 0x324) = uStack_4c;
  if (0x40 - *(int *)(param_4 + 0x38) < 1) {
    bVar1 = FUN_1406d6c7c(param_4,1);
  }
  else {
    *(int *)(param_4 + 0x2c) = *(int *)(param_4 + 0x2c) + 1;
    bVar1 = (byte)((ulonglong)*(longlong *)(param_4 + 0x30) >> 0x3f);
    *(longlong *)(param_4 + 0x30) = *(longlong *)(param_4 + 0x30) * 2;
    *(int *)(param_4 + 0x38) = *(int *)(param_4 + 0x38) + 1;
  }
  *param_3 = bVar1;
  if (0x40 - *(int *)(param_4 + 0x38) < 1) {
    bVar1 = FUN_1406d6c7c(param_4,1);
  }
  else {
    *(int *)(param_4 + 0x2c) = *(int *)(param_4 + 0x2c) + 1;
    bVar1 = (byte)((ulonglong)*(longlong *)(param_4 + 0x30) >> 0x3f);
    *(longlong *)(param_4 + 0x30) = *(longlong *)(param_4 + 0x30) * 2;
    *(int *)(param_4 + 0x38) = *(int *)(param_4 + 0x38) + 1;
  }
  param_3[0x1c] = bVar1;
  FUN_141fcf670(param_3 + 1,param_4);
  puVar7 = (undefined4 *)FUN_1407f2034(local_a4,param_4);
  *(undefined4 *)(param_3 + 0xc) = *puVar7;
  uVar5 = FUN_1406d00ec(param_4);
  *(uint *)(param_3 + 8) = uVar5;
  FUN_14080d69c(uVar5 == 0xffffffff,param_4,param_3 + 0x10,0xffffffff);
  FUN_14080dec4(param_4,"variant_name",param_3 + 0x14);
  puVar7 = (undefined4 *)FUN_14080cd58(local_a0,param_3 + 0x10,*(undefined4 *)(param_3 + 0x14));
  *(undefined4 *)(param_3 + 0x18) = *puVar7;
  if (0x40 - *(int *)(param_4 + 0x38) < 1) {
    bVar1 = FUN_1406d6c7c(param_4,1);
  }
  else {
    *(int *)(param_4 + 0x2c) = *(int *)(param_4 + 0x2c) + 1;
    bVar1 = (byte)((ulonglong)*(longlong *)(param_4 + 0x30) >> 0x3f);
    *(longlong *)(param_4 + 0x30) = *(longlong *)(param_4 + 0x30) * 2;
    *(int *)(param_4 + 0x38) = *(int *)(param_4 + 0x38) + 1;
  }
  param_3[0x1d] = bVar1;
  cVar13 = '\x01';
  if ((uVar5 != 0xffffffff && 3 < uVar5) || (1 < bVar1)) {
    cVar13 = '\0';
  }
  bVar1 = FUN_1406cf008(param_4);
  param_3[2] = bVar1;
  if (param_3[0x1c] == 1) {
    bVar1 = FUN_1406cf008(param_4);
    param_3[0x2dc] = bVar1;
    bVar1 = FUN_1406cf008(param_4);
  }
  else {
    bVar1 = 0;
  }
  param_3[0x2dd] = bVar1;
  if (bVar1 != 0) {
    uVar15 = FUN_1431a0abc(param_4);
    *(undefined4 *)(param_3 + 0x2e0) = uVar15;
    *(undefined4 *)(param_3 + 0x2e4) = uVar15;
  }
  if (*param_3 == 1) {
    FUN_14076dc04();
    param_3[3] = 1;
    if (param_3[0x1c] != 1) {
      return cVar13;
    }
    param_3[0x2f4] = 1;
    *(int *)(param_3 + 0x2e8) = (int)*(undefined8 *)(param_3 + 0x28);
    *(int *)(param_3 + 0x2ec) = (int)((ulonglong)*(undefined8 *)(param_3 + 0x28) >> 0x20);
    *(undefined4 *)(param_3 + 0x2f0) = *(undefined4 *)(param_3 + 0x30);
    uVar5 = FUN_141102ed0(0x24);
    if (uVar5 < 2) {
      return cVar13;
    }
    param_3[0x2f8] = 0;
    param_3[0x2f9] = 0;
    param_3[0x2fa] = 0x80;
    param_3[0x2fb] = 0x3f;
    return cVar13;
  }
  FUN_14080cc68(param_4,param_3 + 0xf8,param_3 + 0x34);
  if ((cVar13 == '\0') || (*(int *)(param_3 + 0xf8) < 0)) {
    cVar13 = '\0';
  }
  else {
    cVar13 = '\x01';
  }
  iVar10 = 0;
  local_98 = 0xc;
  if (0 < *(int *)(param_3 + 0x34)) {
    pbVar11 = param_3 + 0x40;
    do {
      if (0x40 - *(int *)(param_4 + 0x38) < 2) {
        bVar1 = FUN_1406d6c7c(param_4,2);
      }
      else {
        *(int *)(param_4 + 0x2c) = *(int *)(param_4 + 0x2c) + 2;
        bVar1 = (byte)((ulonglong)*(longlong *)(param_4 + 0x30) >> 0x3e);
        *(longlong *)(param_4 + 0x30) = *(longlong *)(param_4 + 0x30) << 2;
        *(int *)(param_4 + 0x38) = *(int *)(param_4 + 0x38) + 2;
      }
      pbVar11[-8] = bVar1;
      bVar1 = FUN_1406cf008(param_4);
      pbVar11[-7] = bVar1;
      if (param_5 == '\0') {
        if (0x40 - *(int *)(param_4 + 0x38) < 0x20) {
          uVar15 = FUN_1406d6c7c(param_4,0x20);
        }
        else {
          *(int *)(param_4 + 0x2c) = *(int *)(param_4 + 0x2c) + 0x20;
          uVar15 = (undefined4)((ulonglong)*(longlong *)(param_4 + 0x30) >> 0x20);
          *(longlong *)(param_4 + 0x30) = *(longlong *)(param_4 + 0x30) << 0x20;
          *(int *)(param_4 + 0x38) = *(int *)(param_4 + 0x38) + 0x20;
        }
        *(undefined4 *)pbVar11 = uVar15;
      }
      else {
        FUN_1406d3140(extraout_XMM0_Da,param_4,1,pbVar11 + -4);
      }
      iVar10 = iVar10 + 1;
      pbVar11 = pbVar11 + 0xc;
    } while (iVar10 < *(int *)(param_3 + 0x34));
  }
  if (*(int *)(param_3 + 0xf8) == 1) {
LAB_14080c602:
    uVar8 = local_98;
    puVar12 = (uint *)(param_3 + 0x110);
    bVar14 = true;
    iVar10 = 0;
    do {
      if (0x40 - *(int *)(param_4 + 0x38) < 4) {
        uVar5 = FUN_1406d6c7c(param_4,4);
      }
      else {
        *(int *)(param_4 + 0x2c) = *(int *)(param_4 + 0x2c) + 4;
        uVar5 = (uint)((ulonglong)*(longlong *)(param_4 + 0x30) >> 0x3c);
        *(longlong *)(param_4 + 0x30) = *(longlong *)(param_4 + 0x30) << 4;
        *(int *)(param_4 + 0x38) = *(int *)(param_4 + 0x38) + 4;
      }
      puVar12[-5] = uVar5;
      cVar2 = FUN_1406cf008(param_4);
      if (cVar2 == '\0') {
        *(byte *)(puVar12 + -4) = (byte)puVar12[-4] & 0xfe;
      }
      else {
        *(byte *)(puVar12 + -4) = (byte)puVar12[-4] | 1;
        iVar6 = FUN_1406d310c(6);
        if (0x40 - *(int *)(param_4 + 0x38) < iVar6) {
          bVar1 = FUN_1406d6c7c(param_4,iVar6);
        }
        else {
          *(int *)(param_4 + 0x2c) = *(int *)(param_4 + 0x2c) + iVar6;
          bVar1 = (byte)(*(ulonglong *)(param_4 + 0x30) >> (0x40 - (byte)iVar6 & 0x3f));
          *(ulonglong *)(param_4 + 0x30) = *(ulonglong *)(param_4 + 0x30) << ((byte)iVar6 & 0x3f);
          *(int *)(param_4 + 0x38) = iVar6 + *(int *)(param_4 + 0x38);
        }
        bVar14 = bVar1 != 0;
        *(byte *)((longlong)puVar12 + -0xf) = bVar1;
        iVar6 = *(int *)(param_4 + 0x38);
        if (*(int *)(param_3 + 0x34) < 3) {
          if (0x40 - iVar6 < 1) {
            uVar9 = 1;
            goto LAB_14080c72e;
          }
          *(int *)(param_4 + 0x2c) = *(int *)(param_4 + 0x2c) + 1;
          uVar5 = (uint)((ulonglong)*(longlong *)(param_4 + 0x30) >> 0x3f);
          *(longlong *)(param_4 + 0x30) = *(longlong *)(param_4 + 0x30) * 2;
          iVar6 = iVar6 + 1;
LAB_14080c723:
          *(int *)(param_4 + 0x38) = iVar6;
        }
        else {
          if (3 < 0x40 - iVar6) {
            *(int *)(param_4 + 0x2c) = *(int *)(param_4 + 0x2c) + 4;
            uVar5 = (uint)((ulonglong)*(longlong *)(param_4 + 0x30) >> 0x3c);
            *(longlong *)(param_4 + 0x30) = *(longlong *)(param_4 + 0x30) << 4;
            iVar6 = iVar6 + 4;
            goto LAB_14080c723;
          }
          uVar9 = 4;
LAB_14080c72e:
          uVar5 = FUN_1406d6c7c(param_4,uVar9);
        }
        *puVar12 = uVar5;
        if (0x40 - *(int *)(param_4 + 0x38) < 0x10) {
          uVar4 = FUN_1406d6c7c(param_4,0x10);
        }
        else {
          *(int *)(param_4 + 0x2c) = *(int *)(param_4 + 0x2c) + 0x10;
          uVar4 = (undefined2)((ulonglong)*(longlong *)(param_4 + 0x30) >> 0x30);
          *(longlong *)(param_4 + 0x30) = *(longlong *)(param_4 + 0x30) << 0x10;
          *(int *)(param_4 + 0x38) = *(int *)(param_4 + 0x38) + 0x10;
        }
        *(undefined2 *)((longlong)puVar12 + -0xe) = uVar4;
        FUN_14102bd24(uVar8 & 0xffffffff,param_3[(longlong)(int)*puVar12 * 0xc + 0x38]);
        FUN_140c1e924(param_4,puVar12 + -3);
      }
      iVar10 = iVar10 + 1;
      puVar12 = puVar12 + 6;
    } while (iVar10 < *(int *)(param_3 + 0xf8));
    if (!bVar14) goto LAB_14080c88d;
  }
  else {
    local_98 = CONCAT44(local_98._4_4_,4);
    if (0 < *(int *)(param_3 + 0xf8)) goto LAB_14080c602;
  }
  cVar2 = '\0';
  local_res20[0] = '\0';
  if (param_5 == '\0') {
    cVar3 = FUN_1406cd5b8(param_4,param_3 + 0x2a0,param_3 + 700);
  }
  else {
    cVar3 = FUN_140c9e4d8(param_3 + 0x27c,param_4,0,local_res20);
    cVar2 = local_res20[0];
  }
  if ((cVar13 == '\0') || (cVar3 == '\0')) {
    cVar13 = '\0';
  }
  else {
    cVar13 = '\x01';
  }
  FUN_1408eff64(param_3 + 0x2c8,param_4,param_5);
  if (cVar2 == '\0') {
    if (0x40 - *(int *)(param_4 + 0x38) < 0x1e) {
      uVar5 = FUN_1406d6c7c(param_4,0x1e);
      uVar8 = (ulonglong)uVar5;
    }
    else {
      *(int *)(param_4 + 0x2c) = *(int *)(param_4 + 0x2c) + 0x1e;
      uVar8 = *(ulonglong *)(param_4 + 0x30) >> 0x22;
      *(ulonglong *)(param_4 + 0x30) = *(ulonglong *)(param_4 + 0x30) << 0x1e;
      *(int *)(param_4 + 0x38) = *(int *)(param_4 + 0x38) + 0x1e;
    }
    FUN_1406d8288(uVar8,param_3 + 0x28,0x1e);
    param_3[3] = 1;
  }
LAB_14080c88d:
  uVar15 = DAT_143cd8374;
  if (param_3[0x2dd] == 0) {
    cVar2 = FUN_1406cf008(param_4);
    if (cVar2 == '\0') {
      param_3[0x2e0] = 0;
      param_3[0x2e1] = 0;
      param_3[0x2e2] = 0;
      param_3[0x2e3] = 0;
    }
    else {
      if (0x40 - *(int *)(param_4 + 0x38) < 6) {
        uVar5 = FUN_1406d6c7c(param_4,6);
        uVar8 = (ulonglong)uVar5;
      }
      else {
        *(int *)(param_4 + 0x2c) = *(int *)(param_4 + 0x2c) + 6;
        uVar8 = *(ulonglong *)(param_4 + 0x30) >> 0x3a;
        *(ulonglong *)(param_4 + 0x30) = *(ulonglong *)(param_4 + 0x30) << 6;
        *(int *)(param_4 + 0x38) = *(int *)(param_4 + 0x38) + 6;
      }
      in_stack_ffffffffffffff38 = in_stack_ffffffffffffff38 & 0xffffff00;
      uVar16 = FUN_1406d8cb0(uVar8,0,uVar15,0x40,in_stack_ffffffffffffff38,1);
      *(undefined4 *)(param_3 + 0x2e0) = uVar16;
    }
    if (0x40 - *(int *)(param_4 + 0x38) < 6) {
      uVar5 = FUN_1406d6c7c(param_4,6);
      uVar8 = (ulonglong)uVar5;
    }
    else {
      *(int *)(param_4 + 0x2c) = *(int *)(param_4 + 0x2c) + 6;
      uVar8 = *(ulonglong *)(param_4 + 0x30) >> 0x3a;
      *(ulonglong *)(param_4 + 0x30) = *(ulonglong *)(param_4 + 0x30) << 6;
      *(int *)(param_4 + 0x38) = *(int *)(param_4 + 0x38) + 6;
    }
    uVar15 = FUN_1406d8cb0(uVar8,0,uVar15,0x40,in_stack_ffffffffffffff38 & 0xffffff00,1);
    *(undefined4 *)(param_3 + 0x2e4) = uVar15;
  }
  if (param_3[0x1c] == 1) {
    local_68 = 0;
    uStack_60 = 0;
    FUN_1431a0cbc(param_4,&local_68,param_3 + 0x2f4);
    *(undefined4 *)(param_3 + 0x2e8) = (undefined4)local_68;
    *(undefined4 *)(param_3 + 0x2ec) = local_68._4_4_;
    *(undefined4 *)(param_3 + 0x2f0) = (undefined4)uStack_60;
    uVar5 = FUN_141102ed0(0x24);
    if (1 < uVar5) {
      uVar15 = FUN_1406d84b4(param_4);
      *(undefined4 *)(param_3 + 0x2f8) = uVar15;
    }
  }
  pbVar11 = param_3 + 0x20;
  if (param_3[0x1c] == 0) {
    uVar4 = FUN_14080cb98(param_4);
    *(undefined2 *)(param_3 + 0x1e) = uVar4;
    FUN_14080cb50(pbVar11,param_4);
    if (*pbVar11 != 0) {
      FUN_14320c36c(param_3 + 0x2fc,param_4);
      FUN_142a40f18(param_3 + 0x304,param_4);
    }
  }
  else {
    param_3[0x1e] = 0xff;
    param_3[0x1f] = 0xff;
    *pbVar11 = 0;
    param_3[0x24] = 0;
    param_3[0x25] = 0;
    param_3[0x26] = 0;
    param_3[0x27] = 0;
  }
  if (0x40 - *(int *)(param_4 + 0x38) < 6) {
    uVar5 = FUN_1406d6c7c(param_4,6);
  }
  else {
    *(int *)(param_4 + 0x2c) = *(int *)(param_4 + 0x2c) + 6;
    uVar5 = (uint)((ulonglong)*(longlong *)(param_4 + 0x30) >> 0x3a);
    *(longlong *)(param_4 + 0x30) = *(longlong *)(param_4 + 0x30) << 6;
    *(int *)(param_4 + 0x38) = *(int *)(param_4 + 0x38) + 6;
  }
  *(uint *)(param_3 + 0x30c) = uVar5;
  bVar1 = FUN_1406cf008(param_4);
  param_3[0x310] = bVar1;
  if (bVar1 != 0) {
    uVar15 = FUN_1406d84b4(param_4);
    *(undefined4 *)(param_3 + 0x314) = uVar15;
  }
  if (param_5 == '\0') {
    bVar1 = FUN_1406cf008(param_4);
    param_3[4] = bVar1;
  }
  cVar2 = FUN_1406cf008(param_4);
  if (cVar2 != '\0') {
    FUN_14076e494(param_4,&local_90,0x10,0,param_5,0);
    uStack_74 = CONCAT31(uStack_74._1_3_,1);
    *(undefined8 *)(param_3 + 0x318) = local_90;
    *(ulonglong *)(param_3 + 800) = CONCAT44(uStack_74,local_88);
  }
  return cVar13;
}

