package main

import (
	"context"
	"os/exec"
	"strings"
	"time"
)

func SpotifyGetCurrentPlaying() (block I3ProtocolBlock, err error) {
	// Create a new context and add a timeout to it
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	// Create the command with our context
	cmd := exec.CommandContext(ctx, "spotifyctl", "get")

	out, err := cmd.Output()

	// We want to check the context error to see if the timeout was executed.
	// The error returned by cmd.Output() will be OS specific based on what
	// happens when a process is killed.
	if ctx.Err() == context.DeadlineExceeded {
		err = ctx.Err()
		return
	}

	block.FullText = strings.TrimSpace(string(out))
	return
}
