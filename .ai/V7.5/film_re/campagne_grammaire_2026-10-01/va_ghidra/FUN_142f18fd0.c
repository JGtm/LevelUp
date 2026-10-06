void FUN_142f18fd0(undefined8 param_1,undefined8 param_2,undefined4 *param_3,longlong param_4)
{
  uint uVar1;
  ulonglong *puVar2;
  char cVar3;
  int iVar4;
  int iVar5;
  ulonglong uVar6;
  ulonglong uVar7;
  uint uVar8;
  FUN_142b549c0(param_4,*param_3,"victim-participant-handle");
  FUN_142b549c0(param_4,param_3[1],"killer-participant-handle");
  iVar5 = *(int *)(param_4 + 0x38);
  uVar1 = param_3[2];
  uVar6 = *(ulonglong *)(param_4 + 0x30);
  iVar4 = 0x40 - iVar5;
  *(int *)(param_4 + 0x2c) = *(int *)(param_4 + 0x2c) + 0x20;
  if (iVar4 < 0x20) {
    *(ulonglong *)(param_4 + 0x30) = (ulonglong)uVar1;
    uVar8 = 0x20 - iVar4;
    *(uint *)(param_4 + 0x38) = uVar8;
    if (uVar8 < 0x40) {
      uVar6 = uVar6 << ((byte)iVar4 & 0x3f) | (ulonglong)(uVar1 >> ((byte)uVar8 & 0x3f));
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
    uVar6 = uVar6 << 0x20 | (ulonglong)uVar1;
    *(int *)(param_4 + 0x38) = iVar5 + 0x20;
    *(ulonglong *)(param_4 + 0x30) = uVar6;
  }
  FUN_1406d49c4(param_4,uVar6,CONCAT31((int3)((uint)iVar5 >> 8),*(undefined1 *)(param_3 + 3)));
  FUN_142b549c0(param_4,param_3[4],"assistant-participant-handle");
  uVar1 = param_3[5];
  iVar5 = 0x40 - *(int *)(param_4 + 0x38);
  uVar6 = *(ulonglong *)(param_4 + 0x30);
  *(int *)(param_4 + 0x2c) = *(int *)(param_4 + 0x2c) + 0x20;
  if (iVar5 < 0x20) {
    uVar8 = 0x20 - iVar5;
    *(ulonglong *)(param_4 + 0x30) = (ulonglong)uVar1;
    *(uint *)(param_4 + 0x38) = uVar8;
    if (uVar8 < 0x40) {
      uVar6 = uVar6 << ((byte)iVar5 & 0x3f) | (ulonglong)(uVar1 >> ((byte)uVar8 & 0x3f));
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
  }
  else {
    *(int *)(param_4 + 0x38) = *(int *)(param_4 + 0x38) + 0x20;
    *(ulonglong *)(param_4 + 0x30) = uVar6 << 0x20 | (ulonglong)uVar1;
  }
  cVar3 = FUN_14076d018();
  if ((cVar3 == '\0') && (cVar3 = FUN_14076cffc(), cVar3 == '\0')) {
    return;
  }
  FUN_1431eb488(param_3 + 6,param_4);
  return;
}
