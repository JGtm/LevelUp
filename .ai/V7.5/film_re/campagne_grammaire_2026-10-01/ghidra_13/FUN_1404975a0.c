
void FUN_1404975a0(longlong param_1,longlong param_2)

{
  uint *puVar1;
  undefined4 uVar2;
  undefined4 uVar3;
  undefined4 uVar4;
  undefined8 uVar5;
  char cVar6;
  undefined1 uVar7;
  char cVar8;
  ushort uVar9;
  uint uVar10;
  uint uVar11;
  longlong lVar12;
  byte *pbVar13;
  undefined8 *puVar14;
  uint uVar15;
  ulonglong uVar16;
  byte *pbVar17;
  int *piVar18;
  uint uVar19;
  byte *pbVar20;
  char *pcVar21;
  int *piVar22;
  uint *puVar23;
  longlong lVar24;
  ulonglong uVar25;
  undefined4 uStackX_10;
  undefined4 uStackX_14;
  char cStackX_18;
  undefined7 uStackX_19;
  longlong lStackX_20;
  undefined4 *puStack_48;
  
  *(undefined4 *)(param_2 + 8) = 0;
  piVar18 = (int *)(param_1 + 0x1d8);
  *(undefined4 *)(param_2 + 0x18e8) = 0;
  lVar24 = param_2 + 0x10;
  uVar15 = 0;
  piVar22 = piVar18;
  do {
    if ((*piVar18 != -1) && (cVar6 = FUN_1406b09c0(piVar22,lVar24), cVar6 != '\0')) {
      *(uint *)(param_2 + 8) = *(uint *)(param_2 + 8) | 1 << (uVar15 & 0x1f);
      lVar12 = FUN_140b763e4(uVar15);
      if (lVar12 != 0) {
        uVar9 = FUN_140496c18(lVar12);
        if ((uVar9 < 0x21) && ((&DAT_144de4348)[(short)uVar9] != '\0')) {
          FUN_1422b6b7c();
          return;
        }
      }
    }
    uVar15 = uVar15 + 1;
    lVar24 = lVar24 + 0xc0;
    piVar22 = piVar22 + 0x3e;
    piVar18 = piVar18 + 0x3e;
  } while ((int)uVar15 < 0x20);
  cVar6 = FUN_14048ee34();
  if (cVar6 == '\0') {
    cVar6 = FUN_1405f1e0c();
    if ((cVar6 != '\0') || (DAT_1451d2628 != -1)) {
      FUN_1422b6be8();
      return;
    }
  }
  else {
    cVar6 = FUN_1404f1ca4();
    if (cVar6 != '\0') {
      uStackX_14 = 0;
      uStackX_10 = 4;
      while (cVar6 = FUN_1405d3b78(param_1,&uStackX_10,&cStackX_18), cVar6 != '\0') {
        lVar24 = CONCAT71(uStackX_19,cStackX_18);
        cVar6 = FUN_1409cbf3c();
        if (cVar6 != '\0') {
          FUN_142f2bb90(*(longlong *)(lVar24 + 0x10) + 0x1b908);
        }
      }
      FUN_140497671();
      return;
    }
  }
  uVar7 = FUN_14048ee34();
  uStackX_10 = CONCAT31(uStackX_10._1_3_,uVar7);
  cStackX_18 = FUN_1404f1ca4();
  FUN_140c73d0c();
  puStack_48 = (undefined4 *)(param_2 + 0x30c8);
  uVar15 = 0;
  lStackX_20 = param_2 + 0x3070;
  pbVar17 = (byte *)(param_2 + 0x18ec);
  puVar23 = (uint *)(param_1 + 0x1e0);
  uVar25 = param_2 + 0x87;
  pcVar21 = &DAT_144de4348;
  pbVar20 = pbVar17;
  do {
    puVar1 = puVar23 + -2;
    lVar24 = lStackX_20;
    if ((*puVar1 != 0xffffffff) && ((0x20 < uVar15 || (*pcVar21 == '\0')))) {
      cVar6 = (char)uStackX_10;
      if (((char)uStackX_10 == '\0') && ((*puVar23 - 4 < 2 || (*puVar23 == 1)))) {
        if (*(int *)(*(longlong *)(puVar23 + 6) + 0x10) == 4) {
          lVar24 = FUN_1405d3b40();
        }
        else {
          lVar24 = FUN_141f864ac();
        }
        if ((lVar24 != 0) && (lVar24 = *(longlong *)(lVar24 + 0x10), lVar24 != -0x1b908)) {
          uVar10 = *puVar1;
          uVar19 = 1 << ((byte)uVar10 & 0x1f);
          if ((*(uint *)(lVar24 + 0x1de64) & uVar19) != 0) {
            puVar14 = (undefined8 *)(lVar24 + 0x1ebf8 + (longlong)(int)uVar10 * 0xbc);
            uVar5 = puVar14[1];
            *(undefined8 *)pbVar17 = *puVar14;
            *(undefined8 *)(pbVar17 + 8) = uVar5;
            uVar5 = puVar14[3];
            *(undefined8 *)(pbVar17 + 0x10) = puVar14[2];
            *(undefined8 *)(pbVar17 + 0x18) = uVar5;
            uVar5 = puVar14[5];
            *(undefined8 *)(pbVar17 + 0x20) = puVar14[4];
            *(undefined8 *)(pbVar17 + 0x28) = uVar5;
            uVar5 = puVar14[7];
            *(undefined8 *)(pbVar17 + 0x30) = puVar14[6];
            *(undefined8 *)(pbVar17 + 0x38) = uVar5;
            uVar5 = puVar14[9];
            *(undefined8 *)(pbVar17 + 0x40) = puVar14[8];
            *(undefined8 *)(pbVar17 + 0x48) = uVar5;
            uVar5 = puVar14[0xb];
            *(undefined8 *)(pbVar17 + 0x50) = puVar14[10];
            *(undefined8 *)(pbVar17 + 0x58) = uVar5;
            uVar5 = puVar14[0xd];
            *(undefined8 *)(pbVar17 + 0x60) = puVar14[0xc];
            *(undefined8 *)(pbVar17 + 0x68) = uVar5;
            uVar5 = puVar14[0xf];
            *(undefined8 *)(pbVar17 + 0x70) = puVar14[0xe];
            *(undefined8 *)(pbVar17 + 0x78) = uVar5;
            uVar5 = puVar14[0x11];
            *(undefined8 *)(pbVar17 + 0x80) = puVar14[0x10];
            *(undefined8 *)(pbVar17 + 0x88) = uVar5;
            uVar2 = *(undefined4 *)((longlong)puVar14 + 0x94);
            uVar3 = *(undefined4 *)(puVar14 + 0x13);
            uVar4 = *(undefined4 *)((longlong)puVar14 + 0x9c);
            *(undefined4 *)(pbVar17 + 0x90) = *(undefined4 *)(puVar14 + 0x12);
            *(undefined4 *)(pbVar17 + 0x94) = uVar2;
            *(undefined4 *)(pbVar17 + 0x98) = uVar3;
            *(undefined4 *)(pbVar17 + 0x9c) = uVar4;
            uVar2 = *(undefined4 *)((longlong)puVar14 + 0xa4);
            uVar3 = *(undefined4 *)(puVar14 + 0x15);
            uVar4 = *(undefined4 *)((longlong)puVar14 + 0xac);
            *(undefined4 *)(pbVar17 + 0xa0) = *(undefined4 *)(puVar14 + 0x14);
            *(undefined4 *)(pbVar17 + 0xa4) = uVar2;
            *(undefined4 *)(pbVar17 + 0xa8) = uVar3;
            *(undefined4 *)(pbVar17 + 0xac) = uVar4;
            *(undefined8 *)(pbVar17 + 0xb0) = puVar14[0x16];
            *(undefined4 *)(pbVar17 + 0xb8) = *(undefined4 *)(puVar14 + 0x17);
            uVar11 = *(uint *)(lVar24 + 0x1de68);
            if ((uVar19 & uVar11) != 0) {
              *pbVar17 = *pbVar17 & 0xfe;
              uVar11 = *(uint *)(lVar24 + 0x1de68);
            }
            *(uint *)(lVar24 + 0x1de68) = uVar11 | 1 << (uVar10 & 0x1f);
            cVar6 = FUN_1407699d0(pbVar17);
            if (cVar6 != '\0') {
              FUN_142f1cf10(pbVar20);
              *(uint *)(param_2 + 0x18e8) = *(uint *)(param_2 + 0x18e8) | 1 << (uVar15 & 0x1f);
            }
          }
        }
        cVar6 = (char)uStackX_10;
      }
      lVar24 = lStackX_20;
      if ((((2 < *puVar23) || (cVar8 = FUN_1405f1e0c(), cVar8 != '\0')) || (DAT_1451d2628 != -1)) &&
         (uVar10 = 1 << ((byte)uVar15 & 0x1f), (*(uint *)(param_2 + 8) & uVar10) != 0)) {
        if (cVar6 == '\0') {
          pbVar13 = (byte *)0x0;
          if ((*(uint *)(param_2 + 0x18e8) & uVar10) != 0) {
            pbVar13 = pbVar20;
          }
          FUN_140747520(uVar25 - 0x77,pbVar13,puVar23[-1]);
          lVar24 = lStackX_20;
        }
        else {
          cVar6 = FUN_142e2eac8(puVar1,lVar24);
          if (cVar6 == '\0') {
            uVar10 = *(uint *)(param_2 + 0x306c) & ~(1 << (uVar15 & 0x1f));
          }
          else {
            uVar10 = *(uint *)(param_2 + 0x306c) | 1 << (uVar15 & 0x1f);
          }
          *(uint *)(param_2 + 0x306c) = uVar10;
          uVar16 = uVar25 + 0x1c & 0xfffffffffffffffc;
          lVar24 = lStackX_20;
          if (((cStackX_18 != '\0') && (*(undefined4 *)(uVar16 + 4) = 0, cVar6 != '\0')) &&
             (cVar6 = FUN_14060f5fc(uVar25 & 0xfffffffffffffffc), lVar24 = lStackX_20, cVar6 != '\0'
             )) {
            *(undefined4 *)(uVar16 + 4) = *puStack_48;
          }
        }
      }
    }
    puStack_48 = puStack_48 + 0x19;
    lStackX_20 = lVar24 + 100;
    pbVar20 = pbVar20 + 0xbc;
    pbVar17 = pbVar17 + 0xbc;
    uVar15 = uVar15 + 1;
    uVar25 = uVar25 + 0xc0;
    puVar23 = puVar23 + 0x3e;
    pcVar21 = pcVar21 + 1;
  } while ((int)uVar15 < 0x20);
  return;
}

