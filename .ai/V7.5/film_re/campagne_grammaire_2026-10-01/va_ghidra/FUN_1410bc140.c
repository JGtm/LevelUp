void FUN_1410bc140(undefined8 param_1,char *param_2)
{
  char *pcVar1;
  uint uVar2;
  ulonglong uVar3;
  char cVar4;
  cVar4 = '\0';
  uVar3 = 0;
  pcVar1 = param_2;
  do {
    if (*pcVar1 != '\0') {
      cVar4 = '\x01';
      break;
    }
    uVar2 = (int)uVar3 + 1;
    uVar3 = (ulonglong)uVar2;
    pcVar1 = pcVar1 + 0x38;
  } while (uVar2 < 4);
  FUN_1406d49c4(param_1,param_2,CONCAT71((int7)(uVar3 >> 8),cVar4));
  if (cVar4 != '\0') {
    FUN_1406d60f4(param_1);
  }
  return;
}
