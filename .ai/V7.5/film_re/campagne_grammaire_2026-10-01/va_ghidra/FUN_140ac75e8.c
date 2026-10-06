void FUN_140ac75e8(undefined8 *param_1,byte param_2,ushort param_3)
{
  undefined8 uVar1;
  ushort *puVar2;
  undefined8 uVar3;
  ushort local_res10 [4];
  ushort local_res18 [8];
  uVar1 = *param_1;
  local_res18[0] = param_3;
  if (param_3 < 6) {
    local_res10[0]._0_1_ = (char)param_3 << 5 | param_2;
    puVar2 = local_res10;
    uVar3 = 1;
  }
  else {
    if (param_3 < 0x100) {
      local_res10[0]._0_1_ = param_2 | 0xc0;
      FUN_140ac78d0(uVar1,local_res10,1);
      puVar2 = local_res10;
      local_res10[0]._0_1_ = (byte)local_res18[0];
      uVar3 = 1;
    }
    else {
      local_res10[0]._0_1_ = param_2 | 0xe0;
      FUN_140ac78d0(uVar1,local_res10,1);
      uVar3 = 2;
      puVar2 = local_res18;
    }
    uVar1 = *param_1;
  }
  FUN_140ac78d0(uVar1,puVar2,uVar3);
  return;
}
