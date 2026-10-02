
void FUN_14076e494(undefined8 param_1,undefined8 *param_2)

{
  char cVar1;
  undefined1 local_res10 [24];
  longlong in_stack_00000030;
  undefined8 local_18;
  undefined4 local_10;
  
  cVar1 = FUN_14076f91c();
  if (cVar1 == '\0') {
    if (in_stack_00000030 == 0) {
      FUN_14076e524(&local_18,param_1,local_res10);
    }
    else {
      FUN_141f85880();
    }
  }
  else {
    FUN_1411b259c(&local_18,param_1);
  }
  *param_2 = local_18;
  *(undefined4 *)(param_2 + 1) = local_10;
  return;
}

