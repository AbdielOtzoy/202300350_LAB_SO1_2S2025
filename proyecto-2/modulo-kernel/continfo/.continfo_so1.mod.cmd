savedcmd_continfo_so1.mod := printf '%s\n'   continfo_so1.o | awk '!x[$$0]++ { print("./"$$0) }' > continfo_so1.mod
