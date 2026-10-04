
undefined1 FUN_142c61310(undefined8 param_1,undefined8 param_2,longlong param_3,undefined8 param_4)

{
  undefined4 *puVar1;
  char cVar2;
  undefined4 local_res18 [4];
  undefined1 local_18 [16];
  
  FUN_142af27f8(param_4);
  local_res18[0] = 0xffffffff;
  cVar2 = FUN_1406cf008(param_4);
  if (cVar2 == '\0') {
    local_res18[0] = 0xffffffff;
  }
  else {
    FUN_14080d6f0();
  }
  cVar2 = FUN_1405838f0(local_res18);
  if (cVar2 == '\0') {
    *(undefined4 *)(param_3 + 4) = 0xffffffff;
  }
  else {
    puVar1 = (undefined4 *)FUN_1407f21b4(local_18,local_res18);
    *(undefined4 *)(param_3 + 4) = *puVar1;
  }
  local_res18[0] = 0xffffffff;
  cVar2 = FUN_1406cf008(param_4);
  if (cVar2 == '\0') {
    local_res18[0] = 0xffffffff;
  }
  else {
    FUN_14080d6f0();
  }
  cVar2 = FUN_1405838f0(local_res18);
  if (cVar2 == '\0') {
    *(undefined4 *)(param_3 + 8) = 0xffffffff;
  }
  else {
    puVar1 = (undefined4 *)FUN_1407f21b4(local_18,local_res18);
    *(undefined4 *)(param_3 + 8) = *puVar1;
  }
  FUN_14080dec4(param_4,"marker-name",param_3 + 0xc);
  return 1;
}

