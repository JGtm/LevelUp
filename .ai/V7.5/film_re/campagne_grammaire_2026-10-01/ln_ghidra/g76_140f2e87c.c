
undefined8 FUN_140f2e87c(undefined8 param_1,undefined8 param_2,longlong param_3,undefined8 param_4)

{
  PSRWLOCK SRWLock;
  undefined4 uVar1;
  uint uVar2;
  char cVar3;
  undefined1 uVar4;
  uint *puVar5;
  longlong lVar6;
  uint uVar7;
  uint local_res18 [2];
  undefined4 local_48 [2];
  undefined4 local_40;
  uint local_3c;
  undefined1 local_38 [8];
  uint local_30;
  undefined4 uStack_2c;
  longlong local_28;
  
  uVar7 = 0xffffffff;
  FUN_14080d69c(param_1,param_4,param_3,0xffffffff);
  uVar1 = DAT_144b404f0;
  local_res18[0] = 0xffffffff;
  cVar3 = FUN_1406cf008(param_4);
  if (cVar3 == '\0') {
    local_res18[0] = 0xffffffff;
  }
  else {
    FUN_14080d6f0();
  }
  uVar2 = local_res18[0];
  local_48[0] = uVar1;
  if (1 < local_res18[0] + 1) {
    local_40 = 0;
    local_3c = local_res18[0];
    puVar5 = (uint *)FUN_140821f44(DAT_144eae7b8,local_38,&local_3c,&local_40,1);
    uVar7 = *puVar5;
    local_3c = uVar7;
    cVar3 = FUN_1404785a0(&local_3c);
    if (cVar3 != '\0') {
      uStack_2c = uVar1;
      local_30 = uVar2;
      lVar6 = FUN_1405a7654();
      SRWLock = (PSRWLOCK)(lVar6 + 0x80);
      AcquireSRWLockShared(SRWLock);
      FUN_140647820(SRWLock,&local_3c,&local_30);
      ReleaseSRWLockShared(SRWLock);
      if ((*(byte *)((ulonglong)(local_3c & 0xffffff) * 0x10 + *(longlong *)(lVar6 + 0xc0) + 7) &
          0x3f) == 3) goto LAB_140f2e96e;
    }
    local_3c = uVar2;
    FUN_140ea752c(&local_3c,local_48,"GetLocalHandleFromGlobalId could not find local tag handle");
    FUN_14064697c(&local_30,local_res18,local_48,2);
    (**(code **)(*(longlong *)CONCAT44(uStack_2c,local_30) + 0x20))
              ((longlong *)CONCAT44(uStack_2c,local_30),DAT_144c2a2a8,0,0);
    local_3c = 0;
    puVar5 = (uint *)FUN_140821f44(DAT_144eae7b8,local_48,local_res18,&local_3c,1);
    uVar7 = *puVar5;
    if (local_28 != 0) {
      FUN_1404f965c();
    }
  }
LAB_140f2e96e:
  *(uint *)(param_3 + 4) = uVar7;
  uVar4 = FUN_1407f2058(param_4);
  *(undefined1 *)(param_3 + 8) = uVar4;
  FUN_1406ce648(param_4);
  return 1;
}

