undefined8 FUN_140b3a118(undefined8 param_1,undefined8 param_2)
{
  char cVar1;
  int iVar2;
  longlong lVar3;
  undefined8 uVar4;
  undefined ***local_res10;
  longlong local_res18;
  undefined **local_18;
  undefined8 local_10;
  lVar3 = FUN_140689b88(0x40,0x52,"D:\\Enlistment\\shared\\engine\\includes\\i343\\STLAllocator.h",
                        0x108);
  if (lVar3 == 0) {
    lVar3 = 0;
  }
  else {
    *(undefined8 *)(lVar3 + 8) = 0;
    *(undefined8 *)(lVar3 + 0x18) = 0;
    *(undefined8 *)(lVar3 + 0x28) = 0;
    *(undefined8 *)(lVar3 + 0x38) = 0;
  }
  local_18 = &PTR_FUN_143686718;
  local_res10 = &local_18;
  local_res18 = lVar3;
  local_10 = param_2;
  FUN_140b3a21c(&local_res10,lVar3);
  iVar2 = FUN_1405f6254();
  if ((((iVar2 == 3) || (iVar2 = FUN_1405f6254(), iVar2 == 4)) && (*(longlong *)(lVar3 + 8) != 0))
     && (cVar1 = FUN_1408f50c8(), cVar1 == '\0')) {
    FUN_140b3a1f8(&local_res18);
    uVar4 = 0;
  }
  else {
    FUN_140958494(param_1,lVar3);
    FUN_141166c78(param_1);
    if (lVar3 != 0) {
      FUN_1409585cc(lVar3);
      FUN_1405a3950(lVar3);
    }
    uVar4 = 1;
  }
  return uVar4;
}
