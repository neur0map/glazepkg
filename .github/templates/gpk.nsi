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
  SetRegView 64
  WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\gpk" "DisplayName" "gpk ${TAG}"
  WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\gpk" "UninstallString" '"$INSTDIR\uninstall.exe"'
  WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\gpk" "DisplayVersion" "${TAG}"
  WriteRegDWORD HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\gpk" "NoModify" 1
  WriteRegDWORD HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\gpk" "NoRepair" 1
  SetRegView lastused

  ; Add $INSTDIR to the system PATH (EnVar handles duplicates and long PATH)
  EnVar::SetHKLM
  EnVar::AddValue "PATH" "$INSTDIR"
  Pop $0
  SendMessage ${HWND_BROADCAST} ${WM_WININICHANGE} 0 "STR:Environment" /TIMEOUT=5000
SectionEnd

; ── Uninstall ─────────────────────────────────────────────────────
Section "Uninstall"
  ; Remove $INSTDIR from system PATH
  EnVar::SetHKLM
  EnVar::DeleteValue "PATH" "$INSTDIR"
  Pop $0
  SendMessage ${HWND_BROADCAST} ${WM_WININICHANGE} 0 "STR:Environment" /TIMEOUT=5000

  ; Delete installed files
  Delete "$INSTDIR\gpk.exe"
  Delete "$INSTDIR\uninstall.exe"

  ; Remove directory if empty
  RMDir "$INSTDIR"

  ; Delete registry keys
  SetRegView 64
  DeleteRegKey HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\gpk"
  SetRegView lastused
SectionEnd
