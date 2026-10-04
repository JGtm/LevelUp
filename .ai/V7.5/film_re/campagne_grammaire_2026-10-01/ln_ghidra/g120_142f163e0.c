
undefined4
FUN_142f163e0(undefined8 param_1,undefined8 param_2,undefined4 *param_3,undefined8 param_4,
             undefined1 param_5)

{
  undefined4 uVar1;
  int iVar2;
  undefined4 *puVar3;
  undefined1 local_res18 [8];
  
  uVar1 = FUN_1424e2f20(param_4);
  puVar3 = (undefined4 *)FUN_140495860(local_res18,uVar1);
  *param_3 = *puVar3;
  FUN_14076e494(param_4,param_3 + 1,0x10,1,param_5,0);
  iVar2 = FUN_1424e1d48(param_4);
  param_3[4] = iVar2;
  for (puVar3 = param_3 + 6; puVar3 != param_3 + (longlong)iVar2 + 6; puVar3 = puVar3 + 1) {
    FUN_1406d3140();
  }
  iVar2 = FUN_1424e1d48(param_4);
  param_3[0x12] = iVar2;
  for (puVar3 = param_3 + 0x14; puVar3 != param_3 + (longlong)iVar2 + 0x14; puVar3 = puVar3 + 1) {
    FUN_1406d3140();
  }
  return 1;
}

