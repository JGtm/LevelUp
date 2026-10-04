
undefined8 FUN_142f17b94(undefined8 param_1,undefined8 param_2,uint *param_3,longlong param_4)

{
  int iVar1;
  ulonglong uVar2;
  undefined1 uVar3;
  ulonglong uVar4;
  ulonglong *puVar5;
  uint uVar6;
  uint uVar7;
  ulonglong uVar8;
  
  iVar1 = *(int *)(param_4 + 0x38);
  uVar7 = (uint)((ulonglong)*(longlong *)(param_4 + 0x30) >> 0x20);
  if (0x40 - iVar1 < 6) {
    puVar5 = *(ulonglong **)(param_4 + 0x40);
    uVar8 = 0;
    uVar6 = 0;
    if (*(ulonglong **)(param_4 + 0x10) < puVar5 + 1) {
      uVar4 = uVar8;
      if (puVar5 < *(ulonglong **)(param_4 + 0x10)) {
        do {
          uVar2 = *puVar5;
          uVar6 = (int)uVar8 + 8;
          uVar8 = (ulonglong)uVar6;
          puVar5 = (ulonglong *)((longlong)puVar5 + 1);
          uVar4 = uVar4 << 8 | (ulonglong)(byte)uVar2;
          *(ulonglong **)(param_4 + 0x40) = puVar5;
        } while (puVar5 < *(ulonglong **)(param_4 + 0x10));
        uVar8 = uVar4 << (0x40U - (char)uVar6 & 0x3f);
      }
    }
    else {
      uVar8 = *puVar5;
      uVar6 = 0x40;
      uVar8 = uVar8 >> 0x38 | (uVar8 & 0xff000000000000) >> 0x28 | (uVar8 & 0xff0000000000) >> 0x18
              | (uVar8 & 0xff00000000) >> 8 | (uVar8 & 0xff000000) << 8 | (uVar8 & 0xff0000) << 0x18
              | (uVar8 & 0xff00) << 0x28 | uVar8 << 0x38;
      *(ulonglong **)(param_4 + 0x40) = puVar5 + 1;
    }
    *(int *)(param_4 + 0x28) = *(int *)(param_4 + 0x28) + uVar6;
    uVar6 = iVar1 - 0x3a;
    *(int *)(param_4 + 0x2c) = *(int *)(param_4 + 0x2c) + 6;
    uVar4 = -(ulonglong)(uVar6 < 0x40) & uVar8 << ((byte)uVar6 & 0x3f);
    uVar7 = (uint)(uVar8 >> (0x40 - (byte)uVar6 & 0x3f)) | uVar7 >> 0x1a;
  }
  else {
    *(int *)(param_4 + 0x2c) = *(int *)(param_4 + 0x2c) + 6;
    uVar4 = *(longlong *)(param_4 + 0x30) << 6;
    uVar6 = iVar1 + 6;
    uVar7 = uVar7 >> 0x1a;
  }
  *(ulonglong *)(param_4 + 0x30) = uVar4;
  *(uint *)(param_4 + 0x38) = uVar6;
  *param_3 = uVar7;
  uVar3 = FUN_1406cf008(param_4);
  *(undefined1 *)(param_3 + 1) = uVar3;
  uVar3 = FUN_140c1e31c(param_4);
  *(undefined1 *)((longlong)param_3 + 5) = uVar3;
  return 1;
}

