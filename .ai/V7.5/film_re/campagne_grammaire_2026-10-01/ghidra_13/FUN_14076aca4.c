
bool FUN_14076aca4(longlong param_1,undefined4 param_2,longlong param_3,undefined8 param_4,
                  int param_5,int *param_6)

{
  int *piVar1;
  char cVar2;
  int iVar3;
  int iVar4;
  void *_Dst;
  longlong *plVar5;
  undefined8 uVar6;
  undefined8 *puVar7;
  uint uVar8;
  void *pvVar9;
  ulonglong uVar10;
  uint *puVar11;
  int iVar12;
  longlong lVar13;
  undefined8 *puVar14;
  int local_d4;
  int local_d0;
  uint local_cc;
  void *local_c8;
  longlong local_c0;
  void *local_b8;
  undefined8 local_b0;
  undefined8 *local_a8;
  void *local_a0;
  void *pvStack_98;
  void *local_90;
  uint local_88;
  uint local_84;
  undefined8 local_80;
  undefined4 local_78;
  undefined8 local_70;
  undefined4 local_68;
  int local_64;
  undefined **local_60;
  char *local_58;
  undefined8 uStack_50;
  undefined1 local_48 [16];
  
  local_d0 = *(int *)(param_3 + 0x2c);
  uVar10 = (ulonglong)DAT_1447061ac;
  local_a0 = (void *)0x0;
  pvStack_98 = (void *)0x0;
  local_90 = (void *)0x0;
  local_c0 = 3;
  if (uVar10 == 0) {
    local_b8 = (void *)0x0;
    _Dst = local_a0;
  }
  else {
    _Dst = (void *)FUN_14068a268(uVar10 * 4);
    local_b8 = _Dst;
    if (_Dst == (void *)0x0) {
      uStack_50 = 0;
      local_58 = "bad allocation";
      local_60 = std::bad_alloc::vftable;
                    /* WARNING: Subroutine does not return */
      _CxxThrowException(&local_60,(ThrowInfo *)&DAT_14436e8a8);
    }
    pvVar9 = (void *)(uVar10 * 4 + (longlong)_Dst);
    local_a0 = _Dst;
    local_90 = pvVar9;
    memset(_Dst,0,uVar10 * 4);
    pvStack_98 = pvVar9;
  }
  uVar8 = (uint)DAT_144706104;
  local_c8 = _Dst;
  if (*(uint *)(param_3 + 0x38) < 0x40) {
    *(uint *)(param_3 + 0x38) = *(uint *)(param_3 + 0x38) + 1;
    *(int *)(param_3 + 0x2c) = *(int *)(param_3 + 0x2c) + 1;
    *(ulonglong *)(param_3 + 0x30) = *(longlong *)(param_3 + 0x30) * 2 | (ulonglong)(uVar8 & 1);
  }
  else {
    FUN_1406d6e28(param_3,(ulonglong)(uVar8 & 1),1);
  }
  cVar2 = FUN_14051a14c();
  if ((cVar2 == '\0') || (*(int *)(param_1 + 400) == 0)) {
    FUN_14076c008(param_1,_Dst);
  }
  else {
    local_c8 = *(void **)(param_1 + 0x188);
  }
  local_d4 = -1;
  iVar3 = FUN_14076b9b0(param_3);
  iVar3 = ((*(int *)(param_3 + 0x18) * 8 - *(int *)(param_1 + 0x170)) - param_5) - iVar3;
  local_cc = FUN_140514010(*(undefined4 *)(param_1 + 0xc));
  if (local_cc < 0x21) {
    cVar2 = (&DAT_144de4348)[(int)local_cc];
    if (cVar2 == '\0') goto LAB_14076add6;
    uVar6 = *(undefined8 *)(param_1 + 0x180);
  }
  else {
    cVar2 = '\0';
LAB_14076add6:
    uVar6 = *(undefined8 *)(param_1 + 0x178);
  }
  local_b0 = FUN_14076bf98(uVar6);
  puVar14 = (undefined8 *)(param_1 + 0x128);
  lVar13 = 3;
  local_a8 = puVar14;
  do {
    puVar7 = puVar14 + 3;
    if (cVar2 == '\0') {
      puVar7 = puVar14;
    }
    (**(code **)(*(longlong *)*puVar7 + 0x28))();
    uVar6 = local_b0;
    puVar14 = puVar14 + 1;
    lVar13 = lVar13 + -1;
  } while (lVar13 != 0);
  *param_6 = *(int *)(param_3 + 0x2c) - local_d0;
  uVar10 = (ulonglong)*(int *)(param_1 + 0x194);
  iVar4 = *(int *)(param_1 + 400);
  if (*(int *)(param_1 + 0x194) < iVar4) {
    puVar11 = (uint *)((longlong)local_c8 + uVar10 * 4);
    iVar12 = local_d4;
    do {
      uVar8 = *puVar11;
      if (cVar2 == '\0') {
        plVar5 = *(longlong **)(param_1 + 0x128 + (ulonglong)(uVar8 >> 0x1e) * 8);
      }
      else {
        plVar5 = *(longlong **)(param_1 + 0x140 + (ulonglong)(uVar8 >> 0x1e) * 8);
      }
      local_88 = uVar8 & 0x1fff;
      if (local_88 == 0x1fff) {
        local_88 = 0xffffffff;
      }
      local_84 = uVar8 >> 0x17 & 0x7f;
      if (local_84 == 0x7f) {
        local_84 = 0xffffffff;
      }
      local_80 = uVar6;
      local_70 = 0;
      local_68 = 0;
      local_d4 = 0;
      local_78 = param_2;
      local_64 = iVar3;
      iVar4 = (**(code **)(*plVar5 + 0x18))(plVar5,&local_88,&local_d4);
      *param_6 = *param_6 + local_d4;
      *(undefined1 *)(param_1 + 0x10) = 1;
      iVar3 = iVar3 - iVar4;
      if (((iVar4 == 0) || (iVar4 != local_d4)) && (iVar12 == -1)) {
        iVar12 = (int)uVar10;
      }
      uVar8 = (int)uVar10 + 1;
      uVar10 = (ulonglong)uVar8;
      puVar11 = puVar11 + 1;
      iVar4 = *(int *)(param_1 + 400);
    } while ((int)uVar8 < iVar4);
    if (iVar12 != -1) goto LAB_14076af08;
  }
  iVar12 = iVar4;
LAB_14076af08:
  uVar8 = local_cc;
  *(int *)(param_1 + 0x194) = iVar12;
  iVar3 = 0;
  puVar14 = local_a8;
  do {
    puVar7 = puVar14 + 3;
    if (cVar2 == '\0') {
      puVar7 = puVar14;
    }
    plVar5 = (longlong *)*puVar7;
    (**(code **)(*plVar5 + 0x20))(plVar5,param_3,uVar8);
    iVar4 = *(int *)(param_3 + 0x2c);
    (**(code **)(*plVar5 + 0x38))(plVar5,param_2,param_3);
    iVar3 = iVar3 + (*(int *)(param_3 + 0x2c) - iVar4);
    puVar14 = puVar14 + 1;
    local_c0 = local_c0 + -1;
  } while (local_c0 != 0);
  *param_6 = *param_6 + iVar3;
  iVar3 = *(int *)(param_3 + 0x2c) - local_d0;
  plVar5 = (longlong *)FUN_140515334();
  plVar5 = (longlong *)(**(code **)(*plVar5 + 0x70))(plVar5,local_48);
  piVar1 = (int *)*plVar5;
  if (piVar1 != (int *)0x0) {
    *piVar1 = iVar3;
    *(longlong *)(piVar1 + 2) = *(longlong *)(piVar1 + 2) + 1;
    *(longlong *)(piVar1 + 4) = *(longlong *)(piVar1 + 4) + (longlong)iVar3;
    *(longlong *)(piVar1 + 6) = *(longlong *)(piVar1 + 6) + 1;
    *(longlong *)(piVar1 + 8) = *(longlong *)(piVar1 + 8) + (longlong)iVar3;
    piVar1[0xb] = piVar1[0xb] + 1;
    piVar1[0xc] = piVar1[0xc] + iVar3;
  }
  iVar4 = *param_6;
  if (local_b8 != (void *)0x0) {
    FUN_1405a3720(local_b8);
  }
  return iVar3 < iVar4;
}

