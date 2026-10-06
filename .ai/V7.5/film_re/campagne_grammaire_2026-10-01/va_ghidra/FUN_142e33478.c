
void FUN_142e33478(uint param_1)

{
  int iVar1;
  char cVar2;
  undefined4 uVar3;
  longlong lVar4;
  undefined8 uVar5;
  ulonglong uVar6;
  undefined8 uVar7;
  undefined4 *puVar8;
  undefined4 *puVar9;
  undefined4 local_res8 [2];
  undefined4 *local_48;
  undefined4 *puStack_40;
  undefined4 *local_38;
  longlong local_30;
  undefined4 local_28;
  undefined4 local_24;
  undefined4 local_20;
  undefined4 local_1c;
  ulonglong local_18;
  undefined4 local_10;
  
  uVar7 = 2;
  if ((param_1 & 2) != 0) {
    FUN_142e34d10();
  }
  iVar1 = *(int *)(DAT_144e61d78 + 0x10);
  if (iVar1 == 0) {
    uVar5 = 0;
LAB_142e334d4:
    FUN_140be9658(uVar5);
  }
  else {
    if (iVar1 == 1) {
      uVar5 = 1;
      goto LAB_142e334d4;
    }
    if (iVar1 == 3) {
      uVar5 = 3;
      goto LAB_142e334d4;
    }
    if ((iVar1 == 4) || (iVar1 == 5)) {
      uVar5 = 2;
      goto LAB_142e334d4;
    }
  }
  iVar1 = *(int *)(DAT_144e61d78 + 0x10);
  if ((iVar1 != 0) && (iVar1 != 1)) {
    if (iVar1 == 2) {
      uVar7 = 1;
      goto LAB_142e334fd;
    }
    if ((iVar1 != 3) && (iVar1 != 4)) goto LAB_142e334fd;
  }
  uVar7 = 0;
LAB_142e334fd:
  FUN_142bba074(uVar7);
  cVar2 = FUN_1404f1ca4();
  if (cVar2 != '\0') {
    FUN_1410fe7f8();
    FUN_142e2eaa0(DAT_144e61d78);
    if (DAT_1452f2f08 != (longlong *)0x0) {
      (**(code **)(*DAT_1452f2f08 + 0x2b0))();
    }
    FUN_142e32dd0(DAT_144e61d78);
    FUN_142e2e8e0(DAT_144e61d78);
    local_48 = (undefined4 *)0x0;
    puStack_40 = (undefined4 *)0x0;
    local_38 = (undefined4 *)0x0;
    lVar4 = FUN_140973988();
    uVar6 = (ulonglong)(longlong)*(int *)(lVar4 + 0x70) >> 1;
    if (uVar6 != 0) {
      if (0x3fffffffffffffff < uVar6) {
                    /* WARNING: Subroutine does not return */
        FUN_141c10040();
      }
      FUN_14270d08c(&local_48);
    }
    local_30 = 0;
    local_10 = 0x86868686;
    local_28 = 0xffffffff;
    local_24 = 0;
    local_20 = 0;
    local_1c = 0xffffffff;
    local_18 = (ulonglong)*(byte *)(*(longlong *)ThreadLocalStoragePointer + 0x810);
    cVar2 = FUN_140477ea0(&local_30);
    puVar9 = puStack_40;
    while (lVar4 = local_30, puVar8 = local_48, cVar2 != '\0') {
      if ((*(int *)(local_30 + 0x114) != -1) && ((*(ulonglong *)(local_30 + 0x20) & 0x10000) == 0))
      {
        local_res8[0] = local_1c;
        if (puVar9 == local_38) {
          FUN_142641200(&local_48,puVar9,local_res8);
          puVar9 = puStack_40;
        }
        else {
          *puVar9 = local_1c;
          puStack_40 = puVar9 + 1;
          puVar9 = puStack_40;
        }
      }
      FUN_1408f1144(lVar4);
      *(undefined4 *)(lVar4 + 0x114) = 0xffffffff;
      cVar2 = FUN_140477ea0(&local_30);
    }
    for (; puVar8 != puVar9; puVar8 = puVar8 + 1) {
      FUN_140e1b494(*puVar8);
    }
    FUN_142b70920();
    if (local_48 != (undefined4 *)0x0) {
      FUN_1405a3720(local_48);
    }
  }
  FUN_142e2dc70(DAT_144e61d80);
  FUN_140beb574(DAT_144e61d78);
  FUN_142e32488(DAT_144e61d80,param_1);
  uVar3 = FUN_1404f16fc();
  FUN_142e32d60(DAT_144e61d78,uVar3);
  FUN_140be91d0();
  DAT_144e61d6e = 0;
  return;
}

