
undefined1 FUN_142f17480(undefined8 param_1,undefined8 param_2,longlong param_3,undefined8 param_4)

{
  undefined4 *puVar1;
  char cVar2;
  undefined4 uVar3;
  undefined1 local_res18 [16];
  
  FUN_14080d69c(param_1,param_4,param_3,0xffffffff);
  cVar2 = FUN_1405838f0();
  if (cVar2 == '\0') {
    uVar3 = 0xffffffff;
  }
  else {
    puVar1 = (undefined4 *)FUN_1407f21b4(local_res18,param_3);
    uVar3 = *puVar1;
  }
  *(undefined4 *)(param_3 + 4) = uVar3;
  *(undefined4 *)(param_3 + 8) = 0xffffffff;
  return 1;
}

