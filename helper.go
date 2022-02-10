package httpc

import (
	"context"
	"io"
	"os"
	"os/exec"
)

type DisplayHelperOutputter struct {
	Helper   string
	Out, Err *os.File
}

func (o DisplayHelperOutputter) Output(ctx context.Context, contentType string, r io.Reader) error {
	cmd := exec.CommandContext(ctx, o.Helper, contentType)
	cmd.Stdout = o.Out
	cmd.Stderr = o.Err
	out, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	defer cmd.Wait()
	if _, err := io.Copy(out, r); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}
