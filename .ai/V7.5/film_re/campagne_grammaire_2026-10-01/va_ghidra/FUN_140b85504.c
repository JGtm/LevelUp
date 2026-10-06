void FUN_140b85504(longlong param_1,undefined8 param_2)
{
  char cVar1;
  int iVar2;
  longlong lVar3;
  undefined ***local_res8;
  undefined1 local_2a8 [8];
  longlong local_2a0;
  longlong local_290;
  longlong local_280;
  longlong local_270;
  undefined **local_268;
  undefined8 local_260;
  undefined8 **local_258;
  undefined1 local_250;
  undefined8 **local_248;
  undefined8 local_240;
  undefined2 local_238;
  undefined4 local_230;
  undefined4 local_22c;
  undefined1 local_228 [256];
  undefined1 *local_128;
  undefined4 local_120;
  undefined4 local_11c;
  undefined1 local_118 [256];
  undefined1 *local_18;
  cVar1 = *(char *)(param_1 + 0xe94e0);
  if ((cVar1 == '\0') || ((cVar1 != '\x01' && (cVar1 != '\x02')))) {
    FUN_141015d54();
  }
  else {
    FUN_141015ce8();
  }
  local_2a0 = 0;
  local_290 = 0;
  local_280 = 0;
  local_270 = 0;
  lVar3 = FUN_1406aed80(param_1);
  iVar2 = *(int *)(lVar3 + 4);
  if (iVar2 == 0) {
    local_2a0 = FUN_140958628();
  }
  else if (iVar2 == 1) {
    local_280 = FUN_140958628();
  }
  else if (iVar2 == 2) {
    local_290 = FUN_140958628();
  }
  else if (iVar2 == 3) {
    local_270 = FUN_140958628();
  }
  local_268 = &PTR_FUN_143d45e08;
  local_res8 = &local_268;
  local_248 = &local_res8;
  local_240 = 0;
  local_238 = 2;
  local_230 = 0;
  local_22c = 0x40;
  local_128 = local_228;
  local_120 = 0;
  local_11c = 0x40;
  local_18 = local_118;
  local_258 = &local_248;
  local_250 = 0;
  local_260 = param_2;
  FUN_140b85798(&local_258,local_2a8);
  FUN_140dc8a80(&local_120);
  FUN_140dc8a80(&local_230);
  if (local_270 != 0) {
    FUN_140eb9820();
    FUN_1405a3950(local_270);
    local_270 = 0;
  }
  if (local_280 != 0) {
    FUN_140eb9820();
    FUN_1405a3950(local_280);
    local_280 = 0;
  }
  if (local_290 != 0) {
    FUN_140eb9820();
    FUN_1405a3950(local_290);
    local_290 = 0;
  }
  if (local_2a0 != 0) {
    FUN_140eb9820();
    FUN_1405a3950(local_2a0);
  }
  return;
}
