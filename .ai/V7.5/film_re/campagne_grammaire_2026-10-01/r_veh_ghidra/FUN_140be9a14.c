
/* WARNING: Globals starting with '_' overlap smaller symbols at the same address */

void FUN_140be9a14(void)

{
  undefined8 uVar1;
  undefined8 uVar2;
  longlong lVar3;
  char cVar4;
  int iVar5;
  longlong lVar6;
  uint uVar7;
  undefined *puVar8;
  undefined8 *puVar9;
  longlong lVar10;
  int iVar11;
  
  lVar3 = DAT_144976b60;
  iVar5 = 0;
  uVar7 = 0;
  iVar11 = *(int *)(DAT_144976b60 + 0x7bc);
  if (0 < iVar11) {
    lVar6 = 0;
    puVar9 = &DAT_14462cbe0;
    puVar8 = &DAT_1445cc9b0;
    do {
      lVar10 = *(longlong *)(lVar3 + 0x7ac) + lVar6;
      cVar4 = FUN_140be9d1c(lVar10 + 0x44);
      if (cVar4 != '\0') {
        iVar11 = 0;
        *(uint *)(puVar8 + (ulonglong)(uVar7 >> 5) * 4 + 0x1b0) =
             *(uint *)(puVar8 + (ulonglong)(uVar7 >> 5) * 4 + 0x1b0) | 1 << ((byte)uVar7 & 0x1f);
        uVar2 = *(undefined8 *)(lVar10 + 0x4c);
        uVar1 = *(undefined8 *)(lVar10 + 0x54);
        *puVar9 = *(undefined8 *)(lVar10 + 0x44);
        puVar9[1] = uVar2;
        puVar9[2] = uVar1;
        do {
          FUN_140be9b88(iVar11,lVar10 + 0x44);
          iVar11 = iVar11 + 1;
        } while (iVar11 < 0x20);
        puVar8 = &DAT_1445cc9b0;
        FUN_140be9cc0(&DAT_1445cc9b0,&DAT_1445cc9b0 + ((longlong)(int)uVar7 * 3 + 0xc046) * 8);
      }
      iVar11 = *(int *)(lVar3 + 0x7bc);
      uVar7 = uVar7 + 1;
      lVar6 = lVar6 + 0xdc;
      puVar9 = puVar9 + 3;
    } while ((int)uVar7 < iVar11);
  }
  if (iVar11 == 1) {
    DAT_144632be0 = 1;
  }
  else {
    DAT_144632be0 = FUN_1406d310c();
  }
  DAT_1445cc9c8 = _DAT_143b8c6b8;
  DAT_1445cc9cc = _UNK_143b8c6bc;
  DAT_1445cc9d0 = _UNK_143b8c6c0;
  DAT_1445cc9d4 = _UNK_143b8c6c4;
  _DAT_1445cc9d8 = DAT_143b8c6c8;
  do {
    FUN_140be9b88(iVar5,&DAT_1445cc9c8);
    iVar5 = iVar5 + 1;
  } while (iVar5 < 0x20);
  return;
}

