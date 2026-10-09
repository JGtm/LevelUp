
/* WARNING: Removing unreachable block (ram,0x00014076b19a) */
/* WARNING: Removing unreachable block (ram,0x00014230bf83) */
/* WARNING: Removing unreachable block (ram,0x00014230bfd2) */
/* WARNING: Removing unreachable block (ram,0x00014230bfe6) */
/* WARNING: Removing unreachable block (ram,0x00014230bff1) */
/* WARNING: Removing unreachable block (ram,0x00014230c00c) */
/* WARNING: Removing unreachable block (ram,0x00014230c012) */
/* WARNING: Removing unreachable block (ram,0x00014230bfff) */
/* WARNING: Removing unreachable block (ram,0x00014230c031) */
/* WARNING: Removing unreachable block (ram,0x00014230bfb8) */
/* WARNING: Removing unreachable block (ram,0x00014230c035) */
/* WARNING: Removing unreachable block (ram,0x00014230c03e) */
/* WARNING: Removing unreachable block (ram,0x00014230c050) */

undefined4 * FUN_14076b0e8(longlong param_1,uint *param_2,uint *param_3)

{
  longlong lVar1;
  byte bVar2;
  ulonglong *puVar3;
  undefined8 *puVar4;
  undefined4 uVar5;
  undefined4 uVar6;
  char cVar7;
  undefined4 uVar8;
  int iVar9;
  int iVar10;
  undefined4 uVar11;
  int iVar12;
  longlong lVar13;
  longlong lVar14;
  longlong lVar15;
  undefined4 *puVar16;
  undefined4 *puVar17;
  ulonglong uVar18;
  uint uVar19;
  uint uVar20;
  uint uVar21;
  longlong lVar22;
  undefined4 *puVar23;
  undefined4 extraout_XMM0_Da;
  undefined4 extraout_XMM0_Da_00;
  undefined4 extraout_XMM0_Da_01;
  undefined4 extraout_XMM0_Da_02;
  char local_res10;
  undefined8 in_stack_ffffffffffffff08;
  undefined8 uVar25;
  undefined8 local_b8;
  undefined8 uStack_b0;
  undefined8 local_a8;
  undefined8 uStack_a0;
  undefined4 local_98;
  undefined4 uStack_94;
  undefined4 uStack_90;
  undefined4 uStack_8c;
  undefined4 local_88;
  undefined4 uStack_84;
  undefined4 uStack_80;
  undefined4 uStack_7c;
  undefined1 local_78 [64];
  undefined4 *puVar24;
  
  uVar11 = (undefined4)((ulonglong)in_stack_ffffffffffffff08 >> 0x20);
  puVar24 = (undefined4 *)0x0;
  puVar23 = (undefined4 *)0x0;
  if (param_2[1] == 0) {
    uVar19 = *param_2;
    lVar22 = (longlong)(int)uVar19;
    uVar8 = FUN_14076c748(**(undefined8 **)(param_2 + 2));
    cVar7 = FUN_140769e58(uVar8);
    lVar13 = DAT_144de4b78;
    if (cVar7 == '\0') {
      lVar13 = DAT_145178b58;
    }
    lVar15 = *(longlong *)(lVar22 * 0x4c8 + 0x480 + lVar13);
    lVar13 = lVar13 + lVar22 * 0x4c8;
    puVar23 = (undefined4 *)(lVar13 + 0x488);
    bVar2 = *(byte *)(lVar22 + 0x5a10 + param_1);
    FUN_1404f1ca4();
    if ((bVar2 != 0xff) &&
       (lVar1 = param_1 + (lVar22 * 5 + 0xa02) * 8,
       *(longlong *)
        (*(longlong *)(lVar1 + 8) +
        ((ulonglong)bVar2 + *(longlong *)(lVar1 + 0x18) & *(longlong *)(lVar1 + 0x10) - 1U) * 8) !=
       0)) {
      puVar23 = (undefined4 *)FUN_14230befc();
      return puVar23;
    }
    iVar12 = 5;
    if (*(int *)(lVar15 + 0x20) != 1 && *(int *)(lVar15 + 0x20) != 2) {
      *(undefined8 *)(lVar15 + 0x40) = *(undefined8 *)(lVar15 + 8);
      *(undefined4 *)(lVar15 + 0x38) = 0;
      *(undefined4 *)(lVar15 + 0x1c) = 1;
      *(undefined4 *)(lVar15 + 0x20) = 1;
      *(undefined8 *)(lVar15 + 0x28) = 0;
      *(undefined4 *)(lVar15 + 0x48) = 0;
      *(undefined1 *)(lVar15 + 0x24) = 0;
      *(undefined8 *)(lVar15 + 0x30) = 0;
      *(undefined8 *)(lVar15 + 0xd0) = 0;
      *puVar23 = 0;
      *(undefined2 *)(lVar13 + 0x48c) = 3;
      *(undefined1 *)(lVar13 + 0x48e) = 0xff;
      if (*(uint *)(lVar15 + 0x38) < 0x40) {
        *(int *)(lVar15 + 0x2c) = *(int *)(lVar15 + 0x2c) + 1;
        *(uint *)(lVar15 + 0x38) = *(uint *)(lVar15 + 0x38) + 1;
        *(ulonglong *)(lVar15 + 0x30) = *(longlong *)(lVar15 + 0x30) * 2 | 1;
      }
      else {
        iVar12 = 5;
        FUN_1406d6e28(lVar15,1,1);
      }
      FUN_1411b13fc(lVar15);
      if (*(uint *)(lVar15 + 0x38) < 0x40) {
        *(int *)(lVar15 + 0x2c) = *(int *)(lVar15 + 0x2c) + 1;
        *(longlong *)(lVar15 + 0x30) = *(longlong *)(lVar15 + 0x30) << 1;
        *(uint *)(lVar15 + 0x38) = *(uint *)(lVar15 + 0x38) + 1;
      }
      else {
        FUN_1406d6e28(lVar15,0,1);
      }
      *puVar23 = *(undefined4 *)(lVar15 + 0x2c);
      iVar9 = 0x40 - *(int *)(lVar15 + 0x38);
      uVar18 = *(ulonglong *)(lVar15 + 0x30);
      *(int *)(lVar15 + 0x2c) = *(int *)(lVar15 + 0x2c) + iVar12;
      if (iVar9 < iVar12) {
        uVar21 = iVar12 - iVar9;
        *(ulonglong *)(lVar15 + 0x30) = (ulonglong)uVar19;
        *(uint *)(lVar15 + 0x38) = uVar21;
        if (uVar21 < 0x40) {
          uVar18 = (ulonglong)(uVar19 >> ((byte)uVar21 & 0x3f)) | uVar18 << ((byte)iVar9 & 0x3f);
        }
        puVar3 = *(ulonglong **)(lVar15 + 0x40);
        if (*(ulonglong **)(lVar15 + 0x10) < puVar3 + 1) {
          if (puVar3 < *(ulonglong **)(lVar15 + 0x10)) {
            do {
              **(undefined1 **)(lVar15 + 0x40) = (char)(uVar18 >> 0x38);
              *(longlong *)(lVar15 + 0x40) = *(longlong *)(lVar15 + 0x40) + 1;
              uVar18 = uVar18 << 8;
            } while (*(ulonglong *)(lVar15 + 0x40) < *(ulonglong *)(lVar15 + 0x10));
          }
        }
        else {
          *puVar3 = uVar18 >> 0x38 | (uVar18 & 0xff000000000000) >> 0x28 |
                    (uVar18 & 0xff0000000000) >> 0x18 | (uVar18 & 0xff00000000) >> 8 |
                    (uVar18 & 0xff000000) << 8 | (uVar18 & 0xff0000) << 0x18 |
                    (uVar18 & 0xff00) << 0x28 | uVar18 << 0x38;
          *(longlong *)(lVar15 + 0x40) = *(longlong *)(lVar15 + 0x40) + 8;
        }
        *(int *)(lVar15 + 0x28) = *(int *)(lVar15 + 0x28) + 0x40;
      }
      else {
        *(int *)(lVar15 + 0x38) = *(int *)(lVar15 + 0x38) + 5;
        *(ulonglong *)(lVar15 + 0x30) = uVar18 << 5 | (ulonglong)uVar19;
      }
      cVar7 = FUN_14048ee34();
      if (cVar7 == '\0') {
        FUN_140769f90(param_1,lVar15,puVar23,uVar19);
        uVar25 = CONCAT44(uVar11,uVar8);
        FUN_14076a064(param_1,lVar15,puVar23,uVar19,uVar25);
        uVar11 = (undefined4)((ulonglong)uVar25 >> 0x20);
      }
      else {
        cVar7 = FUN_1404f25f4();
        if (cVar7 != '\0') {
          FUN_142f2cc28(param_1,lVar15,puVar23,uVar19);
        }
        uVar11 = 0;
        FUN_142f2c164(param_1,lVar15,puVar23,uVar19,0);
      }
      FUN_1406d6d94(lVar15);
    }
    FUN_14047bdd0(&local_b8,8,8,FUN_141161570);
    lVar1 = param_1 + 0x6088;
    uVar8 = *(undefined4 *)(param_1 + 0x60b4);
    uVar5 = *(undefined4 *)(param_1 + 0x60b8);
    uVar6 = *(undefined4 *)(param_1 + 0x60bc);
    lVar14 = (longlong)*(int *)(param_1 + 0x60d0) * 0x20;
    puVar23 = (undefined4 *)(lVar14 + 0x50 + lVar1);
    *puVar23 = *(undefined4 *)(param_1 + 0x60b0);
    puVar23[1] = uVar8;
    puVar23[2] = uVar5;
    puVar23[3] = uVar6;
    uVar8 = *(undefined4 *)(param_1 + 0x60c4);
    uVar5 = *(undefined4 *)(param_1 + 0x60c8);
    uVar6 = *(undefined4 *)(param_1 + 0x60cc);
    puVar23 = (undefined4 *)(lVar14 + 0x60 + lVar1);
    *puVar23 = *(undefined4 *)(param_1 + 0x60c0);
    puVar23[1] = uVar8;
    puVar23[2] = uVar5;
    puVar23[3] = uVar6;
    *(int *)(param_1 + 0x60d0) = *(int *)(param_1 + 0x60d0) + 1;
    iVar9 = FUN_14076b9b0(lVar1);
    local_b8 = *(undefined8 *)(lVar13 + 0x488);
    uStack_b0 = *(undefined8 *)(lVar13 + 0x490);
    local_a8 = *(undefined8 *)(lVar13 + 0x498);
    uStack_a0 = *(undefined8 *)(lVar13 + 0x4a0);
    local_98 = *(undefined4 *)(lVar13 + 0x4a8);
    uStack_94 = *(undefined4 *)(lVar13 + 0x4ac);
    uStack_90 = *(undefined4 *)(lVar13 + 0x4b0);
    uStack_8c = *(undefined4 *)(lVar13 + 0x4b4);
    local_88 = *(undefined4 *)(lVar13 + 0x4b8);
    uStack_84 = *(undefined4 *)(lVar13 + 0x4bc);
    uStack_80 = *(undefined4 *)(lVar13 + 0x4c0);
    uStack_7c = *(undefined4 *)(lVar13 + 0x4c4);
    FUN_1406d5d14(local_98,lVar15);
    iVar10 = FUN_14076b9b0(lVar1);
    *param_3 = iVar10 - iVar9;
    iVar12 = *(int *)(param_1 + 0x60d0);
    uVar21 = param_2[9];
    *(int *)(param_1 + 0x60d0) = iVar12 + -1;
    if ((int)uVar21 < iVar10 - iVar9) {
      lVar13 = (longlong)iVar12 * 0x20;
      puVar4 = (undefined8 *)(lVar13 + 0x30 + lVar1);
      uVar25 = puVar4[1];
      *(undefined8 *)(param_1 + 0x60b0) = *puVar4;
      *(undefined8 *)(param_1 + 0x60b8) = uVar25;
      puVar4 = (undefined8 *)(lVar13 + 0x40 + lVar1);
      uVar25 = puVar4[1];
      *(undefined8 *)(param_1 + 0x60c0) = *puVar4;
      *(undefined8 *)(param_1 + 0x60c8) = uVar25;
      puVar23 = puVar24;
    }
    else {
      cVar7 = FUN_14048ee34();
      if (cVar7 == '\0') {
        *(uint *)(param_1 + 0x2550) = *(uint *)(param_1 + 0x2550) & ~(1 << (uVar19 & 0x1f));
        *(uint *)(param_1 + 0x2558) = *(uint *)(param_1 + 0x2558) & ~(1 << (uVar19 & 0x1f));
      }
      else {
        *(uint *)(param_1 + 0x1f44) = *(uint *)(param_1 + 0x1f44) & ~(1 << (uVar19 & 0x1f));
        lVar15 = FUN_142f28458(param_1 + 0x4fe8,param_2[4],uVar19);
        lVar13 = param_1 + 0x4a70 + lVar22 * 0x28;
        if (*(longlong *)(param_1 + 0x4a90 + lVar22 * 0x28) != 0) {
          puVar4 = *(undefined8 **)
                    (*(longlong *)(param_1 + 0x4a78 + lVar22 * 0x28) +
                    (*(longlong *)(param_1 + 0x4a80 + lVar22 * 0x28) - 1U &
                    *(ulonglong *)(param_1 + 0x4a88 + lVar22 * 0x28)) * 8);
          uVar25 = puVar4[1];
          *(undefined8 *)(lVar15 + 4) = *puVar4;
          *(undefined8 *)(lVar15 + 0xc) = uVar25;
          uVar25 = puVar4[3];
          *(undefined8 *)(lVar15 + 0x14) = puVar4[2];
          *(undefined8 *)(lVar15 + 0x1c) = uVar25;
          uVar25 = puVar4[5];
          *(undefined8 *)(lVar15 + 0x24) = puVar4[4];
          *(undefined8 *)(lVar15 + 0x2c) = uVar25;
          uVar25 = puVar4[7];
          *(undefined8 *)(lVar15 + 0x34) = puVar4[6];
          *(undefined8 *)(lVar15 + 0x3c) = uVar25;
          uVar8 = *(undefined4 *)((longlong)puVar4 + 0x44);
          uVar5 = *(undefined4 *)(puVar4 + 9);
          uVar6 = *(undefined4 *)((longlong)puVar4 + 0x4c);
          *(undefined4 *)(lVar15 + 0x44) = *(undefined4 *)(puVar4 + 8);
          *(undefined4 *)(lVar15 + 0x48) = uVar8;
          *(undefined4 *)(lVar15 + 0x4c) = uVar5;
          *(undefined4 *)(lVar15 + 0x50) = uVar6;
          *(undefined8 *)(lVar15 + 0x54) = puVar4[10];
          *(byte *)(lVar15 + 0xb4) = *(byte *)(lVar15 + 0xb4) | 1;
        }
        cVar7 = FUN_1404f1ca4();
        if (cVar7 == '\0') {
          lVar15 = *(longlong *)(param_1 + 0x4a90 + lVar22 * 0x28);
          if (lVar15 == 0) {
            uVar8 = 0xffffffff;
          }
          else {
            uVar8 = **(undefined4 **)
                      (*(longlong *)(lVar13 + 8) +
                      (*(longlong *)(lVar13 + 0x10) - 1U &
                      (*(longlong *)(param_1 + 0x4a88 + lVar22 * 0x28) + lVar15) - 1U) * 8);
          }
          FUN_142f23098(param_1 + 0x4f70,param_2[4],uVar19,uVar8,CONCAT44(uVar11,0xffffffff));
        }
      }
      uVar11 = FUN_1405f50b8();
      *(undefined4 *)(param_1 + 0x2570 + lVar22 * 4) = uVar11;
      cVar7 = FUN_14048ee34();
      if (cVar7 != '\0') {
        FUN_1404f25f4();
      }
      puVar23 = (undefined4 *)(ulonglong)*param_3;
    }
  }
  else if (param_2[1] == 1) {
    uVar19 = *param_2;
    lVar13 = (longlong)(int)uVar19;
    uVar11 = FUN_14076c748(**(undefined8 **)(param_2 + 2));
    FUN_140769e58(uVar11);
    puVar16 = (undefined4 *)FUN_140769e74(param_1 + 0x4fe8,uVar19);
    puVar23 = puVar24;
    if ((puVar16 == (undefined4 *)0x0) || (puVar17 = puVar16, lVar22 = FUN_142f2b9d8(), lVar22 == 0)
       ) {
      local_res10 = -1;
    }
    else {
      local_res10 = FUN_140c72400(param_2[4],*puVar17);
      if (local_res10 != -1) {
        puVar23 = puVar16;
      }
    }
    FUN_1424cbdf0(local_78);
    lVar22 = param_1 + 0x6088;
    uVar11 = *(undefined4 *)(param_1 + 0x60b4);
    uVar8 = *(undefined4 *)(param_1 + 0x60b8);
    uVar5 = *(undefined4 *)(param_1 + 0x60bc);
    lVar15 = (longlong)*(int *)(param_1 + 0x60d0) * 0x20;
    puVar16 = (undefined4 *)(lVar15 + 0x50 + lVar22);
    *puVar16 = *(undefined4 *)(param_1 + 0x60b0);
    puVar16[1] = uVar11;
    puVar16[2] = uVar8;
    puVar16[3] = uVar5;
    uVar11 = *(undefined4 *)(param_1 + 0x60c4);
    uVar8 = *(undefined4 *)(param_1 + 0x60c8);
    uVar5 = *(undefined4 *)(param_1 + 0x60cc);
    puVar16 = (undefined4 *)(lVar15 + 0x60 + lVar22);
    *puVar16 = *(undefined4 *)(param_1 + 0x60c0);
    puVar16[1] = uVar11;
    puVar16[2] = uVar8;
    puVar16[3] = uVar5;
    *(int *)(param_1 + 0x60d0) = *(int *)(param_1 + 0x60d0) + 1;
    iVar12 = FUN_14076b9b0(lVar22);
    FUN_140769eb4(extraout_XMM0_Da,local_78,1,local_res10);
    uVar18 = *(ulonglong *)(param_1 + 0x60b8);
    *(int *)(param_1 + 0x60b4) = *(int *)(param_1 + 0x60b4) + 5;
    iVar9 = 0x40 - *(int *)(param_1 + 0x60c0);
    if (iVar9 < 5) {
      uVar21 = 5 - iVar9;
      *(ulonglong *)(param_1 + 0x60b8) = (ulonglong)uVar19;
      *(uint *)(param_1 + 0x60c0) = uVar21;
      if (uVar21 < 0x40) {
        uVar18 = (ulonglong)(uVar19 >> ((byte)uVar21 & 0x3f)) | uVar18 << ((byte)iVar9 & 0x3f);
      }
      puVar3 = *(ulonglong **)(param_1 + 0x60c8);
      if (*(ulonglong **)(param_1 + 0x6098) < puVar3 + 1) {
        if (puVar3 < *(ulonglong **)(param_1 + 0x6098)) {
          do {
            **(undefined1 **)(param_1 + 0x60c8) = (char)(uVar18 >> 0x38);
            *(longlong *)(param_1 + 0x60c8) = *(longlong *)(param_1 + 0x60c8) + 1;
            uVar18 = uVar18 << 8;
          } while (*(ulonglong *)(param_1 + 0x60c8) < *(ulonglong *)(param_1 + 0x6098));
        }
      }
      else {
        *puVar3 = uVar18 >> 0x38 | (uVar18 & 0xff000000000000) >> 0x28 |
                  (uVar18 & 0xff0000000000) >> 0x18 | (uVar18 & 0xff00000000) >> 8 |
                  (uVar18 & 0xff000000) << 8 | (uVar18 & 0xff0000) << 0x18 |
                  (uVar18 & 0xff00) << 0x28 | uVar18 << 0x38;
        *(longlong *)(param_1 + 0x60c8) = *(longlong *)(param_1 + 0x60c8) + 8;
      }
      *(int *)(param_1 + 0x60b0) = *(int *)(param_1 + 0x60b0) + 0x40;
    }
    else {
      *(int *)(param_1 + 0x60c0) = *(int *)(param_1 + 0x60c0) + 5;
      *(ulonglong *)(param_1 + 0x60b8) = uVar18 << 5 | (ulonglong)uVar19;
    }
    FUN_142f2c9d4(param_1,lVar22,local_78,uVar19,puVar23);
    uVar11 = (undefined4)((ulonglong)puVar23 >> 0x20);
    iVar9 = FUN_14076b9b0(lVar22);
    uVar20 = iVar9 - iVar12;
    *param_3 = uVar20;
    uVar21 = param_2[9];
    FUN_14076a148(extraout_XMM0_Da_00,(int)uVar21 < (int)uVar20);
    puVar23 = puVar24;
    if ((int)uVar20 <= (int)uVar21) {
      *(uint *)(param_1 + 0x2048) = *(uint *)(param_1 + 0x2048) & ~(1 << (uVar19 & 0x1f));
      lVar22 = FUN_142f28458(param_1 + 0x4fe8,param_2[4],uVar19);
      if (*(longlong *)(param_1 + 0x2070 + lVar13 * 0x28) != 0) {
        puVar4 = *(undefined8 **)
                  (*(longlong *)(param_1 + 0x2058 + lVar13 * 0x28) +
                  (*(longlong *)(param_1 + 0x2060 + lVar13 * 0x28) - 1U &
                  *(ulonglong *)(param_1 + 0x2068 + lVar13 * 0x28)) * 8);
        uVar25 = puVar4[1];
        *(undefined8 *)(lVar22 + 0x5c) = *puVar4;
        *(undefined8 *)(lVar22 + 100) = uVar25;
        uVar25 = puVar4[3];
        *(undefined8 *)(lVar22 + 0x6c) = puVar4[2];
        *(undefined8 *)(lVar22 + 0x74) = uVar25;
        uVar25 = puVar4[5];
        *(undefined8 *)(lVar22 + 0x7c) = puVar4[4];
        *(undefined8 *)(lVar22 + 0x84) = uVar25;
        uVar25 = puVar4[7];
        *(undefined8 *)(lVar22 + 0x8c) = puVar4[6];
        *(undefined8 *)(lVar22 + 0x94) = uVar25;
        uVar8 = *(undefined4 *)((longlong)puVar4 + 0x44);
        uVar5 = *(undefined4 *)(puVar4 + 9);
        uVar6 = *(undefined4 *)((longlong)puVar4 + 0x4c);
        *(undefined4 *)(lVar22 + 0x9c) = *(undefined4 *)(puVar4 + 8);
        *(undefined4 *)(lVar22 + 0xa0) = uVar8;
        *(undefined4 *)(lVar22 + 0xa4) = uVar5;
        *(undefined4 *)(lVar22 + 0xa8) = uVar6;
        *(undefined8 *)(lVar22 + 0xac) = puVar4[10];
        *(byte *)(lVar22 + 0xb4) = *(byte *)(lVar22 + 0xb4) | 2;
      }
      cVar7 = FUN_1404f1ca4();
      if (cVar7 == '\0') {
        lVar22 = *(longlong *)(param_1 + 0x2070 + lVar13 * 0x28);
        if (lVar22 == 0) {
          uVar8 = 0xffffffff;
        }
        else {
          lVar15 = param_1 + 0x2050 + lVar13 * 0x28;
          uVar8 = **(undefined4 **)
                    (*(longlong *)(lVar15 + 8) +
                    (*(longlong *)(lVar15 + 0x10) - 1U &
                    (*(longlong *)(param_1 + 0x2068 + lVar13 * 0x28) + lVar22) - 1U) * 8);
        }
        FUN_142f23098(param_1 + 0x4f70,param_2[4],uVar19,0xffffffff,CONCAT44(uVar11,uVar8));
      }
      uVar11 = FUN_1405f50b8();
      *(undefined4 *)(param_1 + 0x1040 + lVar13 * 4) = uVar11;
      puVar23 = (undefined4 *)(ulonglong)*param_3;
    }
  }
  else if (param_2[1] == 2) {
    lVar13 = param_1 + 0x5b30;
    uVar11 = *(undefined4 *)(param_1 + 0x5b5c);
    uVar8 = *(undefined4 *)(param_1 + 0x5b60);
    uVar5 = *(undefined4 *)(param_1 + 0x5b64);
    lVar22 = (longlong)*(int *)(param_1 + 0x5b78) * 0x20;
    puVar24 = (undefined4 *)(lVar22 + 0x50 + lVar13);
    *puVar24 = *(undefined4 *)(param_1 + 0x5b58);
    puVar24[1] = uVar11;
    puVar24[2] = uVar8;
    puVar24[3] = uVar5;
    uVar11 = *(undefined4 *)(param_1 + 0x5b6c);
    uVar8 = *(undefined4 *)(param_1 + 0x5b70);
    uVar5 = *(undefined4 *)(param_1 + 0x5b74);
    puVar24 = (undefined4 *)(lVar22 + 0x60 + lVar13);
    *puVar24 = *(undefined4 *)(param_1 + 0x5b68);
    puVar24[1] = uVar11;
    puVar24[2] = uVar8;
    puVar24[3] = uVar5;
    *(int *)(param_1 + 0x5b78) = *(int *)(param_1 + 0x5b78) + 1;
    FUN_1424cbdf0(local_78);
    iVar12 = FUN_14076b9b0(lVar13);
    FUN_140769eb4(extraout_XMM0_Da_01,local_78,2,0xff);
    FUN_142f2beac(param_1,local_78,lVar13);
    iVar9 = FUN_14076b9b0(lVar13);
    uVar21 = iVar9 - iVar12;
    *param_3 = uVar21;
    uVar19 = param_2[9];
    FUN_14076a148(extraout_XMM0_Da_02,(int)uVar19 < (int)uVar21);
    if ((int)uVar21 <= (int)uVar19) {
      cVar7 = FUN_1404f1ca4();
      if (cVar7 != '\0') {
        *(short *)(param_1 + 0x256a) =
             *(short *)(param_1 + 0x256a) + (short)*(char *)(param_1 + 0x2567);
        FUN_142f2304c(param_1 + 0x4f70,param_2[4]);
        *(undefined1 *)(param_1 + 0x2564) = 0;
      }
      puVar23 = (undefined4 *)(ulonglong)*param_3;
    }
  }
  else {
    puVar23 = (undefined4 *)0x0;
  }
  return puVar23;
}

