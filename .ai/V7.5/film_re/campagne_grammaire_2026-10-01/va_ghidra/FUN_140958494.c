longlong FUN_140958494(longlong param_1,longlong param_2)
{
  longlong lVar1;
  lVar1 = *(longlong *)(param_2 + 8);
  if (lVar1 == 0) {
    lVar1 = *(longlong *)(param_2 + 0x18);
    if (lVar1 == 0) {
      lVar1 = *(longlong *)(param_2 + 0x28);
      if (lVar1 == 0) {
        lVar1 = *(longlong *)(param_2 + 0x38);
        if (lVar1 != 0) {
          FUN_1411b2ae4();
          FUN_140959400(param_1,lVar1);
          *(undefined1 *)(param_1 + 0xe94e0) = 3;
        }
      }
      else {
        FUN_1411b2ae4();
        FUN_140959380(param_1,lVar1);
        *(undefined1 *)(param_1 + 0xe94e0) = 1;
      }
    }
    else {
      FUN_1411b2ae4();
      FUN_140959380(param_1,lVar1);
      *(undefined1 *)(param_1 + 0xe94e0) = 2;
    }
  }
  else {
    FUN_1411b2ae4();
    FUN_140959400(param_1,lVar1);
    *(undefined1 *)(param_1 + 0xe94e0) = 0;
  }
  return param_1;
}
