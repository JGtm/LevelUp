
char * FUN_14080ada0(int param_1)

{
  char *pcVar1;
  
  pcVar1 = "unknown";
  if ((-1 < param_1) && (param_1 < *(int *)(DAT_144e61d88 + 0x208))) {
    pcVar1 = (char *)(**(code **)(**(longlong **)(DAT_144e61d88 + 0x210 + (longlong)param_1 * 8) + 8
                                 ))();
  }
  return pcVar1;
}

