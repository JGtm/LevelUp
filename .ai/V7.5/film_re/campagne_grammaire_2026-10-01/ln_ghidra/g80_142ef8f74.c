
undefined8
FUN_142ef8f74(undefined8 param_1,undefined8 param_2,undefined4 *param_3,undefined8 param_4)

{
  char cVar1;
  undefined4 *puVar2;
  undefined4 local_res18 [2];
  undefined1 local_res20 [8];
  
  local_res18[0] = 0xffffffff;
  cVar1 = FUN_1406cf008(param_4);
  if (cVar1 == '\0') {
    local_res18[0] = 0xffffffff;
  }
  else {
    FUN_14080d6f0();
  }
  cVar1 = FUN_1405838f0(local_res18);
  if (cVar1 == '\0') {
    *param_3 = 0xffffffff;
  }
  else {
    puVar2 = (undefined4 *)FUN_1407f21b4(local_res20,local_res18);
    *param_3 = *puVar2;
  }
  FUN_142ef15e0(param_4);
  return 1;
}

