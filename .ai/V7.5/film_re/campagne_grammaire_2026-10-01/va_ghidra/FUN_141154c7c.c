undefined * FUN_141154c7c(void)
{
  longlong local_20 [3];
  local_20[0] = 0;
  local_20[1] = 0;
  local_20[0] = FUN_14046aed0(0x60);
  *(longlong *)local_20[0] = local_20[0];
  *(longlong *)(local_20[0] + 8) = local_20[0];
  *(longlong *)(local_20[0] + 0x10) = local_20[0];
  *(undefined2 *)(local_20[0] + 0x18) = 0x101;
  FUN_140a36ebc(&DAT_144a07260,"PlaybackSettings","i343.NetProtocol.GameOptions.PlaybackSettings",
                local_20);
  FUN_140464c80(local_20,local_20,*(undefined8 *)(local_20[0] + 8));
  FUN_1405a3288(local_20[0],0x60);
  return &DAT_144a07260;
}
