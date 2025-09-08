savedcmd_sysinfo_so1.mod := printf '%s\n'   sysinfo_so1.o | awk '!x[$$0]++ { print("./"$$0) }' > sysinfo_so1.mod
