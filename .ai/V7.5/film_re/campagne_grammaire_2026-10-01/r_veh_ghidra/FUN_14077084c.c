
ulonglong FUN_14077084c(float *param_1)

{
  char cVar1;
  uint uVar2;
  longlong lVar3;
  ulonglong uVar4;
  ulonglong uVar5;
  uint uVar6;
  uint local_88 [32];
  
  memset(local_88,0,0x80);
  for (uVar2 = FUN_1404f8928(); uVar4 = 0xffffffff, uVar2 != 0xffffffff;
      uVar2 = FUN_1404f7434((ulonglong)uVar2)) {
    uVar6 = 1 << ((byte)uVar2 & 0x1f);
    if (((&DAT_1445ccb60)[uVar2 >> 5] & uVar6) != 0) {
      lVar3 = (longlong)(int)uVar2;
      if (((((*(float *)(&DAT_14462cbe0 + lVar3 * 3) <= *param_1) &&
            (*param_1 <= *(float *)((longlong)&DAT_14462cbe0 + lVar3 * 0x18 + 4))) &&
           (*(float *)(&DAT_14462cbe8 + lVar3 * 3) <= param_1[1])) &&
          ((param_1[1] <= *(float *)((longlong)&DAT_14462cbe8 + lVar3 * 0x18 + 4) &&
           (*(float *)(&DAT_14462cbf0 + lVar3 * 3) <= param_1[2])))) &&
         (uVar4 = (ulonglong)uVar2,
         param_1[2] <= *(float *)((longlong)&DAT_14462cbf0 + lVar3 * 0x18 + 4))) break;
    }
    local_88[uVar2 >> 5] = local_88[uVar2 >> 5] | uVar6;
  }
  uVar5 = 0;
  lVar3 = (longlong)(int)uVar4;
  while( true ) {
    if (lVar3 != -1) {
      return uVar4;
    }
    if (0x3ff < (int)uVar5) break;
    uVar2 = 1 << ((byte)uVar5 & 0x1f);
    if ((((local_88[uVar5 >> 5] & uVar2) == 0) && (((&DAT_1445ccb60)[uVar5 >> 5] & uVar2) != 0)) &&
       (cVar1 = FUN_140770948(param_1,&DAT_14462cbe0 + (longlong)(int)uVar5 * 3), cVar1 != '\0')) {
      return uVar5 & 0xffffffff;
    }
    uVar5 = (ulonglong)((int)uVar5 + 1);
  }
  return uVar4;
}

