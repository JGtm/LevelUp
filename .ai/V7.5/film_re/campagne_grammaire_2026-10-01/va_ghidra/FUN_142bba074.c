
void FUN_142bba074(int param_1)

{
  undefined8 uVar1;
  longlong lVar2;
  char cVar3;
  
  uVar1 = FUN_140be96d8();
  FUN_142bad944(uVar1,param_1);
  if (param_1 != 0) {
    lVar2 = FUN_140be96d8();
    cVar3 = FUN_140be96b8();
    if ((cVar3 != '\0') && (*(char *)(lVar2 + 0xea720) != '\0')) {
      *(undefined1 *)(lVar2 + 0xea720) = 0;
      FUN_140be9708();
    }
  }
  DAT_1452f2e41 = param_1 == 2;
  return;
}

