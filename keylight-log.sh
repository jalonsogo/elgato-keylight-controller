#!/bin/bash

# Log file location
LOGFILE="/tmp/keylight-debug.log"

# Log timestamp and command
echo "===========================================" >> "$LOGFILE"
echo "Time: $(date)" >> "$LOGFILE"
echo "Full command line: /Users/javieralonso/elgato/keylight-go $@" >> "$LOGFILE"
echo "Number of args: $#" >> "$LOGFILE"
echo "---" >> "$LOGFILE"

# Run the actual command and capture output with verbose errors
# Use PIPESTATUS to get the real exit code
/Users/javieralonso/elgato/keylight-go "$@" 2>&1 | tee -a "$LOGFILE"
EXIT_CODE=${PIPESTATUS[0]}

# Log exit code
echo "Exit code: $EXIT_CODE" >> "$LOGFILE"
echo "" >> "$LOGFILE"

exit $EXIT_CODE
