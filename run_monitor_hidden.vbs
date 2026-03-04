Dim shell
Set shell = CreateObject("WScript.Shell")
shell.Run "cmd /c cd /d ""C:\Users\Guillaume\Downloads\Scripts\LevelUp"" && "".venv\Scripts\python.exe"" scripts\monitor_uptime.py >> data\logs\monitor.log 2>&1", 0, False
Set shell = Nothing
