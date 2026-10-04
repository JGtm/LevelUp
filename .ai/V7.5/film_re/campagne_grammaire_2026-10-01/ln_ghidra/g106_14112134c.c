
undefined8 FUN_14112134c(undefined8 param_1,undefined8 param_2,undefined8 *param_3,longlong param_4)

{
  int iVar1;
  ulonglong uVar2;
  char cVar3;
  ulonglong *puVar4;
  uint uVar5;
  ushort uVar6;
  ulonglong uVar7;
  ulonglong uVar8;
  undefined1 local_res18 [8];
  undefined8 local_18;
  undefined4 local_10;
  
  cVar3 = FUN_14076f91c();
  uVar8 = 0;
  uVar5 = 0;
  if (cVar3 == '\0') {
    FUN_14076e524(&local_18,param_4,local_res18,0xc);
  }
  else {
    FUN_1411b259c(&local_18,param_4);
  }
  *param_3 = local_18;
  *(undefined4 *)(param_3 + 1) = local_10;
  FUN_14076dc04(param_4);
  iVar1 = *(int *)(param_4 + 0x38);
  uVar6 = (ushort)((ulonglong)*(longlong *)(param_4 + 0x30) >> 0x30);
  if (0x40 - iVar1 < 9) {
    puVar4 = *(ulonglong **)(param_4 + 0x40);
    if (*(ulonglong **)(param_4 + 0x10) < puVar4 + 1) {
      uVar7 = uVar8;
      if (puVar4 < *(ulonglong **)(param_4 + 0x10)) {
        do {
          uVar2 = *puVar4;
          uVar5 = (int)uVar8 + 8;
          uVar8 = (ulonglong)uVar5;
          puVar4 = (ulonglong *)((longlong)puVar4 + 1);
          uVar7 = uVar7 << 8 | (ulonglong)(byte)uVar2;
          *(ulonglong **)(param_4 + 0x40) = puVar4;
        } while (puVar4 < *(ulonglong **)(param_4 + 0x10));
        uVar8 = uVar7 << (0x40U - (char)uVar5 & 0x3f);
      }
    }
    else {
      uVar8 = *puVar4;
      uVar5 = 0x40;
      uVar8 = uVar8 >> 0x38 | (uVar8 & 0xff000000000000) >> 0x28 | (uVar8 & 0xff0000000000) >> 0x18
              | (uVar8 & 0xff00000000) >> 8 | (uVar8 & 0xff000000) << 8 | (uVar8 & 0xff0000) << 0x18
              | (uVar8 & 0xff00) << 0x28 | uVar8 << 0x38;
      *(ulonglong **)(param_4 + 0x40) = puVar4 + 1;
    }
    *(int *)(param_4 + 0x28) = *(int *)(param_4 + 0x28) + uVar5;
    uVar5 = iVar1 - 0x37;
    *(int *)(param_4 + 0x2c) = *(int *)(param_4 + 0x2c) + 9;
    uVar7 = -(ulonglong)(uVar5 < 0x40) & uVar8 << ((byte)uVar5 & 0x3f);
    uVar6 = (ushort)(uVar8 >> (0x40 - (byte)uVar5 & 0x3f)) | uVar6 >> 7;
  }
  else {
    *(int *)(param_4 + 0x2c) = *(int *)(param_4 + 0x2c) + 9;
    uVar7 = *(longlong *)(param_4 + 0x30) << 9;
    uVar5 = iVar1 + 9;
    uVar6 = uVar6 >> 7;
  }
  *(ulonglong *)(param_4 + 0x30) = uVar7;
  *(uint *)(param_4 + 0x38) = uVar5;
  FUN_140809d94(param_3 + 3,DAT_144976b50,uVar6 - 1);
  return 1;
}

