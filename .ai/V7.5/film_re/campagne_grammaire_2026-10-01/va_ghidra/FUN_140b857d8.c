/* WARNING: Function: _alloca_probe replaced with injection: alloca_probe */
bool FUN_140b857d8(undefined8 param_1,undefined8 param_2)
{
  char cVar1;
  undefined ***local_res18 [2];
  undefined **local_13f8;
  undefined8 local_13f0;
  undefined8 ***local_13e8;
  undefined1 local_13e0;
  undefined ****local_13d8;
  undefined8 local_13d0;
  undefined2 local_13c8;
  undefined4 local_13c0;
  undefined4 local_13bc;
  undefined1 local_13b8 [256];
  undefined1 *local_12b8;
  undefined4 local_12b0;
  undefined4 local_12ac;
  undefined1 local_12a8 [256];
  undefined1 *local_11a8;
  undefined1 local_1198 [4472];
  undefined1 local_20 [16];
  undefined8 uStack_10;
  uStack_10 = 0x140b857f5;
  FUN_14064c240(local_1198);
  cVar1 = FUN_140b85ab4(param_2,local_1198);
  if (cVar1 != '\0') {
    local_13f8 = &PTR_FUN_143d45e08;
    local_res18[0] = &local_13f8;
    local_13d8 = local_res18;
    local_13d0 = 0;
    local_13c8 = 2;
    local_13c0 = 0;
    local_13bc = 0x40;
    local_12b8 = local_13b8;
    local_12b0 = 0;
    local_12ac = 0x40;
    local_11a8 = local_12a8;
    local_13e8 = &local_13d8;
    local_13e0 = 0;
    local_13f0 = param_1;
    FUN_140b85a0c(&local_13e8,local_1198);
    FUN_140dc8a80(&local_12b0);
    FUN_140dc8a80(&local_13c0);
  }
  FUN_14091a5dc(local_20);
  return cVar1 != '\0';
}
