
void FUN_140be9658(int param_1)

{
  char cVar1;
  longlong lVar2;
  
  lVar2 = FUN_140be96d8();
  cVar1 = FUN_140be96b8();
  if ((cVar1 != '\0') && (*(int *)(lVar2 + 4) != param_1)) {
    *(int *)(lVar2 + 4) = param_1;
    FUN_140be9708();
  }
  DAT_144e4f098 = param_1 == 2;
  FUN_14051c1ec("game_simulation",&DAT_143686030,(&PTR_DAT_143cef4b0)[param_1]);
  return;
}

