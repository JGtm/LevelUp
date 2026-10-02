
ulonglong FUN_1405f0adc(longlong param_1,uint param_2)

{
  ulonglong in_RAX;
  
  if (param_2 < 0x21) {
    return CONCAT71((int7)(int3)(param_2 >> 8),
                    *(undefined1 *)((longlong)(int)param_2 + 0x4a8 + param_1));
  }
  return in_RAX & 0xffffffffffffff00;
}

