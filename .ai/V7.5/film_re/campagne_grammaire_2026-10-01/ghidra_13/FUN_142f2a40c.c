
void FUN_142f2a40c(longlong param_1,uint param_2)

{
  longlong lVar1;
  longlong lVar2;
  undefined1 local_18 [16];
  
  lVar2 = (longlong)(int)param_2;
  *(undefined1 *)(lVar2 + 0x520 + param_1) = 0;
  FUN_140964180();
  *(uint *)(param_1 + 8000) = *(uint *)(param_1 + 8000) & ~(1 << (param_2 & 0x1f));
  *(uint *)(param_1 + 0x1f44) = *(uint *)(param_1 + 0x1f44) & ~(1 << (param_2 & 0x1f));
  *(uint *)(param_1 + 0x2550) = *(uint *)(param_1 + 0x2550) & ~(1 << (param_2 & 0x1f));
  *(uint *)(param_1 + 0x2554) = *(uint *)(param_1 + 0x2554) & ~(1 << (param_2 & 0x1f));
  *(uint *)(param_1 + 0x2558) = *(uint *)(param_1 + 0x2558) & ~(1 << (param_2 & 0x1f));
  *(uint *)(param_1 + 0x255c) = *(uint *)(param_1 + 0x255c) & ~(1 << (param_2 & 0x1f));
  *(uint *)(param_1 + 0x2560) = *(uint *)(param_1 + 0x2560) & ~(1 << (param_2 & 0x1f));
  *(undefined4 *)(param_1 + 0x1f48 + lVar2 * 4) = 0xffffffff;
  *(undefined4 *)(param_1 + 0x1fc8 + lVar2 * 4) = 0xffffffff;
  *(undefined4 *)(param_1 + 0x1040 + lVar2 * 4) = 0;
  *(undefined4 *)(lVar2 * 0x58 + 0x540 + param_1) = 0xffffffff;
  *(undefined4 *)(param_1 + 0x1ec0 + lVar2 * 4) = 0xffffffff;
  *(undefined4 *)(param_1 + 0x2570 + lVar2 * 4) = 0;
  lVar1 = DAT_1451f98c8;
  if (DAT_1451f98c8 != 0) {
    *(undefined2 *)(DAT_1451f98c8 + lVar2 * 2) = 0;
    *(undefined1 *)(lVar1 + 0x40 + lVar2) = 0;
  }
  if (DAT_1451f47b0 != 0) {
    FUN_1404fba0c(&DAT_145123240,local_18);
    if ((*(uint *)(DAT_1451f47b0 + 0xf80) >> (param_2 & 0x1f) & 1) != 0) {
      FUN_142bcf3cc(param_2);
      *(uint *)(DAT_1451f47b0 + 0xf80) = *(uint *)(DAT_1451f47b0 + 0xf80) & ~(1 << (param_2 & 0x1f))
      ;
    }
    FUN_1404fb988(local_18);
  }
  return;
}

