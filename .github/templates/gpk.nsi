!include "WinMessages.nsh"

Name "gpk ${TAG}"
OutFile "gpk-${TAG}-${ARCH}-setup.exe"
InstallDir "$PROGRAMFILES64\gpk"
RequestExecutionLevel admin

Page directory
Page instfiles

UninstPage uninstConfirm
UninstPage instfiles

; ── Install ───────────────────────────────────────────────────────
Section "Install gpk"
  SetOutPath "$INSTDIR"
  File "./bin/windows-${ARCH}/gpk.exe"

  ; Register uninstaller in Add / Remove Programs
  WriteUninstaller "$INSTDIR\uninstall.exe"
  WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\gpk" "DisplayName" "gpk ${TAG}" /reg:64
  WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\gpk" "UninstallString" '"$INSTDIR\uninstall.exe"' /reg:64
  WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\gpk" "DisplayVersion" "${TAG}" /reg:64
  WriteRegDWORD HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\gpk" "NoModify" 1 /reg:64
  WriteRegDWORD HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\gpk" "NoRepair" 1 /reg:64

  ; Add $INSTDIR to the system PATH if not already present
  ExecWait 'powershell -NoProfile -NonInteractive -Command ^
    $p = [Environment]::GetEnvironmentVariable("Path","Machine"); ^
    if ($p -notmatch [regex]::Escape("$INSTDIR")) { ^
      [Environment]::SetEnvironmentVariable("Path","$p;$INSTDIR","Machine") ^
    }'
  SendMessage ${HWND_BROADCAST} ${WM_WININICHANGE} 0 "STR:Environment" /TIMEOUT=5000
  Sleep 1000
SectionEnd

; ── Uninstall ─────────────────────────────────────────────────────
Section "Uninstall gpk"
  ; Remove $INSTDIR from system PATH
  ExecWait 'powershell -NoProfile -NonInteractive -Command ^
    $p = [Environment]::GetEnvironmentVariable("Path","Machine"); ^
    $entries = ($p -split ";") | Where-Object { $_ -ne "$INSTDIR" }; ^
    [Environment]::SetEnvironmentVariable("Path", ($entries -join ";"), "Machine")'
  SendMessage ${HWND_BROADCAST} ${WM_WININICHANGE} 0 "STR:Environment" /TIMEOUT=5000
  Sleep 1000

  ; Delete installed files
  Delete "$INSTDIR\gpk.exe"
  Delete "$INSTDIR\uninstall.exe"

  ; Remove directory if empty
  RMDir "$INSTDIR"

  ; Delete registry keys
  DeleteRegKey HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\gpk" /reg:64
SectionEnd
