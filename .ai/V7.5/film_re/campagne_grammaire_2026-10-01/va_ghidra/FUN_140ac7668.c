void FUN_140ac7668(undefined8 param_1,uint param_2)
{
  byte local_res10 [24];
  local_res10[0] = (byte)param_2;
  if (param_2 >> 7 == 0) {
    FUN_140ac78d0(param_1,local_res10,1);
  }
  else {
    local_res10[0] = local_res10[0] | 0x80;
    FUN_140ac78d0(param_1,local_res10,1);
    thunk_FUN_140ac7668(param_1,param_2 >> 7);
  }
  return;
}
