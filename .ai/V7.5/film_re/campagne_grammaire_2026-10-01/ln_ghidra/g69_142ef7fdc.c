
undefined8 FUN_142ef7fdc(undefined8 param_1,undefined8 param_2,uint *param_3,longlong param_4)

{
  int iVar1;
  char cVar2;
  ulonglong uVar3;
  ulonglong *puVar4;
  int iVar5;
  uint uVar6;
  uint uVar7;
  ulonglong uVar8;
  
  iVar1 = *(int *)(param_4 + 0x38);
  uVar7 = (uint)((ulonglong)*(longlong *)(param_4 + 0x30) >> 0x20);
  if (0x40 - iVar1 < 4) {
    puVar4 = *(ulonglong **)(param_4 + 0x40);
    uVar8 = 0;
    iVar5 = 0;
    if (*(ulonglong **)(param_4 + 0x10) < puVar4 + 1) {
      if (puVar4 < *(ulonglong **)(param_4 + 0x10)) {
        do {
          uVar3 = *puVar4;
          iVar5 = iVar5 + 8;
          puVar4 = (ulonglong *)((longlong)puVar4 + 1);
          uVar8 = uVar8 << 8 | (ulonglong)(byte)uVar3;
          *(ulonglong **)(param_4 + 0x40) = puVar4;
        } while (puVar4 < *(ulonglong **)(param_4 + 0x10));
        uVar8 = uVar8 << (0x40U - (char)iVar5 & 0x3f);
      }
    }
    else {
      uVar8 = *puVar4;
      iVar5 = 0x40;
      uVar8 = uVar8 >> 0x38 | (uVar8 & 0xff000000000000) >> 0x28 | (uVar8 & 0xff0000000000) >> 0x18
              | (uVar8 & 0xff00000000) >> 8 | (uVar8 & 0xff000000) << 8 | (uVar8 & 0xff0000) << 0x18
              | (uVar8 & 0xff00) << 0x28 | uVar8 << 0x38;
      *(ulonglong **)(param_4 + 0x40) = puVar4 + 1;
    }
    *(int *)(param_4 + 0x28) = *(int *)(param_4 + 0x28) + iVar5;
    uVar6 = iVar1 - 0x3c;
    *(int *)(param_4 + 0x2c) = *(int *)(param_4 + 0x2c) + 4;
    uVar3 = -(ulonglong)(uVar6 < 0x40) & uVar8 << ((byte)uVar6 & 0x3f);
    uVar7 = (uint)(uVar8 >> (0x40 - (byte)uVar6 & 0x3f)) | uVar7 >> 0x1c;
  }
  else {
    *(int *)(param_4 + 0x2c) = *(int *)(param_4 + 0x2c) + 4;
    uVar3 = *(longlong *)(param_4 + 0x30) << 4;
    uVar6 = iVar1 + 4;
    uVar7 = uVar7 >> 0x1c;
  }
  *(ulonglong *)(param_4 + 0x30) = uVar3;
  *(uint *)(param_4 + 0x38) = uVar6;
  *param_3 = uVar7;
  FUN_14080dec4(param_4,"campaign_map_event_id",param_3 + 1);
  param_3[2] = 0;
  param_3[3] = 0;
  param_3[4] = 0;
  cVar2 = FUN_1406cf008(param_4);
  if (cVar2 != '\0') {
    FUN_1424e0e38(param_4,param_3 + 2,0x10);
  }
  return 1;
}

