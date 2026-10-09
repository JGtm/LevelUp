
undefined8 FUN_142ef8828(undefined8 param_1,undefined8 param_2,undefined8 *param_3,longlong param_4)

{
  ulonglong uVar1;
  char cVar2;
  longlong lVar3;
  undefined8 uVar4;
  uint uVar5;
  ulonglong *puVar6;
  ulonglong uVar7;
  int iVar8;
  uint uVar9;
  ulonglong uVar10;
  ulonglong uVar11;
  uint uVar12;
  undefined8 extraout_XMM0_Qa;
  undefined4 local_res18 [2];
  undefined4 local_res20 [2];
  undefined4 uStack_34;
  undefined4 uStack_30;
  
  uVar10 = 0;
  uVar12 = 0;
  iVar8 = 0x40 - *(int *)(param_4 + 0x38);
  uVar9 = (uint)((ulonglong)*(longlong *)(param_4 + 0x30) >> 0x20);
  if (iVar8 < 0x20) {
    puVar6 = *(ulonglong **)(param_4 + 0x40);
    uVar5 = 0;
    if (*(ulonglong **)(param_4 + 0x10) < puVar6 + 1) {
      uVar11 = uVar10;
      uVar7 = uVar10;
      if (puVar6 < *(ulonglong **)(param_4 + 0x10)) {
        do {
          uVar5 = (int)uVar11 + 8;
          uVar11 = (ulonglong)uVar5;
          uVar1 = *puVar6;
          puVar6 = (ulonglong *)((longlong)puVar6 + 1);
          uVar7 = (ulonglong)(byte)uVar1 | uVar7 << 8;
          *(ulonglong **)(param_4 + 0x40) = puVar6;
        } while (puVar6 < *(ulonglong **)(param_4 + 0x10));
        uVar7 = uVar7 << (-(char)uVar5 & 0x3fU);
      }
    }
    else {
      uVar11 = *puVar6;
      uVar5 = 0x40;
      *(ulonglong **)(param_4 + 0x40) = puVar6 + 1;
      uVar7 = uVar11 >> 0x38 | (uVar11 & 0xff000000000000) >> 0x28 |
              (uVar11 & 0xff0000000000) >> 0x18 | (uVar11 & 0xff00000000) >> 8 |
              (uVar11 & 0xff000000) << 8 | (uVar11 & 0xff0000) << 0x18 | (uVar11 & 0xff00) << 0x28 |
              uVar11 << 0x38;
    }
    *(int *)(param_4 + 0x28) = *(int *)(param_4 + 0x28) + uVar5;
    *(int *)(param_4 + 0x2c) = *(int *)(param_4 + 0x2c) + 0x20;
    uVar5 = 0x20 - iVar8;
    *(ulonglong *)(param_4 + 0x30) = -(ulonglong)(uVar5 < 0x40) & uVar7 << ((byte)uVar5 & 0x3f);
    *(uint *)(param_4 + 0x38) = uVar5;
    uVar9 = (uint)(uVar7 >> (-(byte)uVar5 & 0x3f)) | uVar9;
  }
  else {
    *(int *)(param_4 + 0x2c) = *(int *)(param_4 + 0x2c) + 0x20;
    *(longlong *)(param_4 + 0x30) = *(longlong *)(param_4 + 0x30) << 0x20;
    *(int *)(param_4 + 0x38) = *(int *)(param_4 + 0x38) + 0x20;
  }
  local_res18[0] = 0xffffffff;
  cVar2 = FUN_1406cf008(param_4);
  if (cVar2 != '\0') {
    FUN_14080d6f0(extraout_XMM0_Qa,param_4,local_res18);
    FUN_1407f21b4(local_res20,local_res18);
    local_res18[0] = local_res20[0];
    lVar3 = FUN_140478680(local_res18,0x73626e6b);
    if (lVar3 != 0) {
      uVar4 = CONCAT44(*(undefined4 *)(lVar3 + 0xc),uVar9);
      goto LAB_142ef8961;
    }
  }
  uVar4 = CONCAT44(0xffffffff,uVar9);
LAB_142ef8961:
  iVar8 = *(int *)(param_4 + 0x38);
  uVar9 = (uint)((ulonglong)*(longlong *)(param_4 + 0x30) >> 0x20);
  if (0x40 - iVar8 < 6) {
    puVar6 = *(ulonglong **)(param_4 + 0x40);
    if (*(ulonglong **)(param_4 + 0x10) < puVar6 + 1) {
      uVar11 = uVar10;
      if (puVar6 < *(ulonglong **)(param_4 + 0x10)) {
        do {
          uVar7 = *puVar6;
          uVar12 = (int)uVar10 + 8;
          uVar10 = (ulonglong)uVar12;
          puVar6 = (ulonglong *)((longlong)puVar6 + 1);
          uVar11 = uVar11 << 8 | (ulonglong)(byte)uVar7;
          *(ulonglong **)(param_4 + 0x40) = puVar6;
        } while (puVar6 < *(ulonglong **)(param_4 + 0x10));
        uVar10 = uVar11 << (-(char)uVar12 & 0x3fU);
      }
    }
    else {
      uVar10 = *puVar6;
      uVar12 = 0x40;
      uVar10 = uVar10 >> 0x38 | (uVar10 & 0xff000000000000) >> 0x28 |
               (uVar10 & 0xff0000000000) >> 0x18 | (uVar10 & 0xff00000000) >> 8 |
               (uVar10 & 0xff000000) << 8 | (uVar10 & 0xff0000) << 0x18 | (uVar10 & 0xff00) << 0x28
               | uVar10 << 0x38;
      *(ulonglong **)(param_4 + 0x40) = puVar6 + 1;
    }
    *(int *)(param_4 + 0x28) = *(int *)(param_4 + 0x28) + uVar12;
    uVar12 = iVar8 - 0x3a;
    *(int *)(param_4 + 0x2c) = *(int *)(param_4 + 0x2c) + 6;
    *(ulonglong *)(param_4 + 0x30) = -(ulonglong)(uVar12 < 0x40) & uVar10 << ((byte)uVar12 & 0x3f);
    uVar9 = (uint)(uVar10 >> (-(byte)uVar12 & 0x3f)) | uVar9 >> 0x1a;
  }
  else {
    *(int *)(param_4 + 0x2c) = *(int *)(param_4 + 0x2c) + 6;
    uVar12 = iVar8 + 6;
    *(longlong *)(param_4 + 0x30) = *(longlong *)(param_4 + 0x30) << 6;
    uVar9 = uVar9 >> 0x1a;
  }
  *(uint *)(param_4 + 0x38) = uVar12;
  uStack_34 = (undefined4)uVar4;
  uStack_30 = (undefined4)((ulonglong)uVar4 >> 0x20);
  *param_3 = CONCAT44(uStack_34,uVar9);
  *(undefined4 *)(param_3 + 1) = uStack_30;
  return CONCAT71((int7)((ulonglong)uVar4 >> 8),1);
}

