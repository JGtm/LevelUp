
undefined8 FUN_141037828(ulonglong param_1,undefined8 param_2,ushort *param_3,longlong param_4)

{
  ushort *puVar1;
  int iVar2;
  char cVar3;
  undefined4 *puVar4;
  ulonglong uVar5;
  byte bVar6;
  ulonglong *puVar7;
  int iVar8;
  undefined4 uVar9;
  uint uVar10;
  ushort uVar11;
  ulonglong uVar12;
  undefined1 local_res18 [16];
  
  iVar2 = *(int *)(param_4 + 0x38);
  uVar11 = (ushort)((ulonglong)*(longlong *)(param_4 + 0x30) >> 0x30);
  if (0x40 - iVar2 < 3) {
    puVar7 = *(ulonglong **)(param_4 + 0x40);
    uVar12 = 0;
    iVar8 = 0;
    if (*(ulonglong **)(param_4 + 0x10) < puVar7 + 1) {
      if (puVar7 < *(ulonglong **)(param_4 + 0x10)) {
        do {
          uVar5 = *puVar7;
          iVar8 = iVar8 + 8;
          puVar7 = (ulonglong *)((longlong)puVar7 + 1);
          uVar12 = uVar12 << 8 | (ulonglong)(byte)uVar5;
          *(ulonglong **)(param_4 + 0x40) = puVar7;
        } while (puVar7 < *(ulonglong **)(param_4 + 0x10));
        uVar12 = uVar12 << (-(char)iVar8 & 0x3fU);
      }
    }
    else {
      uVar12 = *puVar7;
      iVar8 = 0x40;
      uVar12 = uVar12 >> 0x38 | (uVar12 & 0xff000000000000) >> 0x28 |
               (uVar12 & 0xff0000000000) >> 0x18 | (uVar12 & 0xff00000000) >> 8 |
               (uVar12 & 0xff000000) << 8 | (uVar12 & 0xff0000) << 0x18 | (uVar12 & 0xff00) << 0x28
               | uVar12 << 0x38;
      *(ulonglong **)(param_4 + 0x40) = puVar7 + 1;
    }
    *(int *)(param_4 + 0x28) = *(int *)(param_4 + 0x28) + iVar8;
    uVar10 = iVar2 - 0x3d;
    *(int *)(param_4 + 0x2c) = *(int *)(param_4 + 0x2c) + 3;
    bVar6 = 0x40 - (byte)uVar10;
    param_1 = (ulonglong)bVar6;
    uVar5 = -(ulonglong)(uVar10 < 0x40) & uVar12 << ((byte)uVar10 & 0x3f);
    uVar11 = (ushort)(uVar12 >> (bVar6 & 0x3f)) | uVar11 >> 0xd;
  }
  else {
    *(int *)(param_4 + 0x2c) = *(int *)(param_4 + 0x2c) + 3;
    uVar5 = *(longlong *)(param_4 + 0x30) * 8;
    uVar10 = iVar2 + 3;
    uVar11 = uVar11 >> 0xd;
  }
  *(ulonglong *)(param_4 + 0x30) = uVar5;
  puVar1 = param_3 + 2;
  *(uint *)(param_4 + 0x38) = uVar10;
  uVar9 = 0xffffffff;
  *param_3 = uVar11;
  FUN_14080d69c(param_1,param_4,puVar1,0xffffffff);
  cVar3 = FUN_1405838f0(puVar1);
  if (cVar3 != '\0') {
    puVar4 = (undefined4 *)FUN_1407f21b4(local_res18,puVar1);
    uVar9 = *puVar4;
  }
  *(undefined4 *)(param_3 + 4) = uVar9;
  return 1;
}

