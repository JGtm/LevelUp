
undefined8 FUN_142eebcec(undefined8 param_1,undefined8 param_2,longlong param_3,undefined8 param_4)

{
  int iVar1;
  longlong lVar2;
  
  FUN_14076dc04(param_4,param_2,param_3,0x13);
  iVar1 = FUN_142ed0abc(param_4);
  *(int *)(param_3 + 0xc) = iVar1;
  for (lVar2 = param_3 + 0x14; lVar2 != param_3 + 0x14 + (longlong)iVar1 * 0x10;
      lVar2 = lVar2 + 0x10) {
    FUN_1406d3140();
    FUN_14076dc04(param_4);
  }
  return 1;
}

