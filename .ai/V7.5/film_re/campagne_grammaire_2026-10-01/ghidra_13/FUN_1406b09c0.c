
undefined1 FUN_1406b09c0(uint *param_1,undefined8 *param_2)

{
  ulonglong *puVar1;
  undefined4 *puVar2;
  undefined4 uVar3;
  undefined8 uVar4;
  float fVar5;
  void *pvVar6;
  char cVar7;
  undefined1 uVar8;
  uint uVar9;
  int iVar10;
  longlong lVar11;
  ulonglong uVar12;
  longlong lVar13;
  undefined8 uVar14;
  undefined8 *puVar15;
  uint uVar16;
  int iVar17;
  undefined8 *puVar18;
  longlong lVar19;
  byte bVar20;
  uint uVar21;
  int iVar22;
  byte bVar23;
  uint local_res8 [2];
  byte local_res18 [16];
  undefined8 local_1e8;
  uint local_1e0;
  undefined8 uStack_1d8;
  undefined8 uStack_1d0;
  undefined8 uStack_1c8;
  undefined8 uStack_1c0;
  ulonglong uStack_1b8;
  undefined8 uStack_1b0;
  undefined8 uStack_1a8;
  undefined8 uStack_1a0;
  undefined8 uStack_198;
  undefined8 uStack_190;
  ulonglong uStack_188;
  undefined1 local_178;
  undefined1 uStack_177;
  undefined2 uStack_176;
  undefined4 uStack_174;
  undefined4 uStack_170;
  undefined4 uStack_16c;
  undefined8 local_168;
  ulonglong uStack_160;
  undefined8 local_158;
  undefined8 uStack_150;
  undefined8 local_148;
  undefined8 uStack_140;
  undefined8 local_138;
  undefined8 uStack_130;
  undefined4 local_128;
  undefined4 uStack_124;
  undefined4 uStack_120;
  undefined4 uStack_11c;
  undefined8 local_118;
  undefined1 local_108;
  undefined1 uStack_107;
  undefined6 uStack_106;
  ulonglong uStack_100;
  undefined8 local_f8;
  undefined4 uStack_f0;
  undefined4 uStack_ec;
  undefined4 uStack_e8;
  undefined4 uStack_e4;
  undefined4 uStack_e0;
  undefined4 uStack_dc;
  undefined8 local_d8;
  undefined4 uStack_d0;
  undefined4 uStack_cc;
  undefined4 local_c8;
  uint uStack_c4;
  undefined8 uStack_c0;
  undefined4 local_b8;
  undefined4 uStack_b4;
  undefined8 uStack_b0;
  undefined8 local_a8;
  undefined8 uStack_a0;
  undefined8 local_98;
  undefined8 uStack_90;
  undefined8 local_88;
  undefined8 uStack_80;
  undefined8 local_78;
  undefined8 uStack_70;
  undefined8 local_68;
  undefined4 uStack_60;
  undefined2 uStack_5c;
  undefined2 uStack_5a;
  undefined4 local_58;
  undefined4 uStack_54;
  undefined8 uStack_50;
  
  uVar9 = param_1[1];
  local_res8[0] = uVar9;
  lVar11 = FUN_140495b84(local_res8);
  iVar10 = -1;
  bVar20 = 0;
  local_res18[0] = 0;
  lVar13 = 1;
  if ((((1 < param_1[2]) || (iVar10 = -1, cVar7 = FUN_1405f1e0c(), cVar7 != '\0')) ||
      (DAT_1451d2628 != iVar10)) || (0 < (int)param_1[0x3d])) {
    param_1[0x36] = param_1[0x36] + (int)lVar13;
    *(ushort *)(param_1 + 0x37) = *(ushort *)((longlong)param_1 + 0xe2);
    uVar16 = param_1[0x36] * 0x19660d + 0x3c6ef35f >> 0x10 |
             (uint)*(ushort *)((longlong)param_1 + 0xe2) << 0x10;
    uVar21 = param_1[0x36] % 7 + (int)lVar13;
    uVar12 = (ulonglong)uVar21;
    if (uVar21 != 0) {
      do {
        uVar16 = uVar16 * 0x19660d + 0x3c6ef35f;
        uVar12 = uVar12 - lVar13;
      } while (uVar12 != 0);
    }
    param_1[0x38] = uVar16;
    if (*(short *)((longlong)param_1 + 0xe6) != -1) {
      *(ushort *)((longlong)param_1 + 0xe6) =
           *(short *)((longlong)param_1 + 0xe6) + (short)lVar13 & 0xff;
    }
  }
  if (param_1[2] == 0) {
LAB_1406b0ebb:
    FUN_140ae157c(&local_1e8,lVar11,0);
    *(undefined8 *)(param_1 + 0x10) = local_1e8;
    param_1[0x12] = local_1e0;
LAB_1406b0f0e:
    bVar23 = 0;
  }
  else {
    iVar22 = (int)lVar13;
    iVar17 = param_1[2] - iVar22;
    bVar23 = bVar20;
    if (iVar17 == 0) {
      cVar7 = FUN_1405f1e0c();
      if ((cVar7 == '\0') && (DAT_1451d2628 == iVar10)) {
        FUN_140ae157c(&local_1e8,lVar11,0);
        *(undefined8 *)(param_1 + 0x10) = local_1e8;
        param_1[0x12] = local_1e0;
        cVar7 = FUN_14048ee34();
        if (((cVar7 != '\0') && (lVar11 = FUN_1405d3b40(*(undefined8 *)(param_1 + 8)), lVar11 != 0))
           && (lVar11 = *(longlong *)(lVar11 + 0x10) + 0x1b908, lVar11 != 0)) {
          iVar10 = FUN_14049d198(uVar9);
          uStack_1a8 = CONCAT44(0xffffffff,(undefined4)uStack_1a8);
          uStack_1a0 = CONCAT44(uStack_1a0._4_4_,0xffffffff);
          uStack_1b8 = uStack_1b8 & 0xffffffff00000000;
          uStack_1a8 = uStack_1a8 & 0xffffffffffffff00;
          uStack_188 = (uStack_188 >> 0x10 & 0xffff) << 0x10;
          cVar7 = FUN_142f2a5e0(lVar11,*param_1,&uStack_1d8,0);
          if ((cVar7 != '\0') && (iVar10 != -1)) {
            FUN_142bd3ab4(iVar10,&uStack_1d8);
            uVar8 = FUN_1406b0f18();
            return uVar8;
          }
        }
      }
      else {
        lVar11 = FUN_1405d3b40(*(undefined8 *)(param_1 + 8));
        if ((lVar11 != 0) && (lVar11 = *(longlong *)(lVar11 + 0x10) + 0x1b908, lVar11 != 0)) {
          cVar7 = FUN_14048ee34();
          if (cVar7 == '\0') {
            FUN_14076a280(&local_178);
            cVar7 = FUN_14076b058(lVar11,*param_1,&local_178);
            if (cVar7 != '\0') {
              local_b8 = 0xffffffff;
              uStack_d0 = 0;
              uStack_c4 = 0;
              uStack_b4 = 0;
              uStack_5c = 0;
              local_58 = 0;
              FUN_1407ff8cc(&local_178);
              *(ulonglong *)(param_1 + 0xc) = CONCAT62(uStack_106,CONCAT11(uStack_107,local_108));
              *(ulonglong *)(param_1 + 0xe) = uStack_100;
              *(undefined8 *)(param_1 + 0x10) = local_f8;
              *(ulonglong *)(param_1 + 0x12) = CONCAT44(uStack_ec,uStack_f0);
              *(ulonglong *)(param_1 + 0x14) = CONCAT44(uStack_e4,uStack_e8);
              *(ulonglong *)(param_1 + 0x16) = CONCAT44(uStack_dc,uStack_e0);
              *(undefined8 *)(param_1 + 0x18) = local_d8;
              *(ulonglong *)(param_1 + 0x1a) = CONCAT44(uStack_cc,uStack_d0);
              *(ulonglong *)(param_1 + 0x1c) = CONCAT44(uStack_c4,local_c8);
              *(undefined8 *)(param_1 + 0x1e) = uStack_c0;
              *(ulonglong *)(param_1 + 0x20) = CONCAT44(uStack_b4,local_b8);
              *(undefined8 *)(param_1 + 0x22) = uStack_b0;
              *(undefined8 *)(param_1 + 0x24) = local_a8;
              *(undefined8 *)(param_1 + 0x26) = uStack_a0;
              *(undefined8 *)(param_1 + 0x28) = local_98;
              *(undefined8 *)(param_1 + 0x2a) = uStack_90;
              *(undefined8 *)(param_1 + 0x2c) = local_88;
              *(undefined8 *)(param_1 + 0x2e) = uStack_80;
              *(undefined8 *)(param_1 + 0x30) = local_78;
              *(undefined8 *)(param_1 + 0x32) = uStack_70;
              *(undefined8 *)(param_1 + 0x34) = local_68;
              *(ulonglong *)(param_1 + 0x36) = CONCAT26(uStack_5a,CONCAT24(uStack_5c,uStack_60));
              *(ulonglong *)(param_1 + 0x38) = CONCAT44(uStack_54,local_58);
              *(undefined8 *)(param_1 + 0x3a) = uStack_50;
              goto LAB_1422f15fc;
            }
          }
          else {
            uStack_1a8 = CONCAT44(0xffffffff,(undefined4)uStack_1a8);
            uStack_1a0 = CONCAT44(uStack_1a0._4_4_,0xffffffff);
            uStack_1b8 = uStack_1b8 & 0xffffffff00000000;
            uStack_1a8 = uStack_1a8 & 0xffffffffffffff00;
            uStack_188 = (uStack_188 >> 0x10 & 0xffff) << 0x10;
            cVar7 = FUN_142f2a5e0(lVar11,*param_1,&uStack_1d8,0);
            if (cVar7 != '\0') {
              FUN_142bca84c(&uStack_1d8,param_1 + 0xc);
LAB_1422f15fc:
              param_1[0x3c] =
                   *(uint *)(*(longlong *)(*(longlong *)ThreadLocalStoragePointer + 0x6c0) + 0x40);
              goto LAB_1406b0d15;
            }
          }
        }
      }
      goto LAB_1406b0f0e;
    }
    iVar17 = iVar17 - iVar22;
    if (iVar17 == 0) goto LAB_1406b0ebb;
    if ((iVar17 != 2) && (iVar17 + -2 != iVar22)) goto LAB_1406b0f0e;
    if (*(int *)(*(longlong *)(param_1 + 8) + 0x10) != 4) {
      cVar7 = FUN_1404f2b4c();
      if (cVar7 == '\0') {
        lVar13 = FUN_141f864ac(*(longlong *)(param_1 + 8),param_1 + 6);
      }
      else {
        lVar13 = *(longlong *)(*(longlong *)(param_1 + 8) + 0xd8);
      }
      if (lVar13 != 0) {
        FUN_1405f0adc(&DAT_144de3ea0,*(undefined4 *)(lVar13 + 0x2c));
        goto LAB_1406b0afb;
      }
      goto LAB_1406b0f0e;
    }
    lVar13 = FUN_1405d3b40();
    if (lVar13 == 0) goto LAB_1406b0f0e;
LAB_1406b0afb:
    lVar13 = *(longlong *)(lVar13 + 0x10);
    lVar19 = lVar13 + 0x1b908;
    if (lVar19 == 0) goto LAB_1406b0f0e;
    FUN_14076a280(&local_178);
    uVar9 = *param_1;
    cVar7 = FUN_14048ee34();
    if (cVar7 == '\0') {
      if ((*(uint *)(lVar13 + 0x1de5c) >> (uVar9 & 0x1f) & 1) == 0) goto LAB_1406b0f0e;
      local_b8 = 0xffffffff;
      lVar11 = (longlong)(int)uVar9 * 0x68;
      uStack_d0 = 0;
      uStack_c4 = 0;
      uStack_b4 = 0;
      uStack_5c = 0;
      puVar15 = (undefined8 *)(lVar11 + 0x25f0 + lVar19);
      uVar14 = *puVar15;
      uVar4 = puVar15[1];
      local_58 = 0;
      puVar1 = (ulonglong *)(lVar11 + 0x2600 + lVar19);
      local_168 = *puVar1;
      uStack_160 = puVar1[1];
      local_178 = (undefined1)uVar14;
      uStack_177 = (undefined1)((ulonglong)uVar14 >> 8);
      uStack_176 = (undefined2)((ulonglong)uVar14 >> 0x10);
      uStack_174 = (undefined4)((ulonglong)uVar14 >> 0x20);
      uStack_170 = (undefined4)uVar4;
      uStack_16c = (undefined4)((ulonglong)uVar4 >> 0x20);
      puVar15 = (undefined8 *)(lVar11 + 0x2610 + lVar19);
      local_158 = *puVar15;
      uStack_150 = puVar15[1];
      puVar15 = (undefined8 *)(lVar11 + 0x2620 + lVar19);
      local_148 = *puVar15;
      uStack_140 = puVar15[1];
      puVar15 = (undefined8 *)(lVar11 + 0x2630 + lVar19);
      local_138 = *puVar15;
      uStack_130 = puVar15[1];
      puVar2 = (undefined4 *)(lVar11 + 0x2640 + lVar19);
      local_128 = *puVar2;
      uStack_124 = puVar2[1];
      uStack_120 = puVar2[2];
      uStack_11c = puVar2[3];
      local_118 = *(undefined8 *)(lVar11 + 0x2650 + lVar19);
      *(uint *)(lVar13 + 0x1de5c) = *(uint *)(lVar13 + 0x1de5c) & ~(1 << (uVar9 & 0x1f));
      FUN_1408008cc(&local_108);
      local_108 = local_178;
      uStack_107 = uStack_177;
      uStack_ec = uStack_174;
      uStack_e8 = uStack_170;
      uStack_100 = (ulonglong)(local_168._4_2_ & 1);
      uStack_e4 = uStack_16c;
      uStack_dc = (undefined4)local_168;
      if ((local_168 & 0x400000000) != 0) {
        uStack_100 = uStack_100 | 0x80002;
      }
      if ((local_168 & 0x800000000) != 0) {
        uStack_100 = uStack_100 | 0x40200;
      }
      if ((local_168 & 0x1000000000) != 0) {
        uStack_100 = uStack_100 | 0x100000;
      }
      if ((local_168 & 0x200000000) != 0) {
        uStack_100 = uStack_100 | 0x200000;
      }
      if ((local_168 & 0x1000000000) != 0) {
        uStack_100 = uStack_100 | 0x400;
      }
      uStack_100 = uStack_100 | 0xc00000;
      uVar9 = uStack_c4 & 0xfffffffe;
      uStack_c4 = uStack_c4 | 1;
      if ((local_168 & 0x800000000) == 0) {
        uStack_c4 = uVar9;
      }
      FUN_1406dc5b8(&uStack_160,&uStack_cc,&uStack_100,(longlong)&local_98 + 4,&local_78,0);
      *(ulonglong *)(param_1 + 0xc) = CONCAT62(uStack_106,CONCAT11(uStack_107,local_108));
      *(ulonglong *)(param_1 + 0xe) = uStack_100;
      *(undefined8 *)(param_1 + 0x10) = local_f8;
      *(ulonglong *)(param_1 + 0x12) = CONCAT44(uStack_ec,uStack_f0);
      *(ulonglong *)(param_1 + 0x14) = CONCAT44(uStack_e4,uStack_e8);
      *(ulonglong *)(param_1 + 0x16) = CONCAT44(uStack_dc,uStack_e0);
      *(undefined8 *)(param_1 + 0x18) = local_d8;
      *(ulonglong *)(param_1 + 0x1a) = CONCAT44(uStack_cc,uStack_d0);
      *(ulonglong *)(param_1 + 0x1c) = CONCAT44(uStack_c4,local_c8);
      *(undefined8 *)(param_1 + 0x1e) = uStack_c0;
      *(ulonglong *)(param_1 + 0x20) = CONCAT44(uStack_b4,local_b8);
      *(undefined8 *)(param_1 + 0x22) = uStack_b0;
      *(undefined8 *)(param_1 + 0x24) = local_a8;
      *(undefined8 *)(param_1 + 0x26) = uStack_a0;
      *(undefined8 *)(param_1 + 0x28) = local_98;
      *(undefined8 *)(param_1 + 0x2a) = uStack_90;
      *(undefined8 *)(param_1 + 0x2c) = local_88;
      *(undefined8 *)(param_1 + 0x2e) = uStack_80;
      *(undefined8 *)(param_1 + 0x30) = local_78;
      *(undefined8 *)(param_1 + 0x32) = uStack_70;
      pvVar6 = ThreadLocalStoragePointer;
      *(undefined8 *)(param_1 + 0x34) = local_68;
      *(ulonglong *)(param_1 + 0x36) = CONCAT26(uStack_5a,CONCAT24(uStack_5c,uStack_60));
      *(ulonglong *)(param_1 + 0x38) = CONCAT44(uStack_54,local_58);
      *(undefined8 *)(param_1 + 0x3a) = uStack_50;
      param_1[0x3c] = *(uint *)(*(longlong *)(*(longlong *)pvVar6 + 0x6c0) + 0x40);
    }
    else {
      uStack_1b8 = uStack_1b8 & 0xffffffff00000000;
      uStack_1a8 = CONCAT44(0xffffffff,(undefined4)uStack_1a8);
      uStack_1a0 = CONCAT44(uStack_1a0._4_4_,0xffffffff);
      uStack_1a8 = uStack_1a8 & 0xffffffffffffff00;
      uStack_188 = (uStack_188 >> 0x10 & 0xffff) << 0x10;
      cVar7 = FUN_142f2a5e0(lVar19,uVar9,&uStack_1d8,local_res18);
      if (cVar7 == '\0') {
        cVar7 = FUN_142f2864c(lVar19,uVar9,&uStack_1d8);
        if (cVar7 != '\0') {
          uStack_1d8 = CONCAT44(uStack_1d8._4_4_,param_1[0x36]);
          uStack_190._0_6_ =
               CONCAT24(*(undefined2 *)((longlong)param_1 + 0xe6),(undefined4)uStack_190);
          uStack_188 = *(ulonglong *)(param_1 + 0x37);
          FUN_142bca84c(&uStack_1d8,param_1 + 0xc);
        }
        bVar20 = *(byte *)(lVar11 + 0x326);
        bVar23 = 1;
        if (bVar20 < (byte)(bVar20 + 1)) {
          bVar20 = bVar20 + 1;
        }
      }
      else {
        FUN_142bca84c(&uStack_1d8,param_1 + 0xc);
        bVar23 = 0;
        param_1[0x3c] =
             *(uint *)(*(longlong *)(*(longlong *)ThreadLocalStoragePointer + 0x6c0) + 0x40);
        bVar20 = local_res18[0];
      }
      FUN_1404f293c();
    }
  }
LAB_1406b0d15:
  fVar5 = DAT_144989f60;
  if (param_1[2] == 2) {
    return 0;
  }
  if ((DAT_145173378 == '\0') || (cVar7 = FUN_1404f1760(), cVar7 == '\0')) {
LAB_1406b0d34:
    if ((param_1[0x3c] == 0xffffffff) ||
       (fVar5 <= (float)(int)(*(int *)(*(longlong *)(*(longlong *)ThreadLocalStoragePointer + 0x6c0)
                                      + 0x40) - param_1[0x3c]) *
                 *(float *)(*(longlong *)(*(longlong *)ThreadLocalStoragePointer + 0x6c0) + 0x78)))
    goto LAB_1406b0f1d;
  }
  else {
    uVar14 = FUN_1404f2650();
    lVar11 = FUN_1406aed80(uVar14);
    if (*(char *)(lVar11 + 0xe3644) == '\0') goto LAB_1406b0d34;
  }
  cVar7 = FUN_1406d4db8(param_1 + 0xc,1);
  if (cVar7 == '\0') {
LAB_1406b0f1d:
    FUN_1406d4db8(param_1 + 0xc,1);
    return 0;
  }
  if ((DAT_145173378 != '\0') && (cVar7 = FUN_1404f1760(), cVar7 != '\0')) {
    uVar14 = FUN_1404f2650();
    lVar11 = FUN_1406aed80(uVar14);
    if (*(char *)(lVar11 + 0xe3644) != '\0') {
      puVar15 = (undefined8 *)FUN_142bca92c(&local_178,param_1 + 0xc);
      uStack_1d8 = *puVar15;
      uStack_1d0 = puVar15[1];
      uStack_1c8 = puVar15[2];
      uStack_1c0 = puVar15[3];
      uStack_1b8 = puVar15[4];
      uStack_1b0 = puVar15[5];
      uStack_1a8 = puVar15[6];
      uStack_1a0 = puVar15[7];
      uStack_198 = puVar15[8];
      uStack_190 = puVar15[9];
      uStack_188 = puVar15[10];
      FUN_142bca84c(&uStack_1d8,param_2);
      *(byte *)((longlong)param_2 + 0xb4) = bVar20;
      *(byte *)(param_2 + 0x17) = bVar23;
      iVar10 = FUN_1404f16fc();
      if (iVar10 == 1) {
        puVar15 = (undefined8 *)((longlong)param_1 + 0xa7U & 0xfffffffffffffffc);
        puVar18 = (undefined8 *)((longlong)param_2 + 0x77U & 0xfffffffffffffffc);
        uVar3 = *(undefined4 *)(puVar15 + 1);
        *puVar18 = *puVar15;
        *(undefined4 *)(puVar18 + 1) = uVar3;
      }
      goto LAB_1406b0e23;
    }
  }
  uVar14 = *(undefined8 *)(param_1 + 0xe);
  *param_2 = *(undefined8 *)(param_1 + 0xc);
  param_2[1] = uVar14;
  uVar14 = *(undefined8 *)(param_1 + 0x12);
  param_2[2] = *(undefined8 *)(param_1 + 0x10);
  param_2[3] = uVar14;
  uVar14 = *(undefined8 *)(param_1 + 0x16);
  param_2[4] = *(undefined8 *)(param_1 + 0x14);
  param_2[5] = uVar14;
  uVar14 = *(undefined8 *)(param_1 + 0x1a);
  param_2[6] = *(undefined8 *)(param_1 + 0x18);
  param_2[7] = uVar14;
  uVar14 = *(undefined8 *)(param_1 + 0x1e);
  param_2[8] = *(undefined8 *)(param_1 + 0x1c);
  param_2[9] = uVar14;
  uVar14 = *(undefined8 *)(param_1 + 0x22);
  param_2[10] = *(undefined8 *)(param_1 + 0x20);
  param_2[0xb] = uVar14;
  uVar14 = *(undefined8 *)(param_1 + 0x26);
  param_2[0xc] = *(undefined8 *)(param_1 + 0x24);
  param_2[0xd] = uVar14;
  uVar14 = *(undefined8 *)(param_1 + 0x2a);
  param_2[0xe] = *(undefined8 *)(param_1 + 0x28);
  param_2[0xf] = uVar14;
  uVar14 = *(undefined8 *)(param_1 + 0x2e);
  param_2[0x10] = *(undefined8 *)(param_1 + 0x2c);
  param_2[0x11] = uVar14;
  uVar14 = *(undefined8 *)(param_1 + 0x32);
  param_2[0x12] = *(undefined8 *)(param_1 + 0x30);
  param_2[0x13] = uVar14;
  uVar14 = *(undefined8 *)(param_1 + 0x36);
  param_2[0x14] = *(undefined8 *)(param_1 + 0x34);
  param_2[0x15] = uVar14;
  uVar14 = *(undefined8 *)(param_1 + 0x3a);
  param_2[0x16] = *(undefined8 *)(param_1 + 0x38);
  param_2[0x17] = uVar14;
LAB_1406b0e23:
  param_1[0x3d] = param_1[0x3d] + 1;
  *(ulonglong *)(param_1 + 0xe) = *(ulonglong *)(param_1 + 0xe) & 0xfffff7ffffffffff;
  return 1;
}

