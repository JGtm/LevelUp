
void FUN_1428e219c(longlong param_1,int *param_2)

{
  char cVar1;
  int iVar2;
  longlong lVar3;
  
  if (*param_2 == 0x29) {
    cVar1 = FUN_1414e77d0();
    if (cVar1 == '\0') {
      FUN_142ebf08c();
      cVar1 = FUN_1404f8df8();
      if ((cVar1 != '\0') && (cVar1 = FUN_1404f178c(), cVar1 == '\0')) {
        FUN_142e32ff4();
      }
      param_2[0x6e36b] = 2;
      param_2[0x339a5] = 2;
      lVar3 = *(longlong *)(param_1 + 0x120);
      iVar2 = 0;
      if ((lVar3 != 0) && (iVar2 = 0, *(int *)(lVar3 + 0x1e1d18) == 0)) {
        iVar2 = *(int *)(lVar3 + 0xcb798);
      }
      param_2[0x6e36d] = 0;
      param_2[0x6e36e] = iVar2;
      *(undefined1 *)(param_2 + 0x6e36c) = 0;
      *(undefined1 *)((longlong)param_2 + 0x1b9176) = 1;
      *(char *)(param_1 + 0x1ae) = (char)param_2[0x32d17];
      if (DAT_144db4330 != '\0') {
        do {
          cVar1 = FUN_1428e27c0(param_1);
        } while (cVar1 != '\0');
        return;
      }
      cVar1 = FUN_140ad4144(param_2 + 0x339a4);
      if (cVar1 != '\0') {
        if (DAT_145173888 == DAT_145173890) {
          FUN_140caf904(&DAT_145173888,(longlong)DAT_144dbfc90);
        }
        FUN_140b84d04(param_2 + 0x339a4);
        return;
      }
      FUN_1414e76f0(DAT_144875260,0);
    }
    lVar3 = *(longlong *)(param_1 + 0x130);
  }
  else {
    lVar3 = *(longlong *)(param_1 + 0x130);
  }
  if (lVar3 != 0) {
    FUN_142988e98();
  }
  return;
}

