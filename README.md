# ws_checkers
Checkers WebSocket server for playing games online in real-time.

The current implementation may have goroutine leaks, deadlocks etc. in some specific situations.
In particular, a game with no activity just stays in memory.
It's good enough for the current purposes, though.
