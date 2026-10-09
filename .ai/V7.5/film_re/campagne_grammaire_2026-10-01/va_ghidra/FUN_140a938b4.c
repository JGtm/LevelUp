void FUN_140a938b4(void)
{
  longlong *plVar1;
  int *piVar2;
  longlong lVar3;
  undefined8 *puVar4;
  bool bVar5;
  longlong *plVar6;
  char cVar7;
  int iVar8;
  undefined4 uVar9;
  undefined8 uVar10;
  longlong lVar11;
  longlong lVar12;
  undefined4 *puVar13;
  uint uVar14;
  longlong lVar15;
  void *pvVar16;
  undefined4 local_res8 [2];
  undefined4 local_res10 [2];
  undefined4 local_res18 [2];
  undefined4 local_res20 [2];
  undefined4 local_138;
  undefined4 local_134;
  undefined4 local_130 [2];
  longlong local_128;
  longlong *local_120;
  undefined8 local_118;
  longlong *local_110;
  longlong local_108;
  int local_100;
  undefined4 *local_f8 [2];
  undefined1 local_e8 [16];
  undefined1 local_d8 [48];
  undefined1 local_a8 [112];
  lVar3 = *(longlong *)ThreadLocalStoragePointer;
  memset(*(void **)(lVar3 + 0x220),0,0x1e434);
  puVar4 = *(undefined8 **)(lVar3 + 0x518);
  *puVar4 = 0;
  puVar4[1] = 0;
  puVar4[2] = 0;
  puVar4[3] = 0;
  *(undefined4 *)(puVar4 + 4) = 0;
  FUN_140a93ec8();
  uVar10 = FUN_1404f2650();
  lVar11 = FUN_1406aed80(uVar10);
  iVar8 = FUN_14051a4b8(*(undefined4 *)(lVar11 + 4));
  if (iVar8 < 1) goto LAB_140a93def;
  lVar11 = FUN_1406aed80(uVar10);
  iVar8 = FUN_14051a4b8(*(undefined4 *)(lVar11 + 4));
  if (3 < iVar8) goto LAB_140a93def;
  lVar11 = FUN_1406aed80(uVar10);
  uVar9 = FUN_14051a4b8(*(undefined4 *)(lVar11 + 4));
  FUN_140a93ec8(uVar9);
  *(undefined4 *)(*(longlong *)(lVar3 + 0x220) + 0x2c) = 0xffffffff;
  local_res10[0] = 0xffffffff;
  FUN_1404d5f44(*(longlong *)(lVar3 + 0x220) + 0x34,*(longlong *)(lVar3 + 0x220) + 0xf4,local_res10)
  ;
  *(undefined2 *)(*(longlong *)(lVar3 + 0x220) + 0x272) = 0x2a;
  local_res18[0] = 0xffffffff;
  FUN_1404d5f44(*(longlong *)(lVar3 + 0x220) + 0xf4,*(longlong *)(lVar3 + 0x220) + 0x174,local_res18
               );
  lVar11 = *(longlong *)(lVar3 + 0x220);
  lVar12 = lVar11 + 0x174;
  if (lVar11 + 0x174 != lVar11 + 0x1c4) {
    do {
      local_res20[0] = 0xffffffff;
      lVar15 = lVar12 + 0x14;
      FUN_1404d5f44(lVar12,lVar15,local_res20);
      lVar12 = lVar15;
    } while (lVar15 != lVar11 + 0x1c4);
    lVar11 = *(longlong *)(lVar3 + 0x220);
  }
  local_138 = 0xffffffff;
  FUN_1404d5f44(lVar11 + 0x1c4,lVar11 + 0x1d8,&local_138);
  local_134 = 0xffffffff;
  FUN_1404d5f44(*(longlong *)(lVar3 + 0x220) + 0x1d8,*(longlong *)(lVar3 + 0x220) + 0x25c,&local_134
               );
  *(undefined4 *)(*(longlong *)(lVar3 + 0x220) + 0x25c) = 0xffffffff;
  *(undefined4 *)(*(longlong *)(lVar3 + 0x220) + 0x260) = 0xffffffff;
  local_res8[0] = CONCAT31(local_res8[0]._1_3_,0xff);
  FUN_140a93fe0(*(longlong *)(lVar3 + 0x220) + 0x1c,*(longlong *)(lVar3 + 0x220) + 0x24,local_res8);
  lVar11 = FUN_1406aed80(uVar10);
  uVar14 = **(uint **)(lVar3 + 0x220);
  if (*(char *)(lVar11 + 0x10bc) == '\0') {
    uVar14 = uVar14 & 0xfffffffd;
  }
  else {
    uVar14 = uVar14 | 2;
  }
  **(uint **)(lVar3 + 0x220) = uVar14;
  LOCK();
  *(undefined4 *)(*(longlong *)(lVar3 + 0x220) + 0x1e430) = 0;
  UNLOCK();
  lVar11 = 0;
  do {
    lVar12 = FUN_1406aed80(uVar10);
    *(undefined1 *)(lVar11 + 0x24 + *(longlong *)(lVar3 + 0x220)) = *(undefined1 *)(lVar12 + 0x293);
    lVar11 = lVar11 + 1;
  } while (lVar11 < 7);
  cVar7 = FUN_1404f293c();
  if (cVar7 != '\0') {
    **(uint **)(lVar3 + 0x220) = **(uint **)(lVar3 + 0x220) | 1;
  }
  lVar11 = FUN_1404f1614();
  FUN_141e13850();
  lVar12 = FUN_14056ca14();
  FUN_14095c5b8(lVar12);
  local_130[0] = *(undefined4 *)(lVar11 + 0xeaa10);
  cVar7 = FUN_1405838f0(local_130);
  if (cVar7 == '\0') {
    FUN_14061e3c4(lVar12,local_f8);
    local_res8[0] = DAT_1445bbba0;
    puVar13 = (undefined4 *)FUN_1405a5114(local_res8);
    *local_f8[0] = *puVar13;
    FUN_14061e2f4(local_e8);
LAB_140a93c8e:
    plVar6 = DAT_1452f2f08;
    if (DAT_1452f2f08 != (longlong *)0x0) {
      lVar11 = *(longlong *)(lVar3 + 0x760);
      local_100 = *(int *)(lVar11 + 4);
      local_108 = lVar11;
      while (lVar12 = local_108, local_100 != -1) {
        lVar15 = (longlong)local_100 * 0x1abc;
        *(undefined4 *)(lVar15 + 0x18 + local_108) = 0;
        *(undefined4 *)(lVar15 + 0x1ac4 + local_108) = 0;
        FUN_141024854(local_108 + 0x1c + lVar15);
        FUN_141024870(lVar12 + 0xdb0 + lVar15);
        FUN_140873400(&local_108);
      }
      FUN_140a93f88(lVar11);
      FUN_140ad186c();
      cVar7 = (**(code **)(*plVar6 + 0x10))(plVar6);
      if (cVar7 == '\0') {
        FUN_140a93ec8(0);
        if (*(char *)(lVar3 + 200) == '\0') {
          __dyn_tls_on_demand_init();
        }
        FUN_140896f84(*(undefined8 *)(lVar3 + 0x2d8));
      }
      else {
        if (*(char *)(lVar3 + 200) == '\0') {
          __dyn_tls_on_demand_init();
        }
        FUN_140896f84(*(undefined8 *)(lVar3 + 0x2d8));
        cVar7 = FUN_1406aebc0();
        if (cVar7 != '\0') {
          *(undefined2 *)(*(longlong *)(lVar3 + 0x220) + 6) = 0x1ff;
          *(undefined2 *)(*(longlong *)(lVar3 + 0x220) + 8) = 0xff;
        }
      }
      if ((*(char *)(DAT_144c1cfa8 + 0x10c) != '\0') &&
         ((*(char *)(DAT_144c1cfa8 + 0x10d) != '\0' ||
          ((cVar7 = FUN_1405f1e0c(), cVar7 != '\0' &&
           (*(int *)(*(longlong *)(lVar3 + 0x238) + 0x88) != -1)))))) {
        FUN_142c8c99c();
      }
    }
  }
  else {
    FUN_14095c5b8();
    bVar5 = false;
    cVar7 = FUN_1404f2b4c();
    if (cVar7 == '\0') {
LAB_140a93baf:
      cVar7 = FUN_140bce628();
      if ((cVar7 == '\0') && (cVar7 = FUN_142caa528(lVar11), cVar7 == '\0')) {
        FUN_140a95654(&local_118);
        FUN_140a963a4(local_a8,local_118);
        FUN_140bce6f8(lVar12);
        FUN_140a969c4(local_a8);
        plVar6 = local_110;
        if (local_110 != (longlong *)0x0) {
          LOCK();
          plVar1 = local_110 + 1;
          lVar11 = *plVar1;
          *(int *)plVar1 = (int)*plVar1 + -1;
          UNLOCK();
          if ((int)lVar11 == 1) {
            (**(code **)*local_110)(local_110);
            LOCK();
            piVar2 = (int *)((longlong)plVar6 + 0xc);
            iVar8 = *piVar2;
            *piVar2 = *piVar2 + -1;
            UNLOCK();
            if (iVar8 == 1) {
              (**(code **)(*local_110 + 8))();
            }
          }
        }
      }
    }
    else {
      FUN_141cb8374(&DAT_144c23178,&local_128);
      if (local_128 != 0) {
        FUN_140a969e4(local_d8,local_128 + 0x90);
        cVar7 = FUN_140bce6f8(lVar12);
        FUN_140a96a7c(local_d8);
        if (cVar7 != '\0') {
          FUN_141cb831c(&DAT_144c23178);
          FUN_140789aa4(lVar12 + 8);
          bVar5 = true;
        }
      }
      plVar6 = local_120;
      if (local_120 != (longlong *)0x0) {
        LOCK();
        plVar1 = local_120 + 1;
        lVar15 = *plVar1;
        *(int *)plVar1 = (int)*plVar1 + -1;
        UNLOCK();
        if ((int)lVar15 == 1) {
          (**(code **)*local_120)(local_120);
          LOCK();
          piVar2 = (int *)((longlong)plVar6 + 0xc);
          iVar8 = *piVar2;
          *piVar2 = *piVar2 + -1;
          UNLOCK();
          if (iVar8 == 1) {
            (**(code **)(*local_120 + 8))();
          }
        }
      }
      if (!bVar5) goto LAB_140a93baf;
    }
    plVar6 = DAT_1452f2f08;
    if (DAT_1452f2f08 != (longlong *)0x0) {
      (**(code **)(*DAT_1452f2f08 + 0x48))(DAT_1452f2f08);
      (**(code **)(*plVar6 + 0x40))(plVar6);
      goto LAB_140a93c8e;
    }
  }
  *(undefined1 *)(DAT_144ebd098 + 0x59d09) = 1;
LAB_140a93def:
  pvVar16 = (void *)(*(longlong *)(lVar3 + 0x5d8) + 0x13U & 0xfffffffffffffffc);
  memset(pvVar16,0,0xe04);
  *(undefined4 *)((longlong)pvVar16 + 0xe04) = 0xffffffff;
  *(undefined4 *)((longlong)pvVar16 + 0xe08) = 1;
  *(undefined4 *)((longlong)pvVar16 + 0xe0c) = 0x440;
  FUN_1410db788(pvVar16);
  pvVar16 = (void *)FUN_1407d3000();
  memset(pvVar16,0,0x1910);
  lVar11 = FUN_1407d3000();
  *(undefined4 *)(lVar11 + 0x1904) = 0xffffffff;
  *(undefined4 *)(lVar11 + 0x1908) = 1;
  *(undefined4 *)(lVar11 + 0x190c) = 0x40;
  FUN_14061db14(lVar11);
  *(undefined4 *)(*(longlong *)(lVar3 + 0x220) + 0x1e428) = 0;
  *(undefined4 *)(*(longlong *)(lVar3 + 0x220) + 0x1e42c) = 0;
  uVar10 = FUN_1404f2650();
  lVar11 = FUN_1406aed80(uVar10);
  *(undefined2 *)(*(longlong *)(lVar3 + 0x238) + 0xa0) = *(undefined2 *)(lVar11 + 0xe21a6);
  FUN_140a93f58(*(longlong *)(lVar3 + 0x238) + 0xa0);
  return;
}
