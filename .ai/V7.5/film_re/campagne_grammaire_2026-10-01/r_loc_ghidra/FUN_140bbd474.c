
int FUN_140bbd474(longlong param_1,int *param_2,int *param_3)

{
  int iVar1;
  int iVar2;
  longlong lVar3;
  byte bVar4;
  undefined4 uVar5;
  int iVar6;
  int *piVar7;
  longlong *plVar8;
  ulonglong uVar9;
  int *piVar10;
  undefined4 *puVar11;
  int *piVar12;
  uint uVar13;
  longlong lVar14;
  int *piVar16;
  undefined1 local_128 [8];
  undefined *local_120;
  undefined *local_118;
  int local_110;
  undefined4 local_10c;
  int local_108;
  undefined1 local_104;
  int local_100;
  int local_fc;
  ulonglong local_f8;
  uint local_f0;
  undefined *local_e8;
  undefined4 local_e0;
  undefined8 local_58;
  int *piVar15;
  
  piVar10 = (int *)0x0;
  *param_3 = 0;
  iVar2 = *param_2;
  lVar14 = (longlong)iVar2;
  lVar3 = *(longlong *)(*(longlong *)(param_1 + 0x28) + 0x130 + lVar14 * 8);
  uVar5 = FUN_14076c748(**(undefined8 **)(param_2 + 2));
  bVar4 = FUN_1405f0adc(&DAT_144de3ea0,uVar5);
  uVar9 = (ulonglong)bVar4;
  if (bVar4 == 0) {
    puVar11 = &DAT_14521d920;
    piVar7 = &DAT_144706108;
    piVar12 = &DAT_144e61f6c;
  }
  else {
    puVar11 = (undefined4 *)&DAT_145225910;
    piVar7 = &DAT_14470610c;
    piVar12 = &DAT_144e61f70;
  }
  puVar11 = puVar11 + lVar14 * 4;
  iVar6 = puVar11[2];
  if (iVar6 < 1) {
    local_120 = &DAT_14520b920;
    local_10c = 1;
    local_108 = 1;
    piVar16 = (int *)(lVar3 + 0x14);
    if (bVar4 == 0) {
      local_120 = &DAT_1451f9920;
    }
    local_100 = 0;
    local_e0 = 0;
    local_104 = 0;
    local_58 = 0;
    local_120 = local_120 + *piVar12;
    local_110 = 0x12000 - *piVar12;
    local_118 = local_120 + local_110;
    local_f0 = 8;
    local_fc = 8;
    local_f8 = (ulonglong)*(uint *)(lVar3 + 4) | 0x80;
    piVar15 = piVar10;
    local_e8 = local_120;
    do {
      iVar6 = *piVar16;
      plVar8 = *(longlong **)
                (*(longlong *)(*(longlong *)(*(longlong *)(param_1 + 0x28) + 8) + 0x18) + 0x210 +
                (longlong)*(int *)(lVar3 + 4) * 8);
      (**(code **)(*plVar8 + 0x58))(plVar8,piVar15);
      if (local_f0 < 0x40) {
        local_f0 = local_f0 + 1;
        local_fc = local_fc + 1;
        local_f8 = local_f8 * 2 | (ulonglong)(iVar6 != -1);
      }
      else {
        FUN_1406d6e28(local_128,(ulonglong)(iVar6 != -1),1);
      }
      if (iVar6 != -1) {
        FUN_14080ada0(*(undefined4 *)(lVar3 + 4));
        FUN_1406d5110();
      }
      uVar13 = (int)piVar15 + 1;
      piVar15 = (int *)(ulonglong)uVar13;
      piVar16 = piVar16 + 1;
    } while ((int)uVar13 < 3);
    if (0 < *(int *)(lVar3 + 0x28)) {
      FUN_1424d80bc(*(undefined8 *)(param_1 + 0x28),*(undefined4 *)(lVar3 + 4),
                    *(int *)(lVar3 + 0x28),*(undefined8 *)(lVar3 + 0x20),local_128);
    }
    FUN_1406d6d94();
    uVar13 = local_f0;
    if (1 < local_108 - 1U) {
      uVar13 = local_f0 - (-(uint)(local_f0 != 0) & 0x40);
    }
    *param_3 = uVar13 + local_100;
    *puVar11 = *(undefined4 *)(lVar3 + 4);
    iVar6 = *piVar12;
    puVar11[1] = iVar6;
    puVar11[2] = *param_3;
    uVar13 = local_f0;
    if (1 < local_108 - 1U) {
      uVar13 = local_f0 - (-(uint)(local_f0 != 0) & 0x40);
    }
    *(undefined1 *)(puVar11 + 3) = 0;
    iVar1 = uVar13 + local_100 + 7;
    uVar9 = (longlong)iVar1 % 8 & 0xffffffff;
    *piVar12 = iVar1 / 8 + iVar6;
    iVar6 = iVar2;
    if (iVar2 < *piVar7) {
      iVar6 = *piVar7;
    }
    *piVar7 = iVar6;
    *(undefined1 *)((longlong)puVar11 + 0xd) = *(undefined1 *)(lVar3 + 4);
    *(undefined1 *)((longlong)puVar11 + 0xe) = 0xff;
    iVar6 = *param_3;
  }
  else {
    *param_3 = iVar6;
  }
  if (param_2[9] < iVar6) {
    return 0;
  }
  piVar7 = *(int **)(param_1 + 0x18);
  if ((piVar7 == (int *)0x0) || (*piVar7 != param_2[4])) {
    piVar7 = (int *)FUN_1406d096c(0x18,uVar9);
    if (piVar7 != (int *)0x0) {
      *piVar7 = -1;
      piVar7[2] = 0;
      piVar7[3] = 0;
      piVar7[4] = 0;
      piVar7[5] = 0;
      piVar10 = piVar7;
      if (piVar7 != (int *)0x0) {
        *(undefined8 *)(piVar7 + 4) = *(undefined8 *)(param_1 + 0x18);
        *(int *)(param_1 + 0x20) = *(int *)(param_1 + 0x20) + 1;
        *(int **)(param_1 + 0x18) = piVar7;
        piVar7[2] = 0;
        piVar7[3] = 0;
        *piVar7 = param_2[4];
        goto LAB_140bbd73a;
      }
    }
    *(undefined1 *)(param_1 + 0x11) = 1;
    piVar7 = piVar10;
  }
LAB_140bbd73a:
  if (*(char *)(param_1 + 0x11) == '\0') {
    plVar8 = (longlong *)FUN_1406d096c(0x10);
    if (plVar8 != (longlong *)0x0) {
      *plVar8 = 0;
      plVar8[1] = 0;
      if (plVar8 != (longlong *)0x0) {
        plVar8[1] = *(longlong *)(piVar7 + 2);
        *(longlong **)(piVar7 + 2) = plVar8;
        *plVar8 = lVar3;
        *(int *)(param_1 + 0x34) = *(int *)(param_1 + 0x34) + -1;
        *(int *)(param_1 + 0x38) = *(int *)(param_1 + 0x38) + 1;
        *(ulonglong *)(lVar3 + 0x38) =
             *(ulonglong *)(lVar3 + 0x38) | 1L << ((longlong)*(int *)(param_1 + 0x14) & 0x3fU);
        *(int *)(param_1 + 0x40 + (longlong)*(int *)(param_1 + 0x3c) * 4) = iVar2;
        *(int *)(param_1 + 0x3c) = *(int *)(param_1 + 0x3c) + 1;
        return *param_3;
      }
    }
    *(undefined1 *)(param_1 + 0x11) = 1;
  }
  return 0;
}

