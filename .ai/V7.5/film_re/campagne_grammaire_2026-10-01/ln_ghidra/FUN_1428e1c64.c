
undefined4 FUN_1428e1c64(longlong param_1,int param_2)

{
  undefined4 uVar1;
  char cVar2;
  longlong lVar3;
  undefined1 local_18 [16];
  
  FUN_1404f973c(param_1 + 0x100,local_18);
  cVar2 = FUN_1428e1e94(&DAT_144c23178);
  if (cVar2 == '\0') {
    lVar3 = *(longlong *)(param_1 + 0x108);
  }
  else {
    lVar3 = *(longlong *)(param_1 + 0x120) + 0x130;
  }
  uVar1 = *(undefined4 *)(lVar3 + 0xcb208 + (longlong)param_2 * 4);
  FUN_1404f940c(local_18);
  return uVar1;
}

