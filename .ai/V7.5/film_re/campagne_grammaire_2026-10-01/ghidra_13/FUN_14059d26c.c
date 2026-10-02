
/* WARNING: Function: _alloca_probe replaced with injection: alloca_probe */
/* WARNING: Globals starting with '_' overlap smaller symbols at the same address */

void FUN_14059d26c(undefined8 param_1,undefined8 param_2,undefined2 *param_3)

{
  uint *puVar1;
  longlong lVar2;
  longlong lVar3;
  longlong lVar4;
  longlong lVar5;
  char cVar6;
  uint uVar7;
  longlong lVar8;
  undefined8 *puVar9;
  longlong lVar10;
  undefined1 auVar11 [16];
  undefined8 local_dc88;
  undefined8 uStack_dc80;
  int local_dc28;
  byte bStack_dc24;
  undefined1 auStack_9e60 [40];
  undefined1 auStack_9e38 [4];
  int iStack_9e34;
  undefined8 uStack_30;
  
  uStack_30 = 0x14059d295;
  DAT_144989d40 = DAT_144989d40 + 1;
  FUN_140b87bd0(&local_dc28);
  lVar3 = *(longlong *)ThreadLocalStoragePointer;
  *(undefined1 *)(*(longlong *)(lVar3 + 0x238) + 0x91) = 1;
  FUN_14051c1ec("game_tick","time %d",*(undefined4 *)(*(longlong *)(lVar3 + 0x6c0) + 0x40));
  FUN_140be2a70();
  memset(&local_dc28,0,0xdc00);
  local_dc88 = 0;
  uStack_dc80 = 0;
  FUN_1405d3bcc(DAT_144e61d78,1,&local_dc28,&local_dc88);
  if (((*(int *)(DAT_144e61d78 + 0x10) != 4) && (*(int *)(DAT_144e61d78 + 0x20) != 3)) &&
     ((*(int *)(DAT_144e61d78 + 0x10) - 3U & 0xfffffffd) == 0)) {
    FUN_1422d1c5c();
    return;
  }
  DAT_144e61e98 = &local_dc28;
  FUN_140ade854();
  FUN_14067d268(*(undefined8 *)(lVar3 + 0x418));
  lVar4 = *(longlong *)(lVar3 + 0x4c8);
  *(uint *)(lVar4 + 0x1040) = *(uint *)(lVar4 + 0x1040) | 1;
  puVar1 = (uint *)(lVar4 + 0x180e0);
  *puVar1 = *puVar1 | 1;
  FUN_1405f3b08(&local_dc28);
  lVar5 = DAT_144eae7c0;
  cVar6 = FUN_14048ee34();
  if (cVar6 != '\0') {
    FUN_142bd41c8();
    FUN_142bd3de4(&local_dc28);
    FUN_142c40acc();
  }
  if (DAT_144c2f828 != -1) {
    LOCK();
    UNLOCK();
    if (DAT_144c2fc4c == 1) {
      DAT_144c2f828 = -1;
    }
  }
  FUN_1405654c8(*(undefined8 *)(lVar5 + 8),*param_3,&DAT_144f8b610);
  lVar8 = FUN_1404f7f08();
  auVar11 = ZEXT816(0);
  lVar10 = 0;
  do {
    lVar2 = lVar10 * 4;
    lVar10 = lVar10 + 4;
    auVar11 = *(undefined1 (*) [16])(lVar8 + lVar2) | auVar11;
  } while (lVar10 < 0x20);
  auVar11 = auVar11 | auVar11 >> 0x40;
  if (auVar11._0_4_ == 0 && auVar11._4_4_ == 0) {
    if (((bStack_dc24 & 1) == 0) || ((&DAT_1451f88f8)[*(byte *)(lVar3 + 0x810)] == 0)) {
      FUN_14059dc7c(puVar1);
      *puVar1 = *puVar1 & 0xfffffffe;
      FUN_14059d7bc(lVar5,param_3[3]);
      FUN_14059d7bc(lVar5,param_3[1]);
      FUN_14059d7bc(lVar5,param_3[2]);
      FUN_14059d7bc(lVar5,param_3[4]);
      FUN_14059d7bc(lVar5,param_3[5]);
      FUN_14059d7bc(lVar5,param_3[7]);
      FUN_14059d7bc(lVar5,param_3[6]);
    }
    else {
      uVar7 = FUN_1405a70d4();
      if (_DAT_14533fa40 + 5000U < uVar7) {
        _DAT_14533fa40 = FUN_1405a70d4();
      }
      FUN_14059c7e0(param_3);
      FUN_14059d7bc(lVar5,param_3[3]);
      FUN_14059d7bc(lVar5,param_3[4]);
      FUN_14059d7bc(lVar5,param_3[5]);
      FUN_14059d7bc(lVar5,param_3[7]);
      FUN_14059d818();
      FUN_14051a864();
      FUN_140581f24(param_3);
    }
    FUN_14059d7bc(lVar5,param_3[8]);
    FUN_14059d7bc(lVar5,param_3[9]);
    FUN_14059d7bc(lVar5,param_3[10]);
    FUN_14059d7bc(lVar5,param_3[0xb]);
    FUN_140671ee4(lVar4);
    FUN_140a17be4(lVar4);
    FUN_1405f2e74(DAT_144e61d78,&local_dc28);
    if (0 < iStack_9e34) {
      for (puVar9 = (undefined8 *)FUN_141f86518(auStack_9e38); puVar9 != (undefined8 *)0x0;
          puVar9 = (undefined8 *)FUN_141f865a4(auStack_9e38,puVar9)) {
        FUN_141f86238(*puVar9);
      }
    }
    *(int *)(DAT_144e61d78 + 0x3c) = local_dc28 + 1;
    DAT_144e61e98 = (int *)0x0;
    FUN_1405654c8(*(undefined8 *)(lVar5 + 8),param_3[0xb],&DAT_144f8b610);
    FUN_140cdd41c();
    if ((bStack_dc24 & 1) != 0) {
      FUN_140599f80();
    }
    FUN_140bea558(auStack_9e60);
    FUN_140bea51c(auStack_9e38);
    FUN_1405cb464();
    FUN_14051c1ec("game_tick",0);
    return;
  }
  FUN_1422d1c9b();
  return;
}

