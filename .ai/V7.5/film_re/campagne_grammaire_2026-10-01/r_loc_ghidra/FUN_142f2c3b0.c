
/* WARNING: Type propagation algorithm not settling */

void FUN_142f2c3b0(void)

{
  bool bVar1;
  char cVar2;
  char cVar3;
  int iVar4;
  longlong lVar5;
  longlong lVar6;
  undefined1 *puVar7;
  ulonglong uVar8;
  undefined1 *puVar9;
  int iVar10;
  ulonglong uVar11;
  longlong local_res10 [2];
  undefined1 local_2d8 [8];
  longlong local_2d0;
  undefined1 local_2c8 [656];
  
  uVar8 = 0;
  bVar1 = false;
  if (DAT_144eadd40 != '\0') {
    cVar2 = FUN_1409a621c(&DAT_144c23178);
    cVar3 = FUN_14076d018();
    if ((cVar3 != '\0') || (cVar3 = FUN_14076cffc(), cVar3 != '\0')) {
      bVar1 = true;
    }
    if (((cVar2 != '\0') || (bVar1)) && (lVar5 = FUN_140514044(), lVar5 != 0)) {
      local_res10[0] = 0;
      local_res10[1] = 4;
      do {
        cVar3 = FUN_1405d3b78(lVar5,local_res10 + 1,local_res10);
        if (cVar3 == '\0') {
          return;
        }
      } while (*(char *)(*(longlong *)(local_res10[0] + 0x10) + 0x18) == '\0');
      lVar5 = FUN_1406d096c(0x37000);
      local_res10[0] = lVar5;
      _eh_vector_constructor_iterator_
                (local_2c8,0xd8,3,FUN_14064c350,AK::MemoryMgr::GetCategoryStats);
      puVar9 = local_2c8;
      uVar11 = uVar8;
      do {
        iVar10 = (int)uVar11;
        puVar7 = local_2c8 + uVar11 * 0xd8;
        FUN_1411b149c(puVar7,*(int *)((longlong)&DAT_143d0fc80 + uVar8) + lVar5,
                      *(undefined4 *)((longlong)&DAT_143d0fc70 + uVar8));
        *(undefined4 *)(puVar9 + 0x1c) = 1;
        iVar4 = 1;
        FUN_1406d5cc0(puVar9);
        if (iVar10 == 0) {
          FUN_142f2c050(puVar7);
          FUN_1406d49c4(puVar7);
        }
        else if (iVar10 == iVar4) {
          FUN_142f2cc78(puVar7);
        }
        else if (iVar10 - iVar4 == iVar4) {
          lVar5 = 0;
          do {
            lVar6 = *(longlong *)(lVar5 + 0x480 + DAT_145178b58);
            if ((*(int *)(lVar6 + 0x20) - 1U < 2) && (iVar4 = FUN_14076b9b0(), 0 < (iVar4 + 7) / 8))
            {
              FUN_1406d5d14(puVar7,lVar6);
            }
            lVar5 = lVar5 + 0x4c8;
          } while (lVar5 < 0x9900);
          FUN_1406d49c4(puVar7);
          lVar5 = local_res10[0];
        }
        FUN_1406d6d94(puVar7);
        uVar11 = (ulonglong)(iVar10 + 1U);
        uVar8 = uVar8 + 4;
        puVar9 = puVar9 + 0xd8;
      } while ((int)(iVar10 + 1U) < 3);
      if ((cVar2 != '\0') && (FUN_1428e339c(&DAT_144c23178,local_2d8,local_2c8), local_2d0 != 0)) {
        FUN_1404f965c();
      }
      if (bVar1) {
        FUN_142f2bd98(&DAT_144de3ea0,local_2c8);
      }
      FUN_1406d0834(lVar5);
      _eh_vector_destructor_iterator_(local_2c8,0xd8,3,AK::MemoryMgr::GetCategoryStats);
    }
  }
  return;
}

