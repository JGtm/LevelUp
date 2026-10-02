
bool FUN_1406d16b8(longlong param_1,undefined8 param_2,char param_3,undefined8 param_4)

{
  bool bVar1;
  longlong lVar2;
  
  lVar2 = FUN_140497308(*(undefined4 *)(param_1 + 4));
  FUN_1406d175c(param_2,*(undefined4 *)(lVar2 + 0x2e0),param_4);
  if (param_3 == '\0') {
    bVar1 = *(int *)(lVar2 + 0x2e0) != -1;
  }
  else {
    bVar1 = true;
  }
  return bVar1;
}

