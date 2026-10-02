
void FUN_14076ba54(longlong param_1,undefined8 param_2,uint param_3)

{
  undefined4 uVar1;
  int *piVar2;
  int iVar3;
  char cVar4;
  
  if (param_3 < 0x21) {
    cVar4 = (&DAT_144de4348)[(int)param_3];
  }
  else {
    cVar4 = '\0';
  }
  iVar3 = 0;
  if (0 < *(int *)(param_1 + 0x3c)) {
    piVar2 = (int *)(param_1 + 0x40);
    do {
      if (cVar4 == '\0') {
        uVar1 = (&DAT_14521d920)[(longlong)*piVar2 * 4];
      }
      else {
        uVar1 = *(undefined4 *)(&DAT_145225910 + (longlong)*piVar2 * 0x10);
      }
      FUN_14080ada0(uVar1);
      FUN_1406d60f4(param_2);
      iVar3 = iVar3 + 1;
      piVar2 = piVar2 + 1;
    } while (iVar3 < *(int *)(param_1 + 0x3c));
  }
  return;
}

