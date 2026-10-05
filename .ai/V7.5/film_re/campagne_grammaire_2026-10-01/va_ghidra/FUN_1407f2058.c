
ulonglong FUN_1407f2058(longlong param_1)

{
  int iVar1;
  ulonglong uVar2;
  char cVar3;
  ulonglong uVar4;
  ulonglong *puVar5;
  ulonglong uVar6;
  ulonglong uVar7;
  uint uVar8;
  
  cVar3 = FUN_1406cf008();
  uVar7 = 0;
  uVar8 = 0;
  if (cVar3 == '\0') {
    iVar1 = *(int *)(param_1 + 0x38);
    uVar6 = *(ulonglong *)(param_1 + 0x30);
    if (0x40 - iVar1 < 5) {
      puVar5 = *(ulonglong **)(param_1 + 0x40);
      if (*(ulonglong **)(param_1 + 0x10) < puVar5 + 1) {
        uVar4 = uVar7;
        if (puVar5 < *(ulonglong **)(param_1 + 0x10)) {
          do {
            uVar2 = *puVar5;
            uVar8 = (int)uVar7 + 8;
            uVar7 = (ulonglong)uVar8;
            puVar5 = (ulonglong *)((longlong)puVar5 + 1);
            uVar4 = uVar4 << 8 | (ulonglong)(byte)uVar2;
            *(ulonglong **)(param_1 + 0x40) = puVar5;
          } while (puVar5 < *(ulonglong **)(param_1 + 0x10));
          uVar7 = uVar4 << (0x40U - (char)uVar8 & 0x3f);
        }
      }
      else {
        uVar7 = *puVar5;
        uVar8 = 0x40;
        uVar7 = uVar7 >> 0x38 | (uVar7 & 0xff000000000000) >> 0x28 |
                (uVar7 & 0xff0000000000) >> 0x18 | (uVar7 & 0xff00000000) >> 8 |
                (uVar7 & 0xff000000) << 8 | (uVar7 & 0xff0000) << 0x18 | (uVar7 & 0xff00) << 0x28 |
                uVar7 << 0x38;
        *(ulonglong **)(param_1 + 0x40) = puVar5 + 1;
      }
      *(int *)(param_1 + 0x28) = *(int *)(param_1 + 0x28) + uVar8;
      uVar8 = iVar1 - 0x3b;
      *(int *)(param_1 + 0x2c) = *(int *)(param_1 + 0x2c) + 5;
      uVar4 = -(ulonglong)(uVar8 < 0x40) & uVar7 << ((byte)uVar8 & 0x3f);
      uVar6 = uVar7 >> (0x40 - (byte)uVar8 & 0x3f) | uVar6 >> 0x3b;
    }
    else {
      *(int *)(param_1 + 0x2c) = *(int *)(param_1 + 0x2c) + 5;
      uVar4 = uVar6 << 5;
      uVar8 = iVar1 + 5;
      uVar6 = uVar6 >> 0x3b;
    }
    *(ulonglong *)(param_1 + 0x30) = uVar4;
    *(uint *)(param_1 + 0x38) = uVar8;
  }
  else {
    uVar6 = 0xffffffff;
  }
  return uVar6 & 0xffffffff;
}

