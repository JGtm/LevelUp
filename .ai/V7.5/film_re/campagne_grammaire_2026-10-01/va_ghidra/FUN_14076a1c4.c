
int FUN_14076a1c4(longlong param_1,undefined8 param_2,longlong param_3,int param_4,
                 undefined8 param_5,undefined4 *param_6)

{
  undefined4 uVar1;
  int iVar2;
  longlong lVar3;
  bool bVar4;
  
  iVar2 = 0;
  if (*(char *)(param_1 + 0x11) == '\0') {
    do {
      if (*(uint *)(param_3 + 0x38) < 0x40) {
        lVar3 = *(longlong *)(param_3 + 0x30);
        *(int *)(param_3 + 0x2c) = *(int *)(param_3 + 0x2c) + 1;
        *(longlong *)(param_3 + 0x30) = lVar3 * 2;
        bVar4 = -1 < lVar3;
        *(uint *)(param_3 + 0x38) = *(uint *)(param_3 + 0x38) + 1;
      }
      else {
        lVar3 = FUN_1406d6c7c(param_3,1);
        bVar4 = lVar3 == 0;
      }
      if (bVar4) break;
      if (param_4 < 1) {
        iVar2 = 3;
        break;
      }
      uVar1 = FUN_140514010(*(undefined4 *)(param_1 + 0x14));
      iVar2 = FUN_14080a9d4(*(undefined8 *)(*(longlong *)(param_1 + 0x28) + 8),uVar1,param_3);
    } while (iVar2 == 0);
  }
  else {
    iVar2 = 2;
  }
  *param_6 = 0;
  return iVar2;
}

