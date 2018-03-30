@echo off

set PATH=C:\Program Files\mingw-w64\x86_64-6.3.0-win32-seh-rt_v5-rev2\mingw64\bin;%PATH%
set CPATH=C:\Program Files (x86)\WinFsp\inc\fuse

mingw32-make %*
