package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/ingcr3at1on/glas"
)

// Wrap our functionality to allow defer to work with exit.
func _main() error {
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-sc
		cancel()
	}()

	inR, inW := io.Pipe()

	g, err := glas.New(&glas.Config{
		Input:  inR,
		Output: os.Stdout,
	})
	if err != nil {
		return err
	}

	var wg sync.WaitGroup
	errCh := make(chan error, 1)

	wg.Add(1)
	go func() {
		defer wg.Done()

		if err := g.Start(ctx, cancel); err != nil {
			errCh <- err
			return
		}
	}()

	// Don't put this in the waitgroup because it can and will continue running
	// until we stop it.
	go func() {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			nw, err := inW.Write(scanner.Bytes())
			if err != nil {
				errCh <- err
				return
			}

			if nw != len(scanner.Bytes()) {
				errCh <- io.ErrShortWrite
				return
			}
		}

		if err := scanner.Err(); err != nil {
			if err != io.EOF {
				errCh <- err
			}
		}
	}()

	select {
	case <-ctx.Done():
		break
	case err := <-errCh:
		if err != nil {
			return err
		}
	}

	wg.Wait()
	fmt.Println("exiting")
	return nil
}

func main() {
	if err := _main(); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}
