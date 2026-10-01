
void FUN_14076c304(uint *param_1,int param_2)

{
  uint uVar1;
  longlong lVar2;
  uint *puVar3;
  uint uVar4;
  uint *puVar5;
  longlong lVar6;
  uint *puVar7;
  uint *puVar8;
  undefined8 local_1e8 [30];
  longlong alStack_f8 [30];
  
  if (param_2 < 2) {
    return;
  }
  lVar6 = 0;
  puVar5 = param_1 + (longlong)param_2 + -1;
LAB_14076c352:
  lVar2 = ((longlong)puVar5 - (longlong)param_1 >> 2) + 1;
  if (8 < lVar2) {
    puVar7 = puVar5 + 1;
    lVar2 = lVar2 / 2;
    uVar4 = param_1[lVar2];
    param_1[lVar2] = *param_1;
    *param_1 = uVar4;
    puVar3 = param_1;
    do {
      uVar4 = *param_1;
      do {
        puVar3 = puVar3 + 1;
        if (puVar5 < puVar3) break;
        uVar4 = *param_1;
      } while ((uVar4 & 0x7fe000) <= (*puVar3 & 0x7fe000));
      do {
        puVar8 = puVar7;
        puVar7 = puVar8 + -1;
        uVar1 = *puVar7;
        if (puVar7 <= param_1) break;
      } while ((uVar1 & 0x7fe000) <= (uVar4 & 0x7fe000));
      if (puVar7 < puVar3) goto LAB_14230cb27;
      uVar4 = *puVar3;
      *puVar3 = uVar1;
      *puVar7 = uVar4;
    } while( true );
  }
  for (; puVar3 = param_1, puVar7 = param_1, param_1 < puVar5; puVar5 = puVar5 + -1) {
    while (puVar7 = puVar7 + 1, puVar7 <= puVar5) {
      if ((*puVar7 & 0x7fe000) < (*puVar3 & 0x7fe000)) {
        puVar3 = puVar7;
      }
    }
    uVar4 = *puVar3;
    *puVar3 = *puVar5;
    *puVar5 = uVar4;
  }
LAB_14076c3a8:
  if (lVar6 + -1 < 0) {
    return;
  }
  FUN_14230cba1();
  return;
LAB_14230cb27:
  *param_1 = uVar1;
  *puVar7 = uVar4;
  if ((longlong)((longlong)puVar7 + (-4 - (longlong)param_1) & 0xfffffffffffffffcU) <
      (longlong)((longlong)puVar5 - (longlong)puVar3 & 0xfffffffffffffffcU)) {
    if (puVar3 < puVar5) {
      local_1e8[lVar6] = puVar3;
      alStack_f8[lVar6] = (longlong)puVar5;
      lVar6 = lVar6 + 1;
    }
    if (puVar7 <= param_1 + 1) goto LAB_14076c3a8;
    puVar5 = puVar8 + -2;
    goto LAB_14076c352;
  }
  if (param_1 + 1 < puVar7) {
    local_1e8[lVar6] = param_1;
    alStack_f8[lVar6] = (longlong)(puVar8 + -2);
    lVar6 = lVar6 + 1;
  }
  param_1 = puVar3;
  if (puVar5 <= puVar3) goto LAB_14076c3a8;
  goto LAB_14076c352;
}

