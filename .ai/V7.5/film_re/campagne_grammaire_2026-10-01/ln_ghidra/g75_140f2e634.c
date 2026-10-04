
/* WARNING: Type propagation algorithm not settling */

undefined8
FUN_140f2e634(undefined8 param_1,undefined8 param_2,undefined4 *param_3,undefined8 param_4)

{
  PSRWLOCK SRWLock;
  undefined4 uVar1;
  char cVar2;
  undefined1 uVar3;
  undefined2 uVar4;
  int iVar5;
  undefined4 *puVar6;
  longlong lVar7;
  undefined4 uVar8;
  int iVar9;
  int local_res18 [4];
  undefined4 local_78 [2];
  undefined4 local_70;
  int local_6c;
  undefined4 local_68;
  uint local_64;
  int local_60 [3];
  int local_54;
  undefined4 local_50;
  undefined1 local_4c [4];
  undefined1 local_48 [8];
  longlong *local_40;
  longlong local_38;
  
  FUN_140f2ea18(param_4,param_2,(longlong)param_3 + 6);
  uVar1 = DAT_144b404f0;
  uVar8 = 0xffffffff;
  local_res18[0] = -1;
  cVar2 = FUN_1406cf008(param_4);
  iVar9 = 0;
  if (cVar2 == '\0') {
    local_res18[0] = -1;
  }
  else {
    FUN_14080d6f0();
  }
  iVar5 = local_res18[0];
  local_78[0] = uVar1;
  if (1 < local_res18[0] + 1U) {
    local_70 = 0;
    local_6c = local_res18[0];
    puVar6 = (undefined4 *)FUN_140821f44(DAT_144eae7b8,local_4c,&local_6c,&local_70,1);
    uVar8 = *puVar6;
    local_68 = uVar8;
    cVar2 = FUN_1404785a0(&local_68);
    if (cVar2 != '\0') {
      local_50 = uVar1;
      local_54 = iVar5;
      lVar7 = FUN_1405a7654();
      SRWLock = (PSRWLOCK)(lVar7 + 0x80);
      AcquireSRWLockShared(SRWLock);
      FUN_140647820(SRWLock,&local_64,&local_54);
      ReleaseSRWLockShared(SRWLock);
      if ((*(byte *)((ulonglong)(local_64 & 0xffffff) * 0x10 + *(longlong *)(lVar7 + 0xc0) + 7) &
          0x3f) == 3) goto LAB_140f2e72b;
    }
    local_60[0] = iVar5;
    FUN_140ea752c(local_60,local_78,"GetLocalHandleFromGlobalId could not find local tag handle");
    FUN_14064697c(&local_40,local_res18,local_78,2);
    (**(code **)(*local_40 + 0x20))(local_40,DAT_144c2a2a8,0,0);
    local_60[1] = 0;
    local_60[2] = local_res18[0];
    puVar6 = (undefined4 *)FUN_140821f44(DAT_144eae7b8,local_48,local_60 + 2,local_60 + 1,1);
    uVar8 = *puVar6;
    if (local_38 != 0) {
      FUN_1404f965c();
    }
  }
LAB_140f2e72b:
  *param_3 = uVar8;
  uVar4 = FUN_140f2e7a4(param_4);
  *(undefined2 *)(param_3 + 1) = uVar4;
  uVar3 = FUN_1407f2058(param_4);
  *(undefined1 *)((longlong)param_3 + 7) = uVar3;
  iVar5 = FUN_1424e1464(param_4);
  if (0 < iVar5) {
    do {
      uVar3 = FUN_1406cf008(param_4);
      FUN_1407688b0(param_3 + 2,iVar9,uVar3);
      iVar9 = iVar9 + 1;
    } while (iVar9 < iVar5);
  }
  return 1;
}

