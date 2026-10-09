
undefined8 FUN_141118a00(undefined8 param_1,undefined8 param_2,uint *param_3,undefined8 param_4)

{
  char cVar1;
  uint *puVar2;
  uint uVar3;
  uint local_res18 [4];
  undefined4 local_18;
  uint local_14;
  undefined1 local_10 [8];
  
  local_res18[0] = 0xffffffff;
  cVar1 = FUN_1406cf008(param_4);
  if (cVar1 == '\0') {
    local_res18[0] = 0xffffffff;
  }
  else {
    FUN_14080d6f0();
  }
  if (local_res18[0] + 1 < 2) {
    *param_3 = 0xffffffff;
  }
  else {
    local_18 = 0;
    local_14 = local_res18[0];
    puVar2 = (uint *)FUN_140821f44(DAT_144eae7b8,local_10,&local_14,&local_18,1);
    uVar3 = *puVar2;
    local_14 = uVar3;
    cVar1 = FUN_1404785a0(&local_14);
    if (cVar1 == '\0') {
      puVar2 = (uint *)FUN_14080d61c(&local_14,local_res18,DAT_144b404f0);
      uVar3 = *puVar2;
    }
    *param_3 = uVar3;
  }
  return 1;
}

