/* WARNING: Function: _alloca_probe replaced with injection: alloca_probe */
undefined1 FUN_140ee5f40(undefined8 param_1,undefined8 param_2)
{
  char cVar1;
  ulonglong uVar2;
  undefined1 uVar4;
  undefined ***local_res8;
  undefined **local_11a8;
  undefined8 local_11a0;
  undefined1 local_1198 [4168];
  char local_150 [304];
  undefined1 local_20 [16];
  undefined8 uStack_10;
  ulonglong uVar3;
  uStack_10 = 0x140ee5f50;
  FUN_14064c240(local_1198);
  local_11a8 = &PTR_FUN_143686718;
  local_res8 = &local_11a8;
  local_11a0 = param_1;
  cVar1 = FUN_140ee6004(&local_res8);
  uVar4 = 0;
  if (cVar1 != '\0') {
    uVar2 = 0xffffffffffffffff;
    do {
      uVar3 = uVar2;
      uVar2 = uVar3 + 1;
    } while (local_150[uVar3 + 1] != '\0');
    if (uVar2 < 0x104) {
      memset(local_150 + uVar3 + 1,0,0x104 - uVar2);
      cVar1 = FUN_1409a6bec(param_2,local_1198);
      if (cVar1 != '\0') {
        uVar4 = 1;
      }
    }
  }
  FUN_14091a5dc(local_20);
  return uVar4;
}
