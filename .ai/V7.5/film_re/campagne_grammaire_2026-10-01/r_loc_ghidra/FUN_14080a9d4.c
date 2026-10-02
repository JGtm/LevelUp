
undefined8 FUN_14080a9d4(longlong param_1,undefined4 param_2,longlong param_3)

{
  longlong *plVar1;
  ulonglong uVar2;
  char cVar3;
  undefined4 uVar4;
  undefined1 *puVar5;
  undefined8 uVar6;
  ulonglong *puVar7;
  undefined1 *puVar8;
  uint uVar9;
  undefined1 *puVar10;
  int iVar11;
  uint uVar12;
  uint uVar14;
  undefined1 *puVar15;
  int local_res18;
  undefined1 *local_878;
  undefined1 local_870 [24];
  undefined1 local_858 [2080];
  undefined1 *puVar13;
  
  puVar10 = (undefined1 *)0x0;
  iVar11 = 0x40 - *(int *)(param_3 + 0x38);
  uVar12 = 0;
  uVar9 = (uint)((ulonglong)*(longlong *)(param_3 + 0x30) >> 0x20);
  if (iVar11 < 7) {
    puVar7 = *(ulonglong **)(param_3 + 0x40);
    if (*(ulonglong **)(param_3 + 0x10) < puVar7 + 1) {
      puVar8 = puVar10;
      puVar5 = puVar10;
      uVar14 = uVar12;
      if (puVar7 < *(ulonglong **)(param_3 + 0x10)) {
        do {
          uVar14 = (int)puVar8 + 8;
          puVar8 = (undefined1 *)(ulonglong)uVar14;
          uVar2 = *puVar7;
          puVar7 = (ulonglong *)((longlong)puVar7 + 1);
          puVar5 = (undefined1 *)((ulonglong)(byte)uVar2 | (longlong)puVar5 << 8);
          *(ulonglong **)(param_3 + 0x40) = puVar7;
        } while (puVar7 < *(ulonglong **)(param_3 + 0x10));
        puVar5 = (undefined1 *)((longlong)puVar5 << (-(char)uVar14 & 0x3fU));
      }
    }
    else {
      uVar2 = *puVar7;
      *(ulonglong **)(param_3 + 0x40) = puVar7 + 1;
      puVar5 = (undefined1 *)
               (uVar2 >> 0x38 | (uVar2 & 0xff000000000000) >> 0x28 |
                (uVar2 & 0xff0000000000) >> 0x18 | (uVar2 & 0xff00000000) >> 8 |
                (uVar2 & 0xff000000) << 8 | (uVar2 & 0xff0000) << 0x18 | (uVar2 & 0xff00) << 0x28 |
               uVar2 << 0x38);
      uVar14 = 0x40;
    }
    *(int *)(param_3 + 0x28) = *(int *)(param_3 + 0x28) + uVar14;
    *(int *)(param_3 + 0x2c) = *(int *)(param_3 + 0x2c) + 7;
    uVar14 = 7 - iVar11;
    *(ulonglong *)(param_3 + 0x30) =
         -(ulonglong)(uVar14 < 0x40) & (longlong)puVar5 << ((byte)uVar14 & 0x3f);
    *(uint *)(param_3 + 0x38) = uVar14;
    uVar9 = (uint)((ulonglong)puVar5 >> (0x40 - (byte)uVar14 & 0x3f)) | uVar9 >> 0x19;
  }
  else {
    *(int *)(param_3 + 0x2c) = *(int *)(param_3 + 0x2c) + 7;
    *(longlong *)(param_3 + 0x30) = *(longlong *)(param_3 + 0x30) << 7;
    *(int *)(param_3 + 0x38) = *(int *)(param_3 + 0x38) + 7;
    uVar9 = uVar9 >> 0x19;
  }
  if ((ulonglong)(longlong)(int)uVar9 < 0x7b) {
    plVar1 = *(longlong **)(*(longlong *)(param_1 + 0x18) + 0x210 + (longlong)(int)uVar9 * 8);
    cVar3 = FUN_1406cb0cc();
    puVar5 = puVar10;
    puVar8 = puVar10;
    if ((cVar3 == '\0') ||
       (puVar5 = (undefined1 *)FUN_14080abe4(uVar9,plVar1,param_2), puVar5 == (undefined1 *)0x0)) {
      local_878 = local_870;
      local_res18 = (**(code **)(*plVar1 + 0x10))(plVar1);
      puVar15 = puVar10;
      if (0 < local_res18) {
        memset(local_858,0,(longlong)local_res18);
        puVar15 = local_858;
      }
    }
    else {
      local_878 = puVar5 + 0x10;
      local_res18 = *(int *)(puVar5 + 0x1c);
      puVar15 = *(undefined1 **)(puVar5 + 0x20);
    }
    do {
      cVar3 = FUN_1406cf008(param_3);
      if (cVar3 != '\0') {
        (**(code **)(*plVar1 + 0x58))(plVar1,puVar8);
        FUN_14080ada0(uVar9);
        FUN_1406d3140();
      }
      uVar14 = (int)puVar8 + 1;
      puVar8 = (undefined1 *)(ulonglong)uVar14;
    } while ((int)uVar14 < 3);
    if (puVar15 != (undefined1 *)0x0) {
      cVar3 = (**(code **)(*plVar1 + 0x68))(plVar1,local_res18,puVar15,param_3,1);
      if (cVar3 == '\0') {
        if (puVar5 != (undefined1 *)0x0) {
          *(undefined8 *)(puVar5 + 8) = 0;
        }
        goto LAB_1422701d0;
      }
      cVar3 = FUN_14076cea8();
      if ((cVar3 != '\0') && (cVar3 = FUN_1406cf008(param_3), cVar3 != '\0')) {
        iVar11 = *(int *)(param_3 + 0x38);
        if (0x40 - iVar11 < 0x20) {
          puVar7 = *(ulonglong **)(param_3 + 0x40);
          if (*(ulonglong **)(param_3 + 0x10) < puVar7 + 1) {
            puVar8 = puVar10;
            puVar13 = puVar10;
            if (puVar7 < *(ulonglong **)(param_3 + 0x10)) {
              do {
                uVar2 = *puVar7;
                uVar12 = (int)puVar13 + 8;
                puVar13 = (undefined1 *)(ulonglong)uVar12;
                puVar7 = (ulonglong *)((longlong)puVar7 + 1);
                puVar8 = (undefined1 *)((longlong)puVar8 << 8 | (ulonglong)(byte)uVar2);
                *(ulonglong **)(param_3 + 0x40) = puVar7;
              } while (puVar7 < *(ulonglong **)(param_3 + 0x10));
              puVar8 = (undefined1 *)((longlong)puVar8 << (-(char)uVar12 & 0x3fU));
            }
          }
          else {
            uVar2 = *puVar7;
            uVar12 = 0x40;
            *(ulonglong **)(param_3 + 0x40) = puVar7 + 1;
            puVar8 = (undefined1 *)
                     (uVar2 >> 0x38 | (uVar2 & 0xff000000000000) >> 0x28 |
                      (uVar2 & 0xff0000000000) >> 0x18 | (uVar2 & 0xff00000000) >> 8 |
                      (uVar2 & 0xff000000) << 8 | (uVar2 & 0xff0000) << 0x18 |
                      (uVar2 & 0xff00) << 0x28 | uVar2 << 0x38);
          }
          *(int *)(param_3 + 0x28) = *(int *)(param_3 + 0x28) + uVar12;
          uVar9 = iVar11 - 0x20;
          *(int *)(param_3 + 0x2c) = *(int *)(param_3 + 0x2c) + 0x20;
          *(ulonglong *)(param_3 + 0x30) =
               -(ulonglong)(uVar9 < 0x40) & (longlong)puVar8 << ((byte)uVar9 & 0x3f);
          *(uint *)(param_3 + 0x38) = uVar9;
        }
        else {
          *(int *)(param_3 + 0x2c) = *(int *)(param_3 + 0x2c) + 0x20;
          *(longlong *)(param_3 + 0x30) = *(longlong *)(param_3 + 0x30) << 0x20;
          *(int *)(param_3 + 0x38) = iVar11 + 0x20;
        }
      }
    }
    cVar3 = FUN_1406cb0cc();
    if ((cVar3 != '\0') && (puVar5 == (undefined1 *)0x0)) {
      (**(code **)(*plVar1 + 0x70))(plVar1,local_res18,puVar15);
      iVar11 = (**(code **)(*plVar1 + 0x18))(plVar1);
      if (0 < iVar11) {
        puVar10 = local_878;
      }
      uVar4 = (**(code **)(*plVar1 + 0x18))(plVar1);
      FUN_14080a390(plVar1,uVar4,puVar10,local_res18,puVar15,param_2);
    }
    uVar6 = 0;
  }
  else {
LAB_1422701d0:
    uVar6 = 3;
  }
  return uVar6;
}

