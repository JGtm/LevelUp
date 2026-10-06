void FUN_140ac7430(undefined8 *param_1,uint param_2,byte param_3)
{
  byte local_res8 [8];
  byte local_res18 [16];
  local_res8[0] = param_3;
  if ((param_2 < 7) && (*(short *)(param_1 + 2) == 2)) {
    local_res18[0] = ((char)param_2 + '\x01') * ' ' | param_3;
    FUN_140ac78d0(*param_1,local_res18,1);
  }
  else {
    FUN_140ac78d0(*param_1,local_res8,1);
    FUN_140ac7668(*param_1,param_2);
  }
  return;
}
