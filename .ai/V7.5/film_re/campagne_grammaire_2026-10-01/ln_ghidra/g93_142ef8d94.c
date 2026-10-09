
undefined4
FUN_142ef8d94(undefined8 param_1,undefined8 param_2,longlong param_3,undefined8 param_4,
             undefined1 param_5)

{
  char cVar1;
  undefined4 uVar2;
  undefined1 uVar3;
  undefined7 uVar4;
  undefined8 uVar5;
  
  uVar4 = (undefined7)((ulonglong)param_3 >> 8);
  uVar3 = (undefined1)param_3;
  uVar5 = param_4;
  FUN_140f03db8(param_3);
  FUN_14080bd28(CONCAT71(uVar4,uVar3),uVar5);
  cVar1 = FUN_1406cf008(param_4);
  *(char *)(param_3 + 2) = cVar1;
  if (cVar1 == '\0') {
    uVar2 = FUN_141102ed0(0x5d);
    FUN_142f268c4(param_3,param_4,param_5,uVar2);
  }
  return 1;
}

