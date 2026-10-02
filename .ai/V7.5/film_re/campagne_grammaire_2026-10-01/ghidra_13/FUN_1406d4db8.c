
undefined1 FUN_1406d4db8(byte *param_1,int param_2)

{
  byte bVar1;
  byte bVar2;
  char cVar3;
  byte bVar4;
  bool bVar5;
  undefined1 uVar6;
  byte bVar7;
  
  uVar6 = 1;
  bVar7 = 1;
  if ((param_1 != (byte *)0x0) && (*param_1 < 2)) {
    if (*param_1 == 1) {
      bVar4 = 4;
      if (DAT_145121140 == '\x01') {
        bVar4 = 0x10;
      }
      if (param_1[1] < bVar4) {
LAB_1406d4e31:
        if ((((((*(ulonglong *)(param_1 + 8) & 0xff80000000000000) == 0) &&
              ((*(uint *)(param_1 + 0x14) & 0x7f800000) != 0x7f800000)) &&
             ((*(uint *)(param_1 + 0x10) & 0x7f800000) != 0x7f800000)) &&
            ((((*(uint *)(param_1 + 0x28) & 0x7f800000) != 0x7f800000 &&
              ((*(uint *)(param_1 + 0x2c) & 0x7f800000) != 0x7f800000)) &&
             ((DAT_143cd8370 <= *(float *)(param_1 + 0x28) &&
              ((*(float *)(param_1 + 0x28) <= DAT_143cd8374 &&
               ((*(uint *)(param_1 + 0x1c) & 0x7f800000) != 0x7f800000)))))))) &&
           (((*(uint *)(param_1 + 0x20) & 0x7f800000) != 0x7f800000 &&
            ((((DAT_143cd84ec <= *(float *)(param_1 + 0x1c) &&
               (*(float *)(param_1 + 0x1c) <= DAT_143cd8374)) &&
              (DAT_143cd84ec <= *(float *)(param_1 + 0x20))) &&
             (*(float *)(param_1 + 0x20) <= DAT_143cd8374)))))) {
          bVar4 = 1;
          goto LAB_1406d4df6;
        }
      }
    }
    else if (param_1[1] == 0xff) goto LAB_1406d4e31;
  }
  bVar4 = 0;
LAB_1406d4df6:
  bVar1 = param_1[0x3d];
  if (((bVar1 != 0xff) && (3 < bVar1)) ||
     (((param_1[0x3e] != 0xff && (3 < param_1[0x3e])) ||
      ((bVar2 = bVar7, bVar1 != 0xff && (bVar1 == param_1[0x3e])))))) {
    bVar2 = 0;
  }
  if ((param_1[0x40] != 0) && (5 < (byte)(param_1[0x40] - 1))) {
    bVar7 = 0;
  }
  bVar5 = false;
  if ((byte)(param_1[0x42] + 1) < 0x41) {
    bVar5 = (bool)(bVar7 & bVar4 & bVar2);
  }
  if (((((!bVar5) || (*(short *)(param_1 + 0x48) != -1)) && (2 < *(ushort *)(param_1 + 0x48))) ||
      (((*(uint *)(param_1 + 0x38) & 0xfffc0000) != 0 ||
       ((param_2 == 1 &&
        (cVar3 = FUN_1406d5080((ulonglong)(param_1 + 0x77) & 0xfffffffffffffffc,
                               (ulonglong)(param_1 + 0x93) & 0xfffffffffffffffc), cVar3 == '\0')))))
      ) || ((cVar3 = FUN_1406d4f98(param_1 + 0x5c), cVar3 == '\0' ||
            ((*(ulonglong *)(param_1 + 0xa0) & 0xffffe00000000000) != 0)))) {
    uVar6 = 0;
  }
  return uVar6;
}

