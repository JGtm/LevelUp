
longlong * FUN_14076bf98(longlong *param_1,undefined8 param_2,undefined8 param_3,undefined8 param_4)

{
  longlong *plVar1;
  int *piVar2;
  int iVar3;
  uint uVar4;
  undefined8 uVar5;
  int iVar6;
  uint uVar7;
  char cVar8;
  uint uVar9;
  longlong lVar10;
  longlong lVar11;
  uint *puVar12;
  ulonglong uVar13;
  undefined4 uVar14;
  uint uVar15;
  uint uVar16;
  uint local_res8 [2];
  undefined1 local_res10 [8];
  undefined1 local_res18 [8];
  undefined4 local_res20;
  undefined2 local_res24;
  int iStack_60;
  int iStack_5c;
  undefined8 local_58;
  
  uVar16 = 0;
  plVar1 = param_1 + 1;
  *(undefined8 *)((longlong)param_1 + 0xac) = 0;
  uVar14 = 0;
  if ((char)param_1[0x1f] == '\0') {
    lVar10 = *param_1;
    local_res20 = *(undefined4 *)(lVar10 + 0x24);
    uVar5 = *(undefined8 *)(lVar10 + 0x18);
    local_res24 = *(undefined2 *)(lVar10 + 0x28);
    *plVar1 = (longlong)param_1;
    *(undefined4 *)(param_1 + 3) = 0;
    if (*(int *)(lVar10 + 4) == 2) {
      *(bool *)(param_1 + 2) = *(char *)(lVar10 + 200) == '\0';
      cVar8 = FUN_1409cbf3c();
      if (cVar8 != '\0') {
        uVar14 = *(undefined4 *)(lVar10 + 0xcc);
      }
      *(undefined4 *)((longlong)param_1 + 0x14) = uVar14;
      cVar8 = FUN_1404f1b74();
      if (cVar8 != '\0') {
        do {
          lVar10 = FUN_1405f484c(uVar5,&local_res20,uVar16);
          if (lVar10 != 0) {
            local_res8[0] = *(uint *)(lVar10 + 4);
            uVar9 = local_res8[0] >> 1 & 0x7fff;
            lVar10 = (ulonglong)uVar9 * 2 + (ulonglong)uVar9;
            piVar2 = (int *)(DAT_1451d2638 + lVar10 * 8);
            iVar6 = *piVar2;
            iStack_60 = piVar2[2];
            iStack_5c = piVar2[3];
            local_58 = *(undefined8 *)(DAT_1451d2638 + 0x10 + lVar10 * 8);
            iVar3 = *(int *)((longlong)param_1 + 0xac);
            uVar13 = 0x10 - (longlong)iVar3;
            if ((ulonglong)(longlong)iVar6 <= uVar13) {
              uVar13 = (longlong)iVar6;
            }
            if (uVar13 != 0) {
              *(int *)((longlong)param_1 + 0xac) = iVar3 + (int)uVar13;
              FUN_142c3d114(&iStack_60,iVar6,(longlong)param_1 + ((longlong)iVar3 + 0x2d) * 4,
                            param_4,iVar6);
            }
            lVar10 = FUN_140495b84(local_res8);
            uVar15 = *(uint *)(lVar10 + 0x2e0);
            local_res8[0] = uVar15;
            if (uVar15 == 0xffffffff) {
              lVar11 = FUN_1424c76d0(lVar10);
              if (*(char *)(lVar11 + 0x28f) == '\0') {
                cVar8 = FUN_1406ff920(lVar10);
                if ((cVar8 != '\0') && (lVar11 = FUN_142b7a4e0(lVar10), lVar11 != 0)) {
                  puVar12 = (uint *)FUN_1406ff8f0(lVar10,local_res10);
                  uVar15 = *puVar12;
                  local_res8[0] = uVar15;
                }
                lVar10 = FUN_140495abc(lVar10 + 0x15c);
                uVar4 = uVar15;
                uVar7 = local_res8[0];
                if ((((lVar10 != 0) &&
                     (uVar4 = *(uint *)(lVar10 + 0x2e0), uVar7 = uVar4, uVar4 == 0xffffffff)) &&
                    (cVar8 = FUN_1406ff920(lVar10), uVar4 = uVar15, uVar7 = local_res8[0],
                    cVar8 != '\0')) &&
                   (lVar11 = FUN_142b7a4e0(lVar10), uVar7 = local_res8[0], lVar11 != 0)) {
                  puVar12 = (uint *)FUN_1406ff8f0(lVar10,local_res18);
                  uVar4 = *puVar12;
                  uVar7 = *puVar12;
                }
              }
              else {
                *(undefined4 *)((longlong)param_1 + (longlong)(int)param_1[3] * 0x24 + 0x20) =
                     0xffffffff;
                lVar11 = (longlong)(int)param_1[3] + 1;
                *(undefined8 *)((longlong)param_1 + lVar11 * 0x24) = *(undefined8 *)(lVar10 + 0x144)
                ;
                *(undefined4 *)((longlong)param_1 + lVar11 * 0x24 + 8) =
                     *(undefined4 *)(lVar10 + 0x14c);
                lVar11 = param_1[3];
                *(undefined8 *)((longlong)param_1 + (longlong)(int)lVar11 * 0x24 + 0x30) =
                     *(undefined8 *)(lVar10 + 0x150);
                *(undefined4 *)((longlong)param_1 + (longlong)(int)lVar11 * 0x24 + 0x38) =
                     *(undefined4 *)(lVar10 + 0x158);
                *(undefined1 *)((longlong)param_1 + (longlong)(int)param_1[3] * 0x24 + 0x3c) = 0xff;
                *(uint *)((longlong)param_1 + (longlong)(int)param_1[3] * 0x24 + 0x1c) = uVar9;
                *(int *)(param_1 + 3) = (int)param_1[3] + 1;
                uVar4 = uVar15;
                uVar7 = local_res8[0];
              }
              local_res8[0] = uVar7;
              uVar15 = uVar4;
              if (uVar15 == 0xffffffff) goto LAB_14230c9f9;
            }
            lVar11 = FUN_140471c88(local_res8);
            *(undefined4 *)((longlong)param_1 + (longlong)(int)param_1[3] * 0x24 + 0x20) =
                 *(undefined4 *)(lVar11 + 0x114);
            FUN_1406aab68(uVar15,(longlong)param_1 + ((longlong)(int)param_1[3] + 1) * 0x24);
            lVar10 = param_1[3];
            *(undefined8 *)((longlong)param_1 + (longlong)(int)lVar10 * 0x24 + 0x30) =
                 *(undefined8 *)(lVar11 + 0x348);
            *(undefined4 *)((longlong)param_1 + (longlong)(int)lVar10 * 0x24 + 0x38) =
                 *(undefined4 *)(lVar11 + 0x350);
            *(undefined1 *)((longlong)param_1 + (longlong)(int)param_1[3] * 0x24 + 0x3c) =
                 *(undefined1 *)(lVar11 + 0x462);
            *(uint *)((longlong)param_1 + (longlong)(int)param_1[3] * 0x24 + 0x1c) = uVar9;
            *(int *)(param_1 + 3) = (int)param_1[3] + 1;
          }
LAB_14230c9f9:
          uVar9 = uVar16 + 1;
          uVar16 = 0xffffffff;
          if (uVar9 < 4) {
            uVar16 = uVar9;
          }
        } while (uVar16 != 0xffffffff);
      }
    }
    else {
      *(undefined4 *)((longlong)param_1 + 0x14) = 0xffffffff;
      *(undefined1 *)(param_1 + 2) = 0;
    }
  }
  else {
    *(undefined4 *)((longlong)param_1 + 0x14) = 0xffffffff;
    *plVar1 = (longlong)param_1;
    *(undefined4 *)(param_1 + 3) = 0;
    *(undefined1 *)(param_1 + 2) = 0;
  }
  return plVar1;
}

