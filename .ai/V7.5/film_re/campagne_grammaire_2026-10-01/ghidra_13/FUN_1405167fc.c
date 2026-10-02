
/* WARNING: Globals starting with '_' overlap smaller symbols at the same address */

void FUN_1405167fc(longlong param_1,longlong param_2,char param_3,char param_4,int *param_5,
                  int *param_6)

{
  int *piVar1;
  undefined8 uVar2;
  ulonglong *puVar3;
  char cVar4;
  undefined1 uVar5;
  undefined4 uVar6;
  int iVar7;
  int iVar8;
  int iVar9;
  uint uVar10;
  longlong *plVar11;
  longlong lVar12;
  undefined8 *puVar13;
  uint *puVar14;
  ulonglong uVar15;
  uint uVar16;
  uint uVar17;
  undefined4 *puVar18;
  size_t _Size;
  int iVar19;
  uint uVar20;
  uint uVar21;
  undefined4 extraout_XMM0_Da;
  int local_res10 [2];
  int local_res18;
  char local_res20;
  undefined8 in_stack_fffffffffffff978;
  int local_668;
  int local_65c;
  char *local_658;
  undefined8 uStack_650;
  undefined8 local_648;
  int local_640;
  int local_63c;
  int local_638;
  int local_634;
  longlong *local_630;
  int local_628 [2];
  longlong *local_620;
  undefined4 local_618;
  undefined4 uStack_614;
  undefined4 uStack_610;
  undefined4 uStack_60c;
  undefined8 local_608;
  char *local_5f8;
  undefined8 uStack_5f0;
  undefined8 local_5e8;
  undefined1 local_5d8 [24];
  undefined8 local_5c0;
  undefined4 uStack_5b8;
  undefined4 uStack_5b4;
  undefined8 local_5b0;
  undefined4 local_5a8;
  undefined4 local_5a4;
  undefined4 uStack_5a0;
  undefined4 uStack_59c;
  undefined4 uStack_598;
  undefined8 local_594;
  undefined2 local_588;
  undefined8 local_580;
  char *local_578;
  undefined8 local_570;
  undefined2 local_568;
  undefined8 local_560;
  undefined1 local_4b8 [1152];
  
  local_668 = -1;
  iVar19 = 0;
  iVar7 = 0;
  local_res18 = 0;
  local_65c = 0;
  local_res20 = param_4;
  local_620 = (longlong *)FUN_140515334();
  local_634 = *(int *)(param_2 + 0x2c);
  if (param_3 != '\0') {
    uVar6 = FUN_1405a70d4();
    local_668 = FUN_1409cca10(param_1 + 0xac8,uVar6);
    iVar8 = iVar7;
    if (local_668 == -1) goto LAB_140516f09;
  }
  *(undefined4 *)(param_2 + 0x20) = 1;
  *(undefined4 *)(param_2 + 0x1c) = 8;
  *(undefined8 *)(param_2 + 0x40) = *(undefined8 *)(param_2 + 8);
  *(undefined8 *)(param_2 + 0x28) = 0;
  *(undefined4 *)(param_2 + 0x48) = 0;
  *(undefined1 *)(param_2 + 0x24) = 0;
  *(undefined8 *)(param_2 + 0x30) = 0;
  *(undefined4 *)(param_2 + 0x38) = 0;
  *(undefined8 *)(param_2 + 0xd0) = 0;
  lVar12 = *(longlong *)(param_1 + 0xa98);
  iVar7 = FUN_1406d1a44(lVar12,param_1 + 0x18);
  if (iVar7 != -1) {
    piVar1 = (int *)(lVar12 + 0x18 + (longlong)iVar7 * 0x24);
    *piVar1 = *piVar1 + 1;
    uVar6 = FUN_1405a70d4();
    *(undefined4 *)(lVar12 + 0x20 + (longlong)iVar7 * 0x24) = uVar6;
  }
  FUN_140c4d5d4(param_2);
  iVar7 = FUN_14076b9b0();
  iVar7 = *(int *)(param_2 + 0x18) * 8 - iVar7;
  local_640 = iVar7;
  if ((((*(longlong *)(param_1 + 0x3030) == 0) ||
       ((*(byte *)(*(longlong *)(param_1 + 0x3030) + 0x28) & 0xc) != 4)) ||
      (cVar4 = FUN_140c6906c(), cVar4 == '\0')) ||
     ((param_3 == '\0' || (cVar4 = FUN_1406cb0a4(), cVar4 != '\0')))) {
    uVar10 = 1;
    uVar5 = 0;
  }
  else {
    uVar10 = 0;
    uVar5 = 1;
  }
  *(undefined1 *)(param_1 + 0x2fe0) = uVar5;
  local_648 = (ulonglong)local_648._4_4_ << 0x20;
  uVar17 = 0xffffffff;
  local_658 = (char *)0x0;
  uStack_650 = 0;
LAB_140516966:
  do {
    uVar21 = 0xffffffff;
    uVar20 = 0xffffffff;
    if (uVar17 != 0xffffffff) {
      if (-1 < (int)uVar17) {
        uVar16 = (uVar17 & 0x7fffffff) + 1;
        uVar17 = uVar16 & 0x7fffffff;
        if (-1 < (int)uVar16) goto LAB_14051698c;
        goto LAB_1405169ff;
      }
LAB_1405169f5:
      puVar14 = (uint *)0x0;
      uVar17 = uVar21;
      break;
    }
    uVar16 = 0;
    uVar17 = 0;
LAB_14051698c:
    if ((int)uVar16 < *(int *)(param_1 + 0x2fe8)) {
      puVar14 = (uint *)(((longlong)(int)uVar16 + 0x2ff) * 0x10 + param_1);
    }
    else {
LAB_1405169ff:
      uVar16 = *(uint *)(param_1 + 0x2fe8);
      uVar17 = 0x80000000;
      lVar12 = *(longlong *)(param_1 + 0x3030);
      if ((lVar12 == 0) || (*(longlong *)(lVar12 + 0x20) == 0)) goto LAB_1405169f5;
      puVar14 = (uint *)(lVar12 + 0x18);
    }
    uVar20 = uVar16;
  } while ((uVar17 != 0xffffffff) && ((*puVar14 & uVar10) != uVar10));
  if (uVar17 != 0xffffffff) {
    iVar8 = (**(code **)(**(longlong **)(puVar14 + 2) + 0x20))
                      (*(longlong **)(puVar14 + 2),iVar7,iVar7 - iVar19);
    iVar19 = iVar19 + iVar8;
    *(int *)((longlong)&local_658 + (longlong)(int)uVar20 * 4) = iVar8;
    goto LAB_140516966;
  }
  iVar19 = iVar19 + 1;
  if (iVar19 <= iVar7) {
LAB_140516a4e:
    do {
      uVar6 = (undefined4)((ulonglong)in_stack_fffffffffffff978 >> 0x20);
      uVar17 = 0xffffffff;
      if (uVar21 == 0xffffffff) {
        uVar20 = 0;
        uVar21 = 0;
LAB_140516a81:
        if ((int)uVar20 < *(int *)(param_1 + 0x2fe8)) {
          puVar14 = (uint *)(((longlong)(int)uVar20 + 0x2ff) * 0x10 + param_1);
        }
        else {
LAB_140516c39:
          uVar20 = *(uint *)(param_1 + 0x2fe8);
          uVar21 = 0x80000000;
          lVar12 = *(longlong *)(param_1 + 0x3030);
          if ((lVar12 == 0) || (*(longlong *)(lVar12 + 0x20) == 0)) goto LAB_140516c2b;
          puVar14 = (uint *)(lVar12 + 0x18);
        }
        if ((uVar21 != 0xffffffff) && ((*puVar14 & uVar10) != uVar10)) goto LAB_140516a4e;
      }
      else {
        if (-1 < (int)uVar21) {
          uVar20 = (uVar21 & 0x7fffffff) + 1;
          uVar21 = uVar20 & 0x7fffffff;
          if (-1 < (int)uVar20) goto LAB_140516a81;
          goto LAB_140516c39;
        }
LAB_140516c2b:
        puVar14 = (uint *)0x0;
        uVar21 = 0xffffffff;
        uVar20 = 0xffffffff;
      }
      if (uVar21 == 0xffffffff) {
        plVar11 = (longlong *)0x0;
        uVar15 = 0;
      }
      else {
        plVar11 = *(longlong **)(puVar14 + 2);
        uVar15 = (ulonglong)*puVar14;
      }
      uVar16 = uVar21;
      local_630 = plVar11;
      iVar8 = FUN_14076b9b0(param_2);
      if (uVar16 == uVar17) {
        iVar7 = *(int *)(param_2 + 0x2c);
        if ((local_res20 == '\0') ||
           (iVar19 = FUN_14076b9b0(), *(int *)(param_2 + 0x18) * 8 - iVar19 < 0x11)) {
          if (*(uint *)(param_2 + 0x38) < 0x40) {
            *(longlong *)(param_2 + 0x30) = *(longlong *)(param_2 + 0x30) << 1;
            *(uint *)(param_2 + 0x38) = *(uint *)(param_2 + 0x38) + 1;
            *(int *)(param_2 + 0x2c) = iVar7 + 1;
          }
          else {
            FUN_1406d6e28(param_2,0,1);
          }
        }
        else {
          iVar19 = *(int *)(param_2 + 0x18);
          iVar9 = FUN_14076b9b0(param_2);
          uVar10 = (iVar19 * 8 + -0x10) - iVar9;
          _Size = (size_t)((int)(uVar10 + 7) / 8);
          iVar9 = FUN_14076b9b0(extraout_XMM0_Da,(longlong)(int)(uVar10 + 7) % 8 & 0xffffffff);
          local_65c = (iVar19 * 8 - iVar9) + -1;
          FUN_1406d49c4();
          uVar15 = *(ulonglong *)(param_2 + 0x30);
          iVar19 = 0x40 - *(int *)(param_2 + 0x38);
          *(int *)(param_2 + 0x2c) = *(int *)(param_2 + 0x2c) + 0xf;
          if (iVar19 < 0xf) {
            uVar17 = 0xf - iVar19;
            *(ulonglong *)(param_2 + 0x30) = (ulonglong)uVar10;
            *(uint *)(param_2 + 0x38) = uVar17;
            if (uVar17 < 0x40) {
              uVar15 = (ulonglong)(uVar10 >> ((byte)uVar17 & 0x3f)) |
                       uVar15 << ((byte)iVar19 & 0x3f);
            }
            puVar3 = *(ulonglong **)(param_2 + 0x40);
            if (*(ulonglong **)(param_2 + 0x10) < puVar3 + 1) {
              if (puVar3 < *(ulonglong **)(param_2 + 0x10)) {
                do {
                  **(undefined1 **)(param_2 + 0x40) = (char)(uVar15 >> 0x38);
                  *(longlong *)(param_2 + 0x40) = *(longlong *)(param_2 + 0x40) + 1;
                  uVar15 = uVar15 << 8;
                } while (*(ulonglong *)(param_2 + 0x40) < *(ulonglong *)(param_2 + 0x10));
              }
            }
            else {
              *puVar3 = uVar15 >> 0x38 | (uVar15 & 0xff000000000000) >> 0x28 |
                        (uVar15 & 0xff0000000000) >> 0x18 | (uVar15 & 0xff00000000) >> 8 |
                        (uVar15 & 0xff000000) << 8 | (uVar15 & 0xff0000) << 0x18 |
                        (uVar15 & 0xff00) << 0x28 | uVar15 << 0x38;
              *(longlong *)(param_2 + 0x40) = *(longlong *)(param_2 + 0x40) + 8;
            }
            *(int *)(param_2 + 0x28) = *(int *)(param_2 + 0x28) + 0x40;
          }
          else {
            *(int *)(param_2 + 0x38) = *(int *)(param_2 + 0x38) + 0xf;
            *(ulonglong *)(param_2 + 0x30) = uVar15 << 0xf | (ulonglong)uVar10;
          }
          memset(local_4b8,0,_Size);
          FUN_1406d60f4(param_2);
        }
        iVar7 = *(int *)(param_2 + 0x2c) - iVar7;
        plVar11 = (longlong *)(**(code **)(*local_620 + 0x90))();
        piVar1 = (int *)*plVar11;
        if (piVar1 != (int *)0x0) {
          *(longlong *)(piVar1 + 2) = *(longlong *)(piVar1 + 2) + 1;
          *(longlong *)(piVar1 + 6) = *(longlong *)(piVar1 + 6) + 1;
          piVar1[0xb] = piVar1[0xb] + 1;
          *(longlong *)(piVar1 + 4) = *(longlong *)(piVar1 + 4) + (longlong)iVar7;
          *(longlong *)(piVar1 + 8) = *(longlong *)(piVar1 + 8) + (longlong)iVar7;
          piVar1[0xc] = piVar1[0xc] + iVar7;
          *piVar1 = iVar7;
        }
        FUN_1406d6d94(param_2);
        if (local_668 != -1) {
          iVar7 = -1;
          iVar19 = -1;
          if ((*(int *)(param_1 + 0xb10) < local_668) &&
             (iVar19 = -1, local_668 <= *(int *)(param_1 + 0xb0c))) {
            if (*(int *)(param_1 + 0xb1c) != 0) {
              iVar7 = *(int *)(param_1 + 0xb18);
            }
            iVar19 = ((local_668 - *(int *)(param_1 + 0xb10)) + -1 + iVar7) %
                     *(int *)(param_1 + 0xb08);
          }
          puVar18 = (undefined4 *)0x0;
          if (iVar19 != -1) {
            puVar18 = (undefined4 *)(((longlong)iVar19 + 0x86) * 0x10 + param_1 + 0xac8);
          }
          uVar6 = FUN_1405a70d4();
          *puVar18 = uVar6;
        }
        if (*(uint *)(param_1 + 0x3044) < 2) {
          iVar7 = 0;
        }
        else {
          local_5a4 = *(undefined4 *)(param_1 + 0x18);
          uStack_5a0 = *(undefined4 *)(param_1 + 0x1c);
          uStack_59c = *(undefined4 *)(param_1 + 0x20);
          uStack_598 = *(undefined4 *)(param_1 + 0x24);
          local_5a8 = 0;
          local_594 = *(undefined8 *)(param_1 + 0x28);
          uVar2 = *(undefined8 *)(param_1 + 0xa98);
          if (*(int *)(param_2 + 0x18) < 0x481) {
            local_588 = (undefined2)*(int *)(param_2 + 0x18);
            local_580 = *(undefined8 *)(param_2 + 8);
            lVar12 = FUN_140518a4c(&local_5a4);
            (**(code **)(**(longlong **)(lVar12 + 0x3928) + 8))
                      (*(longlong **)(lVar12 + 0x3928),0,local_588);
            FUN_140514fcc(uVar2,&local_5a8);
            iVar7 = FUN_1404e56d0(&local_5a8);
            if (0 < iVar7) {
              local_res18 = *(int *)(param_2 + 0x18);
            }
          }
          else {
            iVar7 = 0;
          }
        }
        if (local_668 != -1) {
          FUN_140cc4d64(param_1 + 0xac8,local_668,iVar7);
        }
        FUN_140518cb8();
        *(longlong *)(param_1 + 0x290) = *(longlong *)(param_1 + 0x290) + 1;
        *(longlong *)(param_1 + 0x2a0) = *(longlong *)(param_1 + 0x2a0) + 1;
        *(int *)(param_1 + 0x2b4) = *(int *)(param_1 + 0x2b4) + 1;
        *(int *)(param_1 + 0x2b8) = *(int *)(param_1 + 0x2b8) + iVar7;
        *(longlong *)(param_1 + 0x380) = *(longlong *)(param_1 + 0x380) + 1;
        *(longlong *)(param_1 + 0x390) = *(longlong *)(param_1 + 0x390) + 1;
        *(int *)(param_1 + 0x3a4) = *(int *)(param_1 + 0x3a4) + 1;
        *(int *)(param_1 + 0x3a8) = *(int *)(param_1 + 0x3a8) + local_65c;
        *(longlong *)(param_1 + 0x388) = *(longlong *)(param_1 + 0x388) + (longlong)local_65c;
        *(longlong *)(param_1 + 0x398) = *(longlong *)(param_1 + 0x398) + (longlong)local_65c;
        lVar12 = (longlong)iVar7;
        *(longlong *)(param_1 + 0x298) = *(longlong *)(param_1 + 0x298) + lVar12;
        *(longlong *)(param_1 + 0x2a8) = *(longlong *)(param_1 + 0x2a8) + lVar12;
        *(int *)(param_1 + 0x288) = iVar7;
        *(int *)(param_1 + 0x378) = local_65c;
        iVar19 = FUN_1405f50b8();
        if (*(int *)(param_1 + 100) != -1) {
          FUN_140b4f920(param_1 + 0x9d0,iVar19 - *(int *)(param_1 + 100));
        }
        *(int *)(param_1 + 100) = iVar19;
        uVar10 = FUN_1405179f8(param_1);
        _DAT_144de7c30 = _DAT_144de7c30 + 1;
        _DAT_144de7c38 = _DAT_144de7c38 + lVar12;
        _DAT_144de7c40 = _DAT_144de7c40 + 1;
        _DAT_144de7c48 = _DAT_144de7c48 + lVar12;
        _DAT_144de7c54 = _DAT_144de7c54 + 1;
        _DAT_144de7c58 = _DAT_144de7c58 + iVar7;
        _DAT_144de7c28 = iVar7;
        if ((DAT_144de7be0 >> (uVar10 & 0x1f) & 1) != 0) {
          lVar12 = (longlong)(int)uVar10 * 0x130;
          *(int *)(&DAT_144de55f4 + lVar12) = *(int *)(&DAT_144de55f4 + lVar12) + iVar7;
          *(int *)(&DAT_144de5630 + lVar12) = *(int *)(&DAT_144de5630 + lVar12) + iVar8 + 1;
          *(int *)(&DAT_144de5634 + lVar12) = *(int *)(&DAT_144de5634 + lVar12) + local_65c;
        }
        iVar19 = *(int *)(param_2 + 0x2c) - local_634;
        plVar11 = (longlong *)(**(code **)(*local_620 + 0xa0))(local_620,local_628);
        piVar1 = (int *)*plVar11;
        iVar8 = local_res18;
        if (piVar1 != (int *)0x0) {
          *(longlong *)(piVar1 + 2) = *(longlong *)(piVar1 + 2) + 1;
          *(longlong *)(piVar1 + 6) = *(longlong *)(piVar1 + 6) + 1;
          piVar1[0xb] = piVar1[0xb] + 1;
          *(longlong *)(piVar1 + 4) = *(longlong *)(piVar1 + 4) + (longlong)iVar19;
          *(longlong *)(piVar1 + 8) = *(longlong *)(piVar1 + 8) + (longlong)iVar19;
          piVar1[0xc] = piVar1[0xc] + iVar19;
          *piVar1 = iVar19;
        }
        goto LAB_140516f09;
      }
      iVar8 = *(int *)(param_2 + 0x18) * 8 - iVar8;
      local_628[0] = *(int *)((longlong)&local_658 + (longlong)(int)uVar20 * 4);
      iVar19 = iVar19 - local_628[0];
      iVar7 = iVar19;
      if ((*(longlong *)(param_1 + 0x3030) != 0) &&
         ((byte)(((byte)(uVar16 >> 0x1f) ^ 1) &
                -((*(byte *)(*(longlong *)(param_1 + 0x3030) + 0x28) & 4) != 0)) != 0)) {
        iVar9 = (int)((float)local_640 * *(float *)(*(longlong *)(param_1 + 0xab8) + 0x14));
        if (iVar19 < iVar9) {
          iVar7 = iVar9;
        }
        if ((uVar15 & 0x40) != 0) {
          iVar9 = (int)((float)local_640 * *(float *)(*(longlong *)(param_1 + 0xab8) + 0x18));
          iVar7 = local_628[0];
          if (local_628[0] < iVar9) {
            iVar7 = iVar9;
          }
          iVar7 = iVar8 - iVar7;
          if (iVar7 < iVar19) {
            iVar7 = iVar19;
          }
        }
      }
      iVar9 = *(int *)(param_2 + 0x2c);
      local_638 = iVar8 - local_628[0];
      if (iVar7 <= local_638) {
        local_638 = iVar7;
      }
      local_res10[0] = 0;
      in_stack_fffffffffffff978 = CONCAT44(uVar6,local_638);
      (**(code **)(*plVar11 + 0x28))
                (plVar11,local_668,param_2,local_628[0],in_stack_fffffffffffff978,local_res10);
      local_63c = *(int *)(param_2 + 0x2c) - iVar9;
      if (local_63c < local_res10[0]) {
        (**(code **)(**(longlong **)(param_1 + 0x3928) + 0x18))();
      }
      (**(code **)(**(longlong **)(param_1 + 0x3928) + 8))
                (*(longlong **)(param_1 + 0x3928),3,local_63c / 8);
      (**(code **)(**(longlong **)(param_1 + 0x3928) + 8))
                (*(longlong **)(param_1 + 0x3928),4,(longlong)local_res10[0] / 8 & 0xffffffff);
      (**(code **)(**(longlong **)(param_1 + 0x3928) + 0x10))();
      iVar7 = *(int *)(param_2 + 0x18);
      iVar8 = FUN_14076b9b0(param_2);
      if (iVar7 * 8 - iVar8 < local_638) {
        FUN_1422c394e();
        return;
      }
    } while( true );
  }
  local_648 = 0;
  local_5b0 = 0;
  local_658 = "MustLeaveSpaceBitsTotal";
  uStack_650 = CONCAT71(uStack_650._1_7_,(char)iVar19);
  local_5c0 = "TotalBitsAvailable";
  uStack_5b8 = CONCAT31(uStack_5b8._1_3_,(char)iVar7);
  puVar13 = (undefined8 *)FUN_140a3a8a4(local_5d8,param_1 + 0x3070);
  local_560 = 0;
  local_578 = "ComponentName";
  local_570 = *puVar13;
  local_568 = *(undefined2 *)(puVar13 + 1);
  local_5f8 = local_658;
  uStack_5f0 = uStack_650;
  local_5e8 = local_648;
  local_618 = (undefined4)local_5c0;
  uStack_614 = local_5c0._4_4_;
  uStack_610 = uStack_5b8;
  uStack_60c = uStack_5b4;
  local_608 = local_5b0;
  FUN_142fced64();
  iVar7 = 0;
  iVar8 = iVar7;
LAB_140516f09:
  if (param_5 != (int *)0x0) {
    *param_5 = iVar7;
  }
  if (param_6 != (int *)0x0) {
    *param_6 = iVar8;
  }
  return;
}

