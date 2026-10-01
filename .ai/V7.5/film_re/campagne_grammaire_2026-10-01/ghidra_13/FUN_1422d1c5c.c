
/* WARNING: Globals starting with '_' overlap smaller symbols at the same address */

void FUN_1422d1c5c(void)

{
  uint *puVar1;
  longlong lVar2;
  longlong lVar3;
  longlong lVar4;
  char cVar5;
  uint uVar6;
  longlong lVar7;
  undefined8 *puVar8;
  longlong lVar9;
  undefined2 *unaff_RBX;
  longlong unaff_RBP;
  longlong unaff_RSI;
  longlong unaff_RDI;
  undefined8 uVar10;
  undefined1 auVar11 [16];
  
  uVar10 = FUN_141f85f0c(*(longlong *)(unaff_RDI + 8) + 0x42d0);
  FUN_141f85a84(uVar10,unaff_RBP + -0x78,unaff_RBP + -0x70);
  DAT_144e61e98 = unaff_RBP + -0x80;
  FUN_140ade854();
  FUN_14067d268(*(undefined8 *)(unaff_RSI + 0x418));
  lVar3 = *(longlong *)(unaff_RSI + 0x4c8);
  *(uint *)(lVar3 + 0x1040) = *(uint *)(lVar3 + 0x1040) | 1;
  puVar1 = (uint *)(lVar3 + 0x180e0);
  *puVar1 = *puVar1 | 1;
  FUN_1405f3b08(unaff_RBP + -0x80);
  lVar4 = DAT_144eae7c0;
  cVar5 = FUN_14048ee34();
  if (cVar5 != '\0') {
    FUN_142bd41c8();
    FUN_142bd3de4(unaff_RBP + -0x80);
    FUN_142c40acc();
  }
  if (DAT_144c2f828 != -1) {
    LOCK();
    UNLOCK();
    if (DAT_144c2fc4c == 1) {
      DAT_144c2f828 = -1;
    }
  }
  FUN_1405654c8(*(undefined8 *)(lVar4 + 8),*unaff_RBX,&DAT_144f8b610);
  lVar7 = FUN_1404f7f08();
  auVar11 = ZEXT816(0);
  lVar9 = 0;
  do {
    lVar2 = lVar9 * 4;
    lVar9 = lVar9 + 4;
    auVar11 = *(undefined1 (*) [16])(lVar7 + lVar2) | auVar11;
  } while (lVar9 < 0x20);
  auVar11 = auVar11 | auVar11 >> 0x40;
  if (auVar11._0_4_ == 0 && auVar11._4_4_ == 0) {
    if (((*(byte *)(unaff_RBP + -0x7c) & 1) == 0) ||
       ((&DAT_1451f88f8)[*(byte *)(unaff_RSI + 0x810)] == 0)) {
      FUN_14059dc7c(puVar1);
      *puVar1 = *puVar1 & 0xfffffffe;
      FUN_14059d7bc(lVar4,unaff_RBX[3]);
      FUN_14059d7bc(lVar4,unaff_RBX[1]);
      FUN_14059d7bc(lVar4,unaff_RBX[2]);
      FUN_14059d7bc(lVar4,unaff_RBX[4]);
      FUN_14059d7bc(lVar4,unaff_RBX[5]);
      FUN_14059d7bc(lVar4,unaff_RBX[7]);
      FUN_14059d7bc(lVar4,unaff_RBX[6]);
    }
    else {
      uVar6 = FUN_1405a70d4();
      if (_DAT_14533fa40 + 5000U < uVar6) {
        _DAT_14533fa40 = FUN_1405a70d4();
      }
      FUN_14059c7e0();
      FUN_14059d7bc(lVar4,unaff_RBX[3]);
      FUN_14059d7bc(lVar4,unaff_RBX[4]);
      FUN_14059d7bc(lVar4,unaff_RBX[5]);
      FUN_14059d7bc(lVar4,unaff_RBX[7]);
      FUN_14059d818();
      FUN_14051a864();
      FUN_140581f24();
    }
    FUN_14059d7bc(lVar4,unaff_RBX[8]);
    FUN_14059d7bc(lVar4,unaff_RBX[9]);
    FUN_14059d7bc(lVar4,unaff_RBX[10]);
    FUN_14059d7bc(lVar4,unaff_RBX[0xb]);
    FUN_140671ee4(lVar3);
    FUN_140a17be4(lVar3);
    FUN_1405f2e74(DAT_144e61d78,unaff_RBP + -0x80);
    if (0 < *(int *)(unaff_RBP + 0x3d74)) {
      for (puVar8 = (undefined8 *)FUN_141f86518(unaff_RBP + 0x3d70); puVar8 != (undefined8 *)0x0;
          puVar8 = (undefined8 *)FUN_141f865a4(unaff_RBP + 0x3d70,puVar8)) {
        FUN_141f86238(*puVar8);
      }
    }
    *(int *)(DAT_144e61d78 + 0x3c) = *(int *)(unaff_RBP + -0x80) + 1;
    DAT_144e61e98 = 0;
    FUN_1405654c8(*(undefined8 *)(lVar4 + 8),unaff_RBX[0xb],&DAT_144f8b610);
    FUN_140cdd41c();
    if ((*(byte *)(unaff_RBP + -0x7c) & 1) != 0) {
      FUN_140599f80();
    }
    FUN_140bea558(unaff_RBP + 0x3d48);
    FUN_140bea51c(unaff_RBP + 0x3d70);
    FUN_1405cb464();
    FUN_14051c1ec("game_tick",0);
    return;
  }
  FUN_1422d1c9b();
  return;
}

