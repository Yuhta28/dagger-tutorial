package dagger

import (
	"context"
	"path"

	"github.com/moby/buildkit/client/llb"
	bkgw "github.com/moby/buildkit/frontend/gateway/client"
	fstypes "github.com/tonistiigi/fsutil/types"
)

type Stat struct {
	*fstypes.Stat
}

type FS struct {
	// Before last solve
	input llb.State
	// After last solve
	output bkgw.Reference
	// How to produce the output
	s Solver
}

func (fs FS) Solver() Solver {
	return fs.s
}

// Compute output from input, if not done already.
//   This method uses a pointer receiver to simplify
//   calling it, since it is called in almost every
//   other method.
func (fs *FS) solve(ctx context.Context) error {
	if fs.output != nil {
		return nil
	}
	output, err := fs.s.Solve(ctx, fs.input)
	if err != nil {
		return err
	}
	fs.output = output
	return nil
}

func (fs FS) ReadFile(ctx context.Context, filename string) ([]byte, error) {
	// Lazy solve
	if err := (&fs).solve(ctx); err != nil {
		return nil, err
	}
	return fs.output.ReadFile(ctx, bkgw.ReadRequest{Filename: filename})
}

func (fs FS) ReadDir(ctx context.Context, dir string) ([]Stat, error) {
	// Lazy solve
	if err := (&fs).solve(ctx); err != nil {
		return nil, err
	}
	st, err := fs.output.ReadDir(ctx, bkgw.ReadDirRequest{
		Path: dir,
	})
	if err != nil {
		return nil, err
	}
	out := make([]Stat, len(st))
	for i := range st {
		out[i] = Stat{
			Stat: st[i],
		}
	}
	return out, nil
}

func (fs FS) walk(ctx context.Context, p string, fn WalkFunc) error {
	files, err := fs.ReadDir(ctx, p)
	if err != nil {
		return err
	}
	for _, f := range files {
		fPath := path.Join(p, f.GetPath())
		if err := fn(fPath, f); err != nil {
			return err
		}
		if f.IsDir() {
			if err := fs.walk(ctx, fPath, fn); err != nil {
				return err
			}
		}
	}
	return nil
}

type WalkFunc func(string, Stat) error

func (fs FS) Walk(ctx context.Context, fn WalkFunc) error {
	// Lazy solve
	if err := (&fs).solve(ctx); err != nil {
		return err
	}
	return fs.walk(ctx, "/", fn)
}

type ChangeFunc func(llb.State) llb.State

func (fs FS) Change(changes ...ChangeFunc) FS {
	for _, change := range changes {
		fs = fs.Set(change(fs.input))
	}
	return fs
}

func (fs FS) Set(st llb.State) FS {
	fs.input = st
	fs.output = nil
	return fs
}

func (fs FS) Solve(ctx context.Context) (FS, error) {
	if err := (&fs).solve(ctx); err != nil {
		return fs, err
	}
	return fs, nil
}

func (fs FS) LLB() llb.State {
	return fs.input
}

func (fs FS) Ref(ctx context.Context) (bkgw.Reference, error) {
	if err := (&fs).solve(ctx); err != nil {
		return nil, err
	}
	return fs.output, nil
}

func (fs FS) Result(ctx context.Context) (*bkgw.Result, error) {
	res := bkgw.NewResult()
	ref, err := fs.Ref(ctx)
	if err != nil {
		return nil, err
	}
	res.SetRef(ref)
	return res, nil
}