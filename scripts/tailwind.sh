#!/bin/bash
TMP_DIR="${1:-"./tmp"}"
LOG_FILE="${2:-"tailwind.log"}"
LINE_FILE="line"

LAST_LINES=0
CURRENT_LINES=0

if [[ -f "$TMP_DIR/$LINE_FILE" ]]; then
    LAST_LINES="$(cat "$TMP_DIR/$LINE_FILE")"
fi

if [[ -f "$TMP_DIR/$LOG_FILE" ]]; then
    CURRENT_LINES="$(wc -l < "$TMP_DIR/$LOG_FILE")"
    CURRENT_LINE="$(tail -n1 "$TMP_DIR/$LOG_FILE")"
fi

if [[ "$LAST_LINES" -gt "$CURRENT_LINES" ]]; then
    echo "tailwind.sh: Invalid line file"
    LAST_LINES=0
fi

if [[ "$LAST_LINES" -eq "$CURRENT_LINES" ]] || ! [[ "$CURRENT_LINE" == *"Done"* ]]; then
    echo "tailwind.sh: Waiting for file updates..."
    while read f; do
        if [[ "$f" == "$LOG_FILE" ]]; then
            lastline="$(tail -n1 "$TMP_DIR/$LOG_FILE")"
            echo "tailwind.sh: Last line is: ${lastline}"
            if [[ "$lastline" == *"Done"* ]]; then
                echo "tailwind.sh: Condition met. Stopping inotifywait."
                pkill -P $$ inotifywait
                break
            fi
        fi
    done < <(inotifywait -m -q -e create,modify --format %f "$TMP_DIR")
fi

echo "tailwind.sh: Done"
echo "$(wc -l < "$TMP_DIR/$LOG_FILE")" > "$TMP_DIR/$LINE_FILE"
