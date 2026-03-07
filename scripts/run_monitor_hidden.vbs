Dim shell, fso, scriptDir, repoRoot
Set shell = CreateObject("WScript.Shell")
Set fso = CreateObject("Scripting.FileSystemObject")
scriptDir = fso.GetParentFolderName(WScript.ScriptFullName)
repoRoot = fso.GetParentFolderName(scriptDir)
shell.Run "cmd /c cd /d """ & repoRoot & """ && "".venv\Scripts\python.exe"" scripts\monitor_uptime.py >> data\logs\monitor.log 2>&1", 0, False
Set shell = Nothing
Set fso = Nothing
