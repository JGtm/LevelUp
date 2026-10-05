
void FUN_142eec4d8(undefined8 param_1,undefined8 param_2,uint *param_3,longlong param_4)

{
  uint uVar1;
  ulonglong *puVar2;
  char cVar3;
  int iVar4;
  int iVar5;
  ulonglong uVar6;
  ulonglong uVar7;
  uint uVar8;
  
  cVar3 = FUN_1404f293c();
  if (cVar3 != '\0') {
    FUN_141fd07f8(param_3 + 0x1a,param_4);
  }
  uVar1 = *param_3;
  uVar6 = *(ulonglong *)(param_4 + 0x30);
  iVar4 = 0x40 - *(int *)(param_4 + 0x38);
  *(int *)(param_4 + 0x2c) = *(int *)(param_4 + 0x2c) + 0xd;
  if (iVar4 < 0xd) {
    uVar8 = 0xd - iVar4;
    *(ulonglong *)(param_4 + 0x30) = (ulonglong)uVar1;
    *(uint *)(param_4 + 0x38) = uVar8;
    if (uVar8 < 0x40) {
      uVar6 = (ulonglong)(uVar1 >> ((byte)uVar8 & 0x3f)) | uVar6 << ((byte)iVar4 & 0x3f);
    }
    puVar2 = *(ulonglong **)(param_4 + 0x40);
    if (*(ulonglong **)(param_4 + 0x10) < puVar2 + 1) {
      if (puVar2 < *(ulonglong **)(param_4 + 0x10)) {
        do {
          **(undefined1 **)(param_4 + 0x40) = (char)(uVar6 >> 0x38);
          *(longlong *)(param_4 + 0x40) = *(longlong *)(param_4 + 0x40) + 1;
          uVar6 = uVar6 << 8;
        } while (*(ulonglong *)(param_4 + 0x40) < *(ulonglong *)(param_4 + 0x10));
      }
    }
    else {
      *puVar2 = uVar6 >> 0x38 | (uVar6 & 0xff000000000000) >> 0x28 |
                (uVar6 & 0xff0000000000) >> 0x18 | (uVar6 & 0xff00000000) >> 8 |
                (uVar6 & 0xff000000) << 8 | (uVar6 & 0xff0000) << 0x18 | (uVar6 & 0xff00) << 0x28 |
                uVar6 << 0x38;
      *(longlong *)(param_4 + 0x40) = *(longlong *)(param_4 + 0x40) + 8;
    }
    *(int *)(param_4 + 0x28) = *(int *)(param_4 + 0x28) + 0x40;
    iVar4 = *(int *)(param_4 + 0x38);
  }
  else {
    iVar4 = *(int *)(param_4 + 0x38) + 0xd;
    *(int *)(param_4 + 0x38) = iVar4;
    *(ulonglong *)(param_4 + 0x30) = uVar6 << 0xd | (ulonglong)uVar1;
  }
  uVar1 = param_3[1];
  uVar6 = *(ulonglong *)(param_4 + 0x30);
  iVar5 = 0x40 - iVar4;
  if (iVar5 < 10) {
    uVar8 = 10 - iVar5;
    *(ulonglong *)(param_4 + 0x30) = (ulonglong)uVar1;
    *(uint *)(param_4 + 0x38) = uVar8;
    *(int *)(param_4 + 0x2c) = *(int *)(param_4 + 0x2c) + 10;
    if (uVar8 < 0x40) {
      uVar6 = uVar6 << ((byte)iVar5 & 0x3f) | (ulonglong)(uVar1 >> ((byte)uVar8 & 0x3f));
    }
    puVar2 = *(ulonglong **)(param_4 + 0x40);
    if (*(ulonglong **)(param_4 + 0x10) < puVar2 + 1) {
      uVar7 = uVar6;
      if (puVar2 < *(ulonglong **)(param_4 + 0x10)) {
        do {
          uVar6 = uVar7 << 8;
          **(undefined1 **)(param_4 + 0x40) = (char)(uVar7 >> 0x38);
          *(longlong *)(param_4 + 0x40) = *(longlong *)(param_4 + 0x40) + 1;
          uVar7 = uVar6;
        } while (*(ulonglong *)(param_4 + 0x40) < *(ulonglong *)(param_4 + 0x10));
      }
    }
    else {
      uVar6 = uVar6 >> 0x38 | (uVar6 & 0xff000000000000) >> 0x28 | (uVar6 & 0xff0000000000) >> 0x18
              | (uVar6 & 0xff00000000) >> 8 | (uVar6 & 0xff000000) << 8 | (uVar6 & 0xff0000) << 0x18
              | (uVar6 & 0xff00) << 0x28 | uVar6 << 0x38;
      *puVar2 = uVar6;
      *(longlong *)(param_4 + 0x40) = *(longlong *)(param_4 + 0x40) + 8;
    }
    *(int *)(param_4 + 0x28) = *(int *)(param_4 + 0x28) + 0x40;
  }
  else {
    *(int *)(param_4 + 0x38) = iVar4 + 10;
    uVar6 = uVar6 << 10 | (ulonglong)uVar1;
    *(ulonglong *)(param_4 + 0x30) = uVar6;
    *(int *)(param_4 + 0x2c) = *(int *)(param_4 + 0x2c) + 10;
  }
  FUN_1406d60f4(param_4,uVar6,param_3 + 2,param_3[1]);
  return;
}

