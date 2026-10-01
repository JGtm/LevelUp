
void FUN_142f2abb8(longlong param_1)

{
  char cVar1;
  char cVar2;
  uint uVar3;
  undefined4 uVar4;
  uint uVar5;
  longlong *plVar6;
  uint uVar7;
  byte *pbVar8;
  undefined4 *puVar9;
  longlong lVar10;
  byte *pbVar11;
  
  uVar7 = 0;
  if (*(char *)(param_1 + 0x19) != '\0') {
    cVar1 = FUN_14048ee34();
    if (cVar1 == '\0') {
      pbVar8 = (byte *)(param_1 + 0x32f0);
      lVar10 = param_1 + 0x25f0;
      pbVar11 = pbVar8;
      uVar3 = uVar7;
      do {
        cVar1 = FUN_1407699d0(pbVar11);
        cVar2 = FUN_1406d33cc(lVar10);
        if (cVar2 != '\0') {
          *(uint *)(param_1 + 0x2550) = *(uint *)(param_1 + 0x2550) | 1 << (uVar3 & 0x1f);
        }
        if (cVar1 != '\0') {
          *pbVar8 = *pbVar8 | 1;
          *(uint *)(param_1 + 0x2558) = *(uint *)(param_1 + 0x2558) | 1 << (uVar3 & 0x1f);
        }
        uVar3 = uVar3 + 1;
        pbVar11 = pbVar11 + 0xbc;
        lVar10 = lVar10 + 0x68;
        pbVar8 = pbVar8 + 0xbc;
      } while ((int)uVar3 < 0x20);
    }
    else {
      uVar3 = *(uint *)(param_1 + 0x1f44);
      plVar6 = (longlong *)(param_1 + 0x4a90);
      uVar5 = uVar7;
      do {
        if (*plVar6 == 0) {
          uVar3 = uVar3 & ~(1 << (uVar5 & 0x1f));
        }
        else {
          uVar3 = uVar3 | 1 << (uVar5 & 0x1f);
        }
        uVar5 = uVar5 + 1;
        *(uint *)(param_1 + 0x1f44) = uVar3;
        plVar6 = plVar6 + 5;
      } while ((int)uVar5 < 0x20);
    }
    puVar9 = (undefined4 *)(param_1 + 0x2570);
    do {
      uVar4 = FUN_1405f50b8();
      uVar7 = uVar7 + 1;
      *puVar9 = uVar4;
      puVar9 = puVar9 + 1;
    } while (uVar7 < 0x20);
  }
  return;
}

