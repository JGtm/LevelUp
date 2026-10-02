
void FUN_14076ac14(longlong param_1,uint param_2,undefined8 *param_3)

{
  undefined8 *puVar1;
  undefined8 uVar2;
  undefined8 uVar3;
  undefined8 uVar4;
  undefined8 uVar5;
  undefined8 uVar6;
  undefined8 uVar7;
  undefined8 uVar8;
  undefined8 uVar9;
  undefined8 uVar10;
  undefined8 uVar11;
  undefined8 uVar12;
  undefined8 uVar13;
  longlong lVar14;
  
  uVar3 = param_3[1];
  uVar4 = param_3[2];
  uVar5 = param_3[3];
  uVar6 = param_3[4];
  uVar7 = param_3[5];
  uVar8 = param_3[6];
  uVar9 = param_3[7];
  uVar10 = param_3[8];
  uVar11 = param_3[9];
  uVar2 = param_3[0xc];
  lVar14 = (longlong)(int)param_2 * 0x68;
  uVar12 = param_3[10];
  uVar13 = param_3[0xb];
  puVar1 = (undefined8 *)(lVar14 + 0x25f0 + param_1);
  *puVar1 = *param_3;
  puVar1[1] = uVar3;
  puVar1 = (undefined8 *)(lVar14 + 0x2600 + param_1);
  *puVar1 = uVar4;
  puVar1[1] = uVar5;
  puVar1 = (undefined8 *)(lVar14 + 0x2610 + param_1);
  *puVar1 = uVar6;
  puVar1[1] = uVar7;
  puVar1 = (undefined8 *)(lVar14 + 0x2620 + param_1);
  *puVar1 = uVar8;
  puVar1[1] = uVar9;
  puVar1 = (undefined8 *)(lVar14 + 0x2630 + param_1);
  *puVar1 = uVar10;
  puVar1[1] = uVar11;
  puVar1 = (undefined8 *)(lVar14 + 0x2640 + param_1);
  *puVar1 = uVar12;
  puVar1[1] = uVar13;
  *(undefined8 *)(lVar14 + 0x2650 + param_1) = uVar2;
  *(uint *)(param_1 + 0x2550) = *(uint *)(param_1 + 0x2550) | 1 << (param_2 & 0x1f);
  return;
}

