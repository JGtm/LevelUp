
ulonglong FUN_140bbd1a4(longlong param_1,int *param_2,int *param_3)

{
  int iVar1;
  longlong lVar2;
  bool bVar3;
  char cVar4;
  int iVar5;
  ulonglong uVar6;
  longlong lVar7;
  uint uVar8;
  int local_res10 [2];
  undefined8 local_48;
  undefined8 uStack_40;
  undefined8 local_38;
  undefined8 uStack_30;
  undefined8 local_28;
  
  uVar8 = 0;
  *param_3 = 0;
  if (param_2[1] == 0) {
    uVar8 = FUN_140bbd474();
  }
  else {
    local_48 = *(undefined8 *)param_2;
    uStack_40 = *(undefined8 *)(param_2 + 2);
    bVar3 = true;
    local_38 = *(undefined8 *)(param_2 + 4);
    uStack_30 = *(undefined8 *)(param_2 + 6);
    lVar7 = *(longlong *)(*(longlong *)(param_1 + 0x28) + 0x130 + (longlong)*param_2 * 8);
    local_28 = *(undefined8 *)(param_2 + 8);
    iVar1 = *(int *)(lVar7 + 8);
    do {
      local_res10[0] = 0;
      iVar5 = FUN_140bbd474(param_1,&local_48,local_res10);
      uVar8 = uVar8 + iVar5;
      *param_3 = *param_3 + local_res10[0];
      if (iVar5 < 1) break;
      local_28 = CONCAT44(local_28._4_4_ - iVar5,(undefined4)local_28);
      lVar2 = *(longlong *)(lVar7 + 0x40);
      if ((lVar2 != 0) && (lVar7 = lVar2, *(int *)(lVar2 + 8) != iVar1)) {
        uVar6 = FUN_1423be8dc();
        return uVar6;
      }
      if (((*(int *)(lVar7 + 8) == iVar1) &&
          (uVar6 = 1L << ((byte)*(undefined4 *)(param_1 + 0x14) & 0x3f),
          (*(ulonglong *)(lVar7 + 0x30) & uVar6) != 0)) &&
         ((*(ulonglong *)(lVar7 + 0x38) & uVar6) == 0)) {
        local_48 = CONCAT44(local_48._4_4_,*(undefined4 *)(lVar7 + 0x48));
        cVar4 = FUN_1408f0074(*(undefined8 *)(*(longlong *)(param_1 + 0x28) + 8),lVar7,uStack_40,0);
        if (cVar4 == '\0') goto LAB_140bbd259;
      }
      else {
LAB_140bbd259:
        bVar3 = false;
      }
    } while (bVar3);
  }
  return (ulonglong)uVar8;
}

