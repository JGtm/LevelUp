
undefined8 FUN_140f04fb8(undefined8 param_1,undefined8 param_2,longlong param_3,undefined8 param_4)

{
  char cVar1;
  undefined1 local_res18 [16];
  undefined8 local_18;
  undefined4 local_10;
  
  FUN_14080d69c(param_1,param_4,param_3,0xffffffff);
  cVar1 = FUN_14076f91c();
  if (cVar1 == '\0') {
    FUN_14076e524(&local_18,param_4,local_res18,0x10);
  }
  else {
    FUN_1411b259c(&local_18,param_4);
  }
  *(undefined8 *)(param_3 + 4) = local_18;
  *(undefined4 *)(param_3 + 0xc) = local_10;
  cVar1 = FUN_14076f91c();
  if (cVar1 == '\0') {
    FUN_14076e524(&local_18,param_4,local_res18,0x10);
  }
  else {
    FUN_1411b259c(&local_18,param_4);
  }
  *(undefined8 *)(param_3 + 0x10) = local_18;
  *(undefined4 *)(param_3 + 0x18) = local_10;
  return CONCAT71((uint7)(uint3)((uint)local_10 >> 8),1);
}

