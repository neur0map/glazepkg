OutFile "gpk-${TAG}-${ARCH}-setup.exe"
InstallDir $PROGRAMFILES\gpk
Page directory
Page instfiles
Section "gpk"
  SetOutPath $INSTDIR
  File "./bin/windows-${ARCH}/gpk.exe"
SectionEnd
